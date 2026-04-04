package consensus

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"bobtorrent/pkg/torrent"

	"github.com/gorilla/websocket"
)

// Hub maintains the active WebSocket subscribers for the real-time lattice
// block feed used by the bobcoin frontend and the Go supernode TUI.
type Hub struct {
	clients map[*websocket.Conn]bool
	mu      sync.RWMutex
}

// NewHub creates an empty WebSocket hub.
func NewHub() *Hub {
	return &Hub{clients: make(map[*websocket.Conn]bool)}
}

// Register adds a WebSocket client to the broadcast set.
func (h *Hub) Register(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = true
	log.Printf("[ws] client connected (%d total)", len(h.clients))
}

// Unregister removes a WebSocket client and closes the socket.
func (h *Hub) Unregister(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[conn]; ok {
		delete(h.clients, conn)
		_ = conn.Close()
		log.Printf("[ws] client disconnected (%d remaining)", len(h.clients))
	}
}

// BlockEvent is the compatibility envelope broadcast whenever a block is
// confirmed. It includes both `type` and `event` so old and new clients can
// consume the same message without modification.
type BlockEvent struct {
	Type        string         `json:"type"`
	Event       string         `json:"event"`
	Block        *torrent.Block `json:"block"`
	StateHash    string         `json:"stateHash"`
	Timestamp    int64          `json:"timestamp"`
	Chains       int            `json:"chains"`
	Accounts     int            `json:"accounts"`
	TotalBlocks  int            `json:"totalBlocks"`
}

// BroadcastBlock sends a NEW_BLOCK event to every connected WebSocket client.
func (h *Hub) BroadcastBlock(block *torrent.Block, stateHash string, chains, totalBlocks int) {
	event := BlockEvent{
		Type:       "NEW_BLOCK",
		Event:      "NEW_BLOCK",
		Block:      block,
		StateHash:  stateHash,
		Timestamp:  time.Now().UnixMilli(),
		Chains:     chains,
		Accounts:   chains,
		TotalBlocks: totalBlocks,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("[ws] marshal failed: %v", err)
		return
	}

	h.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(h.clients))
	for conn := range h.clients {
		clients = append(clients, conn)
	}
	h.mu.RUnlock()

	for _, conn := range clients {
		go func(c *websocket.Conn) {
			_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.WriteMessage(websocket.TextMessage, payload); err != nil {
				log.Printf("[ws] write failed: %v", err)
				h.Unregister(c)
			}
		}(conn)
	}
}

// ClientCount returns the number of active WebSocket subscribers.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
