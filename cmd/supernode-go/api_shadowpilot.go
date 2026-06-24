package main

import (
	"net/http"

	"bobtorrent/internal/shadowpilot"
)

// handleShadowPilotStatus returns the current git anomaly state tracked by Shadow Pilot.
func handleShadowPilotStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}

	state := shadowpilot.GetCurrentState()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"shadow_pilot_state": state,
	})
}
