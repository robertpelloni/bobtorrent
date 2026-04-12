package dht

import (
	"crypto/sha1"
	"encoding/hex"
)

// MapBlobIDToInfoHash takes a 32-byte (64 char hex) Megatorrent Blob ID
// and maps it to a 20-byte (40 char hex) libtorrent/DHT style InfoHash.
func MapBlobIDToInfoHash(blobIDHex string) (string, error) {
	blobIDBytes, err := hex.DecodeString(blobIDHex)
	if err != nil {
		return "", err
	}

	var infoHashBytes []byte
	if len(blobIDBytes) >= 20 {
		infoHashBytes = blobIDBytes[:20]
	} else {
		hash := sha1.Sum(blobIDBytes)
		infoHashBytes = hash[:]
	}

	return hex.EncodeToString(infoHashBytes), nil
}
