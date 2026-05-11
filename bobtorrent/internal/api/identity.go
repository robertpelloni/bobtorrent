package api

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

type Identity struct {
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"secretKey"` // Match JS client's 'secretKey' mapping
}

func (s *Server) handleKeyGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		http.Error(w, "Failed to generate key", http.StatusInternalServerError)
		return
	}

	ident := Identity{
		PublicKey:  hex.EncodeToString(pub),
		PrivateKey: hex.EncodeToString(priv),
	}

	// Persist the identity
	identPath := filepath.Join(s.DataDir, "identity.json")
	data, _ := json.MarshalIndent(ident, "", "  ")
	os.WriteFile(identPath, data, 0600)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ident)
}
