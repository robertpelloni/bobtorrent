package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	fmt.Println("DHT Proxy Utility - Initializing...")
	
	http.HandleFunc("/api/announce", handleAnnounce)
	http.HandleFunc("/api/torrent/add", handleAddTorrent)
	
	port := ":9080"
	fmt.Printf("Starting DHT Proxy on %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleAnnounce(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Announce endpoint (WIP)")
}

func handleAddTorrent(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Add Torrent endpoint (WIP)")
}
