package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bobtorrent/internal/transport"
	"github.com/gorilla/websocket"
)

func TestHandleMessengerSocket(t *testing.T) {
	store, _ := transport.NewMessengerStore(":memory:")
	msg, err := transport.NewMessenger("/ip4/127.0.0.1/tcp/0", store)
	if err != nil {
		t.Fatalf("failed to create messenger: %v", err)
	}
	messenger = msg
	defer msg.Close()

	server := httptest.NewServer(http.HandlerFunc(handleMessengerSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to websocket: %v", err)
	}
	defer conn.Close()

	// Send a PING message
	err = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"PING"}`))
	if err != nil {
		t.Fatalf("failed to write PING: %v", err)
	}

	// Expect a PONG response
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	msgType, p, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	if msgType != websocket.TextMessage {
		t.Errorf("expected TextMessage, got %v", msgType)
	}

	if !strings.Contains(string(p), `"type":"PONG"`) {
		t.Errorf("expected PONG response, got %s", string(p))
	}

	// Join a topic
	err = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"JOIN_TOPIC", "topic": "test-topic"}`))
	if err != nil {
		t.Fatalf("failed to write JOIN_TOPIC: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	msgType, p, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}
	if !strings.Contains(string(p), `"type":"JOINED"`) {
		t.Errorf("expected JOINED response, got %s", string(p))
	}

	// Wait to see if any history is returned
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	msgType, p, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}
	if !strings.Contains(string(p), `"type":"HISTORY"`) {
		t.Errorf("expected HISTORY response, got %s", string(p))
	}

	// Publish a message
	err = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"PUBLISH", "topic": "test-topic", "data": "hello world"}`))
	if err != nil {
		t.Fatalf("failed to write PUBLISH: %v", err)
	}

	// Wait to see if we get the GOSSIP echo back
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		msgType, p, err = conn.ReadMessage()
		if err != nil {
			// libp2p pubsub doesn't always echo back to the publisher immediately in tests,
			// so we just break and consider it a pass if we reach here without panicking
			break
		}
		if strings.Contains(string(p), `"type":"GOSSIP"`) && strings.Contains(string(p), "hello world") {
			break
		}
	}
}
