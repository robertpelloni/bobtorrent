package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Manifest represents the structure of a BobTorrent v3 detached-key file descriptor.
type Manifest struct {
	OriginalFilename string `json:"originalFilename"`
	FileSize         int64  `json:"fileSize"`
	MimeType         string `json:"mimeType"`
	Chunks           []Chunk `json:"chunks"`
}

// Chunk describes a specific encrypted blob, its order in the file, and its AES-256 decryption key.
type Chunk struct {
	BlobID string `json:"blobId"`
	Order  int    `json:"order"`
	KeyHex string `json:"keyHex"` // Detached AES key
	Size   int64  `json:"size"`
}

// ParseManifest reads a JSON manifest file into the Manifest struct.
func ParseManifest(filePath string) (*Manifest, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest: %w", err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(bytes, &m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest json: %w", err)
	}

	return &m, nil
}
