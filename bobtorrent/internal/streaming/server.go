package streaming

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/bobtorrent/bobtorrent/pkg/storage"
)

type Server struct {
	DataDir string
	Store   *storage.BlobStore
}

func NewServer(dataDir string, store *storage.BlobStore) *Server {
	return &Server{
		DataDir: dataDir,
		Store:   store,
	}
}

func (s *Server) StreamHandler(w http.ResponseWriter, r *http.Request) {
	fileID := strings.TrimPrefix(r.URL.Path, "/api/stream/")
	fileID = filepath.Clean(fileID)
	if strings.Contains(fileID, "..") {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}

	manifestPath := filepath.Join(s.DataDir, "manifests", fileID+".json")
	log.Printf("Streaming request for manifest: %s (Range: %s)", manifestPath, r.Header.Get("Range"))

	manifest, err := storage.ParseManifest(manifestPath)
	if err != nil {
		log.Printf("Manifest not found or invalid: %v", err)
		http.Error(w, "Manifest not found", http.StatusNotFound)
		return
	}

	buffer := NewReadaheadBuffer(manifest, s.Store)
	defer buffer.Close()

	buffer.StartPrefetch()

	w.Header().Set("Content-Type", manifest.MimeType)
	w.Header().Set("Accept-Ranges", "bytes")

	http.ServeContent(w, r, manifest.OriginalFilename, time.Now(), buffer)
}
