package main

import (
	"encoding/json"
	"net/http"
)

func handlePublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FileEntry map[string]interface{} `json:"fileEntry"`
		Identity  map[string]interface{} `json:"identity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.FileEntry == nil || req.Identity == nil {
		http.Error(w, "Missing fileEntry or identity", http.StatusBadRequest)
		return
	}

	manifest := map[string]interface{}{
		"publicKey": req.Identity["publicKey"],
		"collections": []interface{}{
			map[string]interface{}{
				"title": "Default Collection",
				"items": []interface{}{req.FileEntry},
			},
		},
	}

	stored, err := publishRegistry.PublishManifest(manifest, publicBaseURL(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "published",
		"manifest": stored.Manifest,
	})
}
