package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bobtorrent/internal/bridges"
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

var (
	nodeWallet    *torrent.Keypair
	httpClient    = resty.New().SetTimeout(10 * time.Second)
	latticeURL    = "http://localhost:4000"
	torrentClient *anacrolixTorrent.Client
	uiProgram     *tea.Program
	fcBridge      *bridges.FilecoinBridge
	dhtNode       *transport.DHTNode
	startedAt     = time.Now()
)

// main boots the Go supernode stack:
//   1. Wallet + torrent engine
//   2. HTTP/UDP tracker services
//   3. Kademlia DHT node
//   4. Lattice market poller + block feed listener
//   5. Terminal dashboard (TUI)
func main() {
	log.SetOutput(os.Stderr)

	loadOrCreateWallet()
	initTorrentClient()
	defer torrentClient.Close()

	fcBridge = bridges.NewFilecoinBridge("f1bobtorrentnode")

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

	http.HandleFunc("/announce", trackerCore.HTTPHandler())
	http.HandleFunc("/spora/", handleSpora)
	http.HandleFunc("/stats", handleStats)
	http.HandleFunc("/add-torrent", handleAddTorrent)
	http.HandleFunc("/remove-torrent", handleRemoveTorrent)
	http.HandleFunc("/storage.wasm", serveStorageWASM)
	http.HandleFunc("/wasm_exec.js", serveWASMExec)
	go func() {
		if err := http.ListenAndServe(":8000", nil); err != nil {
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

func handleStats(w http.ResponseWriter, r *http.Request) {
	uptimeSeconds := time.Since(startedAt).Seconds()
	if uptimeSeconds <= 0 {
		uptimeSeconds = 1
	}

	peerCount := 0
	if dhtNode != nil {
		peerCount = dhtNode.Stats().GoodNodes
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
		"address": nodeWallet.PublicKey,
		"service": "Go Supernode",
		"status":  "online",
		"uptime":  int64(uptimeSeconds),
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
