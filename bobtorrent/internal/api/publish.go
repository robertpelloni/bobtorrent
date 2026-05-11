package api

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"

	"github.com/bobtorrent/bobtorrent/pkg/storage"
)

func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FileEntry map[string]interface{} `json:"fileEntry"`
		Identity  Identity               `json:"identity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	privBytes, err := hex.DecodeString(req.Identity.PrivateKey)
	if err != nil || len(privBytes) != ed25519.PrivateKeySize {
		http.Error(w, "Invalid Private Key", http.StatusBadRequest)
		return
	}
	privKey := ed25519.PrivateKey(privBytes)

	manifestBytes, err := json.Marshal(req.FileEntry)
	if err != nil {
		http.Error(w, "Invalid Manifest", http.StatusBadRequest)
		return
	}

	// Canonicalize and sign
	// In the real system, we'd use fast-json-stable-stringify.
	// For now, we sign the standard JSON marshal.
	// We'll parse it into our internal manifest type to extract properties.
	var manifest storage.Manifest
	json.NewDecoder(bytes.NewReader(manifestBytes)).Decode(&manifest)

	signature := ed25519.Sign(privKey, manifestBytes)

	// Inject the signature into the final manifest map
	req.FileEntry["signature"] = hex.EncodeToString(signature)

	// Pass to engine to broadcast
	if s.Engine != nil {
		// Marshal the modified FileEntry with the signature included for the DHT payload
		signedManifestBytes, _ := json.Marshal(req.FileEntry)
		err := s.Engine.PublishManifest(privKey, signedManifestBytes)
		if err != nil {
			log.Printf("Failed to publish manifest to DHT: %v", err)
		}
	}
	req.FileEntry["publicKey"] = req.Identity.PublicKey

	// Log that we are actively publishing to the DHT network
	log.Printf("Publishing Manifest to Network via DHT (Publisher: %s)\n", req.Identity.PublicKey)
	log.Printf("Manifest Signed: %s\n", hex.EncodeToString(signature)[:16]+"...")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "published",
		"manifest": req.FileEntry,
	})
}
