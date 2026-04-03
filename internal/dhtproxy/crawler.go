package dhtproxy

import (
	"context"
	"fmt"
	"log"
	"time"
	"bobtorrent/pkg/torrent"

	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/torrent/metainfo"
)

type Crawler struct {
	dhtServer *dht.Server
	db        *Database
	geoIP     *torrent.GeoIPService
}

func NewCrawler(db *Database, geoIPPath string) (*Crawler, error) {
	// Start DHT server on a random port
	server, err := dht.NewServer(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start DHT server: %w", err)
	}

	var geoIP *torrent.GeoIPService
	if geoIPPath != "" {
		geoIP, err = torrent.NewGeoIPService(geoIPPath)
		if err != nil {
			log.Printf("Warning: Failed to load GeoIP database: %v. Peers will not be enriched.", err)
		}
	}

	return &Crawler{
		dhtServer: server,
		db:        db,
		geoIP:     geoIP,
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
				country := "XX"
				var lat, lon float64
				
				if c.geoIP != nil {
					if c, lt, ln, err := c.geoIP.Lookup(peer.IP.String()); err == nil {
						country = c
						lat = lt
						lon = ln
					}
				}

				err := c.db.AddPeer(infoHashHex, peer.IP.String(), peer.Port, country, lat, lon)
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
