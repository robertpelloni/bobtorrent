with open('cmd/supernode-go/mega_messenger_bridge.go', 'r') as f:
    content = f.read()

# Enhance the WebSocket bridge to support history sync and better integration with messenger logic
new_bridge = """package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/libp2p/go-libp2p/core/peer"
	"bobtorrent/internal/transport"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var (
	messengerClients = make(map[*websocket.Conn]string) // conn -> clientID
	messengerMutex   sync.Mutex
)

// handleMegaBridge handles WebSocket connections from the Mega-Messenger frontend (e.g., React/Flutter).
func handleMegaBridge(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[MegaBridge] upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		clientID = "unknown_client"
	}

	messengerMutex.Lock()
	messengerClients[conn] = clientID
	messengerMutex.Unlock()

	log.Printf("[MegaBridge] Light Node UI connected to local Heavy Node control plane. ClientID: %s", clientID)

	defer func() {
		messengerMutex.Lock()
		delete(messengerClients, conn)
		messengerMutex.Unlock()
		// Clean up subscriptions
		if messengerSvc != nil {
			messengerSvc.UnregisterAllHandlers(clientID)
		}
		log.Printf("[MegaBridge] Light Node UI disconnected. ClientID: %s", clientID)
	}()

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg struct {
			Action  string          `json:"action"`
			Topic   string          `json:"topic"`
			Payload json.RawMessage `json:"payload"`
			Limit   int             `json:"limit"`
		}
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[MegaBridge] UI read error: %v", err)
			}
			break
		}

		handleMessengerCommand(conn, clientID, msg.Action, msg.Topic, msg.Payload, msg.Limit)
	}
}

func handleMessengerCommand(conn *websocket.Conn, clientID string, action string, topic string, payload []byte, limit int) {
	if messengerSvc == nil {
		sendError(conn, "Messenger service not initialized")
		return
	}

	switch action {
	case "join":
		err := messengerSvc.JoinTopic(topic, clientID, func(data []byte, sender peer.ID) {
			broadcastToClient(conn, topic, "message", data, sender.String())
		})
		if err != nil {
			sendError(conn, "Failed to join topic: "+err.Error())
		} else {
			sendAck(conn, action, topic)
		}

	case "leave":
		messengerSvc.LeaveTopic(topic, clientID)
		sendAck(conn, action, topic)

	case "publish":
		err := messengerSvc.Publish(topic, payload)
		if err != nil {
			sendError(conn, "Failed to publish: "+err.Error())
		} else {
			sendAck(conn, action, topic)
		}

	case "history":
		if limit == 0 {
			limit = 50
		}
		history, err := messengerSvc.GetHistory(topic, limit)
		if err != nil {
			sendError(conn, "Failed to fetch history: "+err.Error())
		} else {
			sendHistory(conn, topic, history)
		}

	default:
		log.Printf("[MegaBridge] Unknown action: %s", action)
		sendError(conn, "Unknown action")
	}
}

func sendAck(conn *websocket.Conn, action, topic string) {
	conn.WriteJSON(map[string]interface{}{
		"type":   "ack",
		"action": action,
		"topic":  topic,
	})
}

func sendError(conn *websocket.Conn, msg string) {
	conn.WriteJSON(map[string]interface{}{
		"type":    "error",
		"message": msg,
	})
}

func broadcastToClient(conn *websocket.Conn, topic string, msgType string, data []byte, sender string) {
	messengerMutex.Lock()
	defer messengerMutex.Unlock()

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_ = conn.WriteJSON(map[string]interface{}{
		"type":   msgType,
		"topic":  topic,
		"sender": sender,
		"data":   json.RawMessage(data),
	})
}

func sendHistory(conn *websocket.Conn, topic string, history []transport.PersistedMessage) {
	messengerMutex.Lock()
	defer messengerMutex.Unlock()

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_ = conn.WriteJSON(map[string]interface{}{
		"type":    "history",
		"topic":   topic,
		"history": history,
	})
}
"""
with open('cmd/supernode-go/mega_messenger_bridge.go', 'w') as f:
    f.write(new_bridge)
