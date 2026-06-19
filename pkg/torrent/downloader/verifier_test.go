package downloader

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifier(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "testfile.txt")
	fileData := []byte("Hello, Bobtorrent! This is a test file for the verifier.")
	if err := os.WriteFile(filePath, fileData, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Calculate full file hash
	hash := sha256.New()
	hash.Write(fileData)
	fileHash := hex.EncodeToString(hash.Sum(nil))

	// Define piece size
	pieceLength := int64(10)
	var pieceHashes []string

	// Calculate piece hashes
	file, _ := os.Open(filePath)
	defer file.Close()
	for {
		piece := make([]byte, pieceLength)
		n, err := io.ReadFull(file, piece)
		if n > 0 {
			h := sha256.New()
			h.Write(piece[:n])
			pieceHashes = append(pieceHashes, hex.EncodeToString(h.Sum(nil)))
		}
		if err != nil {
			break
		}
	}

	verifier := NewVerifier(pieceLength, fileHash, pieceHashes)

	// Test VerifyFile
	valid, err := verifier.VerifyFile(filePath)
	if err != nil {
		t.Errorf("VerifyFile failed with error: %v", err)
	}
	if !valid {
		t.Error("VerifyFile returned false for valid file")
	}

	// Test VerifyPiece for all pieces
	for i := range pieceHashes {
		valid, err := verifier.VerifyPiece(filePath, i)
		if err != nil {
			t.Errorf("VerifyPiece failed for piece %d: %v", i, err)
		}
		if !valid {
			t.Errorf("VerifyPiece returned false for piece %d", i)
		}
	}

	// Test Invalid Piece Index
	_, err = verifier.VerifyPiece(filePath, -1)
	if err == nil {
		t.Error("VerifyPiece expected error for invalid index -1, got nil")
	}

	_, err = verifier.VerifyPiece(filePath, len(pieceHashes))
	if err == nil {
		t.Error("VerifyPiece expected error for out-of-bounds index, got nil")
	}

	// Test corrupted file
	corruptedPath := filepath.Join(tempDir, "corrupted.txt")
	corruptedData := []byte("Xello, Bobtorrent! This is a corrupted file.")
	os.WriteFile(corruptedPath, corruptedData, 0644)

	valid, _ = verifier.VerifyFile(corruptedPath)
	if valid {
		t.Error("VerifyFile returned true for corrupted file")
	}

	valid, _ = verifier.VerifyPiece(corruptedPath, 0)
	if valid {
		t.Error("VerifyPiece returned true for corrupted piece")
	}
}
