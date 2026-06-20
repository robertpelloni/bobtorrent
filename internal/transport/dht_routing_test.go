package transport

import (
	"crypto/sha1"
	"encoding/hex"
	"testing"
)

func TestAddHybridNode(t *testing.T) {
	node, err := NewDHTNode("")
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}
	defer node.Close()

	// Test Clearnet Node Insertion
	err = node.AddHybridNode("127.0.0.1:6881")
	if err != nil {
		t.Errorf("AddHybridNode failed for clearnet address: %v", err)
	}

	// Test I2P Node Insertion
	i2pAddr := "example.b32.i2p"
	err = node.AddHybridNode(i2pAddr)
	if err != nil {
		t.Errorf("AddHybridNode failed for I2P address: %v", err)
	}

	// Validate the I2P node was inserted into our parallel cache
	hash := sha1.New()
	hash.Write([]byte(i2pAddr))
	ih := hex.EncodeToString(hash.Sum(nil))

	i2pMutex.RLock()
	cachedAddr := i2pRoutingTable[ih]
	i2pMutex.RUnlock()

	if cachedAddr != i2pAddr {
		t.Errorf("I2P address not properly cached. Expected %s, got %s", i2pAddr, cachedAddr)
	}

	// Test Invalid Node
	err = node.AddHybridNode("invalid_address_format")
	if err == nil {
		t.Error("AddHybridNode expected error for invalid address, got nil")
	}
}

func TestGetPeersHybrid(t *testing.T) {
	node, err := NewDHTNode("")
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}
	defer node.Close()

	// Add an I2P node
	i2pAddr := "target.b32.i2p"
	node.AddHybridNode(i2pAddr)

	hash := sha1.New()
	hash.Write([]byte(i2pAddr))
	ih := hex.EncodeToString(hash.Sum(nil))

	// Retrieve the hybrid peers
	peers, err := node.GetPeersHybrid(ih)
	if err != nil {
		t.Errorf("GetPeersHybrid failed: %v", err)
	}

	if len(peers) == 0 {
		t.Fatal("GetPeersHybrid returned empty slice")
	}

	if peers[0] != i2pAddr {
		t.Errorf("GetPeersHybrid returned incorrect peer. Expected %s, got %s", i2pAddr, peers[0])
	}
}
