package api

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"

	"github.com/bobtorrent/bobtorrent/internal/streaming"
	"github.com/bobtorrent/bobtorrent/internal/wallet"
	"github.com/bobtorrent/bobtorrent/pkg/dht"
	"github.com/bobtorrent/bobtorrent/pkg/storage"
)

//go:embed web-ui/*
var webUI embed.FS

type Server struct {
	Wallet         *wallet.Wallet
	Engine         *dht.Engine
	DataDir        string
	StreamServer   *streaming.Server
}

func (s *Server) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	store := storage.NewBlobStore(s.DataDir, s.Engine.Client)
	s.StreamServer = streaming.NewServer(s.DataDir, store)

	// API Endpoints
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/wallet", s.handleWallet)
	mux.HandleFunc("/api/wallet/airdrop", s.handleWalletAirdrop)
	mux.HandleFunc("/api/stream/", s.StreamServer.StreamHandler)
	mux.HandleFunc("/api/channels/browse", s.handleBrowseChannels)
	mux.HandleFunc("/api/ingest", s.handleIngest)

	// Serve the embedded static Web UI
	subFS, err := fs.Sub(webUI, "web-ui")
	if err != nil {
		panic("Failed to initialize embedded web-ui: " + err.Error())
	}
	mux.Handle("/", http.FileServer(http.FS(subFS)))

	return mux
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "online",
		"version": "3.0.0",
		"engine":  "bobtorrent-go",
	})
}

func (s *Server) handleWallet(w http.ResponseWriter, r *http.Request) {
	balance, err := s.Wallet.GetBalance()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"address": s.Wallet.GetPublicKey(),
		"balance": balance,
	})
}

func (s *Server) handleWalletAirdrop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sig, err := s.Wallet.RequestAirdrop()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"signature": sig,
	})
}

func (s *Server) handleBrowseChannels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]map[string]interface{}{
		{
			"id":          "mock-channel-1",
			"name":        "Go Port Channel",
			"description": "A placeholder channel indicating the Go port is active.",
			"peers":       42,
		},
	})
}
