package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bobtorrent/bobtorrent/pkg/storage"
)

const chunkSize = 1024 * 1024 // 1MB chunks

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(32 << 20) // 32MB max memory
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	manifest := &storage.Manifest{
		OriginalFilename: header.Filename,
		FileSize:         header.Size,
		MimeType:         header.Header.Get("Content-Type"),
		Chunks:           make([]storage.Chunk, 0),
	}

	buffer := make([]byte, chunkSize)
	order := 0

	for {
		bytesRead, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			http.Error(w, "Failed to read file", http.StatusInternalServerError)
			return
		}
		if bytesRead == 0 {
			break
		}

		chunkData := buffer[:bytesRead]

		// Generate random 32-byte AES key
		key := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			http.Error(w, "Failed to generate key", http.StatusInternalServerError)
			return
		}

		// Encrypt
		ciphertext, blobID, err := storage.EncryptBlob(chunkData, key)
		if err != nil {
			http.Error(w, "Failed to encrypt blob", http.StatusInternalServerError)
			return
		}

		// Save encrypted blob to disk
		blobPath := filepath.Join(s.DataDir, blobID)
		if err := os.WriteFile(blobPath, ciphertext, 0644); err != nil {
			http.Error(w, "Failed to save blob", http.StatusInternalServerError)
			return
		}

		// Add to manifest
		manifest.Chunks = append(manifest.Chunks, storage.Chunk{
			BlobID: blobID,
			Order:  order,
			KeyHex: hex.EncodeToString(key),
			Size:   int64(bytesRead),
		})
		order++

		// Announce blob to DHT
		go s.Engine.AnnounceBlob(blobID)
	}

	// Save Manifest
	manifestID := strings.ReplaceAll(header.Filename, " ", "_")
	manifestPath := filepath.Join(s.DataDir, "manifests", manifestID+".json")

	// Ensure manifests dir exists
	os.MkdirAll(filepath.Join(s.DataDir, "manifests"), 0755)

	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")
	os.WriteFile(manifestPath, manifestBytes, 0644)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"fileId":  manifestID,
		"chunks":  len(manifest.Chunks),
	})
}
