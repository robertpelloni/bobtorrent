package tracker

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/anacrolix/torrent/metainfo"
)

// UDPTracker implements the BEP 15 UDP Tracker Protocol.
type UDPTracker struct {
	tracker *Tracker
	conn    *net.UDPConn
	mu      sync.Mutex
}

// NewUDPTracker creates a new UDPTracker instance and starts listening on the specified address.
func NewUDPTracker(tracker *Tracker, addr string) (*UDPTracker, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	return &UDPTracker{
		tracker: tracker,
		conn:    conn,
	}, nil
}

// Start begins processing incoming UDP packets.
func (u *UDPTracker) Start() {
	buffer := make([]byte, 2048)
	for {
		n, addr, err := u.conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("UDP Read Error: %v", err)
			continue
		}

		go u.handlePacket(addr, buffer[:n])
	}
}

func (u *UDPTracker) handlePacket(addr *net.UDPAddr, data []byte) {
	if len(data) < 16 {
		return
	}

	connectionID := binary.BigEndian.Uint64(data[0:8])
	action := binary.BigEndian.Uint32(data[8:12])
	transactionID := binary.BigEndian.Uint32(data[12:16])

	// Basic connection ID verification (simplified)
	if action != 0 && connectionID == 0 {
		return
	}

	switch action {
	case 0: // Connect
		u.handleConnect(addr, transactionID)
	case 1: // Announce
		u.handleAnnounce(addr, transactionID, data[16:])
	case 2: // Scrape
		u.handleScrape(addr, transactionID, data[16:])
	}
}

func (u *UDPTracker) handleConnect(addr *net.UDPAddr, transactionID uint32) {
	resp := make([]byte, 16)
	binary.BigEndian.PutUint32(resp[0:4], 0) // Action: Connect
	binary.BigEndian.PutUint32(resp[4:8], transactionID)
	
	// Create a random connection ID
	connID := uint64(time.Now().UnixNano())
	binary.BigEndian.PutUint64(resp[8:16], connID)

	u.conn.WriteToUDP(resp, addr)
}

func (u *UDPTracker) handleAnnounce(addr *net.UDPAddr, transactionID uint32, data []byte) {
	if len(data) < 82 {
		return
	}

	var ih metainfo.Hash
	copy(ih[:], data[0:20])
	peerID := string(data[20:40])
	downloaded := binary.BigEndian.Uint64(data[40:48])
	left := binary.BigEndian.Uint64(data[48:56])
	uploaded := binary.BigEndian.Uint64(data[56:64])
	event := binary.BigEndian.Uint32(data[64:68])
	ip := net.IP(data[68:72])
	key := binary.BigEndian.Uint32(data[72:76])
	numWant := int32(binary.BigEndian.Uint32(data[76:80]))
	port := binary.BigEndian.Uint16(data[80:82])

	// Use Remote IP if none specified
	if ip.IsUnspecified() || ip.IsLoopback() {
		ip = addr.IP
	}

	swarm := u.tracker.GetSwarm(ih)
	swarm.Announce(peerID, ip, port)

	peers := swarm.GetPeers(50, peerID)

	// Build Response
	resp := make([]byte, 20 + 6*len(peers))
	binary.BigEndian.PutUint32(resp[0:4], 1) // Action: Announce
	binary.BigEndian.PutUint32(resp[4:8], transactionID)
	binary.BigEndian.PutUint32(resp[8:12], 1800) // Interval
	binary.BigEndian.PutUint32(resp[12:16], 0)    // Leechers
	binary.BigEndian.PutUint32(resp[16:20], uint32(len(swarm.Peers))) // Seeders

	offset := 20
	for _, p := range peers {
		if p.IP.To4() != nil {
			copy(resp[offset:offset+4], p.IP.To4())
			binary.BigEndian.PutUint16(resp[offset+4:offset+6], p.Port)
			offset += 6
		}
	}

	u.conn.WriteToUDP(resp[:offset], addr)
}

func (u *UDPTracker) handleScrape(addr *net.UDPAddr, transactionID uint32, data []byte) {
	// Scrape is optional but good to have. Simplified implementation.
}
