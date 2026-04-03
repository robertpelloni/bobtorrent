package transport

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/torrent/metainfo"
)

// DHTNode represents a BitTorrent Kademlia DHT node.
type DHTNode struct {
	server *dht.Server
}

// NewDHTNode creates a new DHT node and starts listening on the specified address.
func NewDHTNode(addr string) (*DHTNode, error) {
	config := dht.NewDefaultServerConfig()
	if addr != "" {
		config.Addr = addr
	}

	server, err := dht.NewServer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create DHT server: %w", err)
	}

	return &DHTNode{server: server}, nil
}

// Start begins bootstrapping and periodic maintenance.
func (n *DHTNode) Start() {
	log.Printf("DHT Node listening on %v", n.server.Addr())
	// Bootstrap from default nodes
	n.server.Bootstrap()
}

// Announce announces ourselves for an info_hash and returns a peer search.
func (n *DHTNode) Announce(ih metainfo.Hash, port int, impliesPort bool) (*dht.Search, error) {
	return n.server.Announce(ih, port, impliesPort)
}

// GetPeers searches for peers for an info_hash.
func (n *DHTNode) GetPeers(ih metainfo.Hash) (*dht.Search, error) {
	return n.server.Search(ih)
}

// AddNode adds a known node to the routing table.
func (n *DHTNode) AddNode(nodeAddr string) error {
	addr, err := net.ResolveUDPAddr("udp", nodeAddr)
	if err != nil {
		return err
	}
	n.server.AddNode(dht.NewAddr(addr))
	return nil
}

// Stats returns various stats from the DHT node.
func (n *DHTNode) Stats() dht.ServerStats {
	return n.server.Stats()
}

// Close stops the DHT node.
func (n *DHTNode) Close() error {
	n.server.Close()
	return nil
}
