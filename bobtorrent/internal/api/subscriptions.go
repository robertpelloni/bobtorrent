package api

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
)

var (
	subMutex      sync.Mutex
	subscriptions []map[string]interface{}
)

func (s *Server) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PublicKey string `json:"publicKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	subMutex.Lock()
	defer subMutex.Unlock()

	for _, sub := range subscriptions {
		if sub["channelKey"] == req.PublicKey {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "already_subscribed",
			})
			return
		}
	}

	newSub := map[string]interface{}{
		"channelKey": req.PublicKey,
		"lastSeq":    0,
		"status":     "Syncing",
	}
	subscriptions = append(subscriptions, newSub)

	// Start polling the DHT for the channel
	if s.Engine != nil {
		pubKeyBytes, err := hex.DecodeString(req.PublicKey)
		if err == nil && len(pubKeyBytes) == ed25519.PublicKeySize {
			pubKey := ed25519.PublicKey(pubKeyBytes)
			s.Engine.SubscribeToChannel(pubKey)
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "subscribed",
	})
}

func (s *Server) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	subMutex.Lock()
	defer subMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscriptions)
}
