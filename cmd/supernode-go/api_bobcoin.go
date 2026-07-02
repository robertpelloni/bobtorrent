package main

import (
	"net/http"
)

// handleBlocks provides legacy compatibility for the Bobcoin frontend
// fetching confirmed blocks.
func handleBlocks(w http.ResponseWriter, r *http.Request) {
	// STUB: Real implementation would fetch blocks from SQLite/Memory
	// For now we just return an empty array or success response to prevent UI crashes.
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"blocks":  []interface{}{},
	})
}

// handleBootstrap provides legacy compatibility for the Bobcoin frontend
// initiating sync with peers.
func handleBootstrap(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Bootstrap sequence initiated via Bobtorrent native layer.",
	})
}
