package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type BlockInfo struct {
	Hash      string    `json:"hash"`
	Height    int64     `json:"height"`
	Producer  string    `json:"producer"`
	Timestamp time.Time `json:"timestamp"`
}

// handleLattice returns a mocked synchronized lattice state to fulfill the UI integration.
// In Phase 8+, this will be wired directly into the Bobcoin consensus DAG.
func (s *Server) handleLattice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Generate 5 mock blocks descending in height from current time
	var blocks []BlockInfo
	now := time.Now()

	// Get the public key to simulate us being the producer of at least one recent block
	producer := "Unknown"
	if s.Wallet != nil {
		producer = s.Wallet.GetPublicKey()
	}

	for i := 0; i < 5; i++ {
		h := sha256.New()
		h.Write([]byte(fmt.Sprintf("block_data_%d_%s", i, now.String())))
		hashStr := hex.EncodeToString(h.Sum(nil))

		blocks = append(blocks, BlockInfo{
			Hash:      hashStr,
			Height:    int64(1024000 - i),
			Producer:  producer,
			Timestamp: now.Add(-time.Duration(i*30) * time.Second),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blocks)
}
