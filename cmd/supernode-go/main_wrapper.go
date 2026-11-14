package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"bobtorrent/internal/bridges"
)

func main() {
	headless := flag.Bool("headless", false, "run without TUI (HTTP dashboard + all services, no terminal UI)")
	flag.Parse()

	if *headless {
		runHeadless()
	} else {
		realMain()
	}
}

// runHeadless starts the exact same services as the TUI path — wallet,
// torrent client, Filecoin bridge, registries, economy DB, verifier,
// ingestion, tracker + HTTP dashboard on :8000, DHT, messenger, I2P,
// market poller and lattice feed — but blocks on OS signals instead of
// the Bubble Tea program. Previously this branch skipped
// startTrackerServices entirely, so `-headless` (the mode the system
// tray controller launches) served no dashboard at all.
func runHeadless() {
	log.SetOutput(os.Stderr)
	loadOrCreateWallet()
	initTorrentClient()
	defer torrentClient.Close()

	fcBridge = bridges.NewFilecoinBridge("f1bobtorrentnode")
	initPublishRegistry()
	defer publishRegistry.Close()
	initEconomyDatabase()
	defer economyDB.Close()
	initVerifierService()
	initIngestionService()

	startTrackerServices()
	startDHT()
	startMessenger()
	if messenger != nil {
		defer messenger.Close()
	}
	startI2PDatagram()
	if i2pDatagram != nil {
		defer i2pDatagram.Close()
	}
	go startMarketPoller()
	go startLatticeFeedListener()

	log.Printf("Headless supernode v%s started on :8000 (dashboard: http://localhost:8000)", Version)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Printf("Headless supernode shutting down")
}
