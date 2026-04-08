package main

import (
	"fmt"
	"log"
	"net/http"
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

	port := ":4000"
	fmt.Printf("Lattice Node listening on %s with persistence at %s\n", port, dbPath)
	if err := http.ListenAndServe(port, s.HTTPHandler()); err != nil {
		log.Fatalf("Lattice Server failed: %v", err)
	}
}
