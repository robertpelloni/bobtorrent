package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"bobtorrent/pkg/torrent/ingest"
)

// handleIngestGameAsset handles the upload of large game textures/models.
func handleIngestGameAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 100MB max memory limit for the multipart parser, remainder streams to temp files
	err := r.ParseMultipartForm(100 << 20)
	if err != nil {
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("asset")
	if err != nil {
		http.Error(w, "Missing 'asset' field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	assetType := r.FormValue("asset_type")
	if assetType == "" {
		assetType = "unknown"
	}

	// Create a temporary file to hold the upload
	tempDir := filepath.Join("data", "ingest", "temp")
	os.MkdirAll(tempDir, 0755)

	// Randomize filename to prevent concurrent upload collisions
	tempFile, err := os.CreateTemp(tempDir, "upload-*")
	if err != nil {
		log.Printf("Failed to create temp file: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // Ensure cleanup happens even if we panic or return early
	defer tempFile.Close()

	// Stream the upload to disk
	if _, err := io.Copy(tempFile, file); err != nil {
		log.Printf("Failed to save uploaded file: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	tempFile.Sync()
	tempFile.Close() // Close before ingestion so IngestGameAsset can open it

	// Run the game asset ingestion pipeline (using 1MB chunks)
	outputDir := filepath.Join("data", "blobs")
	manifest, err := ingest.IngestGameAsset(tempPath, outputDir, assetType, 1024*1024)
	if err != nil {
		log.Printf("Asset ingestion failed: %v", err)
		http.Error(w, "Asset ingestion pipeline failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manifest)
}
