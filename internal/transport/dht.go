package transport

import (
	"fmt"
	"log"
	"net"

	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/dht/v2/krpc"
	"github.com/anacrolix/torrent/metainfo"
)

// DHTNode wraps an anacrolix DHT server for use by the Go supernode.
type DHTNode struct {
	server *dht.Server
}

// NewDHTNode creates a DHT server bound to the requested UDP address.
func NewDHTNode(addr string) (*DHTNode, error) {
	config := dht.NewDefaultServerConfig()
	if addr != "" {
		conn, err := net.ListenPacket("udp", addr)
		if err != nil {
			return nil, fmt.Errorf("failed to bind DHT listener: %w", err)
		}
		config.Conn = conn
	}

	server, err := dht.NewServer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create DHT server: %w", err)
	}

	return &DHTNode{server: server}, nil
}

// Start bootstraps the DHT node into the public network.
func (n *DHTNode) Start() {
	log.Printf("DHT Node listening on %v", n.server.Addr())
	if _, err := n.server.Bootstrap(); err != nil {
		log.Printf("DHT bootstrap failed: %v", err)
	}
}

// Announce announces the local node for the given infohash.
func (n *DHTNode) Announce(ih metainfo.Hash, port int, impliesPort bool) (*dht.Announce, error) {
	return n.server.Announce(ih, port, impliesPort)
}

// AddNode inserts a bootstrap node directly into the routing table.
func (n *DHTNode) AddNode(nodeAddr string) error {
	addr, err := net.ResolveUDPAddr("udp", nodeAddr)
	if err != nil {
		return err
	}
	return n.server.AddNode(krpc.NodeInfo{Addr: krpc.NodeAddr{IP: addr.IP, Port: addr.Port}})
}

// Stats exposes DHT server metrics for dashboards and health checks.
func (n *DHTNode) Stats() dht.ServerStats {
	return n.server.Stats()
}

// Close stops the DHT server.
func (n *DHTNode) Close() error {
	n.server.Close()
	return nil
}
