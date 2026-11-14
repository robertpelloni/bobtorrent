package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// /lattice must serve REAL blocks from lattice-go, never fabricated ones.
func TestHandleLatticeProxiesRealBlocks(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/blocks" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"blocks": []map[string]interface{}{
				{"hash": "abc123", "account": "producer1", "height": 7, "timestamp": int64(1700000000000)},
			},
			"count": 1,
		})
	}))
	defer upstream.Close()

	old := latticeURL
	latticeURL = upstream.URL
	defer func() { latticeURL = old }()

	rec := httptest.NewRecorder()
	handleLattice(rec, httptest.NewRequest(http.MethodGet, "/lattice", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var blocks []BlockInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &blocks); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(blocks) != 1 || blocks[0].Hash != "abc123" || blocks[0].Producer != "producer1" || blocks[0].Height != 7 {
		t.Fatalf("expected the upstream block mapped through, got %+v", blocks)
	}
}

func TestHandleLatticeEmptyWhenLatticeDown(t *testing.T) {
	old := latticeURL
	latticeURL = "http://127.0.0.1:1" // nothing listens there
	defer func() { latticeURL = old }()

	rec := httptest.NewRecorder()
	handleLattice(rec, httptest.NewRequest(http.MethodGet, "/lattice", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with empty list, got %d", rec.Code)
	}
	var blocks []BlockInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &blocks); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(blocks) != 0 {
		t.Fatalf("expected zero fabricated blocks, got %+v", blocks)
	}
}
