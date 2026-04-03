package tracker

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
)

// Peer represents a BitTorrent peer in a swarm.
type Peer struct {
	ID        string    `json:"id"`
	IP        net.IP    `json:"ip"`
	Port      uint16    `json:"port"`
	LastSeen  time.Time `json:"last_seen"`
}

// Swarm represents a set of peers for a specific info_hash.
type Swarm struct {
	InfoHash metainfo.Hash
	Peers    map[string]*Peer
	mu       sync.RWMutex
}

// Tracker manages multiple swarms and provides peer discovery.
type Tracker struct {
	swarms map[metainfo.Hash]*Swarm
	mu     sync.RWMutex
}

// NewTracker creates a new Tracker instance.
func NewTracker() *Tracker {
	return &Tracker{
		swarms: make(map[metainfo.Hash]*Swarm),
	}
}

// GetSwarm returns or creates a swarm for the specified info_hash.
func (t *Tracker) GetSwarm(ih metainfo.Hash) *Swarm {
	t.mu.Lock()
	defer t.mu.Unlock()

	s, ok := t.swarms[ih]
	if !ok {
		s = &Swarm{
			InfoHash: ih,
			Peers:    make(map[string]*Peer),
		}
		t.swarms[ih] = s
	}
	return s
}

// Announce adds/updates a peer in a swarm.
func (s *Swarm) Announce(peerID string, ip net.IP, port uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Peers[peerID] = &Peer{
		ID:       peerID,
		IP:       ip,
		Port:     port,
		LastSeen: time.Now(),
	}
}

// GetPeers returns a list of up to limit peers from the swarm.
func (s *Swarm) GetPeers(limit int, excludeID string) []Peer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Peer
	for id, p := range s.Peers {
		if id == excludeID {
			continue
		}
		// Skip stale peers (> 30 mins)
		if time.Since(p.LastSeen) > 30*time.Minute {
			continue
		}
		result = append(result, *p)
		if len(result) >= limit {
			break
		}
	}
	return result
}

// AnnounceResponse represents the Bencoded response for a tracker announce.
type AnnounceResponse struct {
	Interval int         `bencode:"interval"`
	Peers    interface{} `bencode:"peers"`
}

type CompactPeer []byte

// HTTPHandler returns an http.Handler for the tracker's announce endpoint.
func (t *Tracker) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		infoHash := query.Get("info_hash")
		peerID := query.Get("peer_id")
		portStr := query.Get("port")
		compact := query.Get("compact") == "1"

		if infoHash == "" || peerID == "" || portStr == "" {
			http.Error(w, "missing required parameters", http.StatusBadRequest)
			return
		}

		var ih metainfo.Hash
		if len(infoHash) == 20 {
			copy(ih[:], infoHash)
		} else if err := ih.FromHexString(infoHash); err != nil {
			http.Error(w, "invalid info_hash", http.StatusBadRequest)
			return
		}

		port, _ := strconv.ParseUint(portStr, 10, 16)

		// Determine IP
		ipStr, _, _ := net.SplitHostPort(r.RemoteAddr)
		ip := net.ParseIP(ipStr)
		if ip.To4() != nil {
			ip = ip.To4()
		}

		swarm := t.GetSwarm(ih)
		swarm.Announce(peerID, ip, uint16(port))

		peers := swarm.GetPeers(50, peerID)

		var resp AnnounceResponse
		resp.Interval = 1800

		if compact {
			var buf bytes.Buffer
			for _, p := range peers {
				if len(p.IP) == 4 {
					buf.Write(p.IP)
					buf.WriteByte(byte(p.Port >> 8))
					buf.WriteByte(byte(p.Port & 0xff))
				}
			}
			resp.Peers = buf.Bytes()
		} else {
			var peerList []map[string]interface{}
			for _, p := range peers {
				peerList = append(peerList, map[string]interface{}{
					"peer id": p.ID,
					"ip":      p.IP.String(),
					"port":    p.Port,
				})
			}
			resp.Peers = peerList
		}

		data, err := bencode.Marshal(resp)
		if err != nil {
			http.Error(w, "encoding error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write(data)
	}
}
