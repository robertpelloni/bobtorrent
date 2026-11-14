package main

import (
	"encoding/json"
	"net/http"
	"time"
)

type BlockInfo struct {
	Hash      string    `json:"hash"`
	Height    int64     `json:"height"`
	Producer  string    `json:"producer"`
	Timestamp time.Time `json:"timestamp"`
}

// handleLattice serves REAL confirmed blocks by proxying lattice-go's
// GET /blocks (ordered confirmed chain). It previously fabricated five
// synthetic blocks (heights 1024000-i, hashed from the current time) on
// every call, which made the dashboard ledger look alive when the chain
// was empty or lattice-go was down. Now: unreachable/empty upstream
// yields an honest empty list, never invented blocks.
func handleLattice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	blocks := make([]BlockInfo, 0)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(latticeURL + "/blocks?limit=25")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var upstream struct {
				Blocks []struct {
					Hash      string `json:"hash"`
					Account   string `json:"account"`
					Height    int    `json:"height"`
					Timestamp int64  `json:"timestamp"` // milliseconds
				} `json:"blocks"`
			}
			if json.NewDecoder(resp.Body).Decode(&upstream) == nil {
				for _, b := range upstream.Blocks {
					blocks = append(blocks, BlockInfo{
						Hash:      b.Hash,
						Height:    int64(b.Height),
						Producer:  b.Account,
						Timestamp: time.UnixMilli(b.Timestamp),
					})
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(blocks)
}
