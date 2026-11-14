package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func registerBobcoinProxies(mux *http.ServeMux) {
	target, err := url.Parse(latticeURL)
	if err != nil {
		log.Printf("Failed to parse lattice URL: %v", err)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Proxy legacy endpoints to the lattice daemon.
	mux.Handle("/blocks", proxy)
	mux.Handle("/proposals", proxy)
	mux.Handle("/governance/proposals", proxy)
	mux.Handle("/process", proxy)
	// Read-only lattice state the dashboard renders (NFTs, swaps, anchors,
	// market bids, account frontiers/balances). Without these the
	// Wallet & Lattice tab could only show the block list.
	mux.Handle("/nfts", proxy)
	mux.Handle("/nfts/", proxy)
	mux.Handle("/swaps", proxy)
	mux.Handle("/anchors", proxy)
	mux.Handle("/anchors/", proxy)
	mux.Handle("/market/bids", proxy)
	mux.Handle("/frontier/", proxy)
	mux.Handle("/balance/", proxy)
	mux.Handle("/peers", proxy)
	// /ws is intentionally skipped here because the frontend may conflict with our native websocket.
	// We will implement an interceptor for the bobcoin frontend to use /lattice-ws.
	mux.Handle("/lattice-ws", proxy)
}
