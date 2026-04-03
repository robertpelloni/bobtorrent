package consensus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"bobtorrent/pkg/torrent"

	"github.com/gorilla/websocket"
)

type Server struct {
	lattice *Lattice
	upgrader websocket.Upgrader
}

func NewServer() *Server {
	return &Server{
		lattice: NewLattice(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (s *Server) HTTPHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/process", s.handleProcess)
	mux.HandleFunc("/balance/", s.handleBalance)
	mux.HandleFunc("/frontier/", s.handleFrontier)
	mux.HandleFunc("/market/bids", s.handleMarketBids)
	return mux
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.lattice.mu.RLock()
	defer s.lattice.mu.RUnlock()
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "online",
		"service": "Go Block Lattice Node",
		"chains": len(s.lattice.chains),
		"blocks": len(s.lattice.blocks),
		"stateHash": s.lattice.stateHash,
	})
}

func (s *Server) handleProcess(w http.ResponseWriter, r *http.Request) {
	var block torrent.Block
	if err := json.NewDecoder(r.Body).Decode(&block); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.lattice.ProcessBlock(&block); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "hash": block.Hash})
}

func (s *Server) handleBalance(w http.ResponseWriter, r *http.Request) {
	account := r.URL.Path[len("/balance/"):]
	balance := s.lattice.GetBalance(account)
	json.NewEncoder(w).Encode(map[string]interface{}{"balance": balance})
}

func (s *Server) handleFrontier(w http.ResponseWriter, r *http.Request) {
	account := r.URL.Path[len("/frontier/"):]
	f := s.lattice.GetFrontier(account)
	
	var hash *string
	staked := int64(0)
	if f != nil {
		hash = &f.Hash
		staked = f.StakedBalance
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"frontier": hash,
		"balance": s.lattice.GetBalance(account),
		"staked_balance": staked,
	})
}

func (s *Server) handleMarketBids(w http.ResponseWriter, r *http.Request) {
	s.lattice.mu.RLock()
	defer s.lattice.mu.RUnlock()

	var bids []*MarketBid
	for _, b := range s.lattice.marketBids {
		if b.Status == "OPEN" {
			bids = append(bids, b)
		}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"bids": bids})
}
