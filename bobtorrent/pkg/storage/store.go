package storage

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anacrolix/torrent"
	"github.com/bobtorrent/bobtorrent/pkg/dht"
)

type BlobStore struct {
	DataDir string
	Client  *torrent.Client
}

func NewBlobStore(dataDir string, client *torrent.Client) *BlobStore {
	return &BlobStore{
		DataDir: dataDir,
		Client:  client,
	}
}

// FetchAndDecrypt Blob handles the core logic of taking a BobTorrent Chunk descriptor,
// finding the BitTorrent InfoHash, downloading the encrypted payload via the torrent client,
// and decrypting it using the Chunk's detached AES key.
func (s *BlobStore) FetchAndDecryptBlob(chunk Chunk) ([]byte, error) {
	// 1. Map 32-byte BlobID to 20-byte InfoHash
	infoHashHex, err := dht.MapBlobIDToInfoHash(chunk.BlobID)
	if err != nil {
		return nil, fmt.Errorf("failed to map blob ID to infohash: %w", err)
	}

	// Currently stubbed. Here we would:
	// s.Client.AddTorrentInfoHash(...)
	// <-t.GotInfo()
	// t.DownloadAll()
	// Wait for piece completion...

	// For demonstration, we attempt to read directly from a local stub file if it exists,
	// simulating the completed download.
	encryptedPath := filepath.Join(s.DataDir, infoHashHex)
	encryptedData, err := os.ReadFile(encryptedPath)
	if err != nil {
		return nil, fmt.Errorf("blob %s (infohash: %s) not yet available: %w", chunk.BlobID, infoHashHex, err)
	}

	// 2. Parse the AES Key
	keyBytes, err := hex.DecodeString(chunk.KeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid chunk key hex: %w", err)
	}

	// 3. Decrypt
	plaintext, err := DecryptBlob(encryptedData, keyBytes)
	if err != nil {
		return nil, fmt.Errorf("decryption failed for blob %s: %w", chunk.BlobID, err)
	}

	return plaintext, nil
}
