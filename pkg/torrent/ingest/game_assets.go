package ingest

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// AssetManifest represents the output of a game engine asset ingestion.
type AssetManifest struct {
	OriginalName string   `json:"original_name"`
	AssetType    string   `json:"asset_type"` // e.g., "texture", "model"
	TotalSize    int64    `json:"total_size"`
	PieceLength  int      `json:"piece_length"`
	Pieces       []string `json:"pieces"`     // SHA-256 hashes of encrypted pieces
	EncryptionKey string  `json:"encryption_key"` // Detached AES-256-GCM key
}

// IngestGameAsset processes a large game asset file, slicing it into AES-256-GCM encrypted chunks,
// and returns a manifest ready for publishing to the swarm.
func IngestGameAsset(sourceFile string, outputDir string, assetType string, pieceSize int) (*AssetManifest, error) {
	file, err := os.Open(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open source asset: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat source asset: %w", err)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate a 256-bit (32 byte) AES key for the asset
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	manifest := &AssetManifest{
		OriginalName: filepath.Base(sourceFile),
		AssetType:    assetType,
		TotalSize:    stat.Size(),
		PieceLength:  pieceSize,
		EncryptionKey: hex.EncodeToString(key),
	}

	buffer := make([]byte, pieceSize)
	pieceIndex := 0

	for {
		n, err := io.ReadFull(file, buffer)
		if n > 0 {
			chunkData := buffer[:n]

			// Generate a unique nonce for this chunk
			nonce := make([]byte, gcm.NonceSize())
			if _, randErr := io.ReadFull(rand.Reader, nonce); randErr != nil {
				return nil, fmt.Errorf("failed to generate nonce for piece %d: %w", pieceIndex, randErr)
			}

			// Encrypt chunk
			encryptedChunk := gcm.Seal(nonce, nonce, chunkData, nil)

			// Calculate Piece Hash
			hash := sha256.New()
			hash.Write(encryptedChunk)
			pieceHash := hex.EncodeToString(hash.Sum(nil))

			manifest.Pieces = append(manifest.Pieces, pieceHash)

			// Write encrypted chunk to disk
			chunkPath := filepath.Join(outputDir, pieceHash)
			if err := os.WriteFile(chunkPath, encryptedChunk, 0644); err != nil {
				return nil, fmt.Errorf("failed to write encrypted piece %d: %w", pieceIndex, err)
			}

			pieceIndex++
		}

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading source asset: %w", err)
		}
	}

	return manifest, nil
}
