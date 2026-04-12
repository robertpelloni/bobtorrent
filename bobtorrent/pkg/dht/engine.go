package dht

import (
	"log"

	"github.com/anacrolix/torrent"
)

type Engine struct {
	Client *torrent.Client
}

func NewEngine(dataDir string) (*Engine, error) {
	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = dataDir
	// Disable seed-only for testing
	cfg.Seed = false

	client, err := torrent.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Engine{
		Client: client,
	}, nil
}

func (e *Engine) AnnounceBlob(blobIDHex string) error {
	infoHashHex, err := MapBlobIDToInfoHash(blobIDHex)
	if err != nil {
		return err
	}

	log.Printf("Announcing BlobID: %s mapped to InfoHash: %s", blobIDHex, infoHashHex)
	return nil
}

func (e *Engine) Close() {
	if e.Client != nil {
		e.Client.Close()
	}
}
