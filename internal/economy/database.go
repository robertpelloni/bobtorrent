package economy

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Transaction records a mint/burn/economic-orchestration event exposed by the
// Go supernode compatibility layer. This mirrors the lightweight history the
// legacy Node game-server kept for UI visibility, but stores it durably using
// the same SQLite stack already adopted elsewhere in the Go workspace.
type Transaction struct {
	ID      string  `json:"id"`
	Date    string  `json:"date"`
	Amount  float64 `json:"amount"`
	Type    string  `json:"type"`
	Hash    string  `json:"hash"`
	Reason  string  `json:"reason,omitempty"`
	Address string  `json:"address,omitempty"`
}

// Database provides a minimal durable transaction history for the economic
// compatibility endpoints that are being ported away from the legacy Node
// game-server. It intentionally stays small and focused: a single transactions
// table plus append/list operations.
type Database struct {
	db *sql.DB
}

func NewDatabase(path string) (*Database, error) {
	if path == "" {
		return nil, fmt.Errorf("economy database path required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("failed to create economy database directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open economy database: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping economy database: %w", err)
	}

	d := &Database{db: db}
	if err := d.init(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize economy database schema: %w", err)
	}
	return d, nil
}

func (d *Database) init() error {
	queries := []string{
		`PRAGMA journal_mode = WAL`,
		`PRAGMA synchronous = FULL`,
		`CREATE TABLE IF NOT EXISTS transactions (
			id TEXT PRIMARY KEY,
			date TEXT NOT NULL,
			amount REAL NOT NULL,
			type TEXT NOT NULL,
			hash TEXT NOT NULL,
			reason TEXT,
			address TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(date DESC)`,
	}
	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (d *Database) RecordTransaction(tx Transaction) error {
	if tx.ID == "" {
		return fmt.Errorf("transaction id required")
	}
	if tx.Date == "" {
		tx.Date = time.Now().UTC().Format("2006-01-02 15:04")
	}
	if tx.Hash == "" {
		tx.Hash = "pending"
	}
	_, err := d.db.Exec(
		`INSERT INTO transactions (id, date, amount, type, hash, reason, address) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		tx.ID,
		tx.Date,
		tx.Amount,
		tx.Type,
		tx.Hash,
		tx.Reason,
		tx.Address,
	)
	if err != nil {
		return fmt.Errorf("failed to record economy transaction %s: %w", tx.ID, err)
	}
	return nil
}

func (d *Database) ListTransactions() ([]Transaction, error) {
	rows, err := d.db.Query(`SELECT id, date, amount, type, hash, reason, address FROM transactions ORDER BY date DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to list economy transactions: %w", err)
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(&tx.ID, &tx.Date, &tx.Amount, &tx.Type, &tx.Hash, &tx.Reason, &tx.Address); err != nil {
			return nil, fmt.Errorf("failed to scan economy transaction row: %w", err)
		}
		transactions = append(transactions, tx)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating economy transactions: %w", err)
	}
	return transactions, nil
}

func (d *Database) Close() error {
	if d == nil || d.db == nil {
		return nil
	}
	return d.db.Close()
}
