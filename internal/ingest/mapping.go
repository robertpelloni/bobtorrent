package ingest

import (
	"bobtorrent/pkg/dht"
	"bobtorrent/pkg/storage"
)

// MapEntryToTorrents derives the BitTorrent InfoHashes for each shard in a file entry.
func MapEntryToTorrents(entry *storage.FileEntry) ([]string, error) {
	var infoHashes []string
	for _, chunk := range entry.Chunks {
		for _, blobID := range chunk.BlobIDs {
			ih, err := dht.MapBlobIDToInfoHash(blobID)
			if err != nil {
				return nil, err
			}
			infoHashes = append(infoHashes, ih)
		}
	}
	return infoHashes, nil
}
