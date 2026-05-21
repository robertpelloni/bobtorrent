package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// The Mega-Messenger Bridge
// This endpoint acts as the Local Control Plane interface for "Light Nodes"
// (e.g. the React/Flutter chat frontend derived from element-web).
// Instead of the frontend participating directly in DHT routing or P2P swarms,
// it connects to this local WebSocket to push and pull blinded Protobuf Envelopes.

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow local UI clients
		return true
	},
}

func handleMegaBridge(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[MegaBridge] upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Println("[MegaBridge] Light Node UI connected to local Heavy Node control plane.")

	// Set connection timeouts
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		// Expecting JSON-wrapped payloads from the UI that will eventually be
		// serialized into Protobuf Envelopes and routed via go-libp2p-pubsub.
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[MegaBridge] UI read error: %v", err)
			}
			break
		}

		// STUB: Log incoming payload.
		// Future: Parse msg -> envelope.proto -> broadcast to libp2p mesh.
		msgType, _ := msg["type"].(string)
		log.Printf("[MegaBridge] Received Light Node payload type: %s", msgType)

		// Echo acknowledgment to the UI
		_ = conn.WriteJSON(map[string]interface{}{
			"status": "ack",
			"type":   msgType,
		})
	}

	log.Println("[MegaBridge] Light Node UI disconnected.")
}
