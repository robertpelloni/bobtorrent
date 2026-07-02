package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"

	"bobtorrent/internal/consensus"
)

func main() {
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

	s.StartBackgroundSync(0) // Default interval

	mux := http.NewServeMux()
	mux.Handle("/", s.HTTPHandler())

	// Profiling
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	port := ":4000"
	fmt.Printf("Lattice Node listening on %s with persistence at %s\n", port, dbPath)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Lattice Server failed: %v", err)
	}
}
