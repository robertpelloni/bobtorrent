package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"strings"
)

import "bobtorrent/internal/consensus"

func main() {
	port := flag.Int("port", 4000, "HTTP listen port (two-node setups need distinct ports per node)")
	peers := flag.String("peers", "", "comma-separated peer lattice URLs for autonomous background sync, e.g. http://127.0.0.1:4000")
	flag.Parse()

	fmt.Println("Bobcoin Lattice Node (Go Port) - Initializing...")

	dbPath := os.Getenv("BOBTORRENT_LATTICE_DB")
	if dbPath == "" {
		dbPath = filepath.Join("data", "lattice", "lattice.db")
	}

	s, err := consensus.NewPersistentServer(dbPath)
	if err != nil {
		log.Fatalf("failed to initialize persistent lattice server: %v", err)
	}
	defer func() {
		s.StopBackgroundSync()
		if err := s.Lattice().Close(); err != nil {
			log.Printf("failed to close lattice persistence cleanly: %v", err)
		}
	}()

	for _, peer := range strings.Split(*peers, ",") {
		if peer = strings.TrimSpace(peer); peer != "" {
			s.Lattice().AddPeer(peer)
			log.Printf("registered lattice peer: %s", peer)
		}
	}

	s.StartBackgroundSync(0) // Default interval

	mux := http.NewServeMux()
	mux.Handle("/", s.HTTPHandler())

	// Profiling
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("Lattice Node listening on %s with persistence at %s\n", addr, dbPath)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Lattice Server failed: %v", err)
	}
}
