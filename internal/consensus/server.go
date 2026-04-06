package consensus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"bobtorrent/pkg/torrent"

	"github.com/gorilla/websocket"
)

// Server exposes the Go lattice over HTTP and WebSocket transport.
//
// Compatibility goals:
//   - Serve the newer Go-native endpoints used by the supernode and TUI.
//   - Remain compatible with the existing bobcoin frontend, which expects
//     legacy paths such as /proposals, /pending/:account, and a WebSocket
//     connection on the lattice root URL.
type Server struct {
	lattice  *Lattice
	hub      *Hub
	upgrader websocket.Upgrader
}

// NewServer creates a lattice server with a fresh in-memory consensus state.
func NewServer() *Server {
	return newServerWithLattice(NewLattice())
}

// NewPersistentServer creates a lattice server backed by durable SQLite block
// persistence. Confirmed blocks are replayed on startup to restore consensus
// state after restart.
func NewPersistentServer(path string) (*Server, error) {
	lattice, err := NewPersistentLattice(path)
	if err != nil {
		return nil, err
	}
	return newServerWithLattice(lattice), nil
}

func newServerWithLattice(lattice *Lattice) *Server {
	return &Server{
		lattice: lattice,
		hub:     NewHub(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// Lattice exposes the underlying consensus engine so entrypoints can manage
// lifecycle concerns such as durable-store shutdown.
func (s *Server) Lattice() *Lattice {
	return s.lattice
}

// HTTPHandler wires all HTTP and WebSocket routes.
func (s *Server) HTTPHandler() http.Handler {
	mux := http.NewServeMux()

	// Root compatibility handler:
	// - If a frontend opens ws://host:4000, upgrade to WebSocket.
	// - If a browser hits http://host:4000/, return status JSON.
	mux.HandleFunc("/", s.handleRoot)

	// Core lattice endpoints.
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/process", s.handleProcess)
	mux.HandleFunc("/blocks", s.handleBlocks)
	mux.HandleFunc("/bootstrap", s.handleBootstrap)
	mux.HandleFunc("/balance/", s.handleBalance)
	mux.HandleFunc("/frontier/", s.handleFrontier)
	mux.HandleFunc("/chain/", s.handleChain)
	mux.HandleFunc("/block/", s.handleBlock)
	mux.HandleFunc("/pending/", s.handlePending)

	// Domain-specific endpoints.
	mux.HandleFunc("/market/bids", s.handleMarketBids)
	mux.HandleFunc("/proposals", s.handleProposals)
	mux.HandleFunc("/governance/proposals", s.handleProposals)
	mux.HandleFunc("/nfts", s.handleNFTs)
	mux.HandleFunc("/nfts/", s.handleNFTsByOwner)
	mux.HandleFunc("/anchors", s.handleAnchors)
	mux.HandleFunc("/anchors/", s.handleAnchorsByOwner)
	mux.HandleFunc("/swaps", s.handleSwaps)
	mux.HandleFunc("/peers", s.handlePeers)
	mux.HandleFunc("/persistence/verify", s.handlePersistenceVerify)
	mux.HandleFunc("/persistence/repair", s.handlePersistenceRepair)
	mux.HandleFunc("/persistence/export", s.handlePersistenceExport)
	mux.HandleFunc("/persistence/backup", s.handlePersistenceBackup)
	mux.HandleFunc("/persistence/backup-bundle", s.handlePersistenceBackupBundle)
	mux.HandleFunc("/persistence/import", s.handlePersistenceImport)
	mux.HandleFunc("/persistence/restore", s.handlePersistenceRestore)
	mux.HandleFunc("/persistence/restore-bundle", s.handlePersistenceRestoreBundle)
	mux.HandleFunc("/ws", s.handleWebSocket)

	return mux
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if websocket.IsWebSocketUpgrade(r) {
		s.handleWebSocket(w, r)
		return
	}

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	s.handleStatus(w, r)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.lattice.mu.RLock()
	defer s.lattice.mu.RUnlock()

	persistenceEnabled := s.lattice.store != nil
	persistencePath := ""
	persistedBlocks := int64(0)
	snapshotCount := int64(0)
	snapshotSequence := int64(0)
	snapshotInterval := int64(0)
	snapshotRetention := 0
	latestBlockHash := ""
	if len(s.lattice.blockOrder) > 0 {
		latestBlockHash = s.lattice.blockOrder[len(s.lattice.blockOrder)-1]
	}
	if persistenceEnabled {
		persistencePath = s.lattice.store.Path()
		persistedBlocks, _ = s.lattice.store.CountBlocks()
		snapshotCount, _ = s.lattice.store.CountSnapshots()
		snapshotSequence = s.lattice.snapshotSequence
		snapshotInterval = s.lattice.store.SnapshotInterval()
		snapshotRetention = s.lattice.store.SnapshotRetention()
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":          "online",
		"service":         "Go Block Lattice Node",
		"chains":          len(s.lattice.chains),
		"accounts":        len(s.lattice.chains),
		"blocks":          len(s.lattice.blocks),
		"totalBlocks":     len(s.lattice.blocks),
		"stateHash":       s.lattice.stateHash,
		"latestBlockHash": latestBlockHash,
		"peers":           len(s.lattice.peers),
		"wsClients":       s.hub.ClientCount(),
		"proposals":       len(s.lattice.proposals),
		"nfts":            len(s.lattice.nfts),
		"marketBids":      len(s.lattice.marketBids),
		"activeSwaps":     len(s.lattice.swaps),
		"anchors":         len(s.lattice.anchors),
		"persistence": map[string]interface{}{
			"enabled":           persistenceEnabled,
			"path":              persistencePath,
			"persistedBlocks":   persistedBlocks,
			"persistedSequence": s.lattice.persistedSequence,
			"snapshotCount":     snapshotCount,
			"snapshotSequence":  snapshotSequence,
			"snapshotInterval":  snapshotInterval,
			"snapshotRetention": snapshotRetention,
		},
	})
}

// handleProcess accepts either:
//  1. a raw block JSON object, or
//  2. a wrapper object in the shape {"block": {...}}
//
// Supporting both formats keeps the Go node compatible with both the
// bobcoin frontend and the Go supernode poller.
func (s *Server) handlePersistenceVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}

	report, err := s.lattice.VerifyPersistence()
	if err != nil {
		http.Error(w, fmt.Sprintf("persistence verification unavailable: %v", err), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handlePersistenceRepair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	before, err := s.lattice.VerifyPersistence()
	if err != nil {
		http.Error(w, fmt.Sprintf("persistence repair unavailable: %v", err), http.StatusBadRequest)
		return
	}
	if !before.Repairable {
		http.Error(w, "persistence repair refused: confirmed block log corruption requires manual recovery", http.StatusConflict)
		return
	}

	after, err := s.lattice.RepairPersistence()
	if err != nil {
		http.Error(w, fmt.Sprintf("persistence repair failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"before":  before,
		"after":   after,
	})
}

func (s *Server) handlePersistenceExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}

	bundle, err := s.lattice.ExportPersistence()
	if err != nil {
		http.Error(w, fmt.Sprintf("persistence export unavailable: %v", err), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, bundle)
}

func (s *Server) handlePersistenceBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Path string `json:"path"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}

	result, err := s.lattice.BackupPersistence(body.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("persistence backup failed: %v", err), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"backup":  result,
	})
}

func (s *Server) handlePersistenceBackupBundle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Path              string `json:"path"`
		Passphrase        string `json:"passphrase"`
		SigningPrivateKey string `json:"signingPrivateKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, fmt.Sprintf("invalid backup bundle payload: %v", err), http.StatusBadRequest)
		return
	}

	result, err := s.lattice.CreateSignedEncryptedBackupBundle(body.Path, body.Passphrase, body.SigningPrivateKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("persistence backup bundle failed: %v", err), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"bundle":  result,
	})
}

func (s *Server) handlePersistenceImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Path   string               `json:"path"`
		Bundle *LatticeExportBundle `json:"bundle"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, fmt.Sprintf("invalid import payload: %v", err), http.StatusBadRequest)
		return
	}

	result, err := s.lattice.ImportPersistenceBundle(body.Path, body.Bundle)
	if err != nil {
		http.Error(w, fmt.Sprintf("persistence import failed: %v", err), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"restore": result,
	})
}

func (s *Server) handlePersistenceRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		SourcePath string `json:"sourcePath"`
		TargetPath string `json:"targetPath"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, fmt.Sprintf("invalid restore payload: %v", err), http.StatusBadRequest)
		return
	}

	result, err := s.lattice.RestorePersistenceBackup(body.SourcePath, body.TargetPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("persistence restore failed: %v", err), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"restore": result,
	})
}

func (s *Server) handlePersistenceRestoreBundle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		SourcePath       string `json:"sourcePath"`
		Passphrase       string `json:"passphrase"`
		TargetPath       string `json:"targetPath"`
		RequireSignature bool   `json:"requireSignature"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, fmt.Sprintf("invalid restore bundle payload: %v", err), http.StatusBadRequest)
		return
	}

	result, err := s.lattice.RestoreSignedEncryptedBackupBundle(body.SourcePath, body.Passphrase, body.TargetPath, body.RequireSignature)
	if err != nil {
		http.Error(w, fmt.Sprintf("persistence restore bundle failed: %v", err), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"restore": result,
	})
}

func (s *Server) handleProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	var block torrent.Block
	if raw, ok := body["block"]; ok {
		if err := json.Unmarshal(raw, &block); err != nil {
			http.Error(w, fmt.Sprintf("invalid wrapped block: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		reencoded, err := json.Marshal(body)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to re-encode request: %v", err), http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal(reencoded, &block); err != nil {
			http.Error(w, fmt.Sprintf("invalid block payload: %v", err), http.StatusBadRequest)
			return
		}
	}

	accepted, err := s.lattice.ProcessBlockDetailed(&block)
	if err != nil {
		http.Error(w, fmt.Sprintf("block rejected: %v", err), http.StatusBadRequest)
		return
	}

	s.lattice.mu.RLock()
	chains := len(s.lattice.chains)
	blocks := len(s.lattice.blocks)
	stateHash := s.lattice.stateHash
	s.lattice.mu.RUnlock()

	if accepted {
		go s.broadcastBlock(&block)
		go s.hub.BroadcastBlock(&block, stateHash, chains, blocks)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"accepted":  accepted,
		"duplicate": !accepted,
		"hash":      block.Hash,
		"stateHash": stateHash,
	})
}

func (s *Server) handleBlocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}

	after := strings.TrimSpace(r.URL.Query().Get("after"))
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if _, err := fmt.Sscanf(raw, "%d", &limit); err != nil {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
	}

	blocks, cursorFound, hasMore := s.lattice.GetOrderedBlocksAfter(after, limit)
	s.lattice.mu.RLock()
	totalBlocks := len(s.lattice.blocks)
	latestHash := ""
	if len(s.lattice.blockOrder) > 0 {
		latestHash = s.lattice.blockOrder[len(s.lattice.blockOrder)-1]
	}
	s.lattice.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"blocks":      blocks,
		"count":       len(blocks),
		"after":       after,
		"cursorFound": cursorFound,
		"hasMore":     hasMore,
		"latestHash":  latestHash,
		"totalBlocks": totalBlocks,
	})
}

func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		peers := s.lattice.GetPeers()
		s.lattice.mu.RLock()
		totalBlocks := len(s.lattice.blocks)
		stateHash := s.lattice.stateHash
		latestBlockHash := ""
		if len(s.lattice.blockOrder) > 0 {
			latestBlockHash = s.lattice.blockOrder[len(s.lattice.blockOrder)-1]
		}
		s.lattice.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":          "online",
			"service":         "Go Block Lattice Bootstrap",
			"latestBlockHash": latestBlockHash,
			"totalBlocks":     totalBlocks,
			"stateHash":       stateHash,
			"peers":           peers,
			"peerCount":       len(peers),
		})
	case http.MethodPost:
		var req struct {
			Peer string `json:"peer"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Peer) == "" {
			http.Error(w, "valid peer required", http.StatusBadRequest)
			return
		}
		result, err := s.syncPeer(req.Peer)
		if err != nil {
			http.Error(w, fmt.Sprintf("bootstrap sync failed: %v", err), http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"sync":    result,
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleBalance(w http.ResponseWriter, r *http.Request) {
	account := strings.TrimPrefix(r.URL.Path, "/balance/")
	if account == "" {
		http.Error(w, "account required", http.StatusBadRequest)
		return
	}

	s.lattice.mu.RLock()
	balance := s.lattice.GetBalance(account)
	s.lattice.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"account": account,
		"balance": balance,
	})
}

func (s *Server) handleFrontier(w http.ResponseWriter, r *http.Request) {
	account := strings.TrimPrefix(r.URL.Path, "/frontier/")
	if account == "" {
		http.Error(w, "account required", http.StatusBadRequest)
		return
	}

	s.lattice.mu.RLock()
	frontier := s.lattice.GetFrontier(account)
	balance := s.lattice.GetBalance(account)
	s.lattice.mu.RUnlock()

	var hash *string
	staked := int64(0)
	height := 0
	if frontier != nil {
		hash = &frontier.Hash
		staked = frontier.StakedBalance
		height = frontier.Height
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"account":        account,
		"frontier":       hash,
		"balance":        balance,
		"staked_balance": staked,
		"height":         height,
	})
}

func (s *Server) handleChain(w http.ResponseWriter, r *http.Request) {
	account := strings.TrimPrefix(r.URL.Path, "/chain/")
	if account == "" {
		http.Error(w, "account required", http.StatusBadRequest)
		return
	}

	s.lattice.mu.RLock()
	chain := append([]*torrent.Block(nil), s.lattice.chains[account]...)
	s.lattice.mu.RUnlock()

	// Return both "blocks" and "chain" keys for compatibility.
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"account": account,
		"blocks":  chain,
		"chain":   chain,
		"length":  len(chain),
	})
}

func (s *Server) handleBlock(w http.ResponseWriter, r *http.Request) {
	hash := strings.TrimPrefix(r.URL.Path, "/block/")
	if hash == "" {
		http.Error(w, "hash required", http.StatusBadRequest)
		return
	}

	s.lattice.mu.RLock()
	block, ok := s.lattice.blocks[hash]
	s.lattice.mu.RUnlock()
	if !ok {
		http.Error(w, "block not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, block)
}

func (s *Server) handlePending(w http.ResponseWriter, r *http.Request) {
	account := strings.TrimPrefix(r.URL.Path, "/pending/")
	if account == "" {
		http.Error(w, "account required", http.StatusBadRequest)
		return
	}

	s.lattice.mu.RLock()
	pending := append([]PendingTx(nil), s.lattice.pending[account]...)
	s.lattice.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"account": account,
		"pending": pending,
	})
}

func (s *Server) handleMarketBids(w http.ResponseWriter, r *http.Request) {
	s.lattice.mu.RLock()
	defer s.lattice.mu.RUnlock()

	var bids []*MarketBid
	for _, bid := range s.lattice.marketBids {
		if bid.Status == "OPEN" {
			bids = append(bids, bid)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"bids": bids})
}

func (s *Server) handleProposals(w http.ResponseWriter, r *http.Request) {
	s.lattice.mu.RLock()
	defer s.lattice.mu.RUnlock()

	var proposals []*Proposal
	for _, proposal := range s.lattice.proposals {
		proposals = append(proposals, proposal)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"proposals": proposals,
	})
}

func (s *Server) handleNFTs(w http.ResponseWriter, r *http.Request) {
	s.lattice.mu.RLock()
	defer s.lattice.mu.RUnlock()

	var nfts []*NFT
	for _, nft := range s.lattice.nfts {
		nfts = append(nfts, nft)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"nfts": nfts})
}

func (s *Server) handleNFTsByOwner(w http.ResponseWriter, r *http.Request) {
	owner := strings.TrimPrefix(r.URL.Path, "/nfts/")
	if owner == "" {
		http.Error(w, "owner required", http.StatusBadRequest)
		return
	}

	s.lattice.mu.RLock()
	defer s.lattice.mu.RUnlock()

	var nfts []*NFT
	for _, nft := range s.lattice.nfts {
		if nft.Owner == owner {
			nfts = append(nfts, nft)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"owner": owner,
		"nfts":  nfts,
	})
}

func (s *Server) handleAnchors(w http.ResponseWriter, r *http.Request) {
	s.lattice.mu.RLock()
	defer s.lattice.mu.RUnlock()

	anchors := make([]*ManifestAnchor, 0, len(s.lattice.anchors))
	for _, anchor := range s.lattice.anchors {
		anchors = append(anchors, anchor)
	}
	sort.Slice(anchors, func(i, j int) bool {
		return anchors[i].Timestamp > anchors[j].Timestamp
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{"anchors": anchors})
}

func (s *Server) handleAnchorsByOwner(w http.ResponseWriter, r *http.Request) {
	owner := strings.TrimPrefix(r.URL.Path, "/anchors/")
	if owner == "" {
		http.Error(w, "owner required", http.StatusBadRequest)
		return
	}

	s.lattice.mu.RLock()
	defer s.lattice.mu.RUnlock()

	var anchors []*ManifestAnchor
	for _, anchor := range s.lattice.anchors {
		if anchor.Owner == owner {
			anchors = append(anchors, anchor)
		}
	}
	sort.Slice(anchors, func(i, j int) bool {
		return anchors[i].Timestamp > anchors[j].Timestamp
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{"owner": owner, "anchors": anchors})
}

func (s *Server) handleSwaps(w http.ResponseWriter, r *http.Request) {
	s.lattice.mu.RLock()
	defer s.lattice.mu.RUnlock()

	var swaps []*Swap
	for _, swap := range s.lattice.swaps {
		swaps = append(swaps, swap)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"swaps": swaps})
}

func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		peers := s.lattice.GetPeers()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"peers": peers,
			"count": len(peers),
		})
	case http.MethodPost:
		var req struct {
			Addr string `json:"addr"`
			Sync *bool  `json:"sync,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Addr) == "" {
			http.Error(w, "valid addr required", http.StatusBadRequest)
			return
		}
		addr, err := normalizePeerAddr(req.Addr)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid peer addr: %v", err), http.StatusBadRequest)
			return
		}
		s.lattice.AddPeer(addr)

		syncRequested := true
		if req.Sync != nil {
			syncRequested = *req.Sync
		}
		if !syncRequested {
			writeJSON(w, http.StatusOK, map[string]interface{}{"registered": addr, "sync": nil})
			return
		}

		result, err := s.syncPeer(addr)
		if err != nil {
			http.Error(w, fmt.Sprintf("peer registration sync failed: %v", err), http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"registered": addr, "sync": result})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) broadcastBlock(block *torrent.Block) {
	peers := s.lattice.GetPeers()
	if len(peers) == 0 {
		return
	}

	payload, err := json.Marshal(map[string]interface{}{"block": block})
	if err != nil {
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	for _, peerAddr := range peers {
		go func(addr string) {
			resp, err := client.Post(peerBaseURL(addr)+"/process", "application/json", bytes.NewBuffer(payload))
			if err != nil {
				log.Printf("[consensus] broadcast to %s failed: %v", addr, err)
				return
			}
			defer resp.Body.Close()
		}(peerAddr)
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[consensus] websocket upgrade failed: %v", err)
		return
	}

	s.hub.Register(conn)

	s.lattice.mu.RLock()
	welcome := map[string]interface{}{
		"type":        "STATS",
		"event":       "CONNECTED",
		"service":     "Go Block Lattice Node",
		"accounts":    len(s.lattice.chains),
		"chains":      len(s.lattice.chains),
		"totalBlocks": len(s.lattice.blocks),
		"stateHash":   s.lattice.stateHash,
	}
	s.lattice.mu.RUnlock()

	if err := conn.WriteJSON(welcome); err != nil {
		s.hub.Unregister(conn)
		return
	}

	go func() {
		defer s.hub.Unregister(conn)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

type peerBlocksResponse struct {
	Blocks      []*torrent.Block `json:"blocks"`
	Count       int              `json:"count"`
	CursorFound bool             `json:"cursorFound"`
	HasMore     bool             `json:"hasMore"`
	LatestHash  string           `json:"latestHash"`
	TotalBlocks int              `json:"totalBlocks"`
}

type peerListResponse struct {
	Peers []string `json:"peers"`
}

type PeerSyncResult struct {
	Peer            string   `json:"peer"`
	RequestedCursor string   `json:"requestedCursor,omitempty"`
	AppliedBlocks   int      `json:"appliedBlocks"`
	DuplicateBlocks int      `json:"duplicateBlocks"`
	FetchedPages    int      `json:"fetchedPages"`
	CursorReset     bool     `json:"cursorReset"`
	DiscoveredPeers []string `json:"discoveredPeers,omitempty"`
	LatestBlockHash string   `json:"latestBlockHash,omitempty"`
}

func normalizePeerAddr(addr string) (string, error) {
	trimmed := strings.TrimSpace(addr)
	trimmed = strings.TrimSuffix(trimmed, "/")
	if trimmed == "" {
		return "", fmt.Errorf("peer address is empty")
	}
	if strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return "", err
		}
		if parsed.Host == "" {
			return "", fmt.Errorf("peer URL host missing")
		}
		return parsed.Host, nil
	}
	if strings.Contains(trimmed, "/") {
		return "", fmt.Errorf("peer address must not include a path")
	}
	return trimmed, nil
}

func peerBaseURL(addr string) string {
	if strings.Contains(addr, "://") {
		return strings.TrimSuffix(addr, "/")
	}
	return "http://" + strings.TrimSuffix(addr, "/")
}

func (s *Server) syncPeer(addr string) (*PeerSyncResult, error) {
	normalized, err := normalizePeerAddr(addr)
	if err != nil {
		return nil, err
	}

	result := &PeerSyncResult{Peer: normalized}
	result.RequestedCursor = s.lattice.LatestBlockHash()
	baseURL := peerBaseURL(normalized)
	client := &http.Client{Timeout: 10 * time.Second}
	cursor := result.RequestedCursor
	usedCursor := cursor != ""

	for {
		page, err := fetchPeerBlocks(client, baseURL, cursor)
		if err != nil {
			return result, err
		}
		result.FetchedPages++
		if usedCursor && !page.CursorFound {
			cursor = ""
			usedCursor = false
			result.CursorReset = true
			continue
		}
		if len(page.Blocks) == 0 {
			result.LatestBlockHash = page.LatestHash
			break
		}

		for _, block := range page.Blocks {
			accepted, err := s.lattice.ProcessBlockDetailed(block)
			if err != nil {
				return result, fmt.Errorf("failed to apply peer block %s: %w", block.Hash, err)
			}
			if accepted {
				result.AppliedBlocks++
			} else {
				result.DuplicateBlocks++
			}
			cursor = block.Hash
		}
		result.LatestBlockHash = page.LatestHash
		usedCursor = false
		if !page.HasMore {
			break
		}
	}

	peers, err := fetchPeerPeers(client, baseURL)
	if err == nil {
		known := make(map[string]bool)
		for _, existing := range s.lattice.GetPeers() {
			known[existing] = true
		}
		for _, peer := range peers {
			normalizedPeer, normalizeErr := normalizePeerAddr(peer)
			if normalizeErr != nil || normalizedPeer == "" || normalizedPeer == normalized {
				continue
			}
			if known[normalizedPeer] {
				continue
			}
			s.lattice.AddPeer(normalizedPeer)
			known[normalizedPeer] = true
			result.DiscoveredPeers = append(result.DiscoveredPeers, normalizedPeer)
		}
	}

	return result, nil
}

func fetchPeerBlocks(client *http.Client, baseURL, after string) (*peerBlocksResponse, error) {
	endpoint := baseURL + "/blocks?limit=200"
	if after != "" {
		endpoint += "&after=" + url.QueryEscape(after)
	}
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("fetch peer blocks failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("peer blocks returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var page peerBlocksResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, fmt.Errorf("decode peer blocks failed: %w", err)
	}
	return &page, nil
}

func fetchPeerPeers(client *http.Client, baseURL string) ([]string, error) {
	resp, err := client.Get(baseURL + "/peers")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("peer list returned %d", resp.StatusCode)
	}
	var result peerListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Peers, nil
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
