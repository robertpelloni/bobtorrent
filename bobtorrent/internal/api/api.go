package api

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/pprof"

	"github.com/bobtorrent/bobtorrent/internal/streaming"
	"github.com/bobtorrent/bobtorrent/internal/wallet"
	"github.com/bobtorrent/bobtorrent/pkg/dht"
	"github.com/bobtorrent/bobtorrent/pkg/storage"
)

//go:embed web-ui/*
var webUI embed.FS

type Server struct {
	Wallet       *wallet.Wallet
	Engine       *dht.Engine
	DataDir      string
	StreamServer *streaming.Server
	Registry     *storage.Registry
}

func (s *Server) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	store := storage.NewBlobStore(s.DataDir, s.Engine.Client)
	s.StreamServer = streaming.NewServer(s.DataDir, store)

	registry, err := storage.NewRegistry(s.DataDir)
	if err != nil {
		panic("Failed to initialize SQLite registry: " + err.Error())
	}
	s.Registry = registry

	// API Endpoints
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/wallet", s.handleWallet)
	mux.HandleFunc("/api/wallet", s.handleWallet)
	mux.HandleFunc("/wallet/airdrop", s.handleWalletAirdrop)
	mux.HandleFunc("/api/wallet/airdrop", s.handleWalletAirdrop)
	mux.HandleFunc("/wallet/status", s.handleWalletStatus)
	mux.HandleFunc("/api/wallet/status", s.handleWalletStatus)
	mux.HandleFunc("/stream/", s.StreamServer.StreamHandler)
	mux.HandleFunc("/api/stream/", s.StreamServer.StreamHandler)
	mux.HandleFunc("/channels/browse", s.handleBrowseChannels)
	mux.HandleFunc("/api/channels/browse", s.handleBrowseChannels)
	mux.HandleFunc("/ingest", s.handleIngest)
	mux.HandleFunc("/api/ingest", s.handleIngest)
	mux.HandleFunc("/key/generate", s.handleKeyGenerate)
	mux.HandleFunc("/api/key/generate", s.handleKeyGenerate)
	mux.HandleFunc("/publish", s.handlePublish)
	mux.HandleFunc("/api/publish", s.handlePublish)
	mux.HandleFunc("/subscribe", s.handleSubscribe)
	mux.HandleFunc("/api/subscribe", s.handleSubscribe)
	mux.HandleFunc("/subscriptions", s.handleSubscriptions)
	mux.HandleFunc("/api/subscriptions", s.handleSubscriptions)
	mux.HandleFunc("/blobs", s.handleBlobs)
	mux.HandleFunc("/api/blobs", s.handleBlobs)
	mux.HandleFunc("/assets", s.handleAssets)
	mux.HandleFunc("/api/assets", s.handleAssets)
	mux.HandleFunc("/api/verify", s.handleVerify)
	mux.HandleFunc("/api/lattice", s.handleLattice)

	// Profiling Endpoints
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Serve the embedded static Web UI
	subFS, err := fs.Sub(webUI, "web-ui")
	if err != nil {
		panic("Failed to initialize embedded web-ui: " + err.Error())
	}
	mux.Handle("/", http.FileServer(http.FS(subFS)))

	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var peersCount int64 = 0
	if s.Engine != nil && s.Engine.Client != nil {
		torrents := s.Engine.Client.Torrents()
		for _, t := range torrents {
			peersCount += int64(len(t.PeerConns()))
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"peers":   peersCount,
		"storage": "online",
		"engine":  "bobtorrent-go",
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	var peersCount int64 = 0
	var dhtNodes int64 = 0

	if s.Engine != nil && s.Engine.Client != nil {
		torrents := s.Engine.Client.Torrents()
		for _, t := range torrents {
			peersCount += int64(len(t.PeerConns()))
		}

		dhtServers := s.Engine.Client.DhtServers()
		for _, _ = range dhtServers {
			// Skip dht stats for now to avoid compilation errors without dht/v2 import

			// dhtNodes += ...
		}
	}

	subMutex.Lock()
	subsCount := len(subscriptions)
	subMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "online",
		"version":       "3.0.0",
		"engine":        "bobtorrent-go",
		"dht":           dhtNodes,
		"subscriptions": subsCount,
		"storage": map[string]interface{}{
			"blobs":       0,                       // To be implemented later by counting blobs in dir
			"size":        0,                       // To be implemented
			"max":         10 * 1024 * 1024 * 1024, // 10 GB dummy value
			"utilization": 0,
		},
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

func (s *Server) handleWalletStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	bridgeStatus := "offline"
	if s.Wallet != nil {
		// Verify basic RPC connectivity
		_, err := s.Wallet.GetBalance()
		if err == nil {
			bridgeStatus = "connected"
		} else {
			bridgeStatus = "error"
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"network": "solana-devnet",
		"bridge":  bridgeStatus,
		"syncing": false,
	})
}

func (s *Server) handleBrowseChannels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Dynamically count total active connections from the anacrolix client
	var peersCount int64 = 0
	var dhtNodes int64 = 0

	if s.Engine != nil && s.Engine.Client != nil {
		// Just a simple wrapper counting known torrents and pieces as a placeholder for UI rendering
		// Since anacrolix stats structures change rapidly across versions
		torrents := s.Engine.Client.Torrents()
		for _, t := range torrents {
			peersCount += int64(len(t.PeerConns()))
		}

		dhtServers := s.Engine.Client.DhtServers()
		for _, _ = range dhtServers {
			// Skip dht stats for now to avoid compilation errors without dht/v2 import

			// dhtNodes += ...
		}
	}

	json.NewEncoder(w).Encode([]map[string]interface{}{
		{
			"id":          "bobtorrent-public-dht",
			"name":        "BobTorrent Public DHT Swarm",
			"description": "The active public DHT swarm powered by anacrolix/torrent mapping AES blobs.",
			"peers":       peersCount,
			"dhtNodes":    dhtNodes,
		},
	})
}
