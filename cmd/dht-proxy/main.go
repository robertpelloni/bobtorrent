package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"bobtorrent/internal/dhtproxy"
)

var db *dhtproxy.Database
var crawler *dhtproxy.Crawler

func main() {
	fmt.Println("DHT Proxy Utility - Initializing...")
	
	var err error
	db, err = dhtproxy.NewDatabase("dht-proxy.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	crawler, err = dhtproxy.NewCrawler(db)
	if err != nil {
		log.Fatalf("Failed to initialize crawler: %v", err)
	}
	defer crawler.Close()

	http.HandleFunc("/api/announce", handleAnnounce)
	http.HandleFunc("/api/torrent/add", handleAddTorrent)
	
	port := ":9080"
	fmt.Printf("Starting DHT Proxy on %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleAnnounce(w http.ResponseWriter, r *http.Request) {
	infoHash := r.URL.Query().Get("info_hash")
	if infoHash == "" {
		http.Error(w, "info_hash is required", http.StatusBadRequest)
		return
	}

	peers, err := db.GetPeers(infoHash, 50)
	if err != nil {
		http.Error(w, "failed to get peers", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(peers)
}

func handleAddTorrent(w http.ResponseWriter, r *http.Request) {
	infoHash := r.URL.Query().Get("info_hash")
	if infoHash == "" {
		http.Error(w, "info_hash is required", http.StatusBadRequest)
		return
	}

	err := db.UpsertTorrent(infoHash, "New Torrent")
	if err != nil {
		http.Error(w, "failed to save torrent", http.StatusInternalServerError)
		return
	}

	// Trigger async crawl
	go crawler.Crawl(infoHash)

	fmt.Fprintln(w, "Torrent added and crawl triggered")
}
