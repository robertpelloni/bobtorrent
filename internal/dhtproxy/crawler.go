package dhtproxy

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/torrent/metainfo"
)

type Crawler struct {
	dhtServer *dht.Server
	db        *Database
}

func NewCrawler(db *Database) (*Crawler, error) {
	// Start DHT server on a random port
	server, err := dht.NewServer(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start DHT server: %w", err)
	}

	return &Crawler{
		dhtServer: server,
		db:        db,
	}, nil
}

func (c *Crawler) Start(ctx context.Context) {
	log.Println("DHT Crawler started")
}

func (c *Crawler) Crawl(infoHashHex string) {
	var ih metainfo.Hash
	if err := ih.FromHexString(infoHashHex); err != nil {
		log.Printf("Invalid info hash: %v", err)
		return
	}

	log.Printf("Starting DHT crawl for %s", infoHashHex)
	
	// Use Announce to find peers and announce ourselves
	search, err := c.dhtServer.Announce(ih, 0, true)
	if err != nil {
		log.Printf("Failed to start DHT announce: %v", err)
		return
	}
	defer search.Close()

	for {
		select {
		case result, ok := <-search.Peers:
			if !ok {
				log.Printf("DHT search finished for %s", infoHashHex)
				return
			}
			
			for _, peer := range result.Peers {
				// For now, we skip GeoIP enrichment
				err := c.db.AddPeer(infoHashHex, peer.IP.String(), peer.Port, "XX", 0, 0)
				if err != nil {
					log.Printf("Failed to save peer: %v", err)
				}
			}
		case <-time.After(30 * time.Second):
			log.Printf("DHT search timeout for %s", infoHashHex)
			return
		}
	}
}

func (c *Crawler) Close() error {
	c.dhtServer.Close()
	return nil
}
