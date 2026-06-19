package transport

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMessengerStore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "messenger-test-*")
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

	topic := "test-topic"
	messages := []struct {
		sender string
		data   string
	}{
		{"alice", "hello"},
		{"bob", "hi alice"},
		{"alice", "how are you?"},
	}

	for _, msg := range messages {
		if err := store.SaveMessage(topic, msg.sender, msg.data); err != nil {
			t.Errorf("failed to save message from %s: %v", msg.sender, err)
		}
	}

	history, err := store.QueryHistory(topic, 10, 0)
	if err != nil {
		t.Fatalf("failed to query history: %v", err)
	}

	if len(history) != len(messages) {
		t.Errorf("expected %d messages, got %d", len(messages), len(history))
	}

	for i, msg := range messages {
		if history[i].Sender != msg.sender || history[i].Data != msg.data {
			t.Errorf("message %d mismatch: expected %s: %s, got %s: %s",
				i, msg.sender, msg.data, history[i].Sender, history[i].Data)
		}
	}

	// Test limit
	history2, err := store.QueryHistory(topic, 2, 0)
	if err != nil {
		t.Fatalf("failed to query history with limit: %v", err)
	}
	if len(history2) != 2 {
		t.Errorf("expected 2 messages, got %d", len(history2))
	}
	// Last two messages in chronological order
	if history2[0].Data != "hi alice" || history2[1].Data != "how are you?" {
		t.Errorf("history with limit returned wrong messages or order")
	}

	// Test offset
	history3, err := store.QueryHistory(topic, 2, 1)
	if err != nil {
		t.Fatalf("failed to query history with offset: %v", err)
	}
	if len(history3) != 2 {
		t.Errorf("expected 2 messages, got %d", len(history3))
	}
	if history3[0].Data != "hello" || history3[1].Data != "hi alice" {
		t.Errorf("history with offset returned wrong messages or order")
	}
}
