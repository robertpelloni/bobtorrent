package ingest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIngestGameAsset(t *testing.T) {
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.obj")
	outputDir := filepath.Join(tempDir, "output")

	// Create a dummy 5MB game asset
	data := make([]byte, 5*1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	if err := os.WriteFile(sourceFile, data, 0644); err != nil {
		t.Fatalf("failed to create source asset: %v", err)
	}

	// Ingest the asset with 1MB chunk size
	pieceSize := 1024 * 1024
	manifest, err := IngestGameAsset(sourceFile, outputDir, "model", pieceSize)
	if err != nil {
		t.Fatalf("IngestGameAsset failed: %v", err)
	}

	if manifest.OriginalName != "source.obj" {
		t.Errorf("expected OriginalName 'source.obj', got %s", manifest.OriginalName)
	}

	if manifest.AssetType != "model" {
		t.Errorf("expected AssetType 'model', got %s", manifest.AssetType)
	}

	if manifest.TotalSize != int64(len(data)) {
		t.Errorf("expected TotalSize %d, got %d", len(data), manifest.TotalSize)
	}

	if len(manifest.Pieces) != 5 {
		t.Errorf("expected 5 pieces, got %d", len(manifest.Pieces))
	}

	if len(manifest.EncryptionKey) != 64 { // 32 bytes hex encoded
		t.Errorf("expected 64 char hex encryption key, got %d chars", len(manifest.EncryptionKey))
	}

	// Verify the chunks were actually written to disk
	for _, pieceHash := range manifest.Pieces {
		chunkPath := filepath.Join(outputDir, pieceHash)
		if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
			t.Errorf("expected chunk file %s does not exist on disk", chunkPath)
		}
	}
}
