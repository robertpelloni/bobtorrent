package downloader

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// Verifier handles the verification of downloaded file pieces against expected hashes.
type Verifier struct {
	PieceLength int64
	FileHash    string
	PieceHashes []string
}

// NewVerifier creates a new Verifier.
func NewVerifier(pieceLength int64, fileHash string, pieceHashes []string) *Verifier {
	return &Verifier{
		PieceLength: pieceLength,
		FileHash:    fileHash,
		PieceHashes: pieceHashes,
	}
}

// VerifyFile checks the entire file against the expected file hash.
func (v *Verifier) VerifyFile(filepath string) (bool, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return false, fmt.Errorf("failed to open file for verification: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false, fmt.Errorf("failed to read file for hashing: %w", err)
	}

	calculatedHash := hex.EncodeToString(hash.Sum(nil))
	return calculatedHash == v.FileHash, nil
}

// VerifyPiece checks a specific piece of the file against its expected hash.
func (v *Verifier) VerifyPiece(filepath string, pieceIndex int) (bool, error) {
	if pieceIndex < 0 || pieceIndex >= len(v.PieceHashes) {
		return false, fmt.Errorf("invalid piece index: %d", pieceIndex)
	}

	file, err := os.Open(filepath)
	if err != nil {
		return false, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	offset := int64(pieceIndex) * v.PieceLength
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return false, fmt.Errorf("failed to seek to piece offset: %w", err)
	}

	pieceData := make([]byte, v.PieceLength)
	n, err := io.ReadFull(file, pieceData)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return false, fmt.Errorf("failed to read piece data: %w", err)
	}

	// Truncate to actual read length in case of the last piece
	pieceData = pieceData[:n]

	hash := sha256.New()
	hash.Write(pieceData)
	calculatedHash := hex.EncodeToString(hash.Sum(nil))

	return calculatedHash == v.PieceHashes[pieceIndex], nil
}
