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
	"sync"
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
	lattice      *Lattice
	hub          *Hub
	upgrader     websocket.Upgrader
	peerStatusMu sync.RWMutex
	peerStatus   map[string]*PeerStatus
	syncStop     chan struct{}
	syncInterval time.Duration
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
		lattice:      lattice,
		hub:          NewHub(),
		peerStatus:   make(map[string]*PeerStatus),
		syncInterval: 30 * time.Second, // Default sync check every 30s
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
	mux.HandleFunc("/reconcile", s.handleReconcile)
	mux.HandleFunc("/reconcile/apply", s.handleReconcileApply)
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

// StopBackgroundSync terminates the autonomous peer synchronization loop.
func (s *Server) StopBackgroundSync() {
	if s.syncStop != nil {
		close(s.syncStop)
		s.syncStop = nil
	}
}

// StartBackgroundSync initiates the autonomous peer synchronization loop,
// which periodically analyzes and catches up with the network.
func (s *Server) StartBackgroundSync(interval time.Duration) {
	if interval > 0 {
		s.syncInterval = interval
	}
	if s.syncStop != nil {
		return
	}
	s.syncStop = make(chan struct{})
	go s.syncLoop()
}

func (s *Server) syncLoop() {
	ticker := time.NewTicker(s.syncInterval)
	defer ticker.Stop()

	log.Printf("[Consensus] Starting autonomous peer sync loop (interval: %v)", s.syncInterval)

	for {
		select {
		case <-s.syncStop:
			return
		case <-ticker.C:
			s.performAutonomousSync()
		}
	}
}

func (s *Server) performAutonomousSync() {
	peers := s.lattice.GetPeers()
	if len(peers) == 0 {
		return
	}

	for _, peer := range peers {
		// Only sync from one peer per loop iteration to avoid overwhelming 
		// the node or network.
		report, err := s.analyzePeerReconciliation(peer)
		if err != nil {
			// Telemetry is updated inside analyze/sync methods via mark helpers.
			continue
		}

		if report.Relationship == "remote_ahead" || report.Relationship == "local_empty_remote_has_state" {
			log.Printf("[Consensus] Autonomous sync: Peer %s is ahead. Catching up...", peer)
			// Apply reconciliation safely.
			_, _, _ = s.applyPeerReconciliation(peer, false)
			// Stop after first successful catch-up attempt this cycle.
			return
		}
	}
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
	peerStatuses := s.peerStatusSnapshot()
	peerSummary := summarizePeerStatuses(peerStatuses)

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
		"peerSync": map[string]interface{}{
			"summary": peerSummary,
			"peers":   peerStatuses,
		},
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
		peerStatuses := s.peerStatusSnapshot()
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
			"peerSync": map[string]interface{}{
				"summary": summarizePeerStatuses(peerStatuses),
				"peers":   peerStatuses,
			},
		})
	case http.MethodPost:
		var req struct {
			Peer  string `json:"peer"`
			Force bool   `json:"force"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Peer) == "" {
			http.Error(w, "valid peer required", http.StatusBadRequest)
			return
		}
		result, err := s.syncPeer(req.Peer, req.Force)
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

func (s *Server) handleReconcile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Peer string `json:"peer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Peer) == "" {
		http.Error(w, "valid peer required", http.StatusBadRequest)
		return
	}

	report, err := s.analyzePeerReconciliation(req.Peer)
	if err != nil {
		http.Error(w, fmt.Sprintf("reconciliation analysis failed: %v", err), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":        true,
		"reconciliation": report,
	})
}

func (s *Server) handleReconcileApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Peer  string `json:"peer"`
		Force bool   `json:"force"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Peer) == "" {
		http.Error(w, "valid peer required", http.StatusBadRequest)
		return
	}

	result, statusCode, err := s.applyPeerReconciliation(req.Peer, req.Force)
	if err != nil {
		writeJSON(w, statusCode, map[string]interface{}{
			"success": false,
			"apply":   result,
			"error":   err.Error(),
		})
		return
	}
	writeJSON(w, statusCode, map[string]interface{}{
		"success": true,
		"apply":   result,
	})
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
		statuses := s.peerStatusSnapshot()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"peers":         peers,
			"count":         len(peers),
			"diagnostics":   statuses,
			"healthSummary": summarizePeerStatuses(statuses),
		})
	case http.MethodPost:
		var req struct {
			Addr  string `json:"addr"`
			Sync  *bool  `json:"sync,omitempty"`
			Force bool   `json:"force"`
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
		s.ensurePeerStatus(addr)

		syncRequested := true
		if req.Sync != nil {
			syncRequested = *req.Sync
		}
		if !syncRequested {
			writeJSON(w, http.StatusOK, map[string]interface{}{"registered": addr, "sync": nil})
			return
		}

		result, err := s.syncPeer(addr, req.Force)
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
			if remaining := s.peerCooldownRemaining(addr); remaining > 0 {
				s.markPeerBroadcastSkipped(addr, remaining)
				return
			}
			s.markPeerBroadcastAttempt(addr)
			retriesUsed, err := retryFetch(2, 100*time.Millisecond, func() error {
				resp, postErr := client.Post(peerBaseURL(addr)+"/process", "application/json", bytes.NewBuffer(payload))
				if postErr != nil {
					return postErr
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
					return fmt.Errorf("peer process returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
				}
				return nil
			})
			if retriesUsed > 0 {
				s.markPeerRetryCount(addr, retriesUsed)
			}
			if err != nil {
				log.Printf("[consensus] broadcast to %s failed: %v", addr, err)
				s.markPeerBroadcastFailed(addr, err)
				return
			}
			s.markPeerBroadcastSucceeded(addr)
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
	Peer                 string   `json:"peer"`
	RequestedCursor      string   `json:"requestedCursor,omitempty"`
	AppliedBlocks        int      `json:"appliedBlocks"`
	DuplicateBlocks      int      `json:"duplicateBlocks"`
	FetchedPages         int      `json:"fetchedPages"`
	CursorReset          bool     `json:"cursorReset"`
	DiscoveredPeers      []string `json:"discoveredPeers,omitempty"`
	LatestBlockHash      string   `json:"latestBlockHash,omitempty"`
	RemoteStateHash      string   `json:"remoteStateHash,omitempty"`
	RemoteTotalBlocks    int      `json:"remoteTotalBlocks,omitempty"`
	RemotePeerCount      int      `json:"remotePeerCount,omitempty"`
	LagBlocks            int      `json:"lagBlocks,omitempty"`
	RetryCount           int      `json:"retryCount"`
	SkippedDueToCooldown bool     `json:"skippedDueToCooldown,omitempty"`
	CooldownRemainingMs  int64    `json:"cooldownRemainingMs,omitempty"`
	DivergenceSuspected  bool     `json:"divergenceSuspected,omitempty"`
	DivergenceReason     string   `json:"divergenceReason,omitempty"`
}

type peerBootstrapResponse struct {
	Status          string   `json:"status"`
	Service         string   `json:"service"`
	LatestBlockHash string   `json:"latestBlockHash"`
	TotalBlocks     int      `json:"totalBlocks"`
	StateHash       string   `json:"stateHash"`
	Peers           []string `json:"peers"`
	PeerCount       int      `json:"peerCount"`
}

type PeerReconciliationReport struct {
	Peer                      string   `json:"peer"`
	LocalLatestHash           string   `json:"localLatestHash,omitempty"`
	RemoteLatestHash          string   `json:"remoteLatestHash,omitempty"`
	LocalStateHash            string   `json:"localStateHash,omitempty"`
	RemoteStateHash           string   `json:"remoteStateHash,omitempty"`
	LocalTotalBlocks          int      `json:"localTotalBlocks"`
	RemoteTotalBlocks         int      `json:"remoteTotalBlocks"`
	PeerCooldownRemainingMs   int64    `json:"peerCooldownRemainingMs,omitempty"`
	RemoteContainsLocalCursor bool     `json:"remoteContainsLocalCursor"`
	LocalContainsRemoteLatest bool     `json:"localContainsRemoteLatest"`
	Relationship              string   `json:"relationship"`
	SuggestedAction           string   `json:"suggestedAction"`
	Notes                     []string `json:"notes,omitempty"`
}

type ReconciliationApplyResult struct {
	Peer            string                    `json:"peer"`
	Relationship    string                    `json:"relationship"`
	SuggestedAction string                    `json:"suggestedAction"`
	Executed        bool                      `json:"executed"`
	ExecutionMode   string                    `json:"executionMode"`
	Reason          string                    `json:"reason,omitempty"`
	Reconciliation  *PeerReconciliationReport `json:"reconciliation,omitempty"`
	Sync            *PeerSyncResult           `json:"sync,omitempty"`
}

// PeerStatus captures operator-visible sync and delivery telemetry for one
// known lattice peer. This is intentionally diagnostic rather than consensus-
// critical state: it helps operators understand peer lag, flaky bootstrap
// attempts, and repeated broadcast failures without influencing validation.
type PeerStatus struct {
	Peer                     string   `json:"peer"`
	Health                   string   `json:"health"`
	LastStatus               string   `json:"lastStatus"`
	LastError                string   `json:"lastError,omitempty"`
	LastSyncStartedAt        int64    `json:"lastSyncStartedAt,omitempty"`
	LastSyncCompletedAt      int64    `json:"lastSyncCompletedAt,omitempty"`
	LastSyncSucceededAt      int64    `json:"lastSyncSucceededAt,omitempty"`
	LastSyncFailedAt         int64    `json:"lastSyncFailedAt,omitempty"`
	LastSkippedSyncAt        int64    `json:"lastSkippedSyncAt,omitempty"`
	TotalSyncAttempts        int64    `json:"totalSyncAttempts"`
	TotalSyncSuccesses       int64    `json:"totalSyncSuccesses"`
	TotalSyncFailures        int64    `json:"totalSyncFailures"`
	SkippedSyncs             int64    `json:"skippedSyncs"`
	SkippedBroadcasts        int64    `json:"skippedBroadcasts"`
	ConsecutiveFailures      int      `json:"consecutiveFailures"`
	CooldownUntil            int64    `json:"cooldownUntil,omitempty"`
	CooldownRemainingMs      int64    `json:"cooldownRemainingMs,omitempty"`
	DivergenceCount          int64    `json:"divergenceCount"`
	LastDivergenceAt         int64    `json:"lastDivergenceAt,omitempty"`
	LastDivergenceReason     string   `json:"lastDivergenceReason,omitempty"`
	LastAppliedBlocks        int      `json:"lastAppliedBlocks"`
	LastDuplicateBlocks      int      `json:"lastDuplicateBlocks"`
	LastFetchedPages         int      `json:"lastFetchedPages"`
	LastCursorReset          bool     `json:"lastCursorReset"`
	LastRetryCount           int      `json:"lastRetryCount"`
	LastKnownLagBlocks       int      `json:"lastKnownLagBlocks"`
	LastRemoteTotalBlocks    int      `json:"lastRemoteTotalBlocks"`
	LastKnownRemotePeerCount int      `json:"lastKnownRemotePeerCount"`
	LastAdvertisedLatestHash string   `json:"lastAdvertisedLatestHash,omitempty"`
	LastRemoteStateHash      string   `json:"lastRemoteStateHash,omitempty"`
	LastDiscoveredPeers      []string `json:"lastDiscoveredPeers,omitempty"`
	LastBroadcastAttemptAt   int64    `json:"lastBroadcastAttemptAt,omitempty"`
	LastBroadcastSucceededAt int64    `json:"lastBroadcastSucceededAt,omitempty"`
	LastBroadcastFailedAt    int64    `json:"lastBroadcastFailedAt,omitempty"`
	BroadcastSuccesses       int64    `json:"broadcastSuccesses"`
	BroadcastFailures        int64    `json:"broadcastFailures"`
	LastBroadcastError       string   `json:"lastBroadcastError,omitempty"`
}

func clonePeerStatus(src *PeerStatus) *PeerStatus {
	if src == nil {
		return nil
	}
	copyStatus := *src
	copyStatus.LastDiscoveredPeers = append([]string(nil), src.LastDiscoveredPeers...)
	return &copyStatus
}

func peerCooldownDuration(consecutiveFailures int) time.Duration {
	if consecutiveFailures <= 0 {
		return 0
	}
	cooldown := time.Duration(consecutiveFailures) * 5 * time.Second
	if cooldown > time.Minute {
		cooldown = time.Minute
	}
	return cooldown
}

func peerCooldownRemaining(status *PeerStatus, now time.Time) int64 {
	if status == nil || status.CooldownUntil == 0 {
		return 0
	}
	remaining := status.CooldownUntil - now.UnixMilli()
	if remaining < 0 {
		return 0
	}
	return remaining
}

func peerHealth(status *PeerStatus) string {
	if status == nil {
		return "unknown"
	}
	now := time.Now()
	if status.LastDivergenceReason != "" && status.LastDivergenceAt >= status.LastSyncSucceededAt {
		return "diverged"
	}
	if peerCooldownRemaining(status, now) > 0 {
		return "cooldown"
	}
	if status.ConsecutiveFailures >= 3 {
		return "failing"
	}
	if status.ConsecutiveFailures > 0 || status.LastKnownLagBlocks > 0 || status.LastCursorReset {
		return "degraded"
	}
	if status.TotalSyncSuccesses > 0 || status.BroadcastSuccesses > 0 {
		return "healthy"
	}
	if status.TotalSyncAttempts > 0 || status.BroadcastFailures > 0 {
		return "warning"
	}
	return "idle"
}

func (s *Server) ensurePeerStatus(peer string) *PeerStatus {
	s.peerStatusMu.Lock()
	defer s.peerStatusMu.Unlock()
	status := s.peerStatus[peer]
	if status == nil {
		status = &PeerStatus{Peer: peer, Health: "idle", LastStatus: "registered"}
		s.peerStatus[peer] = status
	}
	return status
}

func (s *Server) peerStatusSnapshot() []*PeerStatus {
	s.peerStatusMu.RLock()
	defer s.peerStatusMu.RUnlock()
	now := time.Now()
	statuses := make([]*PeerStatus, 0, len(s.peerStatus))
	for _, status := range s.peerStatus {
		copyStatus := clonePeerStatus(status)
		copyStatus.CooldownRemainingMs = peerCooldownRemaining(copyStatus, now)
		copyStatus.Health = peerHealth(copyStatus)
		statuses = append(statuses, copyStatus)
	}
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Peer < statuses[j].Peer
	})
	return statuses
}

func summarizePeerStatuses(statuses []*PeerStatus) map[string]int {
	summary := map[string]int{
		"total":    len(statuses),
		"healthy":  0,
		"degraded": 0,
		"failing":  0,
		"warning":  0,
		"idle":     0,
		"cooldown": 0,
		"diverged": 0,
	}
	for _, status := range statuses {
		summary[status.Health]++
	}
	return summary
}

func (s *Server) markPeerSyncStarted(peer string) {
	now := time.Now().UnixMilli()
	s.peerStatusMu.Lock()
	defer s.peerStatusMu.Unlock()
	status := s.peerStatus[peer]
	if status == nil {
		status = &PeerStatus{Peer: peer}
		s.peerStatus[peer] = status
	}
	status.LastStatus = "syncing"
	status.LastSyncStartedAt = now
	status.TotalSyncAttempts++
	status.Health = peerHealth(status)
}

func (s *Server) markPeerSyncSucceeded(peer string, result *PeerSyncResult) {
	now := time.Now().UnixMilli()
	s.peerStatusMu.Lock()
	defer s.peerStatusMu.Unlock()
	status := s.peerStatus[peer]
	if status == nil {
		status = &PeerStatus{Peer: peer}
		s.peerStatus[peer] = status
	}
	status.LastStatus = "synced"
	status.LastError = ""
	status.LastSyncCompletedAt = now
	status.LastSyncSucceededAt = now
	status.TotalSyncSuccesses++
	status.ConsecutiveFailures = 0
	status.CooldownUntil = 0
	if result != nil {
		status.LastAppliedBlocks = result.AppliedBlocks
		status.LastDuplicateBlocks = result.DuplicateBlocks
		status.LastFetchedPages = result.FetchedPages
		status.LastCursorReset = result.CursorReset
		status.LastRetryCount = result.RetryCount
		status.LastKnownLagBlocks = result.LagBlocks
		status.LastRemoteTotalBlocks = result.RemoteTotalBlocks
		status.LastKnownRemotePeerCount = result.RemotePeerCount
		status.LastAdvertisedLatestHash = result.LatestBlockHash
		status.LastRemoteStateHash = result.RemoteStateHash
		status.LastDiscoveredPeers = append([]string(nil), result.DiscoveredPeers...)
		if !result.DivergenceSuspected {
			status.LastDivergenceReason = ""
		}
	}
	status.Health = peerHealth(status)
}

func (s *Server) markPeerSyncFailed(peer string, result *PeerSyncResult, err error) {
	now := time.Now().UnixMilli()
	s.peerStatusMu.Lock()
	defer s.peerStatusMu.Unlock()
	status := s.peerStatus[peer]
	if status == nil {
		status = &PeerStatus{Peer: peer}
		s.peerStatus[peer] = status
	}
	status.LastStatus = "sync_failed"
	status.LastSyncCompletedAt = now
	status.LastSyncFailedAt = now
	status.TotalSyncFailures++
	status.ConsecutiveFailures++
	status.CooldownUntil = now + peerCooldownDuration(status.ConsecutiveFailures).Milliseconds()
	if err != nil {
		status.LastError = err.Error()
	}
	if result != nil {
		status.LastAppliedBlocks = result.AppliedBlocks
		status.LastDuplicateBlocks = result.DuplicateBlocks
		status.LastFetchedPages = result.FetchedPages
		status.LastCursorReset = result.CursorReset
		status.LastRetryCount = result.RetryCount
		status.LastKnownLagBlocks = result.LagBlocks
		status.LastRemoteTotalBlocks = result.RemoteTotalBlocks
		status.LastKnownRemotePeerCount = result.RemotePeerCount
		status.LastAdvertisedLatestHash = result.LatestBlockHash
		status.LastRemoteStateHash = result.RemoteStateHash
		status.LastDiscoveredPeers = append([]string(nil), result.DiscoveredPeers...)
		if result.DivergenceSuspected {
			status.DivergenceCount++
			status.LastDivergenceAt = now
			status.LastDivergenceReason = result.DivergenceReason
		}
	}
	status.Health = peerHealth(status)
}

func (s *Server) markPeerBroadcastAttempt(peer string) {
	now := time.Now().UnixMilli()
	s.peerStatusMu.Lock()
	defer s.peerStatusMu.Unlock()
	status := s.peerStatus[peer]
	if status == nil {
		status = &PeerStatus{Peer: peer}
		s.peerStatus[peer] = status
	}
	status.LastBroadcastAttemptAt = now
}

func (s *Server) markPeerBroadcastSucceeded(peer string) {
	now := time.Now().UnixMilli()
	s.peerStatusMu.Lock()
	defer s.peerStatusMu.Unlock()
	status := s.peerStatus[peer]
	if status == nil {
		status = &PeerStatus{Peer: peer}
		s.peerStatus[peer] = status
	}
	status.LastBroadcastSucceededAt = now
	status.BroadcastSuccesses++
	status.LastBroadcastError = ""
	status.Health = peerHealth(status)
}

func (s *Server) markPeerBroadcastFailed(peer string, err error) {
	now := time.Now().UnixMilli()
	s.peerStatusMu.Lock()
	defer s.peerStatusMu.Unlock()
	status := s.peerStatus[peer]
	if status == nil {
		status = &PeerStatus{Peer: peer}
		s.peerStatus[peer] = status
	}
	status.LastBroadcastFailedAt = now
	status.BroadcastFailures++
	if err != nil {
		status.LastBroadcastError = err.Error()
		status.LastError = err.Error()
	}
	status.Health = peerHealth(status)
}

func (s *Server) markPeerRetryCount(peer string, retries int) {
	s.peerStatusMu.Lock()
	defer s.peerStatusMu.Unlock()
	status := s.peerStatus[peer]
	if status == nil {
		status = &PeerStatus{Peer: peer}
		s.peerStatus[peer] = status
	}
	status.LastRetryCount = retries
	status.Health = peerHealth(status)
}

func (s *Server) peerCooldownRemaining(peer string) int64 {
	s.peerStatusMu.RLock()
	defer s.peerStatusMu.RUnlock()
	return peerCooldownRemaining(s.peerStatus[peer], time.Now())
}

func (s *Server) markPeerSyncSkipped(peer string, remainingMs int64) {
	now := time.Now().UnixMilli()
	s.peerStatusMu.Lock()
	defer s.peerStatusMu.Unlock()
	status := s.peerStatus[peer]
	if status == nil {
		status = &PeerStatus{Peer: peer}
		s.peerStatus[peer] = status
	}
	status.LastStatus = "sync_skipped_cooldown"
	status.LastSkippedSyncAt = now
	status.SkippedSyncs++
	status.CooldownRemainingMs = remainingMs
	status.Health = peerHealth(status)
}

func (s *Server) markPeerBroadcastSkipped(peer string, remainingMs int64) {
	s.peerStatusMu.Lock()
	defer s.peerStatusMu.Unlock()
	status := s.peerStatus[peer]
	if status == nil {
		status = &PeerStatus{Peer: peer}
		s.peerStatus[peer] = status
	}
	status.LastStatus = "broadcast_skipped_cooldown"
	status.SkippedBroadcasts++
	status.CooldownRemainingMs = remainingMs
	status.Health = peerHealth(status)
}

func shortenHash(hash string) string {
	if len(hash) <= 16 {
		return hash
	}
	return hash[:16]
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

func (s *Server) syncPeer(addr string, force bool) (*PeerSyncResult, error) {
	normalized, err := normalizePeerAddr(addr)
	if err != nil {
		return nil, err
	}

	result := &PeerSyncResult{Peer: normalized}
	if remaining := s.peerCooldownRemaining(normalized); remaining > 0 && !force {
		result.SkippedDueToCooldown = true
		result.CooldownRemainingMs = remaining
		s.markPeerSyncSkipped(normalized, remaining)
		return result, nil
	}

	s.markPeerSyncStarted(normalized)
	result.RequestedCursor = s.lattice.LatestBlockHash()
	baseURL := peerBaseURL(normalized)
	client := &http.Client{Timeout: 10 * time.Second}
	cursor := result.RequestedCursor
	usedCursor := cursor != ""

	bootstrap, retries, err := fetchPeerBootstrapWithRetry(client, baseURL)
	result.RetryCount += retries
	if err == nil && bootstrap != nil {
		result.RemoteTotalBlocks = bootstrap.TotalBlocks
		result.RemotePeerCount = bootstrap.PeerCount
		result.LatestBlockHash = bootstrap.LatestBlockHash
		result.RemoteStateHash = bootstrap.StateHash
	}

	for {
		page, retries, err := fetchPeerBlocksWithRetry(client, baseURL, cursor, 200)
		result.RetryCount += retries
		if err != nil {
			s.markPeerSyncFailed(normalized, result, err)
			return result, err
		}
		result.FetchedPages++
		if page.TotalBlocks > result.RemoteTotalBlocks {
			result.RemoteTotalBlocks = page.TotalBlocks
		}
		if page.LatestHash != "" {
			result.LatestBlockHash = page.LatestHash
		}
		if usedCursor && !page.CursorFound {
			result.CursorReset = true
			s.lattice.mu.RLock()
			localBlocks := len(s.lattice.blocks)
			s.lattice.mu.RUnlock()
			if localBlocks > 0 {
				result.DivergenceSuspected = true
				result.DivergenceReason = fmt.Sprintf("remote peer %s does not contain local cursor %s", normalized, shortenHash(result.RequestedCursor))
				divergenceErr := fmt.Errorf("peer divergence suspected: %s", result.DivergenceReason)
				s.markPeerSyncFailed(normalized, result, divergenceErr)
				return result, divergenceErr
			}
			cursor = ""
			usedCursor = false
			continue
		}
		if len(page.Blocks) == 0 {
			break
		}

		for _, block := range page.Blocks {
			accepted, err := s.lattice.ProcessBlockDetailed(block)
			if err != nil {
				wrapped := fmt.Errorf("failed to apply peer block %s: %w", block.Hash, err)
				s.markPeerSyncFailed(normalized, result, wrapped)
				return result, wrapped
			}
			if accepted {
				result.AppliedBlocks++
			} else {
				result.DuplicateBlocks++
			}
			cursor = block.Hash
		}
		usedCursor = false
		if !page.HasMore {
			break
		}
	}

	peers, retries, err := fetchPeerPeersWithRetry(client, baseURL)
	result.RetryCount += retries
	if err == nil {
		result.RemotePeerCount = len(peers)
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
			s.ensurePeerStatus(normalizedPeer)
			known[normalizedPeer] = true
			result.DiscoveredPeers = append(result.DiscoveredPeers, normalizedPeer)
		}
	}

	s.lattice.mu.RLock()
	localBlocks := len(s.lattice.blocks)
	s.lattice.mu.RUnlock()
	if result.RemoteTotalBlocks > localBlocks {
		result.LagBlocks = result.RemoteTotalBlocks - localBlocks
	}

	s.markPeerSyncSucceeded(normalized, result)
	return result, nil
}

func (s *Server) applyPeerReconciliation(addr string, force bool) (*ReconciliationApplyResult, int, error) {
	report, err := s.analyzePeerReconciliation(addr)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}

	result := &ReconciliationApplyResult{
		Peer:            report.Peer,
		Relationship:    report.Relationship,
		SuggestedAction: report.SuggestedAction,
		Reconciliation:  report,
	}

	switch report.Relationship {
	case "both_empty", "in_sync":
		result.Executed = false
		result.ExecutionMode = "noop"
		result.Reason = "Local and remote state do not require any reconciliation action."
		return result, http.StatusOK, nil
	case "remote_ahead", "local_empty_remote_has_state":
		result.ExecutionMode = "remote_to_local_sync"
		syncResult, syncErr := s.syncPeer(addr, force)
		result.Sync = syncResult
		if syncErr != nil {
			result.Reason = "Safe remote-to-local reconciliation was attempted but the sync failed."
			return result, http.StatusBadGateway, syncErr
		}
		if syncResult != nil && syncResult.SkippedDueToCooldown {
			result.Executed = false
			result.Reason = "Safe reconciliation is currently suppressed by peer cooldown policy. Retry later or force explicitly if appropriate."
			return result, http.StatusConflict, fmt.Errorf("reconciliation skipped due to cooldown")
		}
		result.Executed = true
		result.Reason = "Safe remote-to-local reconciliation completed."
		return result, http.StatusOK, nil
	case "local_ahead":
		result.Executed = false
		result.ExecutionMode = "refused"
		result.Reason = "Local node is ahead of the remote peer. Remote-to-local reconciliation is unnecessary, and remote push execution is not implemented in this safe workflow."
		return result, http.StatusConflict, fmt.Errorf("reconciliation refused: local node is ahead of peer")
	case "remote_empty":
		result.Executed = false
		result.ExecutionMode = "refused"
		result.Reason = "Remote peer is empty while local node already has history. Safe reconciliation refuses to overwrite local state or infer that the remote should become canonical."
		return result, http.StatusConflict, fmt.Errorf("reconciliation refused: remote peer is empty")
	case "partially_overlapping", "divergent":
		result.Executed = false
		result.ExecutionMode = "refused"
		result.Reason = "Safe reconciliation refuses to execute when histories are ambiguous or divergent. Use analysis output to investigate before attempting any manual recovery plan."
		return result, http.StatusConflict, fmt.Errorf("reconciliation refused: %s relationship requires manual investigation", report.Relationship)
	default:
		result.Executed = false
		result.ExecutionMode = "refused"
		result.Reason = "Relationship classification is not executable under the current safe reconciliation policy."
		return result, http.StatusConflict, fmt.Errorf("reconciliation refused: unsupported relationship %s", report.Relationship)
	}
}

func (s *Server) analyzePeerReconciliation(addr string) (*PeerReconciliationReport, error) {
	normalized, err := normalizePeerAddr(addr)
	if err != nil {
		return nil, err
	}

	s.lattice.mu.RLock()
	localLatest := ""
	if len(s.lattice.blockOrder) > 0 {
		localLatest = s.lattice.blockOrder[len(s.lattice.blockOrder)-1]
	}
	localStateHash := s.lattice.stateHash
	localTotalBlocks := len(s.lattice.blocks)
	localContains := func(hash string) bool {
		_, ok := s.lattice.blocks[hash]
		return ok
	}
	s.lattice.mu.RUnlock()

	report := &PeerReconciliationReport{
		Peer:                    normalized,
		LocalLatestHash:         localLatest,
		LocalStateHash:          localStateHash,
		LocalTotalBlocks:        localTotalBlocks,
		PeerCooldownRemainingMs: s.peerCooldownRemaining(normalized),
	}

	baseURL := peerBaseURL(normalized)
	client := &http.Client{Timeout: 10 * time.Second}
	bootstrap, _, err := fetchPeerBootstrapWithRetry(client, baseURL)
	if err != nil {
		return nil, err
	}
	report.RemoteLatestHash = bootstrap.LatestBlockHash
	report.RemoteStateHash = bootstrap.StateHash
	report.RemoteTotalBlocks = bootstrap.TotalBlocks

	if report.RemoteLatestHash != "" {
		report.LocalContainsRemoteLatest = localContains(report.RemoteLatestHash)
	}

	switch {
	case localTotalBlocks == 0 && report.RemoteTotalBlocks == 0:
		report.Relationship = "both_empty"
		report.SuggestedAction = "no_action"
		report.Notes = append(report.Notes, "Both local and remote peers are empty.")
		return report, nil
	case localTotalBlocks == 0 && report.RemoteTotalBlocks > 0:
		report.RemoteContainsLocalCursor = true
		report.Relationship = "local_empty_remote_has_state"
		report.SuggestedAction = "bootstrap_from_peer"
		report.Notes = append(report.Notes, "Local node has no blocks while the remote peer already has confirmed history.")
		return report, nil
	case localTotalBlocks > 0 && report.RemoteTotalBlocks == 0:
		report.Relationship = "remote_empty"
		report.SuggestedAction = "do_not_sync_reset_remote_or_wait"
		report.Notes = append(report.Notes, "Remote peer is empty while local node already has confirmed history.")
		return report, nil
	}

	page, _, err := fetchPeerBlocksWithRetry(client, baseURL, localLatest, 1)
	if err != nil {
		return nil, err
	}
	report.RemoteContainsLocalCursor = page.CursorFound

	if report.RemoteContainsLocalCursor {
		switch {
		case report.RemoteStateHash == report.LocalStateHash && report.RemoteTotalBlocks == report.LocalTotalBlocks:
			report.Relationship = "in_sync"
			report.SuggestedAction = "no_action"
			report.Notes = append(report.Notes, "Local and remote peers report the same state hash and block count.")
		case report.RemoteTotalBlocks > report.LocalTotalBlocks:
			report.Relationship = "remote_ahead"
			report.SuggestedAction = "bootstrap_from_peer"
			report.Notes = append(report.Notes, fmt.Sprintf("Remote peer appears ahead by approximately %d block(s).", report.RemoteTotalBlocks-report.LocalTotalBlocks))
		case report.LocalContainsRemoteLatest && report.LocalTotalBlocks >= report.RemoteTotalBlocks:
			report.Relationship = "local_ahead"
			report.SuggestedAction = "wait_or_sync_remote_from_local"
			report.Notes = append(report.Notes, "Local node already contains the remote peer's advertised head block.")
		default:
			report.Relationship = "partially_overlapping"
			report.SuggestedAction = "investigate_state_hash_mismatch"
			report.Notes = append(report.Notes, "Remote peer recognizes the local cursor, but state summary still differs.")
		}
		return report, nil
	}

	report.Relationship = "divergent"
	report.SuggestedAction = "investigate_divergence"
	report.Notes = append(report.Notes, "Remote peer does not contain the local ordered-history cursor on a non-empty local node.")
	if report.LocalContainsRemoteLatest {
		report.Notes = append(report.Notes, "Local node still contains the remote head block, suggesting the remote may be stale or truncated rather than entirely unrelated.")
	}
	return report, nil
}

func retryFetch(attempts int, delay time.Duration, op func() error) (int, error) {
	if attempts <= 0 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if err := op(); err != nil {
			lastErr = err
			if attempt < attempts-1 {
				time.Sleep(delay * time.Duration(attempt+1))
			}
			continue
		}
		return attempt, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("retry operation failed")
	}
	return attempts - 1, lastErr
}

func fetchPeerBootstrap(client *http.Client, baseURL string) (*peerBootstrapResponse, error) {
	resp, err := client.Get(baseURL + "/bootstrap")
	if err != nil {
		return nil, fmt.Errorf("fetch peer bootstrap failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("peer bootstrap returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var summary peerBootstrapResponse
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, fmt.Errorf("decode peer bootstrap failed: %w", err)
	}
	return &summary, nil
}

func fetchPeerBootstrapWithRetry(client *http.Client, baseURL string) (*peerBootstrapResponse, int, error) {
	var result *peerBootstrapResponse
	retriesUsed, err := retryFetch(3, 150*time.Millisecond, func() error {
		var fetchErr error
		result, fetchErr = fetchPeerBootstrap(client, baseURL)
		return fetchErr
	})
	return result, retriesUsed, err
}

func fetchPeerBlocks(client *http.Client, baseURL, after string, limit int) (*peerBlocksResponse, error) {
	if limit <= 0 {
		limit = 200
	}
	endpoint := fmt.Sprintf("%s/blocks?limit=%d", baseURL, limit)
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

func fetchPeerBlocksWithRetry(client *http.Client, baseURL, after string, limit int) (*peerBlocksResponse, int, error) {
	var result *peerBlocksResponse
	retriesUsed, err := retryFetch(3, 150*time.Millisecond, func() error {
		var fetchErr error
		result, fetchErr = fetchPeerBlocks(client, baseURL, after, limit)
		return fetchErr
	})
	return result, retriesUsed, err
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

func fetchPeerPeersWithRetry(client *http.Client, baseURL string) ([]string, int, error) {
	var result []string
	retriesUsed, err := retryFetch(2, 100*time.Millisecond, func() error {
		var fetchErr error
		result, fetchErr = fetchPeerPeers(client, baseURL)
		return fetchErr
	})
	return result, retriesUsed, err
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
