package streaming

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"io"
	"strconv"
	"fmt"

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

    // Explicit Range handling to bypass ServeContent blocking bugs during tests
	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		w.Header().Set("Content-Type", manifest.MimeType)
		w.Header().Set("Accept-Ranges", "bytes")

		var start, end int64
		fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
		if end == 0 {
		    end = manifest.FileSize - 1
		}

		contentLength := end - start + 1
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, manifest.FileSize))
		w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
		w.WriteHeader(http.StatusPartialContent)

		buffer.Seek(start, io.SeekStart)

		// Copy strictly the requested length to prevent unexpected EOF panic loops on test assertions
		io.CopyN(w, buffer, contentLength)
		return
	}

	w.Header().Set("Content-Type", manifest.MimeType)
	http.ServeContent(w, r, manifest.OriginalFilename, time.Now(), buffer)
}
