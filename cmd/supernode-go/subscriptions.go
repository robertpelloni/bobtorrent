package main

import (
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

	subStoreMutex.Lock()
	subscriptionStore[req.PublicKey] = map[string]interface{}{}
	subStoreMutex.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}
