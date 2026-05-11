package api

import (
	"encoding/json"
	"net/http"
	"time"
)

type AssetInfo struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"createdAt"`
	Chunks    int       `json:"chunks"`
}

func (s *Server) handleAssets(w http.ResponseWriter, r *http.Request) {
	if s.Registry == nil {
		http.Error(w, "Registry not initialized", http.StatusInternalServerError)
		return
	}

	records, err := s.Registry.ListAssets()
	if err != nil {
		http.Error(w, "Failed to read registry", http.StatusInternalServerError)
		return
	}

	var assets []AssetInfo
	for _, rec := range records {
		assets = append(assets, AssetInfo{
			ID:        rec.ID,
			Filename:  rec.Filename,
			Size:      rec.Size,
			CreatedAt: rec.CreatedAt,
			Chunks:    rec.Chunks,
		})
	}

	if assets == nil {
		assets = []AssetInfo{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assets)
}
