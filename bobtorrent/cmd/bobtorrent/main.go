package main

import (
	"log"
	"net/http"
	"os"

	"github.com/bobtorrent/bobtorrent/internal/api"
	"github.com/bobtorrent/bobtorrent/internal/wallet"
	"github.com/bobtorrent/bobtorrent/pkg/dht"
)

func main() {
	log.Println("Starting BobTorrent v3.0.0...")

	dataDir := "./data"
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		log.Fatalf("Failed to create data dir: %v", err)
	}

	// Initialize Wallet
	w, err := wallet.NewWallet(dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize Wallet: %v", err)
	}
	log.Printf("Wallet active: %s", w.GetPublicKey())

	// Initialize DHT/Torrent Engine
	engine, err := dht.NewEngine(dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize Torrent Engine: %v", err)
	}
	defer engine.Close()
	log.Println("Torrent Engine Initialized successfully")

	// Set up the unified API and UI server
	server := &api.Server{
		Wallet:  w,
		Engine:  engine,
		DataDir: dataDir,
	}

	mux := server.SetupRoutes()

	log.Println("Web API & UI serving on http://127.0.0.1:8080")
	err = http.ListenAndServe("127.0.0.1:8080", mux)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
