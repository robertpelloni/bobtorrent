package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
)

func buildUploadedTorrentFromMultipartWithFile(file *os.File, originalName string, dataDir string) (*uploadedTorrentDescriptor, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("torrent data directory required")
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create torrent data directory: %w", err)
	}

	if originalName == "" || originalName == "." || originalName == string(filepath.Separator) {
		originalName = "upload.bin"
	}
	storedName := fmt.Sprintf("upload_%d_%s", time.Now().UnixNano(), originalName)
	storedPath := filepath.Join(dataDir, storedName)

	out, err := os.Create(storedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create upload destination: %w", err)
	}
	size, copyErr := io.Copy(out, file)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(storedPath)
		return nil, fmt.Errorf("failed to persist uploaded file: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(storedPath)
		return nil, fmt.Errorf("failed to finalize uploaded file: %w", closeErr)
	}

	info := metainfo.Info{PieceLength: metainfo.ChoosePieceLength(size)}
	if err := info.BuildFromFilePath(storedPath); err != nil {
		_ = os.Remove(storedPath)
		return nil, fmt.Errorf("failed to build torrent metadata: %w", err)
	}
	infoBytes, err := bencode.Marshal(info)
	if err != nil {
		_ = os.Remove(storedPath)
		return nil, fmt.Errorf("failed to encode torrent info: %w", err)
	}

	mi := &metainfo.MetaInfo{
		CreationDate: time.Now().Unix(),
		CreatedBy:    "Bobtorrent Go Supernode",
		InfoBytes:    infoBytes,
	}
	mi.SetDefaults()
	infoHash := mi.HashInfoBytes()
	magnet := mi.Magnet(&infoHash, &info).String()

	return &uploadedTorrentDescriptor{
		MetaInfo:     mi,
		InfoHash:     infoHash.HexString(),
		Magnet:       magnet,
		Size:         size,
		StoredPath:   storedPath,
		StoredName:   storedName,
		OriginalName: originalName,
	}, nil
}

func handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FilePath string `json:"filePath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.FilePath == "" {
		http.Error(w, "Missing filePath", http.StatusBadRequest)
		return
	}

	// SECURITY FIX: Prevent arbitrary file reads by ensuring the path is relative and clean.
	// Since this is a shim, we only allow ingesting files from a specific upload/temp directory.
	// Or better yet, we just require the client to use standard multipart form upload (handled by /upload)
	// but the web UI uses `filePath` JSON. So we sanitize.

	cleanPath := filepath.Clean(req.FilePath)
	if strings.Contains(cleanPath, "..") || filepath.IsAbs(cleanPath) {
		http.Error(w, "Invalid file path. Absolute paths or directory traversal are not allowed.", http.StatusBadRequest)
		return
	}

	// We resolve it relative to the current working directory,
	// but further restrict it to a specific directory if possible. For now, preventing traversal and absolute is a start.
	// A better approach is prepending a known directory to cleanPath.
	// Let's assume we allow files in `data/ingest/` relative to pwd.

	ingestDir := filepath.Join("data", "ingest")
	if err := os.MkdirAll(ingestDir, 0o755); err != nil {
		http.Error(w, "Failed to initialize ingest directory", http.StatusInternalServerError)
		return
	}

	fullPath := filepath.Join(ingestDir, cleanPath)

	file, err := os.Open(fullPath)
	if err != nil {
		http.Error(w, "File not found in ingest directory", http.StatusNotFound)
		return
	}
	defer file.Close()

	originalName := filepath.Base(fullPath)

	uploaded, err := buildUploadedTorrentFromMultipartWithFile(file, originalName, torrentDataDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := registerUploadedTorrent(uploaded); err != nil {
		_ = os.Remove(uploaded.StoredPath)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"fileEntry": map[string]interface{}{
			"id":   uploaded.InfoHash,
			"name": uploaded.OriginalName,
			"size": uploaded.Size,
			"type": "application/octet-stream",
		},
		"blobCount": 1,
	})
}
