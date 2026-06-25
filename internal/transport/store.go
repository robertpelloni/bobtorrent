package transport

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// MessengerStore handles the durable persistence of gossip messages using SQLite.
type MessengerStore struct {
	db *sql.DB
}

// PersistedMessage represents a message stored in the database.
type PersistedMessage struct {
	ID        int64     `json:"id"`
	Topic     string    `json:"topic"`
	Sender    string    `json:"sender"`
	Data      string    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
}

// NewMessengerStore initializes a new SQLite store at the given path.
func NewMessengerStore(dbPath string) (*MessengerStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open messenger database: %w", err)
	}

	// Create messages table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			topic TEXT NOT NULL,
			sender TEXT NOT NULL,
			data TEXT NOT NULL,
			timestamp DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_topic_timestamp ON messages(topic, timestamp);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize messenger schema: %w", err)
	}

	return &MessengerStore{db: db}, nil
}

// SaveMessage persists a gossip message to the database.
func (s *MessengerStore) SaveMessage(topic, sender, data string) error {
	_, err := s.db.Exec(
		"INSERT INTO messages (topic, sender, data, timestamp) VALUES (?, ?, ?, ?)",
		topic, sender, data, time.Now(),
	)
	return err
}

// QueryHistory retrieves the last N messages for a given topic.
func (s *MessengerStore) QueryHistory(topic string, limit int) ([]PersistedMessage, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.Query(`
		SELECT id, topic, sender, data, timestamp
		FROM messages
		WHERE topic = ?
		ORDER BY timestamp DESC
		LIMIT ?`,
		topic, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []PersistedMessage
	for rows.Next() {
		var m PersistedMessage
		if err := rows.Scan(&m.ID, &m.Topic, &m.Sender, &m.Data, &m.Timestamp); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}

	// Reverse to return in chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// Close closes the database connection.
func (s *MessengerStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}
