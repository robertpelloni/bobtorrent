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

// Live delivery regression: two dashboard WS clients on the same node and
// topic — when A publishes, B must receive the GOSSIP frame live (not only
// via history on rejoin). libp2p readLoop skips own-host messages, so the
// messenger must fan out locally to registered handlers.
func TestMessengerSocketDeliversToPeerClient(t *testing.T) {
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

	dial := func() *websocket.Conn {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		return conn
	}
	a, b := dial(), dial()
	defer a.Close()
	defer b.Close()

	joinAndDrain := func(conn *websocket.Conn) {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"JOIN_TOPIC","topic":"delivery-topic"}`)); err != nil {
			t.Fatalf("join write: %v", err)
		}
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		for i := 0; i < 2; i++ { // JOINED + HISTORY
			if _, _, err := conn.ReadMessage(); err != nil {
				t.Fatalf("join drain: %v", err)
			}
		}
	}
	joinAndDrain(a)
	joinAndDrain(b)

	if err := a.WriteMessage(websocket.TextMessage, []byte(`{"type":"PUBLISH","topic":"delivery-topic","data":"peer delivery check"}`)); err != nil {
		t.Fatalf("publish write: %v", err)
	}

	b.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		_, p, err := b.ReadMessage()
		if err != nil {
			t.Fatalf("peer client never received live GOSSIP: %v", err)
		}
		if strings.Contains(string(p), `"type":"GOSSIP"`) && strings.Contains(string(p), "peer delivery check") {
			return
		}
	}
}
