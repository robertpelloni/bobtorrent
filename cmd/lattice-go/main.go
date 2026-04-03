package main

import (
	"fmt"
	"log"
	"net/http"
	"bobtorrent/internal/consensus"
)

func main() {
	fmt.Println("Bobcoin Lattice Node (Go Port) - Initializing...")

	s := consensus.NewServer()
	
	port := ":4000"
	fmt.Printf("Lattice Node listening on %s\n", port)
	if err := http.ListenAndServe(port, s.HTTPHandler()); err != nil {
		log.Fatalf("Lattice Server failed: %v", err)
	}
}
