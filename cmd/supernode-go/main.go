package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"bobtorrent/internal/tracker"
	"bobtorrent/internal/transport"
	"bobtorrent/pkg/torrent"
)

var nodeWallet *torrent.Keypair

func main() {
	fmt.Println("Bobtorrent Supernode (Go Port) - Initializing...")

	// 1. Initialize Wallet
	loadOrCreateWallet()

	// 2. Initialize Tracker
	t := tracker.NewTracker()

	// 3. Start Kademlia DHT Node
	dhtAddr := ":6882"
	dhtNode, err := transport.NewDHTNode(dhtAddr)
	if err != nil {
		log.Fatalf("Failed to start DHT Node: %v", err)
	}
	go dhtNode.Start()
	fmt.Printf("DHT Node listening on %s\n", dhtAddr)

	// 4. Start HTTP Handlers (Tracker + SPoRA)
	http.HandleFunc("/announce", t.HTTPHandler())
	http.HandleFunc("/spora/", handleSpora)
	http.HandleFunc("/stats", handleStats)
	
	httpPort := ":8000"
	fmt.Printf("Supernode API listening on %s\n", httpPort)

	// 5. Start UDP Tracker (BEP 15)
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

func handleSpora(w http.ResponseWriter, r *http.Request) {
	challenge := r.URL.Path[len("/spora/"):]
	if challenge == "" {
		http.Error(w, "challenge required", http.StatusBadRequest)
		return
	}

	// For the prototype, we simulate SPoRA by hashing infohash + challenge
	// In production, this would read actual file chunks from disk
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
