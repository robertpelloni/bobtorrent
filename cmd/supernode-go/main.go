package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"bobtorrent/internal/tracker"
	"bobtorrent/internal/transport"
	"bobtorrent/internal/tui"
	"bobtorrent/pkg/torrent"

	anacrolixTorrent "github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-resty/resty/v2"
)

var (
	nodeWallet    *torrent.Keypair
	httpClient    = resty.New()
	latticeURL    = "http://localhost:4000"
	torrentClient *anacrolixTorrent.Client
	uiProgram     *tea.Program
)

func main() {
	// Disable standard logging for TUI
	log.SetOutput(os.Stderr)

	// 1. Initialize Wallet
	loadOrCreateWallet()

	// 2. Initialize Torrent Client
	initTorrentClient()
	defer torrentClient.Close()

	// 3. Initialize Tracker
	t := tracker.NewTracker()

	// 4. Start Kademlia DHT Node
	dhtAddr := ":6882"
	dhtNode, err := transport.NewDHTNode(dhtAddr)
	if err == nil {
		go dhtNode.Start()
	}

	// 5. Start Market Poller
	go startMarketPoller()

	// 6. Start HTTP Handlers (Tracker + SPoRA)
	http.HandleFunc("/announce", t.HTTPHandler())
	http.HandleFunc("/spora/", handleSpora)
	http.HandleFunc("/stats", handleStats)
	
	go http.ListenAndServe(":8000", nil)

	// 7. Start UDP Tracker (BEP 15)
	udpTracker, _ := tracker.NewUDPTracker(t, ":6881")
	if udpTracker != nil {
		go udpTracker.Start()
	}

	// 8. Start TUI
	m := tui.NewModel()
	uiProgram = tea.NewProgram(m, tea.WithAltScreen())
	if _, err := uiProgram.Run(); err != nil {
		fmt.Printf("Error running TUI: %v", err)
		os.Exit(1)
	}
}

func initTorrentClient() {
	cfg := anacrolixTorrent.NewDefaultClientConfig()
	cfg.DataDir = "./downloads"
	cfg.ListenPort = 4242

	var err error
	torrentClient, err = anacrolixTorrent.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to start Torrent Client: %v", err)
	}
}

func loadOrCreateWallet() {
	walletFile := "wallet.json"
	data, err := os.ReadFile(walletFile)
	if err == nil {
		if err := json.Unmarshal(data, &nodeWallet); err == nil {
			return
		}
	}

	nodeWallet, _ = torrent.GenerateKeypair()
	data, _ = json.MarshalIndent(nodeWallet, "", "  ")
	os.WriteFile(walletFile, data, 0644)
}

func startMarketPoller() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		pollMarket()
	}
}

func pollMarket() {
	var result struct {
		Bids []tui.Bid `json:"bids"`
	}

	resp, err := httpClient.R().SetResult(&result).Get(latticeURL + "/market/bids")
	if err != nil || !resp.IsSuccess() {
		if uiProgram != nil {
			uiProgram.Send(tui.StatusMsg{Text: "Lattice API Offline", Balance: 0})
		}
		return
	}

	if uiProgram != nil {
		uiProgram.Send(tui.BidsMsg{Bids: result.Bids})
	}

	for _, bid := range result.Bids {
		if bid.Status == "OPEN" {
			spec, err := metainfo.ParseMagnetUri(bid.Magnet)
			if err != nil {
				continue
			}

			// Check if already seeding
			if _, exists := torrentClient.Torrent(spec.InfoHash); exists {
				continue
			}

			// 1. Start Seeding
			t, err := torrentClient.AddMagnet(bid.Magnet)
			if err == nil {
				go func(target *anacrolixTorrent.Torrent, bID string, bAmount int64) {
					<-target.GotInfo()
					target.DownloadAll()
					acceptBid(bID, bAmount, spec.InfoHash.HexString())
				}(t, bid.ID, bid.Amount)
			}
		}
	}
}

func acceptBid(bidID string, amount int64, infoHash string) {
	var status struct {
		Frontier      *string `json:"frontier"`
		Balance       int64   `json:"balance"`
		StakedBalance int64   `json:"staked_balance"`
	}

	resp, err := httpClient.R().SetResult(&status).Get(latticeURL + "/frontier/" + nodeWallet.PublicKey)
	if err != nil || !resp.IsSuccess() {
		return
	}

	if uiProgram != nil {
		uiProgram.Send(tui.StatusMsg{Text: "Accepting Bid...", Balance: status.Balance})
	}

	challenge := 12345
	if status.Frontier != nil {
		fmt.Sscanf((*status.Frontier)[:8], "%x", &challenge)
	}
	chunkHash := torrent.HashSHA256(infoHash + fmt.Sprintf("%d", challenge))

	spora := map[string]interface{}{
		"infoHash":  infoHash,
		"challenge": challenge,
		"chunkHash": chunkHash,
	}

	newBalance := status.Balance + amount
	block := torrent.NewBlock("accept_bid", nodeWallet.PublicKey, status.Frontier, newBalance, status.StakedBalance, 0, bidID, spora, nil)
	block.Sign(nodeWallet.PrivateKey)

	submitResp, err := httpClient.R().SetBody(map[string]interface{}{"block": block}).Post(latticeURL + "/process")
	if err == nil && submitResp.IsSuccess() {
		if uiProgram != nil {
			uiProgram.Send(tui.StatusMsg{Text: "Bid Accepted!", Balance: newBalance})
		}
	}
}

func handleSpora(w http.ResponseWriter, r *http.Request) {
	challenge := r.URL.Path[len("/spora/"):]
	infoHash := "1234567890abcdef1234567890abcdef12345678"
	chunkHash := torrent.HashSHA256(infoHash + challenge)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "spora": {"infoHash": "%s", "challenge": %s, "chunkHash": "%s"}}`, infoHash, challenge, chunkHash)
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"address": "%s", "service": "Go Supernode", "status": "online"}`, nodeWallet.PublicKey)
}
