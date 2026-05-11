package storage

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	ChunkSize       = 1024 * 1024 // 1MB
	KeySize         = chacha20poly1305.KeySize
	NonceSize       = chacha20poly1305.NonceSize
	AuthTagSize     = 16
	FixedBlobSize   = ChunkSize + AuthTagSize
	DefaultData     = 4
	DefaultParity   = 2
)

// Storage handles the ingestion and retrieval of files with encryption and erasure coding.
type Storage struct {
	outputDir string
	coder     *ErasureCoder
}

// NewStorage creates a new Storage instance with the specified output directory and coding parameters.
func NewStorage(outputDir string, dataShards, parityShards int) (*Storage, error) {
	// The WASM bridge uses an in-memory-only Storage instance and passes an
	// empty outputDir. In that mode we skip on-disk directory creation.
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	coder, err := NewErasureCoder(dataShards, parityShards)
	if err != nil {
		return nil, err
	}

	return &Storage{
		outputDir: outputDir,
		coder:     coder,
	}, nil
}

// Ingest reads an input file, chunks it, encrypts each chunk, applies erasure coding, and saves the blobs.
func (s *Storage) Ingest(inputPath string) (*FileEntry, error) {
	file, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file stats: %w", err)
	}

	entry := &FileEntry{
		Name: filepath.Base(inputPath),
		Size: stat.Size(),
	}

	buffer := make([]byte, ChunkSize)
	for {
		n, err := file.Read(buffer)
		if n > 0 {
			chunkData := buffer[:n]
			
			// Encrypt the chunk
			blob, key, nonce, err := s.encryptChunk(chunkData)
			if err != nil {
				return nil, fmt.Errorf("encryption failed: %w", err)
			}

			// Apply Erasure Coding (optional for individual chunks, usually applied to a block of blobs)
			// For now, we apply it to each chunk as per Supernode Java's streaming parity logic
			shards, err := s.coder.Encode(blob)
			if err != nil {
				return nil, fmt.Errorf("erasure encoding failed: %w", err)
			}

			chunkMeta := ChunkMeta{
				Key:      hex.EncodeToString(key),
				Nonce:    hex.EncodeToString(nonce),
				RealSize: int64(n),
			}

			// Save shards as individual blobs
			for i, shard := range shards {
				blobID := s.sha256(shard)
				blobPath := filepath.Join(s.outputDir, blobID)
				if err := os.WriteFile(blobPath, shard, 0644); err != nil {
					return nil, fmt.Errorf("failed to write blob %d: %w", i, err)
				}
				chunkMeta.BlobIDs = append(chunkMeta.BlobIDs, blobID)
			}

			entry.Chunks = append(entry.Chunks, chunkMeta)
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("file read error: %w", err)
		}
	}

	return entry, nil
}

// Reassemble retrieves blobs, reconstructs shards if necessary, decrypts, and writes the original file.
func (s *Storage) Reassemble(entry *FileEntry, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	for _, chunkMeta := range entry.Chunks {
		shards := make([][]byte, s.coder.dataShards+s.coder.parityShards)
		presentCount := 0

		for i, blobID := range chunkMeta.BlobIDs {
			blobPath := filepath.Join(s.outputDir, blobID)
			shard, err := os.ReadFile(blobPath)
			if err == nil {
				shards[i] = shard
				presentCount++
			}
		}

		if presentCount < s.coder.dataShards {
			return fmt.Errorf("insufficient shards for chunk: got %d, need %d", presentCount, s.coder.dataShards)
		}

		// Reconstruct if shards are missing
		if presentCount < len(chunkMeta.BlobIDs) {
			if err := s.coder.Reconstruct(shards); err != nil {
				return fmt.Errorf("failed to reconstruct chunk: %w", err)
			}
		}

		// Join shards back into the encrypted blob
		blob, err := s.coder.Join(shards, FixedBlobSize)
		if err != nil {
			return fmt.Errorf("failed to join shards: %w", err)
		}

		// Decrypt the blob
		key, _ := hex.DecodeString(chunkMeta.Key)
		nonce, _ := hex.DecodeString(chunkMeta.Nonce)
		plaintext, err := s.decryptChunk(blob, key, nonce)
		if err != nil {
			return fmt.Errorf("decryption failed: %w", err)
		}

		// Write only the original data (unpad)
		if _, err := file.Write(plaintext[:chunkMeta.RealSize]); err != nil {
			return fmt.Errorf("file write failed: %w", err)
		}
	}

	return nil
}

// EncryptChunk encrypts a single data chunk and returns the ciphertext, key, and nonce.
func (s *Storage) EncryptChunk(data []byte) (ciphertext, key, nonce []byte, err error) {
	return s.encryptChunk(data)
}

// DecryptChunk decrypts a ciphertext chunk using the provided key and nonce.
func (s *Storage) DecryptChunk(ciphertext, key, nonce []byte) ([]byte, error) {
	return s.decryptChunk(ciphertext, key, nonce)
}

func (s *Storage) encryptChunk(data []byte) (ciphertext, key, nonce []byte, err error) {
	key = make([]byte, KeySize)
	nonce = make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, nil, nil, err
	}
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, nil, err
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, nil, nil, err
	}

	// Pad plaintext to exactly ChunkSize
	padded := make([]byte, ChunkSize)
	copy(padded, data)
	if len(data) < ChunkSize {
		// Secure padding with random data
		if _, err := io.ReadFull(rand.Reader, padded[len(data):]); err != nil {
			return nil, nil, nil, err
		}
	}

	ciphertext = aead.Seal(nil, nonce, padded, nil)
	return ciphertext, key, nonce, nil
}

func (s *Storage) decryptChunk(ciphertext, key, nonce []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (s *Storage) sha256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

type FileEntry struct {
	Name   string      `json:"name"`
	Size   int64       `json:"size"`
	Chunks []ChunkMeta `json:"chunks"`
}

type ChunkMeta struct {
	BlobIDs  []string `json:"blob_ids"`
	Key      string   `json:"key"`
	Nonce    string   `json:"nonce"`
	RealSize int64    `json:"real_size"`
}
