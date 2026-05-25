package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	headless := flag.Bool("headless", false, "run without TUI")
	flag.Parse()

	if *headless {
		log.SetOutput(os.Stderr)
		loadOrCreateWallet()
		initTorrentClient()
		defer torrentClient.Close()
		initPublishRegistry()
		defer publishRegistry.Close()
		initEconomyDatabase()
		defer economyDB.Close()
		initVerifierService()
		startTrackerServices()
		startDHT()
		startMessenger()
		startI2PDatagram()
		log.Printf("Headless supernode started on :8000")
		select {}
	} else {
		realMain()
	}
}
