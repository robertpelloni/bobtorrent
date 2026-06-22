package main

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"

	"github.com/anacrolix/torrent/metainfo"
	"bobtorrent/internal/transport"
)

// SwarmDiscoveryRequest represents the incoming JSON payload querying asset pieces.
type SwarmDiscoveryRequest struct {
	InfoHashHex string `json:"info_hash"`
}

// SwarmDiscoveryResponse represents the returned piece availability across the DHT.
type SwarmDiscoveryResponse struct {
	InfoHashHex   string   `json:"info_hash"`
	PeersFound    int      `json:"peers_found"`
	PeerAddresses []string `json:"peer_addresses"`
}

// Global reference to the DHT node, initialized in main.go
var globalDHTNode *transport.DHTNode

// handleSwarmDiscovery queries the DHT to resolve peers currently holding the requested piece/infohash.
func handleSwarmDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if globalDHTNode == nil {
		http.Error(w, "DHT node is not initialized", http.StatusServiceUnavailable)
		return
	}

	var req SwarmDiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if len(req.InfoHashHex) != 40 {
		http.Error(w, "Invalid infohash length. Must be 40 hex characters.", http.StatusBadRequest)
		return
	}

	hashBytes, err := hex.DecodeString(req.InfoHashHex)
	if err != nil {
		http.Error(w, "Invalid hex encoding for infohash", http.StatusBadRequest)
		return
	}

	var ih metainfo.Hash
	copy(ih[:], hashBytes)

	// Since we previously implemented GetPeersHybrid, we should leverage it here.
	// We'll return the hybrid mapped addresses (like I2P) plus standard ones if they existed.
	peers, err := globalDHTNode.GetPeersHybrid(req.InfoHashHex)
	if err != nil {
		log.Printf("Swarm discovery failed for %s: %v", req.InfoHashHex, err)
		http.Error(w, "Failed to query DHT", http.StatusInternalServerError)
		return
	}

	resp := SwarmDiscoveryResponse{
		InfoHashHex:   req.InfoHashHex,
		PeersFound:    len(peers),
		PeerAddresses: peers,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
