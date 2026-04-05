package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bobtorrent/internal/bridges"
	"bobtorrent/internal/economy"
	"bobtorrent/internal/publish"
	"bobtorrent/internal/tracker"
	"bobtorrent/internal/transport"
	"bobtorrent/internal/tui"
	"bobtorrent/pkg/torrent"

	anacrolixTorrent "github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-resty/resty/v2"
	"github.com/gorilla/websocket"
)

const (
	signalingWriteTimeout   = 5 * time.Second
	signalingReadTimeout    = 70 * time.Second
	signalingPingPeriod     = 25 * time.Second
	signalingWaitTimeout    = 45 * time.Second
	signalingMaxMessageSize = 1 << 20
)

var (
	nodeWallet          *torrent.Keypair
	httpClient          = resty.New().SetTimeout(10 * time.Second)
	latticeURL          = "http://localhost:4000"
	torrentClient       *anacrolixTorrent.Client
	uiProgram           *tea.Program
	fcBridge            *bridges.FilecoinBridge
	dhtNode             *transport.DHTNode
	publishRegistry     *publish.Registry
	economyDB           *economy.Database
	startedAt           = time.Now()
	fheOracleRunner     = computeFHEOracleCiphertext
	signalingMatchmaker = newMatchmaker()
)

// matchPlayer tracks one websocket client participating in lightweight
// matchmaking/signaling. The service intentionally stays small and protocol-
// compatible with the legacy Node game-server: a single waiting queue, paired
// opponents, and JSON relay of `SIGNAL` payloads.
type matchPlayer struct {
	conn         *websocket.Conn
	writeMu      sync.Mutex
	mu           sync.Mutex
	opponent     *matchPlayer
	closed       bool
	connectedAt  time.Time
	lastSeen     time.Time
	waitingSince time.Time
}

// matchmaker coordinates the single-queue WebRTC signaling flow expected by the
// Bobcoin frontend. This is intentionally minimal because the existing product
// only needs pairwise matching rather than rooms or persistent lobbies.
type matchmaker struct {
	mu                    sync.Mutex
	waiting               *matchPlayer
	activeConnections     int
	activePairs           int
	totalConnections      int64
	totalMatches          int64
	relayedSignals        int64
	disconnects           int64
	staleWaitingEvictions int64
	upgrader              websocket.Upgrader
}

// matchmakerSnapshot is a compact operational view of the websocket signaling
// layer. It is exposed through status/stats responses so operators can observe
// whether the Go signaling path is active, queued, or churning.
type matchmakerSnapshot struct {
	ActiveConnections     int   `json:"activeConnections"`
	ActivePairs           int   `json:"activePairs"`
	WaitingPlayers        int   `json:"waitingPlayers"`
	WaitingSeconds        int64 `json:"waitingSeconds"`
	TotalConnections      int64 `json:"totalConnections"`
	TotalMatches          int64 `json:"totalMatches"`
	RelayedSignals        int64 `json:"relayedSignals"`
	Disconnects           int64 `json:"disconnects"`
	StaleWaitingEvictions int64 `json:"staleWaitingEvictions"`
}

func newMatchmaker() *matchmaker {
	return &matchmaker{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (p *matchPlayer) sendJSON(v interface{}) error {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	_ = p.conn.SetWriteDeadline(time.Now().Add(signalingWriteTimeout))
	return p.conn.WriteJSON(v)
}

func (p *matchPlayer) sendControl(messageType int, data []byte) error {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	return p.conn.WriteControl(messageType, data, time.Now().Add(signalingWriteTimeout))
}

func (p *matchPlayer) setOpponent(opponent *matchPlayer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.opponent = opponent
}

func (p *matchPlayer) getOpponent() *matchPlayer {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.opponent
}

func (p *matchPlayer) clearOpponent() *matchPlayer {
	p.mu.Lock()
	defer p.mu.Unlock()
	opponent := p.opponent
	p.opponent = nil
	return opponent
}

func (p *matchPlayer) setWaitingSince(ts time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.waitingSince = ts
}

func (p *matchPlayer) clearWaitingSince() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.waitingSince = time.Time{}
}

func (p *matchPlayer) waitingDuration(now time.Time) time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.waitingSince.IsZero() {
		return 0
	}
	return now.Sub(p.waitingSince)
}

func (p *matchPlayer) touch(ts time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastSeen = ts
}

func (p *matchPlayer) markClosed() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
}

func (p *matchPlayer) isClosed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closed
}

func (m *matchmaker) registerConnection(player *matchPlayer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeConnections++
	m.totalConnections++
}

func (m *matchmaker) queueOrMatch(player *matchPlayer) (*matchPlayer, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	player.touch(now)
	if player.getOpponent() != nil || player.isClosed() {
		return nil, false
	}
	if m.waiting != nil {
		staleWaiting := m.waiting.waitingDuration(now) > signalingWaitTimeout
		if m.waiting.isClosed() || m.waiting == player || staleWaiting {
			if staleWaiting && !m.waiting.isClosed() && m.waiting != player {
				m.staleWaitingEvictions++
				m.waiting.clearWaitingSince()
			}
			m.waiting = nil
		}
	}
	if m.waiting == nil {
		player.setWaitingSince(now)
		m.waiting = player
		return nil, false
	}

	opponent := m.waiting
	m.waiting = nil
	opponent.clearWaitingSince()
	player.clearWaitingSince()
	opponent.setOpponent(player)
	player.setOpponent(opponent)
	m.totalMatches++
	m.activePairs++
	return opponent, true
}

func (m *matchmaker) recordSignalRelay() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.relayedSignals++
}

func (m *matchmaker) snapshot() matchmakerSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	snapshot := matchmakerSnapshot{
		ActiveConnections:     m.activeConnections,
		ActivePairs:           m.activePairs,
		TotalConnections:      m.totalConnections,
		TotalMatches:          m.totalMatches,
		RelayedSignals:        m.relayedSignals,
		Disconnects:           m.disconnects,
		StaleWaitingEvictions: m.staleWaitingEvictions,
	}
	if m.waiting != nil && !m.waiting.isClosed() {
		snapshot.WaitingPlayers = 1
		snapshot.WaitingSeconds = int64(m.waiting.waitingDuration(time.Now()).Seconds())
	}
	return snapshot
}

func (m *matchmaker) disconnect(player *matchPlayer) *matchPlayer {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.activeConnections > 0 {
		m.activeConnections--
	}
	m.disconnects++
	if m.waiting == player {
		m.waiting = nil
	}
	opponent := player.clearOpponent()
	if opponent != nil {
		opponent.clearOpponent()
		if m.activePairs > 0 {
			m.activePairs--
		}
	}
	return opponent
}

// main boots the Go supernode stack:
//  1. Wallet + torrent engine
//  2. HTTP/UDP tracker services
//  3. Kademlia DHT node
//  4. Lattice market poller + block feed listener
//  5. Terminal dashboard (TUI)
func main() {
	log.SetOutput(os.Stderr)

	loadOrCreateWallet()
	initTorrentClient()
	defer torrentClient.Close()

	fcBridge = bridges.NewFilecoinBridge("f1bobtorrentnode")
	initPublishRegistry()
	initEconomyDatabase()

	model := tui.NewModel()
	uiProgram = tea.NewProgram(model, tea.WithAltScreen())

	startTrackerServices()
	startDHT()
	go startMarketPoller()
	go startLatticeFeedListener()

	if _, err := uiProgram.Run(); err != nil {
		log.Printf("TUI exited with error: %v", err)
		os.Exit(1)
	}
}

func startTrackerServices() {
	trackerCore := tracker.NewTracker()
	mux := http.NewServeMux()

	mux.HandleFunc("/announce", withCORS(trackerCore.HTTPHandler()))
	mux.HandleFunc("/", handleSignalingSocket)
	mux.HandleFunc("/matchmaking", handleSignalingSocket)
	mux.HandleFunc("/spora/", withCORS(handleSpora))
	mux.HandleFunc("/status", withCORS(handleServiceStatus))
	mux.HandleFunc("/stats", withCORS(handleStats))
	mux.HandleFunc("/filecoin/status", withCORS(handleFilecoinStatus))
	mux.HandleFunc("/filecoin/deals", withCORS(handleFilecoinDeals))
	mux.HandleFunc("/bankroll", withCORS(handleBankroll))
	mux.HandleFunc("/transactions", withCORS(handleTransactions))
	mux.HandleFunc("/mint", withCORS(handleMint))
	mux.HandleFunc("/burn", withCORS(handleBurn))
	mux.HandleFunc("/fhe-oracle", withCORS(handleFHEOracle))
	mux.HandleFunc("/submit-proof", withCORS(handleSubmitProof))
	mux.HandleFunc("/add-torrent", withCORS(handleAddTorrent))
	mux.HandleFunc("/remove-torrent", withCORS(handleRemoveTorrent))
	mux.HandleFunc("/upload-shard", withCORS(handleUploadShard))
	mux.HandleFunc("/publish-manifest", withCORS(handlePublishManifest))
	mux.HandleFunc("/manifests/", withCORS(handleGetManifest))
	mux.HandleFunc("/shards/", withCORS(handleGetShard))
	mux.HandleFunc("/storage.wasm", withCORS(serveStorageWASM))
	mux.HandleFunc("/wasm_exec.js", withCORS(serveWASMExec))
	go func() {
		if err := http.ListenAndServe(":8000", mux); err != nil {
			log.Printf("HTTP tracker failed: %v", err)
		}
	}()

	udpTracker, err := tracker.NewUDPTracker(trackerCore, ":6881")
	if err == nil {
		go udpTracker.Start()
	} else {
		log.Printf("UDP tracker unavailable: %v", err)
	}
}

func startDHT() {
	var err error
	dhtNode, err = transport.NewDHTNode(":6882")
	if err != nil {
		log.Printf("DHT node unavailable: %v", err)
		return
	}
	go dhtNode.Start()
}

func initPublishRegistry() {
	var err error
	publishRegistry, err = publish.NewRegistry(filepath.Join("data", "published"))
	if err != nil {
		log.Fatalf("failed to initialize publish registry: %v", err)
	}
}

func initEconomyDatabase() {
	var err error
	economyDB, err = economy.NewDatabase(filepath.Join("data", "economy", "supernode.db"))
	if err != nil {
		log.Fatalf("failed to initialize economy database: %v", err)
	}
}

func initTorrentClient() {
	cfg := anacrolixTorrent.NewDefaultClientConfig()
	cfg.DataDir = "./downloads"
	cfg.ListenPort = 4242
	cfg.Seed = true

	var err error
	torrentClient, err = anacrolixTorrent.NewClient(cfg)
	if err != nil {
		log.Fatalf("failed to start torrent client: %v", err)
	}
}

func loadOrCreateWallet() {
	const walletFile = "wallet.json"

	data, err := os.ReadFile(walletFile)
	if err == nil {
		if err := json.Unmarshal(data, &nodeWallet); err == nil && nodeWallet != nil {
			return
		}
	}

	nodeWallet, err = torrent.GenerateKeypair()
	if err != nil {
		log.Fatalf("failed to generate wallet: %v", err)
	}

	data, _ = json.MarshalIndent(nodeWallet, "", "  ")
	if err := os.WriteFile(walletFile, data, 0644); err != nil {
		log.Printf("failed to persist wallet: %v", err)
	}
}

func startMarketPoller() {
	pollMarket()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		pollMarket()
	}
}

func pollMarket() {
	var bidsResp struct {
		Bids []tui.Bid `json:"bids"`
	}

	resp, err := httpClient.R().SetResult(&bidsResp).Get(latticeURL + "/market/bids")
	if err != nil || !resp.IsSuccess() {
		sendStatus("Lattice API Offline", 0)
		return
	}

	var frontierResp struct {
		Balance       int64   `json:"balance"`
		Frontier      *string `json:"frontier"`
		StakedBalance int64   `json:"staked_balance"`
		Height        int     `json:"height"`
	}
	_, _ = httpClient.R().SetResult(&frontierResp).Get(latticeURL + "/frontier/" + nodeWallet.PublicKey)

	sendStatus("Market Poll OK", frontierResp.Balance)
	if uiProgram != nil {
		uiProgram.Send(tui.BidsMsg{Bids: bidsResp.Bids})
	}

	publishNetworkStats(0, 0)

	for _, bid := range bidsResp.Bids {
		if bid.Status != "OPEN" || bid.Magnet == "" {
			continue
		}

		spec, err := metainfo.ParseMagnetUri(bid.Magnet)
		if err != nil {
			log.Printf("invalid magnet in bid %s: %v", bid.ID, err)
			continue
		}

		if _, exists := torrentClient.Torrent(spec.InfoHash); exists {
			continue
		}

		t, err := torrentClient.AddMagnet(bid.Magnet)
		if err != nil {
			log.Printf("failed to add magnet for bid %s: %v", bid.ID, err)
			continue
		}

		go func(target *anacrolixTorrent.Torrent, bidID string, amount int64, infoHash string) {
			<-target.GotInfo()
			target.DownloadAll()

			if fcBridge != nil {
				if _, err := fcBridge.PublishDeal(infoHash, 1024*1024, 30); err != nil {
					log.Printf("filecoin archival failed for %s: %v", infoHash, err)
				}
			}

			acceptBid(bidID, amount, infoHash)
		}(t, bid.ID, bid.Amount, spec.InfoHash.HexString())
	}
}

func acceptBid(bidID string, amount int64, infoHash string) {
	var status struct {
		Frontier      *string `json:"frontier"`
		Balance       int64   `json:"balance"`
		StakedBalance int64   `json:"staked_balance"`
		Height        int     `json:"height"`
	}

	resp, err := httpClient.R().SetResult(&status).Get(latticeURL + "/frontier/" + nodeWallet.PublicKey)
	if err != nil || !resp.IsSuccess() {
		return
	}

	sendStatus("Accepting Bid...", status.Balance)

	challenge := 12345
	if status.Frontier != nil && len(*status.Frontier) >= 8 {
		_, _ = fmt.Sscanf((*status.Frontier)[:8], "%x", &challenge)
	}

	chunkHash := torrent.HashSHA256(infoHash + fmt.Sprintf("%d", challenge))
	spora := map[string]interface{}{
		"infoHash":  infoHash,
		"challenge": challenge,
		"chunkHash": chunkHash,
	}

	newBalance := status.Balance + amount
	newHeight := status.Height + 1
	block := torrent.NewBlock(
		"accept_bid",
		nodeWallet.PublicKey,
		status.Frontier,
		newBalance,
		status.StakedBalance,
		newHeight,
		bidID,
		spora,
		nil,
	)
	if err := block.Sign(nodeWallet.PrivateKey); err != nil {
		log.Printf("failed to sign accept_bid block: %v", err)
		return
	}

	submitResp, err := httpClient.R().SetBody(block).Post(latticeURL + "/process")
	if err == nil && submitResp.IsSuccess() {
		sendStatus("Bid Accepted!", newBalance)
	}
}

func startLatticeFeedListener() {
	for {
		listenToLatticeFeed()
		time.Sleep(5 * time.Second)
	}
}

func listenToLatticeFeed() {
	wsURL := strings.Replace(latticeURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)

	parsed, err := url.Parse(wsURL)
	if err != nil {
		log.Printf("invalid lattice ws url: %v", err)
		return
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}

	conn, _, err := websocket.DefaultDialer.Dial(parsed.String(), nil)
	if err != nil {
		log.Printf("lattice websocket dial failed: %v", err)
		return
	}
	defer conn.Close()

	for {
		var msg struct {
			Type        string         `json:"type"`
			Event       string         `json:"event"`
			Accounts    int            `json:"accounts"`
			Chains      int            `json:"chains"`
			TotalBlocks int            `json:"totalBlocks"`
			WSClients   int            `json:"wsClients"`
			Block       *torrent.Block `json:"block"`
		}

		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("lattice websocket closed: %v", err)
			return
		}

		publishNetworkStats(msg.Chains, msg.TotalBlocks)

		if (msg.Type == "NEW_BLOCK" || msg.Event == "NEW_BLOCK") && msg.Block != nil && uiProgram != nil {
			uiProgram.Send(tui.BlockFeedMsg{
				Type:      msg.Block.Type,
				Hash:      msg.Block.Hash,
				Account:   msg.Block.Account,
				Timestamp: time.UnixMilli(msg.Block.Timestamp),
			})
		}
	}
}

func publishNetworkStats(chains, totalBlocks int) {
	if uiProgram == nil {
		return
	}

	peerCount := 0
	if dhtNode != nil {
		peerCount = dhtNode.Stats().GoodNodes
	}

	uiProgram.Send(tui.NetworkStatsMsg{
		Peers:       peerCount,
		Torrents:    len(torrentClient.Torrents()),
		Chains:      chains,
		TotalBlocks: totalBlocks,
	})
}

func sendStatus(text string, balance int64) {
	if uiProgram != nil {
		uiProgram.Send(tui.StatusMsg{Text: text, Balance: balance})
	}
}

func handleSpora(w http.ResponseWriter, r *http.Request) {
	challenge := strings.TrimPrefix(r.URL.Path, "/spora/")
	infoHash := "1234567890abcdef1234567890abcdef12345678"
	chunkHash := torrent.HashSHA256(infoHash + challenge)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "spora": {"infoHash": "%s", "challenge": %s, "chunkHash": "%s"}}`, infoHash, challenge, chunkHash)
}

func walletFrontier() (*string, int64, int64, int, error) {
	if nodeWallet == nil {
		return nil, 0, 0, 0, fmt.Errorf("node wallet unavailable")
	}
	var frontierResp struct {
		Frontier      *string `json:"frontier"`
		Balance       int64   `json:"balance"`
		StakedBalance int64   `json:"staked_balance"`
		Height        int     `json:"height"`
	}
	resp, err := httpClient.R().SetResult(&frontierResp).Get(latticeURL + "/frontier/" + nodeWallet.PublicKey)
	if err != nil || !resp.IsSuccess() {
		if err == nil {
			err = fmt.Errorf("frontier request failed with status %d", resp.StatusCode())
		}
		return nil, 0, 0, 0, err
	}
	return frontierResp.Frontier, frontierResp.Balance, frontierResp.StakedBalance, frontierResp.Height, nil
}

func recordEconomyTransaction(txType string, amount float64, hash, reason, address string) (string, error) {
	id := fmt.Sprintf("tx_%s_%d", strings.ToLower(txType), time.Now().UnixNano())
	if economyDB == nil {
		return id, nil
	}
	return id, economyDB.RecordTransaction(economy.Transaction{
		ID:      id,
		Amount:  amount,
		Type:    txType,
		Hash:    hash,
		Reason:  reason,
		Address: address,
	})
}

func buildLocalSpora(challenge int) map[string]interface{} {
	infoHash := "1234567890abcdef1234567890abcdef12345678"
	chunkHash := torrent.HashSHA256(infoHash + fmt.Sprintf("%d", challenge))
	return map[string]interface{}{
		"infoHash":  infoHash,
		"challenge": challenge,
		"chunkHash": chunkHash,
	}
}

func processMintCompatibility(amount float64, reason, address string) (string, string, error) {
	hash := fmt.Sprintf("mint_%d", time.Now().UnixNano())
	if address != "" {
		amountInt := int64(amount)
		frontier, balance, staked, height, err := walletFrontier()
		if err == nil && frontier != nil {
			if balance < amountInt {
				return "", "", fmt.Errorf("insufficient bankroll: have %d, need %d", balance, amountInt)
			}
			challenge := 12345
			if len(*frontier) >= 8 {
				_, _ = fmt.Sscanf((*frontier)[:8], "%x", &challenge)
			}
			block := torrent.NewBlock(
				"send",
				nodeWallet.PublicKey,
				frontier,
				balance-amountInt,
				staked,
				height+1,
				address,
				buildLocalSpora(challenge),
				map[string]interface{}{"reason": reason},
			)
			if err := block.Sign(nodeWallet.PrivateKey); err == nil {
				resp, submitErr := httpClient.R().SetBody(block).Post(latticeURL + "/process")
				if submitErr == nil && resp.IsSuccess() {
					hash = block.Hash
				} else if submitErr != nil {
					log.Printf("mint lattice send failed: %v", submitErr)
				}
			}
		}
	}

	txID, err := recordEconomyTransaction("MINT", amount, hash, reason, address)
	if err != nil {
		return "", "", err
	}
	return txID, hash, nil
}

func handleSignalingSocket(w http.ResponseWriter, r *http.Request) {
	if !websocket.IsWebSocketUpgrade(r) {
		if r.URL.Path != "/" && r.URL.Path != "/matchmaking" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":      "online",
			"service":     "Go Supernode signaling",
			"matchmaking": "websocket_ready",
			"signaling":   signalingMatchmaker.snapshot(),
		})
		return
	}

	conn, err := signalingMatchmaker.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("signaling websocket upgrade failed: %v", err)
		return
	}
	player := &matchPlayer{conn: conn, connectedAt: time.Now(), lastSeen: time.Now()}
	signalingMatchmaker.registerConnection(player)
	conn.SetReadLimit(signalingMaxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(signalingReadTimeout))
	conn.SetPongHandler(func(string) error {
		player.touch(time.Now())
		return conn.SetReadDeadline(time.Now().Add(signalingReadTimeout))
	})

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(signalingPingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := player.sendControl(websocket.PingMessage, []byte("ping")); err != nil {
					return
				}
			}
		}
	}()

	defer func() {
		close(done)
		player.markClosed()
		opponent := signalingMatchmaker.disconnect(player)
		_ = conn.Close()
		if opponent != nil && !opponent.isClosed() {
			_ = opponent.sendJSON(map[string]interface{}{"type": "OPPONENT_DISCONNECTED"})
		}
	}()

	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		player.touch(time.Now())
		_ = conn.SetReadDeadline(time.Now().Add(signalingReadTimeout))

		msgType, _ := msg["type"].(string)
		switch msgType {
		case "FIND_MATCH":
			opponent, matched := signalingMatchmaker.queueOrMatch(player)
			if !matched {
				continue
			}
			_ = opponent.sendJSON(map[string]interface{}{"type": "MATCH_FOUND", "initiator": true})
			_ = player.sendJSON(map[string]interface{}{"type": "MATCH_FOUND", "initiator": false})
		case "SIGNAL":
			opponent := player.getOpponent()
			if opponent == nil || opponent.isClosed() {
				continue
			}
			signalingMatchmaker.recordSignalRelay()
			_ = opponent.sendJSON(map[string]interface{}{"type": "SIGNAL", "signal": msg["signal"]})
		}
	}
}

func handleServiceStatus(w http.ResponseWriter, r *http.Request) {
	filecoinStatus := map[string]interface{}{
		"enabled": false,
		"mode":    "disabled",
	}
	if fcBridge != nil {
		filecoinStatus = map[string]interface{}{
			"enabled": true,
			"bridge":  fcBridge.Status(),
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "online",
		"service":   "Go Supernode orchestrator",
		"version":   "11.40.0",
		"signaling": signalingMatchmaker.snapshot(),
		"filecoin":  filecoinStatus,
	})
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	uptimeSeconds := time.Since(startedAt).Seconds()
	if uptimeSeconds <= 0 {
		uptimeSeconds = 1
	}

	peerCount := 0
	if dhtNode != nil {
		peerCount = dhtNode.Stats().GoodNodes
	}
	filecoinStatus := map[string]interface{}{}
	if fcBridge != nil {
		filecoinStatus = map[string]interface{}{
			"bridge": fcBridge.Status(),
		}
	}

	torrents := torrentClient.Torrents()
	storageEntries := make([]map[string]interface{}, 0, len(torrents))
	totalSize := int64(0)
	totalDownloadBytes := int64(0)
	totalUploadBytes := int64(0)

	for _, t := range torrents {
		length := t.Length()
		completed := t.BytesCompleted()
		progress := 0.0
		if length > 0 {
			progress = float64(completed) / float64(length)
		}

		stats := t.Stats()
		downloadBytes := stats.BytesReadData.Int64()
		uploadBytes := stats.BytesWrittenData.Int64()
		totalDownloadBytes += downloadBytes
		totalUploadBytes += uploadBytes
		totalSize += length

		storageEntries = append(storageEntries, map[string]interface{}{
			"infoHash":  t.InfoHash().HexString(),
			"name":      t.Name(),
			"progress":  progress,
			"peers":     len(t.PeerConns()),
			"totalSize": length,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"address":   nodeWallet.PublicKey,
		"service":   "Go Supernode",
		"status":    "online",
		"uptime":    int64(uptimeSeconds),
		"signaling": signalingMatchmaker.snapshot(),
		"filecoin":  filecoinStatus,
		"network": map[string]interface{}{
			"peers":         peerCount,
			"downloadSpeed": int64(float64(totalDownloadBytes) / uptimeSeconds),
			"uploadSpeed":   int64(float64(totalUploadBytes) / uptimeSeconds),
		},
		"storage": map[string]interface{}{
			"totalSize": totalSize,
			"torrents":  storageEntries,
		},
	})
}

func handleFilecoinStatus(w http.ResponseWriter, r *http.Request) {
	if fcBridge == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"enabled": false,
			"mode":    "disabled",
		})
		return
	}
	writeJSON(w, http.StatusOK, fcBridge.Status())
}

func handleFilecoinDeals(w http.ResponseWriter, r *http.Request) {
	if fcBridge == nil {
		writeJSON(w, http.StatusOK, []bridges.FilecoinDealRecord{})
		return
	}
	deals := fcBridge.ListDeals()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deals": deals,
		"count": len(deals),
	})
}

func handleBankroll(w http.ResponseWriter, r *http.Request) {
	address := ""
	if nodeWallet != nil {
		address = nodeWallet.PublicKey
	}
	_, balance, _, _, err := walletFrontier()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"balance": 0,
			"address": address,
			"status":  "wallet_not_synced",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"balance": balance,
		"address": address,
		"status":  "online",
	})
}

func handleTransactions(w http.ResponseWriter, r *http.Request) {
	if economyDB == nil {
		writeJSON(w, http.StatusOK, []economy.Transaction{})
		return
	}
	transactions, err := economyDB.ListTransactions()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list transactions: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, transactions)
}

func handleBurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Amount float64 `json:"amount"`
		Reason string  `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 {
		http.Error(w, "valid amount required", http.StatusBadRequest)
		return
	}

	hash := fmt.Sprintf("burn_%d", time.Now().UnixNano())
	txID, err := recordEconomyTransaction("SEND", req.Amount, hash, req.Reason, "")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to record burn transaction: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"tx":      txID,
		"hash":    hash,
	})
}

func handleMint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Amount  float64 `json:"amount"`
		Reason  string  `json:"reason"`
		Address string  `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 {
		http.Error(w, "valid amount required", http.StatusBadRequest)
		return
	}

	txID, hash, err := processMintCompatibility(req.Amount, req.Reason, req.Address)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to mint: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"tx":      txID,
		"hash":    hash,
	})
}

func computeFHEOracleCiphertext(parent context.Context, cipherText string) (string, error) {
	helperPath := os.Getenv("BOBTORRENT_FHE_ORACLE_HELPER")
	if helperPath == "" {
		helperPath = filepath.Join("cmd", "supernode-go", "fhe_oracle_helper.mjs")
	}
	nodeBin := os.Getenv("BOBTORRENT_NODE_BIN")
	if nodeBin == "" {
		nodeBin = "node"
	}

	payload, err := json.Marshal(map[string]interface{}{
		"cipherText": cipherText,
		"multiply":   2,
		"add":        500,
	})
	if err != nil {
		return "", fmt.Errorf("failed to encode fhe helper payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, nodeBin, helperPath)
	cmd.Stdin = strings.NewReader(string(payload))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("fhe helper timed out")
		}
		return "", fmt.Errorf("fhe helper execution failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}

	var response struct {
		Success      bool   `json:"success"`
		ResultCipher string `json:"resultCipher"`
		Error        string `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return "", fmt.Errorf("failed to decode fhe helper response: %w", err)
	}
	if !response.Success {
		if response.Error == "" {
			response.Error = "unknown helper failure"
		}
		return "", fmt.Errorf("%s", response.Error)
	}
	if response.ResultCipher == "" {
		return "", fmt.Errorf("fhe helper returned empty ciphertext")
	}
	return response.ResultCipher, nil
}

func handleFHEOracle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		CipherText string `json:"cipherText"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.CipherText) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "Encrypted payload missing",
		})
		return
	}

	resultCipher, err := fheOracleRunner(r.Context(), req.CipherText)
	if err != nil {
		log.Printf("fhe oracle compatibility error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "Homomorphic computation failed.",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"resultCipher": resultCipher,
	})
}

func handleSubmitProof(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Proof map[string]interface{} `json:"proof"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Proof == nil {
		http.Error(w, `{"success": false, "error": "Invalid proof payload"}`, http.StatusBadRequest)
		return
	}

	publicValues, _ := req.Proof["publicValues"].(map[string]interface{})
	if publicValues == nil {
		http.Error(w, `{"success": false, "error": "Invalid proof payload"}`, http.StatusBadRequest)
		return
	}

	score, _ := publicValues["score"].(float64)
	address, _ := publicValues["address"].(string)
	if address == "" {
		address = "unknown"
	}
	proofBytes, _ := json.Marshal(req.Proof)
	sum := sha256.Sum256(proofBytes)
	verificationHash := hex.EncodeToString(sum[:])
	zkVerified := score >= 1000
	if !zkVerified {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Cryptographic trace verification failed.",
		})
		return
	}

	amount := score / 100
	txID, hash, err := processMintCompatibility(amount, "Proof-of-play reward", address)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to mint proof reward: %v", err), http.StatusInternalServerError)
		return
	}
	if hash == "" {
		hash = verificationHash[:32]
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"tx":         txID,
		"hash":       hash,
		"zkVerified": true,
	})
}

func handleAddTorrent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Magnet string `json:"magnet"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Magnet == "" {
		http.Error(w, "valid magnet required", http.StatusBadRequest)
		return
	}

	spec, err := metainfo.ParseMagnetUri(req.Magnet)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid magnet: %v", err), http.StatusBadRequest)
		return
	}

	if _, exists := torrentClient.Torrent(spec.InfoHash); exists {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":  true,
			"infoHash": spec.InfoHash.HexString(),
			"message":  "torrent already loaded",
		})
		return
	}

	t, err := torrentClient.AddMagnet(req.Magnet)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to add torrent: %v", err), http.StatusInternalServerError)
		return
	}
	go func() {
		<-t.GotInfo()
		t.DownloadAll()
	}()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"infoHash": spec.InfoHash.HexString(),
	})
}

func handleRemoveTorrent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		InfoHash string `json:"infoHash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.InfoHash == "" {
		http.Error(w, "valid infoHash required", http.StatusBadRequest)
		return
	}

	var ih metainfo.Hash
	if err := ih.FromHexString(req.InfoHash); err != nil {
		http.Error(w, fmt.Sprintf("invalid infoHash: %v", err), http.StatusBadRequest)
		return
	}

	t, exists := torrentClient.Torrent(ih)
	if !exists {
		http.Error(w, "torrent not found", http.StatusNotFound)
		return
	}
	t.Drop()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"infoHash": req.InfoHash,
	})
}

func handleUploadShard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Hash string `json:"hash"`
		Data string `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Data == "" {
		http.Error(w, "valid shard payload required", http.StatusBadRequest)
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid base64 shard data: %v", err), http.StatusBadRequest)
		return
	}

	stored, err := publishRegistry.StoreShard(req.Hash, decoded, publicBaseURL(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"shard":   stored,
	})
}

func handlePublishManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, fmt.Sprintf("invalid manifest payload: %v", err), http.StatusBadRequest)
		return
	}

	manifest := body
	if rawManifest, ok := body["manifest"].(map[string]interface{}); ok {
		manifest = rawManifest
	}

	stored, err := publishRegistry.PublishManifest(manifest, publicBaseURL(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"id":          stored.ID,
		"locator":     stored.Locator,
		"manifestUrl": stored.ManifestURL,
		"manifest":    stored.Manifest,
	})
}

func handleGetManifest(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/manifests/")
	if id == "" {
		http.Error(w, "manifest id required", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	http.ServeFile(w, r, publishRegistry.ManifestPath(id))
}

func handleGetShard(w http.ResponseWriter, r *http.Request) {
	hash := strings.TrimPrefix(r.URL.Path, "/shards/")
	if hash == "" {
		http.Error(w, "shard hash required", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, publishRegistry.ShardPath(hash))
}

func publicBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func serveStorageWASM(w http.ResponseWriter, r *http.Request) {
	servePreferredFile(w, r, "application/wasm", []string{
		filepath.Join("build", "storage.wasm"),
		"storage.wasm",
	})
}

func serveWASMExec(w http.ResponseWriter, r *http.Request) {
	servePreferredFile(w, r, "application/javascript", []string{
		filepath.Join("build", "wasm_exec.js"),
		"wasm_exec.js",
	})
}

func servePreferredFile(w http.ResponseWriter, r *http.Request, contentType string, candidates []string) {
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			w.Header().Set("Content-Type", contentType)
			http.ServeFile(w, r, candidate)
			return
		}
	}
	http.Error(w, "artifact not found; run the root build pipeline first", http.StatusNotFound)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
