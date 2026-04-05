package consensus

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"bobtorrent/pkg/torrent"

	_ "modernc.org/sqlite"
)

const (
	// defaultLatticeSnapshotInterval controls how frequently the lattice writes a
	// materialized snapshot on top of the append-only confirmed block log.
	//
	// Why 25?
	//   - small enough to make local/dev cold boots noticeably faster even on
	//     modest histories
	//   - large enough to avoid snapshotting on nearly every block during normal
	//     usage
	//   - easy to reason about in tests and status output
	defaultLatticeSnapshotInterval int64 = 25

	// defaultLatticeSnapshotRetention keeps the newest few materialized snapshots
	// around so operators retain rollback/inspection room without unbounded
	// growth. The confirmed block log remains the durable source of truth.
	defaultLatticeSnapshotRetention = 3
)

// LatticeStore provides durable persistence for confirmed lattice blocks.
//
// Design rationale:
//   - We persist the canonical confirmed block log rather than trying to map
//     every derived in-memory index directly into relational tables on day one.
//   - The lattice already has deterministic state transition logic, so replaying
//     the confirmed block stream is enough to reconstruct chains, pending txs,
//     anchors, NFT ownership, governance state, and swaps after restart.
//   - Materialized snapshots now accelerate cold boot by letting recovery start
//     from a recent derived-state checkpoint before replaying only the tail.
//
// Durability contract:
//  1. every confirmed block is appended transactionally to SQLite
//  2. snapshots are best-effort optimizations layered on top of the block log
//  3. startup restores the newest snapshot if present, then replays newer blocks
//  4. if block append fails, the in-memory mutation is rolled back
type LatticeStore struct {
	db               *sql.DB
	path             string
	snapshotInterval int64
}

type storedBlock struct {
	Sequence int64
	Block    *torrent.Block
}

type storedSnapshot struct {
	LastSequence int64
	Snapshot     *persistedLatticeSnapshot
}

type LatticeIntegrityReport struct {
	Path                      string   `json:"path"`
	CheckedAt                 int64    `json:"checkedAt"`
	QuickCheckResult          string   `json:"quickCheckResult"`
	QuickCheckOK              bool     `json:"quickCheckOk"`
	Healthy                   bool     `json:"healthy"`
	Repairable                bool     `json:"repairable"`
	BlockCount                int64    `json:"blockCount"`
	LatestBlockSequence       int64    `json:"latestBlockSequence"`
	SnapshotCount             int64    `json:"snapshotCount"`
	LatestSnapshotSequence    int64    `json:"latestSnapshotSequence"`
	InvalidBlockSequences     []int64  `json:"invalidBlockSequences,omitempty"`
	InvalidSnapshotSequences  []int64  `json:"invalidSnapshotSequences,omitempty"`
	OrphanedSnapshotSequences []int64  `json:"orphanedSnapshotSequences,omitempty"`
	Notes                     []string `json:"notes,omitempty"`
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

	store := &LatticeStore{
		db:               db,
		path:             path,
		snapshotInterval: defaultLatticeSnapshotInterval,
	}
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
		`CREATE TABLE IF NOT EXISTS lattice_snapshots (
			snapshot_sequence INTEGER PRIMARY KEY,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			snapshot_json TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_lattice_snapshots_created_at ON lattice_snapshots(created_at)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

func (s *LatticeStore) AppendBlock(block *torrent.Block) (int64, error) {
	encoded, err := json.Marshal(block)
	if err != nil {
		return 0, fmt.Errorf("failed to encode block %s: %w", block.Hash, err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin lattice store transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	res, err := tx.Exec(
		`INSERT OR IGNORE INTO confirmed_blocks (hash, account, type, height, timestamp, block_json) VALUES (?, ?, ?, ?, ?, ?)`,
		block.Hash,
		block.Account,
		block.Type,
		block.Height,
		block.Timestamp,
		string(encoded),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to append confirmed block %s: %w", block.Hash, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to inspect append result for block %s: %w", block.Hash, err)
	}
	if affected == 0 {
		var existingSequence int64
		if err := tx.QueryRow(`SELECT sequence FROM confirmed_blocks WHERE hash = ?`, block.Hash).Scan(&existingSequence); err != nil {
			return 0, fmt.Errorf("failed to load existing sequence for duplicate block %s: %w", block.Hash, err)
		}
		if err := tx.Commit(); err != nil {
			return 0, fmt.Errorf("failed to commit duplicate lookup for block %s: %w", block.Hash, err)
		}
		return existingSequence, nil
	}

	sequence, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to read inserted sequence for block %s: %w", block.Hash, err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit confirmed block %s: %w", block.Hash, err)
	}

	return sequence, nil
}

func (s *LatticeStore) LoadBlocks() ([]storedBlock, error) {
	return s.LoadBlocksAfter(0)
}

func (s *LatticeStore) LoadBlocksAfter(sequence int64) ([]storedBlock, error) {
	rows, err := s.db.Query(`SELECT sequence, block_json FROM confirmed_blocks WHERE sequence > ? ORDER BY sequence ASC`, sequence)
	if err != nil {
		return nil, fmt.Errorf("failed to load confirmed blocks after sequence %d: %w", sequence, err)
	}
	defer rows.Close()

	var blocks []storedBlock
	for rows.Next() {
		var rowSequence int64
		var raw string
		if err := rows.Scan(&rowSequence, &raw); err != nil {
			return nil, fmt.Errorf("failed to scan confirmed block row: %w", err)
		}

		var block torrent.Block
		if err := json.Unmarshal([]byte(raw), &block); err != nil {
			return nil, fmt.Errorf("failed to decode confirmed block at sequence %d: %w", rowSequence, err)
		}

		blocks = append(blocks, storedBlock{Sequence: rowSequence, Block: &block})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating confirmed blocks: %w", err)
	}

	return blocks, nil
}

func (s *LatticeStore) StoreSnapshot(snapshot *persistedLatticeSnapshot) error {
	if snapshot == nil {
		return fmt.Errorf("snapshot required")
	}

	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to encode lattice snapshot at sequence %d: %w", snapshot.LastSequence, err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin snapshot transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.Exec(
		`INSERT INTO lattice_snapshots (snapshot_sequence, snapshot_json) VALUES (?, ?)
		 ON CONFLICT(snapshot_sequence) DO UPDATE SET snapshot_json = excluded.snapshot_json, created_at = CURRENT_TIMESTAMP`,
		snapshot.LastSequence,
		string(encoded),
	)
	if err != nil {
		return fmt.Errorf("failed to store lattice snapshot at sequence %d: %w", snapshot.LastSequence, err)
	}

	_, err = tx.Exec(
		`DELETE FROM lattice_snapshots
		 WHERE snapshot_sequence NOT IN (
			SELECT snapshot_sequence FROM lattice_snapshots ORDER BY snapshot_sequence DESC LIMIT ?
		 )`,
		defaultLatticeSnapshotRetention,
	)
	if err != nil {
		return fmt.Errorf("failed to trim old lattice snapshots: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit lattice snapshot at sequence %d: %w", snapshot.LastSequence, err)
	}

	return nil
}

func (s *LatticeStore) LoadLatestSnapshot() (*storedSnapshot, error) {
	row := s.db.QueryRow(`SELECT snapshot_sequence, snapshot_json FROM lattice_snapshots ORDER BY snapshot_sequence DESC LIMIT 1`)

	var sequence int64
	var raw string
	if err := row.Scan(&sequence, &raw); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load latest lattice snapshot: %w", err)
	}

	var snapshot persistedLatticeSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return nil, fmt.Errorf("failed to decode lattice snapshot at sequence %d: %w", sequence, err)
	}

	return &storedSnapshot{LastSequence: sequence, Snapshot: &snapshot}, nil
}

func (s *LatticeStore) CountBlocks() (int64, error) {
	var count int64
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM confirmed_blocks`).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count confirmed blocks: %w", err)
	}
	return count, nil
}

func (s *LatticeStore) CountSnapshots() (int64, error) {
	var count int64
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM lattice_snapshots`).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count lattice snapshots: %w", err)
	}
	return count, nil
}

func (s *LatticeStore) LatestSnapshotSequence() (int64, error) {
	var sequence sql.NullInt64
	if err := s.db.QueryRow(`SELECT MAX(snapshot_sequence) FROM lattice_snapshots`).Scan(&sequence); err != nil {
		return 0, fmt.Errorf("failed to read latest lattice snapshot sequence: %w", err)
	}
	if !sequence.Valid {
		return 0, nil
	}
	return sequence.Int64, nil
}

func (s *LatticeStore) ShouldSnapshot(currentSequence, lastSnapshotSequence int64) bool {
	if s == nil || s.snapshotInterval <= 0 || currentSequence <= 0 {
		return false
	}
	if currentSequence-lastSnapshotSequence < s.snapshotInterval {
		return false
	}
	return currentSequence%s.snapshotInterval == 0
}

func (s *LatticeStore) SnapshotInterval() int64 {
	if s == nil {
		return 0
	}
	return s.snapshotInterval
}

func (s *LatticeStore) VerifyIntegrity() (*LatticeIntegrityReport, error) {
	report := &LatticeIntegrityReport{
		Path:       s.path,
		CheckedAt:  time.Now().UnixMilli(),
		Healthy:    true,
		Repairable: true,
	}

	if err := s.db.QueryRow(`PRAGMA quick_check`).Scan(&report.QuickCheckResult); err != nil {
		return nil, fmt.Errorf("failed to run SQLite quick_check: %w", err)
	}
	report.QuickCheckOK = report.QuickCheckResult == "ok"
	if !report.QuickCheckOK {
		report.Healthy = false
		report.Notes = append(report.Notes, fmt.Sprintf("sqlite quick_check returned %q", report.QuickCheckResult))
	}

	if err := s.db.QueryRow(`SELECT COUNT(*), COALESCE(MAX(sequence), 0) FROM confirmed_blocks`).Scan(&report.BlockCount, &report.LatestBlockSequence); err != nil {
		return nil, fmt.Errorf("failed to summarize confirmed blocks: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*), COALESCE(MAX(snapshot_sequence), 0) FROM lattice_snapshots`).Scan(&report.SnapshotCount, &report.LatestSnapshotSequence); err != nil {
		return nil, fmt.Errorf("failed to summarize lattice snapshots: %w", err)
	}

	blockRows, err := s.db.Query(`SELECT sequence, block_json FROM confirmed_blocks ORDER BY sequence ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to iterate confirmed blocks for integrity verification: %w", err)
	}
	defer blockRows.Close()
	for blockRows.Next() {
		var sequence int64
		var raw string
		if err := blockRows.Scan(&sequence, &raw); err != nil {
			return nil, fmt.Errorf("failed to scan confirmed block during integrity verification: %w", err)
		}
		var block torrent.Block
		if err := json.Unmarshal([]byte(raw), &block); err != nil {
			report.InvalidBlockSequences = append(report.InvalidBlockSequences, sequence)
			continue
		}
		if block.Hash == "" || block.CalculateHash() != block.Hash {
			report.InvalidBlockSequences = append(report.InvalidBlockSequences, sequence)
		}
	}
	if err := blockRows.Err(); err != nil {
		return nil, fmt.Errorf("failed while verifying confirmed block integrity: %w", err)
	}

	rows, err := s.db.Query(`SELECT snapshot_sequence, snapshot_json FROM lattice_snapshots ORDER BY snapshot_sequence ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to iterate lattice snapshots for integrity verification: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sequence int64
		var raw string
		if err := rows.Scan(&sequence, &raw); err != nil {
			return nil, fmt.Errorf("failed to scan lattice snapshot during integrity verification: %w", err)
		}
		var snapshot persistedLatticeSnapshot
		if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
			report.InvalidSnapshotSequences = append(report.InvalidSnapshotSequences, sequence)
			continue
		}
		if snapshot.LastSequence != sequence || sequence > report.LatestBlockSequence {
			report.OrphanedSnapshotSequences = append(report.OrphanedSnapshotSequences, sequence)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while verifying lattice snapshots: %w", err)
	}

	if len(report.InvalidBlockSequences) > 0 {
		report.Healthy = false
		report.Repairable = false
		report.Notes = append(report.Notes, "confirmed block log contains invalid or hash-mismatched rows; manual recovery from the block log is required")
	}
	if len(report.InvalidSnapshotSequences) > 0 || len(report.OrphanedSnapshotSequences) > 0 {
		report.Healthy = false
		report.Notes = append(report.Notes, "snapshot layer can be safely rebuilt from the confirmed block log")
	}

	return report, nil
}

func (s *LatticeStore) ReplaceSnapshots(snapshot *persistedLatticeSnapshot) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin snapshot repair transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.Exec(`DELETE FROM lattice_snapshots`); err != nil {
		return fmt.Errorf("failed to clear lattice snapshots during repair: %w", err)
	}

	if snapshot != nil && snapshot.LastSequence > 0 {
		encoded, err := json.Marshal(snapshot)
		if err != nil {
			return fmt.Errorf("failed to encode rebuilt snapshot at sequence %d: %w", snapshot.LastSequence, err)
		}
		if _, err := tx.Exec(`INSERT INTO lattice_snapshots (snapshot_sequence, snapshot_json) VALUES (?, ?)`, snapshot.LastSequence, string(encoded)); err != nil {
			return fmt.Errorf("failed to persist rebuilt snapshot at sequence %d: %w", snapshot.LastSequence, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit snapshot repair transaction: %w", err)
	}
	return nil
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
