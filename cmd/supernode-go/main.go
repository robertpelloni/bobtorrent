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
	"bobtorrent/pkg/torrent"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/go-resty/resty/v2"
)

var (
	nodeWallet    *torrent.Keypair
	httpClient    = resty.New()
	latticeURL    = "http://localhost:4000"
	torrentClient *torrent.Client
)

func main() {
	fmt.Println("Bobtorrent Supernode (Go Port) - Initializing...")

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
	if err != nil {
		log.Fatalf("Failed to start DHT Node: %v", err)
	}
	go dhtNode.Start()
	fmt.Printf("DHT Node listening on %s\n", dhtAddr)

	// 5. Start Market Poller
	go startMarketPoller()

	// 6. Start HTTP Handlers (Tracker + SPoRA)
	http.HandleFunc("/announce", t.HTTPHandler())
	http.HandleFunc("/spora/", handleSpora)
	http.HandleFunc("/stats", handleStats)
	
	httpPort := ":8000"
	fmt.Printf("Supernode API listening on %s\n", httpPort)

	// 7. Start UDP Tracker (BEP 15)
	udpAddr := ":6881"
	udpTracker, err := tracker.NewUDPTracker(t, udpAddr)
	if err != nil {
		log.Fatalf("Failed to start UDP Tracker: %v", err)
	}
	go udpTracker.Start()
	fmt.Printf("UDP Tracker listening on %s\n", udpAddr)

	if err := http.ListenAndServe(httpPort, nil); err != nil {
		log.Fatalf("API Server failed: %v", err)
	}
}

func initTorrentClient() {
	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = "./downloads"
	cfg.ListenPort = 4242

	var err error
	torrentClient, err = torrent.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to start Torrent Client: %v", err)
	}
	fmt.Println("[Torrent] Client started on port 4242")
}

func loadOrCreateWallet() {
	walletFile := "wallet.json"
	data, err := os.ReadFile(walletFile)
	if err == nil {
		if err := json.Unmarshal(data, &nodeWallet); err == nil {
			fmt.Printf("Loaded existing wallet: %s...\n", nodeWallet.PublicKey[:16])
			return
		}
	}

	nodeWallet, _ = torrent.GenerateKeypair()
	data, _ = json.MarshalIndent(nodeWallet, "", "  ")
	os.WriteFile(walletFile, data, 0644)
	fmt.Printf("Generated new wallet: %s...\n", nodeWallet.PublicKey[:16])
}

func startMarketPoller() {
	fmt.Println("[Poller] Starting Lattice Market Poller...")
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		pollMarket()
	}
}

func pollMarket() {
	var result struct {
		Bids []struct {
			ID     string `json:"id"`
			Magnet string `json:"magnet"`
			Amount int64  `json:"amount"`
			Status string `json:"status"`
		} `json:"bids"`
	}

	resp, err := httpClient.R().SetResult(&result).Get(latticeURL + "/market/bids")
	if err != nil || !resp.IsSuccess() {
		return
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

			fmt.Printf("[Poller] Found OPEN bid: %s... Accepting...\n", bid.ID[:8])
			
			// 1. Start Seeding
			t, err := torrentClient.AddMagnet(bid.Magnet)
			if err == nil {
				go func(target *torrent.Torrent, bID string, bAmount int64) {
					<-target.GotInfo()
					target.DownloadAll()
					fmt.Printf("[Torrent] Started seeding: %s\n", target.Name())
					
					// 2. Accept Bid on Lattice
					acceptBid(bID, bAmount, spec.InfoHash.HexString())
				}(t, bid.ID, bid.Amount)
			}
		}
	}
}

func acceptBid(bidID string, amount int64, infoHash string) {
	// 1. Get Frontier and Balance
	var status struct {
		Frontier      *string `json:"frontier"`
		Balance       int64   `json:"balance"`
		StakedBalance int64   `json:"staked_balance"`
	}

	resp, err := httpClient.R().SetResult(&status).Get(latticeURL + "/frontier/" + nodeWallet.PublicKey)
	if err != nil || !resp.IsSuccess() {
		return
	}

	// 2. Create SPoRA proof
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

	// 3. Construct Block
	height := 0 
	newBalance := status.Balance + amount
	
	block := torrent.NewBlock("accept_bid", nodeWallet.PublicKey, status.Frontier, newBalance, status.StakedBalance, height, bidID, spora, nil)
	block.Sign(nodeWallet.PrivateKey)

	// 4. Submit to Lattice
	submitResp, err := httpClient.R().SetBody(map[string]interface{}{"block": block}).Post(latticeURL + "/process")
	if err == nil && submitResp.IsSuccess() {
		fmt.Printf("[Poller] ✅ Bid %s accepted! New Balance: %d\n", bidID[:8], newBalance)
	} else {
		fmt.Printf("[Poller] ❌ Failed to accept bid: %v\n", err)
	}
}

func handleSpora(w http.ResponseWriter, r *http.Request) {
	challenge := r.URL.Path[len("/spora/"):]
	if challenge == "" {
		http.Error(w, "challenge required", http.StatusBadRequest)
		return
	}

	infoHash := "1234567890abcdef1234567890abcdef12345678"
	chunkHash := torrent.HashSHA256(infoHash + challenge)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "spora": {"infoHash": "%s", "challenge": %s, "chunkHash": "%s"}}`, 
		infoHash, challenge, chunkHash)
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"address": "%s", "service": "Go Supernode", "status": "online"}`, nodeWallet.PublicKey)
}
