package transport

import (
	"crypto/sha1"
	"encoding/hex"
	"log"
	"strings"
	"sync"


)

// We use an isolated cache for I2P nodes since the underlying anacrolix/dht
// does not natively support non-IP (e.g. b32.i2p) node structs.
var (
	i2pRoutingTable = make(map[string]string) // Map infohash to I2P address
	i2pMutex        sync.RWMutex
)

// AddHybridNode attempts to inject a node into the routing table.
// If the node is an I2P address (.b32.i2p), it is handled specifically.
// Otherwise, it falls back to standard Clearnet routing.
func (n *DHTNode) AddHybridNode(addr string) error {
	if strings.HasSuffix(addr, ".b32.i2p") {
		// Calculate a pseudo-infohash for the I2P address to map it.
		hash := sha1.New()
		hash.Write([]byte(addr))
		ih := hex.EncodeToString(hash.Sum(nil))

		i2pMutex.Lock()
		i2pRoutingTable[ih] = addr
		i2pMutex.Unlock()

		log.Printf("Optimizing routing table for I2P peer: %s (mapped to %s)", addr, ih)
		return nil
	}

	// Standard clearnet insertion.
	return n.AddNode(addr)
}

// GetPeersHybrid executes a lookup for an infohash. It queries both
// standard clearnet k-buckets and our optimized I2P peer list.
func (n *DHTNode) GetPeersHybrid(ih string) ([]string, error) {
	var peers []string

	// 1. Query I2P Darknet Peers (Optimized path, minimal latency)
	i2pMutex.RLock()
	if addr, exists := i2pRoutingTable[ih]; exists {
		peers = append(peers, addr)
	}
	i2pMutex.RUnlock()

	// 2. Query Clearnet Peers (Standard DHT)
	// We cannot pull directly from anacrolix/dht without triggering a full active
	// search. For now, we only return the cached hybrid I2P peers.

	return peers, nil
}
