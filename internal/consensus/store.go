package consensus

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"bobtorrent/pkg/torrent"

	_ "modernc.org/sqlite"
)

// LatticeStore provides durable persistence for confirmed lattice blocks.
//
// Design rationale:
//   - We persist the canonical confirmed block log rather than trying to map
//     every derived in-memory index directly into relational tables on day one.
//   - The lattice already has deterministic state transition logic, so replaying
//     the confirmed block stream is enough to reconstruct chains, pending txs,
//     anchors, NFT ownership, governance state, and swaps after restart.
//   - This gives us durable crash recovery immediately while keeping the schema
//     small and migration-friendly.
//
// Future directions:
//   - persist periodic materialized snapshots for faster cold boots
//   - persist peer metadata / health information
//   - add retention/integrity checks for historical block log pruning
//   - store derived analytics if replay cost ever becomes significant
//
// For now, the durability contract is:
//  1. every confirmed block is appended transactionally to SQLite
//  2. startup replays blocks in commit order
//  3. if persistence fails, the in-memory mutation is rolled back
type LatticeStore struct {
	db   *sql.DB
	path string
}

type storedBlock struct {
	Sequence int64
	Block    *torrent.Block
}

func NewLatticeStore(path string) (*LatticeStore, error) {
	if path == "" {
		return nil, fmt.Errorf("lattice store path required")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("failed to create lattice store directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open lattice store: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping lattice store: %w", err)
	}

	store := &LatticeStore{db: db, path: path}
	if err := store.init(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize lattice store schema: %w", err)
	}

	return store, nil
}

func (s *LatticeStore) init() error {
	queries := []string{
		`PRAGMA journal_mode = WAL`,
		`PRAGMA synchronous = FULL`,
		`CREATE TABLE IF NOT EXISTS confirmed_blocks (
			sequence INTEGER PRIMARY KEY AUTOINCREMENT,
			hash TEXT NOT NULL UNIQUE,
			account TEXT NOT NULL,
			type TEXT NOT NULL,
			height INTEGER NOT NULL,
			timestamp INTEGER NOT NULL,
			block_json TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_confirmed_blocks_account_height ON confirmed_blocks(account, height)`,
		`CREATE INDEX IF NOT EXISTS idx_confirmed_blocks_timestamp ON confirmed_blocks(timestamp)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

func (s *LatticeStore) AppendBlock(block *torrent.Block) error {
	encoded, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to encode block %s: %w", block.Hash, err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin lattice store transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.Exec(
		`INSERT OR IGNORE INTO confirmed_blocks (hash, account, type, height, timestamp, block_json) VALUES (?, ?, ?, ?, ?, ?)`,
		block.Hash,
		block.Account,
		block.Type,
		block.Height,
		block.Timestamp,
		string(encoded),
	)
	if err != nil {
		return fmt.Errorf("failed to append confirmed block %s: %w", block.Hash, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit confirmed block %s: %w", block.Hash, err)
	}

	return nil
}

func (s *LatticeStore) LoadBlocks() ([]storedBlock, error) {
	rows, err := s.db.Query(`SELECT sequence, block_json FROM confirmed_blocks ORDER BY sequence ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to load confirmed blocks: %w", err)
	}
	defer rows.Close()

	var blocks []storedBlock
	for rows.Next() {
		var sequence int64
		var raw string
		if err := rows.Scan(&sequence, &raw); err != nil {
			return nil, fmt.Errorf("failed to scan confirmed block row: %w", err)
		}

		var block torrent.Block
		if err := json.Unmarshal([]byte(raw), &block); err != nil {
			return nil, fmt.Errorf("failed to decode confirmed block at sequence %d: %w", sequence, err)
		}

		blocks = append(blocks, storedBlock{Sequence: sequence, Block: &block})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating confirmed blocks: %w", err)
	}

	return blocks, nil
}

func (s *LatticeStore) CountBlocks() (int64, error) {
	var count int64
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM confirmed_blocks`).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count confirmed blocks: %w", err)
	}
	return count, nil
}

func (s *LatticeStore) Path() string {
	return s.path
}

func (s *LatticeStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
