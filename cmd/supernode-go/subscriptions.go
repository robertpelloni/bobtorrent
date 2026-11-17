package main

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
)

var (
	subscriptionStore = make(map[string]interface{})
	subStoreMutex     sync.RWMutex
)

func handleGetSubscriptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}

	subStoreMutex.RLock()
	subs := make([]map[string]interface{}, 0, len(subscriptionStore))
	for pubKey, data := range subscriptionStore {
		sub := map[string]interface{}{
			"publicKey": pubKey,
			"data":      data,
		}
		subs = append(subs, sub)
	}
	subStoreMutex.RUnlock()

	writeJSON(w, http.StatusOK, subs)
}

func handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PublicKey string `json:"publicKey"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.PublicKey == "" {
		http.Error(w, "Missing public key", http.StatusBadRequest)
		return
	}

	// Publisher keys are Ed25519 public keys: exactly 64 hex characters.
	// Anything else can never resolve to a channel, so reject it instead
	// of storing a dead subscription.
	if raw, err := hex.DecodeString(req.PublicKey); err != nil || len(raw) != 32 {
		http.Error(w, "publicKey must be a 64-character hex Ed25519 public key", http.StatusBadRequest)
		return
	}

	subStoreMutex.Lock()
	subscriptionStore[req.PublicKey] = map[string]interface{}{}
	subStoreMutex.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}
