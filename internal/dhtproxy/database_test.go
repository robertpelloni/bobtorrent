package dhtproxy

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

// Empty peer lookups must serialize as [] not null so API consumers
// can range over the result without a nil check.
func TestGetPeersEmptyReturnsNonNilSlice(t *testing.T) {
	db, err := NewDatabase(filepath.Join(t.TempDir(), "dht.db"))
	if err != nil {
		t.Fatalf("NewDatabase failed: %v", err)
	}
	defer db.Close()

	peers, err := db.GetPeers("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 50, 0, 0)
	if err != nil {
		t.Fatalf("GetPeers failed: %v", err)
	}
	if peers == nil {
		t.Fatal("expected non-nil empty slice, got nil (JSON null)")
	}
	encoded, err := json.Marshal(peers)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(encoded) != "[]" {
		t.Fatalf("expected [], got %s", encoded)
	}
}
