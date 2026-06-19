package transport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

func TestMessengerTypingIndicatorPersistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "messenger-typing-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "messenger.db")
	store, err := NewMessengerStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	messenger, err := NewMessenger("/ip4/127.0.0.1/tcp/0", store)
	if err != nil {
		t.Fatalf("failed to create messenger: %v", err)
	}
	defer messenger.Close()

	topicName := "test-typing-topic"
	handlerCalled := make(chan []byte, 10)
	err = messenger.JoinTopic(topicName, "test-handler", func(data []byte, from peer.ID) {
		// Mock handler
		handlerCalled <- data
	})
	if err != nil {
		t.Fatalf("failed to join topic: %v", err)
	}

	// We'll directly use Publish to test persistence logic of outbound messages.
	// We'll also test inbound persistence using a separate test or just trust the logic
	// is the same since we modified both to use !isTypingIndicator.

	// Normal message
	normalEvent := MatrixEvent{
		Type:    "m.room.message",
		Sender:  "@alice:example.com",
		RoomID:  "!room:example.com",
		Content: map[string]interface{}{"body": "Hello"},
	}
	if err := messenger.PublishMatrixEvent(topicName, normalEvent); err != nil {
		t.Fatalf("failed to publish normal event: %v", err)
	}

	// Typing indicator message
	typingEvent := MatrixEvent{
		Type:    "m.typing",
		Sender:  "@alice:example.com",
		RoomID:  "!room:example.com",
		Content: map[string]interface{}{"user_ids": []string{"@alice:example.com"}},
	}
	if err := messenger.PublishMatrixEvent(topicName, typingEvent); err != nil {
		t.Fatalf("failed to publish typing event: %v", err)
	}

	// Wait briefly for persistence to settle (Publish is synchronous to store, but readLoop is not, we're testing Publish here)
	time.Sleep(100 * time.Millisecond)

	history, err := store.QueryHistory(topicName, 10)
	if err != nil {
		t.Fatalf("failed to query history: %v", err)
	}

	if len(history) != 1 {
		t.Fatalf("expected exactly 1 message persisted, got %d", len(history))
	}

	var parsed MatrixEvent
	if err := json.Unmarshal([]byte(history[0].Data), &parsed); err != nil {
		t.Fatalf("failed to parse persisted history: %v", err)
	}
	if parsed.Type != "m.room.message" {
		t.Errorf("expected m.room.message, got %s", parsed.Type)
	}
}

func TestMessengerRateLimiting(t *testing.T) {
	messenger, err := NewMessenger("/ip4/127.0.0.1/tcp/0", nil)
	if err != nil {
		t.Fatalf("failed to create messenger: %v", err)
	}
	defer messenger.Close()

	topicName := "test-rate-limit"
	err = messenger.JoinTopic(topicName, "test-handler", func(data []byte, from peer.ID) {})
	if err != nil {
		t.Fatalf("failed to join topic: %v", err)
	}

	// The limiter is configured for 5 req/sec with a burst of 10.
	// So publishing 15 messages back-to-back should hit the limit.
	successCount := 0
	errorCount := 0
	for i := 0; i < 15; i++ {
		err := messenger.Publish(topicName, []byte("test data"))
		if err == nil {
			successCount++
		} else {
			errorCount++
		}
	}

	if successCount > 10 {
		t.Errorf("expected max 10 successes due to burst limit, got %d", successCount)
	}
	if errorCount == 0 {
		t.Errorf("expected rate limit errors, but got 0")
	}
}
