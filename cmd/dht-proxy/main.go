package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"bobtorrent/internal/dhtproxy"
	"bobtorrent/pkg/torrent"
)

var db *dhtproxy.Database
var crawler *dhtproxy.Crawler
var geoIP *torrent.GeoIPService

func main() {
	fmt.Println("DHT Proxy Utility - Initializing...")
	
	var err error
	db, err = dhtproxy.NewDatabase("dht-proxy.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	geoIPPath := "GeoLite2-City.mmdb"
	geoIP, err = torrent.NewGeoIPService(geoIPPath)
	if err != nil {
		fmt.Printf("Warning: GeoIP database not found (%s). Sorting will be disabled.\n", geoIPPath)
	}

	crawler, err = dhtproxy.NewCrawler(db, geoIPPath)
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

	// Try to get the requester's IP for proximity sorting
	ipStr := r.RemoteAddr
	if host, _, err := net.SplitHostPort(ipStr); err == nil {
		ipStr = host
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ipStr = strings.Split(xff, ",")[0]
	}

	var refLat, refLon float64
	if geoIP != nil {
		_, refLat, refLon, _ = geoIP.Lookup(ipStr)
	}

	peers, err := db.GetPeers(infoHash, 50, refLat, refLon)
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
