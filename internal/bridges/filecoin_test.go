package bridges

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilecoinBridgePublishDealFallsBackWhenLotusUnavailable(t *testing.T) {
	t.Setenv("BOBTORRENT_FILECOIN_RPC_URL", "")
	t.Setenv("BOBTORRENT_FILECOIN_WALLET", "")
	t.Setenv("BOBTORRENT_FILECOIN_MINER", "")
	recordsPath := filepath.Join(t.TempDir(), "deals.json")
	t.Setenv("BOBTORRENT_FILECOIN_RECORDS", recordsPath)

	bridge := NewFilecoinBridge("f1fallback")
	dealID, err := bridge.PublishDeal("bafy-test-root", 1024, 30)
	if err != nil {
		t.Fatalf("PublishDeal fallback failed: %v", err)
	}
	if !strings.HasPrefix(dealID, "sim-f0") {
		t.Fatalf("expected simulated deal prefix, got %s", dealID)
	}

	status := bridge.Status()
	if status.Mode != "fallback-simulated" || status.DealCount != 1 {
		t.Fatalf("unexpected fallback status: %#v", status)
	}

	records := bridge.ListDeals()
	if len(records) != 1 {
		t.Fatalf("expected one persisted deal record, got %d", len(records))
	}
	if records[0].State != "simulated" {
		t.Fatalf("unexpected fallback record: %#v", records[0])
	}

	active, err := bridge.VerifyStorage(dealID)
	if err != nil {
		t.Fatalf("VerifyStorage fallback failed: %v", err)
	}
	if !active {
		t.Fatal("expected simulated verification to remain active")
	}
}

func TestFilecoinBridgePublishAndVerifyViaLotusRPC(t *testing.T) {
	var observedMethods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode lotus request: %v", err)
		}
		method, _ := req["method"].(string)
		observedMethods = append(observedMethods, method)

		w.Header().Set("Content-Type", "application/json")
		switch method {
		case "Filecoin.ClientStartDeal":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result":  "f01234",
			})
		case "Filecoin.StateMarketStorageDeal":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result": map[string]interface{}{
					"State": map[string]interface{}{
						"SectorStartEpoch": 123,
						"SlashEpoch":       -1,
					},
				},
			})
		default:
			t.Fatalf("unexpected lotus method: %s", method)
		}
	}))
	defer server.Close()

	t.Setenv("BOBTORRENT_FILECOIN_RPC_URL", server.URL)
	t.Setenv("BOBTORRENT_FILECOIN_WALLET", "f1wallet")
	t.Setenv("BOBTORRENT_FILECOIN_MINER", "f012miner")
	recordsPath := filepath.Join(t.TempDir(), "deals.json")
	t.Setenv("BOBTORRENT_FILECOIN_RECORDS", recordsPath)

	bridge := NewFilecoinBridge("f1node")
	dealID, err := bridge.PublishDeal("bafy-real-root", 2048, 45)
	if err != nil {
		t.Fatalf("PublishDeal lotus failed: %v", err)
	}
	if dealID != "f01234" {
		t.Fatalf("unexpected lotus deal id: %s", dealID)
	}

	active, err := bridge.VerifyStorage(dealID)
	if err != nil {
		t.Fatalf("VerifyStorage lotus failed: %v", err)
	}
	if !active {
		t.Fatal("expected lotus deal verification to report active")
	}

	if len(observedMethods) != 2 || observedMethods[0] != "Filecoin.ClientStartDeal" || observedMethods[1] != "Filecoin.StateMarketStorageDeal" {
		t.Fatalf("unexpected lotus methods: %#v", observedMethods)
	}

	records := bridge.ListDeals()
	if len(records) != 1 {
		t.Fatalf("expected one lotus deal record, got %d", len(records))
	}
	if records[0].Mode != "lotus-rpc" || records[0].State != "active" {
		t.Fatalf("unexpected lotus deal record: %#v", records[0])
	}

	status := bridge.Status()
	if !status.Configured || status.Mode != "lotus-rpc" || status.LatestDealID != "f01234" {
		t.Fatalf("unexpected lotus bridge status: %#v", status)
	}
}
