package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

type BlobInfo struct {
	ID        string `json:"id"`
	Size      int64  `json:"size"`
	Timestamp int64  `json:"timestamp"`
}

func (s *Server) handleBlobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var blobs []BlobInfo

	// Read all files in the data directory
	entries, err := os.ReadDir(s.DataDir)
	if err != nil {
		http.Error(w, "Failed to read storage directory", http.StatusInternalServerError)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Filter out internal application files. Blobs are mapped InfoHashes (40 chars)
		// but let's just filter out known files for now
		if name == "identity.json" || name == "wallet.json" || strings.HasSuffix(name, ".db") || strings.HasPrefix(name, ".torrent") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		blobs = append(blobs, BlobInfo{
			ID:        name,
			Size:      info.Size(),
			Timestamp: info.ModTime().UnixMilli(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"blobs": blobs,
	})
}
