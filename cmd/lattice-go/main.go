package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

import "bobtorrent/internal/consensus"

func main() {
	port := flag.Int("port", 4000, "HTTP listen port (two-node setups need distinct ports per node)")
	peers := flag.String("peers", "", "comma-separated peer lattice URLs for autonomous background sync, e.g. http://127.0.0.1:4000")
	requireSigs := flag.Bool("require-signatures", false, "reject live blocks without a valid Ed25519 signature (replay of existing history is never gated; peer sync of unsigned history will be refused)")
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
	if *requireSigs {
		s.Lattice().RequireSignatures = true
		log.Printf("signature enforcement ENABLED: unsigned live blocks will be rejected")
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
	// Serve in a goroutine so SIGINT/SIGTERM can run the deferred
	// StopBackgroundSync + store Close above. Previously ListenAndServe
	// blocked main forever, those defers never ran, and every stop was
	// an unclean kill left for WAL recovery to sort out.
	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("Lattice Server failed: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Printf("Lattice Node shutting down gracefully")
}
