package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bobtorrent/internal/transport"
	"github.com/gorilla/websocket"
)

func TestElementWebIntegrationEdgeCases(t *testing.T) {
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

	// 1. Join topic
	topic := "matrix-room-123"
	err = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"JOIN_TOPIC", "topic": "` + topic + `"}`))
	if err != nil {
		t.Fatalf("failed to write JOIN_TOPIC: %v", err)
	}

	time.Sleep(100 * time.Millisecond) // Allow join to process

	// 2. Avatar/profile metadata propagation via MatrixEvent
	matrixEvent := transport.MatrixEvent{
		Type:   "m.room.member",
		Sender: "@alice:bobtorrent.local",
		RoomID: topic,
		Content: map[string]interface{}{
			"membership": "join",
			"avatar_url": "mxc://bobtorrent.local/avatar123",
			"displayname": "Alice",
		},
		EventID:  "$event1",
		OriginTS: time.Now().UnixMilli(),
	}

	eventBytes, _ := json.Marshal(matrixEvent)

	// Simulate frontend publishing matrix event
	publishMsg := map[string]interface{}{
		"type": "PUBLISH",
		"topic": topic,
		"data": string(eventBytes),
	}

	err = conn.WriteJSON(publishMsg)
	if err != nil {
		t.Fatalf("failed to publish matrix event: %v", err)
	}

	// 3. Topic history persistence
	time.Sleep(200 * time.Millisecond) // Allow persistence

	history, err := store.QueryHistory(topic, 10)
	if err != nil {
		t.Fatalf("failed to query history: %v", err)
	}

	found := false
	for _, h := range history {
		if strings.Contains(h.Data, "mxc://bobtorrent.local/avatar123") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("avatar metadata not persisted in history")
	}

	// 4. Simulate high latency dispatching
	// (Just a conceptual delay test to ensure no panics/timeouts block the loop)
	time.Sleep(500 * time.Millisecond)

	// Fetch telemetry
	req, _ := http.NewRequest("GET", "/status", nil)
	rr := httptest.NewRecorder()
	// Assuming handleServiceStatus is mapped, we can just call it or mock it.
	// Since handleServiceStatus is in another file, we can invoke it if exported or just check basic HTTP response.
	handleServiceStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK from /status, got %d", rr.Code)
	}

	// Fetch peers telemetry
	resp, err := http.Get(server.URL + "/peers")
	if err == nil {
		defer resp.Body.Close()
	}
}
