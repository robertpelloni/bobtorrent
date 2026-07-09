package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func registerBobcoinProxies(mux *http.ServeMux) {
	latticeURL, err := url.Parse("http://127.0.0.1:4000")
	if err != nil {
		log.Printf("Failed to parse lattice URL: %v", err)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(latticeURL)

	// Proxy legacy endpoints to the lattice daemon.
	mux.Handle("/blocks", proxy)
	mux.Handle("/proposals", proxy)
	mux.Handle("/governance/proposals", proxy)
	mux.Handle("/process", proxy)
	// /ws is intentionally skipped here because the frontend may conflict with our native websocket.
	// We will implement an interceptor for the bobcoin frontend to use /lattice-ws.
	mux.Handle("/lattice-ws", proxy)
}
