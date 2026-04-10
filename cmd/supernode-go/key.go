package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"net/http"
)

func handleKeyGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		http.Error(w, "Failed to generate key", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"publicKey": hex.EncodeToString(pub),
		"secretKey": hex.EncodeToString(priv),
	})
}
