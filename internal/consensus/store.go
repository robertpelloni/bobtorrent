package consensus

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bobtorrent/pkg/torrent"

	"github.com/mr-tron/base58/base58"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/scrypt"
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

	// secureBackupBundleFormatVersion identifies the operator-facing encrypted
	// backup package format introduced on top of the existing portable SQLite
	// backup flow. This is intentionally a side-channel artifact wrapper rather
	// than a replacement for the plain backup/restore primitives.
	secureBackupBundleFormatVersion = "bobtorrent-secure-backup-bundle-v1"

	// Scrypt parameters are deliberately moderate: expensive enough to make
	// casual offline guessing more annoying, but still practical for operators on
	// modest hardware and CI test environments.
	secureBackupScryptN       = 1 << 15
	secureBackupScryptR       = 8
	secureBackupScryptP       = 1
	secureBackupDerivedKeyLen = chacha20poly1305.KeySize
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
type SnapshotConfig struct {
	Interval  int64 `json:"interval"`
	Retention int   `json:"retention"`
}

type LatticeStore struct {
	db                *sql.DB
	path              string
	snapshotInterval  int64
	snapshotRetention int
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

type ExportedConfirmedBlock struct {
	Sequence int64          `json:"sequence"`
	Block    *torrent.Block `json:"block"`
}

type LatticeExportBundle struct {
	Path                   string                    `json:"path"`
	ExportedAt             int64                     `json:"exportedAt"`
	SnapshotInterval       int64                     `json:"snapshotInterval"`
	SnapshotRetention      int                       `json:"snapshotRetention"`
	Integrity              *LatticeIntegrityReport   `json:"integrity,omitempty"`
	LatestSnapshot         *persistedLatticeSnapshot `json:"latestSnapshot,omitempty"`
	ConfirmedBlocks        []ExportedConfirmedBlock  `json:"confirmedBlocks"`
	LatestBlockSequence    int64                     `json:"latestBlockSequence"`
	LatestSnapshotSequence int64                     `json:"latestSnapshotSequence"`
}

type LatticeBackupResult struct {
	SourcePath    string `json:"sourcePath"`
	BackupPath    string `json:"backupPath"`
	CreatedAt     int64  `json:"createdAt"`
	BlockCount    int64  `json:"blockCount"`
	SnapshotCount int64  `json:"snapshotCount"`
}

type LatticeRestoreResult struct {
	Mode           string                  `json:"mode"`
	SourcePath     string                  `json:"sourcePath,omitempty"`
	TargetPath     string                  `json:"targetPath"`
	CreatedAt      int64                   `json:"createdAt"`
	BlockCount     int64                   `json:"blockCount"`
	SnapshotCount  int64                   `json:"snapshotCount"`
	LatestSequence int64                   `json:"latestSequence"`
	Integrity      *LatticeIntegrityReport `json:"integrity,omitempty"`
}

// SecureBundleKDF captures how the operator backup bundle derived its symmetric
// encryption key from the supplied passphrase. The values are persisted so the
// restore side can deterministically re-derive the same key.
type SecureBundleKDF struct {
	Name   string `json:"name"`
	Salt   string `json:"salt"`
	N      int    `json:"n"`
	R      int    `json:"r"`
	P      int    `json:"p"`
	KeyLen int    `json:"keyLen"`
}

// SecureBundleSignature records optional Ed25519 metadata for operator bundle
// authenticity. The signature is over a deterministic hash of the bundle's
// critical metadata and ciphertext hashes so tampering can be detected before
// restore.
type SecureBundleSignature struct {
	PublicKey   string `json:"publicKey"`
	MessageHash string `json:"messageHash"`
	Signature   string `json:"signature"`
}

// SecureLatticeBackupBundle is a portable encrypted wrapper around a verified
// SQLite backup copy. It deliberately packages a side-channel backup artifact so
// the live node still follows the conservative no-hot-swap durability model.
type SecureLatticeBackupBundle struct {
	Format         string                 `json:"format"`
	CreatedAt      int64                  `json:"createdAt"`
	SourcePath     string                 `json:"sourcePath"`
	ArtifactType   string                 `json:"artifactType"`
	Cipher         string                 `json:"cipher"`
	KDF            SecureBundleKDF        `json:"kdf"`
	Nonce          string                 `json:"nonce"`
	Ciphertext     string                 `json:"ciphertext"`
	PlaintextHash  string                 `json:"plaintextHash"`
	CiphertextHash string                 `json:"ciphertextHash"`
	PlaintextSize  int64                  `json:"plaintextSize"`
	Backup         *LatticeBackupResult   `json:"backup,omitempty"`
	Signature      *SecureBundleSignature `json:"signature,omitempty"`
}

// LatticeSecureBackupResult describes the newly created encrypted bundle plus
// whether it was signed. The actual bundle payload is persisted to disk rather
// than returned inline over every API path, but callers still receive the core
// metadata for logging and operator UX.
type LatticeSecureBackupResult struct {
	BundlePath     string                 `json:"bundlePath"`
	CreatedAt      int64                  `json:"createdAt"`
	SourcePath     string                 `json:"sourcePath"`
	ArtifactType   string                 `json:"artifactType"`
	PlaintextSize  int64                  `json:"plaintextSize"`
	PlaintextHash  string                 `json:"plaintextHash"`
	CiphertextHash string                 `json:"ciphertextHash"`
	Signed         bool                   `json:"signed"`
	Signature      *SecureBundleSignature `json:"signature,omitempty"`
	Backup         *LatticeBackupResult   `json:"backup,omitempty"`
}

// LatticeSecureRestoreResult describes the safe restore path from an encrypted
// operator bundle into a fresh verified lattice database.
type LatticeSecureRestoreResult struct {
	BundlePath         string                `json:"bundlePath"`
	TargetPath         string                `json:"targetPath"`
	CreatedAt          int64                 `json:"createdAt"`
	ArtifactType       string                `json:"artifactType"`
	PlaintextHash      string                `json:"plaintextHash"`
	CiphertextHash     string                `json:"ciphertextHash"`
	SignatureVerified  bool                  `json:"signatureVerified"`
	SignaturePublicKey string                `json:"signaturePublicKey,omitempty"`
	Restore            *LatticeRestoreResult `json:"restore"`
}

func DefaultSnapshotConfig() SnapshotConfig {
	return SnapshotConfig{
		Interval:  defaultLatticeSnapshotInterval,
		Retention: defaultLatticeSnapshotRetention,
	}
}

func normalizeSnapshotConfig(config SnapshotConfig) (SnapshotConfig, error) {
	if config.Interval < 0 {
		return SnapshotConfig{}, fmt.Errorf("snapshot interval cannot be negative")
	}
	if config.Retention <= 0 {
		return SnapshotConfig{}, fmt.Errorf("snapshot retention must be at least 1")
	}
	return config, nil
}

func SnapshotConfigFromEnv() (SnapshotConfig, error) {
	config := DefaultSnapshotConfig()
	if raw := strings.TrimSpace(os.Getenv("BOBTORRENT_LATTICE_SNAPSHOT_INTERVAL")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return SnapshotConfig{}, fmt.Errorf("invalid BOBTORRENT_LATTICE_SNAPSHOT_INTERVAL %q: %w", raw, err)
		}
		config.Interval = value
	}
	if raw := strings.TrimSpace(os.Getenv("BOBTORRENT_LATTICE_SNAPSHOT_RETENTION")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return SnapshotConfig{}, fmt.Errorf("invalid BOBTORRENT_LATTICE_SNAPSHOT_RETENTION %q: %w", raw, err)
		}
		config.Retention = value
	}
	return normalizeSnapshotConfig(config)
}

func NewLatticeStore(path string) (*LatticeStore, error) {
	return NewLatticeStoreWithConfig(path, DefaultSnapshotConfig())
}

func NewLatticeStoreWithConfig(path string, config SnapshotConfig) (*LatticeStore, error) {
	config, err := normalizeSnapshotConfig(config)
	if err != nil {
		return nil, err
	}
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
		db:                db,
		path:              path,
		snapshotInterval:  config.Interval,
		snapshotRetention: config.Retention,
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
		s.snapshotRetention,
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

func (s *LatticeStore) SnapshotRetention() int {
	if s == nil {
		return 0
	}
	return s.snapshotRetention
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

func (s *LatticeStore) ExportBundle() (*LatticeExportBundle, error) {
	integrity, err := s.VerifyIntegrity()
	if err != nil {
		return nil, err
	}

	storedBlocks, err := s.LoadBlocks()
	if err != nil {
		return nil, err
	}
	confirmedBlocks := make([]ExportedConfirmedBlock, 0, len(storedBlocks))
	for _, entry := range storedBlocks {
		confirmedBlocks = append(confirmedBlocks, ExportedConfirmedBlock{
			Sequence: entry.Sequence,
			Block:    entry.Block,
		})
	}

	var latestSnapshot *persistedLatticeSnapshot
	if integrity.QuickCheckOK && len(integrity.InvalidSnapshotSequences) == 0 {
		if storedSnapshot, err := s.LoadLatestSnapshot(); err == nil && storedSnapshot != nil {
			latestSnapshot = storedSnapshot.Snapshot
		}
	}

	return &LatticeExportBundle{
		Path:                   s.path,
		ExportedAt:             time.Now().UnixMilli(),
		SnapshotInterval:       s.snapshotInterval,
		SnapshotRetention:      s.snapshotRetention,
		Integrity:              integrity,
		LatestSnapshot:         latestSnapshot,
		ConfirmedBlocks:        confirmedBlocks,
		LatestBlockSequence:    integrity.LatestBlockSequence,
		LatestSnapshotSequence: integrity.LatestSnapshotSequence,
	}, nil
}

func (s *LatticeStore) CreateBackup(targetPath string) (*LatticeBackupResult, error) {
	if targetPath == "" {
		backupDir := filepath.Join(filepath.Dir(s.path), "backups")
		targetPath = filepath.Join(backupDir, fmt.Sprintf("lattice-%s.db", time.Now().UTC().Format("20060102-150405")))
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create lattice backup directory: %w", err)
	}
	if _, err := os.Stat(targetPath); err == nil {
		return nil, fmt.Errorf("backup path already exists: %s", targetPath)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to inspect backup path %s: %w", targetPath, err)
	}

	if _, err := s.db.Exec(`PRAGMA wal_checkpoint(FULL)`); err != nil {
		return nil, fmt.Errorf("failed to checkpoint WAL before backup: %w", err)
	}

	quotedPath := strings.ReplaceAll(targetPath, "'", "''")
	if _, err := s.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", quotedPath)); err != nil {
		return nil, fmt.Errorf("failed to create SQLite backup at %s: %w", targetPath, err)
	}

	blockCount, err := s.CountBlocks()
	if err != nil {
		return nil, err
	}
	snapshotCount, err := s.CountSnapshots()
	if err != nil {
		return nil, err
	}

	return &LatticeBackupResult{
		SourcePath:    s.path,
		BackupPath:    targetPath,
		CreatedAt:     time.Now().UnixMilli(),
		BlockCount:    blockCount,
		SnapshotCount: snapshotCount,
	}, nil
}

func randomBase64(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

func deriveSecureBundleKey(passphrase string, kdf SecureBundleKDF) ([]byte, error) {
	if strings.TrimSpace(passphrase) == "" {
		return nil, fmt.Errorf("bundle passphrase required")
	}
	salt, err := base64.StdEncoding.DecodeString(kdf.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode bundle salt: %w", err)
	}
	return scrypt.Key([]byte(passphrase), salt, kdf.N, kdf.R, kdf.P, kdf.KeyLen)
}

func secureBundleSigningMessage(bundle *SecureLatticeBackupBundle) string {
	if bundle == nil {
		return ""
	}
	message := strings.Join([]string{
		bundle.Format,
		bundle.ArtifactType,
		fmt.Sprintf("%d", bundle.CreatedAt),
		bundle.SourcePath,
		bundle.Cipher,
		bundle.PlaintextHash,
		bundle.CiphertextHash,
		fmt.Sprintf("%d", bundle.PlaintextSize),
		bundle.KDF.Name,
		bundle.KDF.Salt,
		fmt.Sprintf("%d", bundle.KDF.N),
		fmt.Sprintf("%d", bundle.KDF.R),
		fmt.Sprintf("%d", bundle.KDF.P),
		fmt.Sprintf("%d", bundle.KDF.KeyLen),
		bundle.Nonce,
	}, "|")
	return torrent.HashSHA256(message)
}

func signerPublicKeyFromPrivateKey(privateKeyBase58 string) (string, error) {
	keypairBytes, err := base58Decode(privateKeyBase58)
	if err != nil {
		return "", err
	}
	if len(keypairBytes) != 64 {
		return "", fmt.Errorf("invalid operator signing private key size: %d", len(keypairBytes))
	}
	return base58Encode(keypairBytes[32:]), nil
}

func base58Decode(value string) ([]byte, error) {
	return base58.Decode(value)
}

func base58Encode(value []byte) string {
	return base58.Encode(value)
}

func buildSecureBackupBundle(sourcePath string, backup *LatticeBackupResult, backupBytes []byte, passphrase, signingPrivateKey string) (*SecureLatticeBackupBundle, error) {
	salt, err := randomBase64(16)
	if err != nil {
		return nil, fmt.Errorf("failed to generate bundle salt: %w", err)
	}
	nonceValue, err := randomBase64(chacha20poly1305.NonceSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate bundle nonce: %w", err)
	}
	kdf := SecureBundleKDF{
		Name:   "scrypt",
		Salt:   salt,
		N:      secureBackupScryptN,
		R:      secureBackupScryptR,
		P:      secureBackupScryptP,
		KeyLen: secureBackupDerivedKeyLen,
	}
	key, err := deriveSecureBundleKey(passphrase, kdf)
	if err != nil {
		return nil, err
	}
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize bundle cipher: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(nonceValue)
	if err != nil {
		return nil, fmt.Errorf("failed to decode generated nonce: %w", err)
	}
	ciphertext := aead.Seal(nil, nonce, backupBytes, nil)
	bundle := &SecureLatticeBackupBundle{
		Format:         secureBackupBundleFormatVersion,
		CreatedAt:      time.Now().UnixMilli(),
		SourcePath:     sourcePath,
		ArtifactType:   "sqlite-backup",
		Cipher:         "chacha20poly1305",
		KDF:            kdf,
		Nonce:          nonceValue,
		Ciphertext:     base64.StdEncoding.EncodeToString(ciphertext),
		PlaintextHash:  torrent.HashSHA256(string(backupBytes)),
		CiphertextHash: torrent.HashSHA256(string(ciphertext)),
		PlaintextSize:  int64(len(backupBytes)),
		Backup:         backup,
	}
	if strings.TrimSpace(signingPrivateKey) != "" {
		messageHash := secureBundleSigningMessage(bundle)
		signature, err := torrent.Sign(messageHash, signingPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to sign secure backup bundle: %w", err)
		}
		publicKey, err := signerPublicKeyFromPrivateKey(signingPrivateKey)
		if err != nil {
			return nil, err
		}
		bundle.Signature = &SecureBundleSignature{
			PublicKey:   publicKey,
			MessageHash: messageHash,
			Signature:   signature,
		}
	}
	return bundle, nil
}

func writeSecureBackupBundle(path string, bundle *SecureLatticeBackupBundle) error {
	if path == "" {
		return fmt.Errorf("secure bundle target path required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create secure bundle directory: %w", err)
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("secure bundle target already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect secure bundle target %s: %w", path, err)
	}
	encoded, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode secure backup bundle: %w", err)
	}
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		return fmt.Errorf("failed to write secure backup bundle %s: %w", path, err)
	}
	return nil
}

func (s *LatticeStore) CreateSignedEncryptedBackupBundle(targetPath, passphrase, signingPrivateKey string) (*LatticeSecureBackupResult, error) {
	tempBackupPath := filepath.Join(os.TempDir(), fmt.Sprintf("bobtorrent-secure-backup-%d.db", time.Now().UnixNano()))
	backup, err := s.CreateBackup(tempBackupPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = os.Remove(tempBackupPath)
	}()
	backupBytes, err := os.ReadFile(tempBackupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read temporary lattice backup: %w", err)
	}
	bundle, err := buildSecureBackupBundle(s.path, backup, backupBytes, passphrase, signingPrivateKey)
	if err != nil {
		return nil, err
	}
	if targetPath == "" {
		bundleDir := filepath.Join(filepath.Dir(s.path), "backups")
		targetPath = filepath.Join(bundleDir, fmt.Sprintf("lattice-%s.secure-backup.json", time.Now().UTC().Format("20060102-150405")))
	}
	if err := writeSecureBackupBundle(targetPath, bundle); err != nil {
		return nil, err
	}
	return &LatticeSecureBackupResult{
		BundlePath:     targetPath,
		CreatedAt:      bundle.CreatedAt,
		SourcePath:     s.path,
		ArtifactType:   bundle.ArtifactType,
		PlaintextSize:  bundle.PlaintextSize,
		PlaintextHash:  bundle.PlaintextHash,
		CiphertextHash: bundle.CiphertextHash,
		Signed:         bundle.Signature != nil,
		Signature:      bundle.Signature,
		Backup:         backup,
	}, nil
}

func RestoreSignedEncryptedBackupBundleToPath(bundlePath, passphrase, targetPath string, requireSignature bool) (*LatticeSecureRestoreResult, error) {
	if strings.TrimSpace(bundlePath) == "" {
		return nil, fmt.Errorf("secure bundle source path required")
	}
	raw, err := os.ReadFile(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secure backup bundle %s: %w", bundlePath, err)
	}
	var bundle SecureLatticeBackupBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return nil, fmt.Errorf("failed to decode secure backup bundle: %w", err)
	}
	if bundle.Format != secureBackupBundleFormatVersion {
		return nil, fmt.Errorf("unsupported secure bundle format: %s", bundle.Format)
	}
	if bundle.ArtifactType != "sqlite-backup" {
		return nil, fmt.Errorf("unsupported secure bundle artifact type: %s", bundle.ArtifactType)
	}
	signatureVerified := false
	if bundle.Signature != nil {
		expected := secureBundleSigningMessage(&bundle)
		if bundle.Signature.MessageHash != expected {
			return nil, fmt.Errorf("secure bundle signature metadata mismatch")
		}
		if !torrent.Verify(bundle.Signature.MessageHash, bundle.Signature.Signature, bundle.Signature.PublicKey) {
			return nil, fmt.Errorf("secure bundle signature verification failed")
		}
		signatureVerified = true
	} else if requireSignature {
		return nil, fmt.Errorf("secure bundle signature required but missing")
	}
	key, err := deriveSecureBundleKey(passphrase, bundle.KDF)
	if err != nil {
		return nil, err
	}
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize secure bundle cipher: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(bundle.Nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to decode secure bundle nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(bundle.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode secure bundle ciphertext: %w", err)
	}
	if torrent.HashSHA256(string(ciphertext)) != bundle.CiphertextHash {
		return nil, fmt.Errorf("secure bundle ciphertext hash mismatch")
	}
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secure backup bundle: %w", err)
	}
	if int64(len(plaintext)) != bundle.PlaintextSize {
		return nil, fmt.Errorf("secure bundle plaintext size mismatch: expected %d, got %d", bundle.PlaintextSize, len(plaintext))
	}
	if torrent.HashSHA256(string(plaintext)) != bundle.PlaintextHash {
		return nil, fmt.Errorf("secure bundle plaintext hash mismatch")
	}
	tempSourcePath := filepath.Join(os.TempDir(), fmt.Sprintf("bobtorrent-decrypted-backup-%d.db", time.Now().UnixNano()))
	if err := os.WriteFile(tempSourcePath, plaintext, 0600); err != nil {
		return nil, fmt.Errorf("failed to materialize decrypted backup: %w", err)
	}
	defer func() {
		_ = os.Remove(tempSourcePath)
	}()
	restore, err := RestoreBackupToPath(tempSourcePath, targetPath)
	if err != nil {
		return nil, err
	}
	result := &LatticeSecureRestoreResult{
		BundlePath:        bundlePath,
		TargetPath:        restore.TargetPath,
		CreatedAt:         time.Now().UnixMilli(),
		ArtifactType:      bundle.ArtifactType,
		PlaintextHash:     bundle.PlaintextHash,
		CiphertextHash:    bundle.CiphertextHash,
		SignatureVerified: signatureVerified,
		Restore:           restore,
	}
	if bundle.Signature != nil {
		result.SignaturePublicKey = bundle.Signature.PublicKey
	}
	return result, nil
}

func defaultRestoreTarget(basePath, prefix string) string {
	name := fmt.Sprintf("%s-%s.db", prefix, time.Now().UTC().Format("20060102-150405"))
	return filepath.Join(filepath.Dir(basePath), "restored", name)
}

func validateExportBundle(bundle *LatticeExportBundle) error {
	if bundle == nil {
		return fmt.Errorf("export bundle required")
	}
	lastSequence := int64(0)
	for index, entry := range bundle.ConfirmedBlocks {
		if entry.Block == nil {
			return fmt.Errorf("confirmed block entry %d is nil", index)
		}
		if entry.Sequence <= lastSequence {
			return fmt.Errorf("confirmed block sequence %d is not strictly increasing", entry.Sequence)
		}
		if entry.Block.Hash == "" || entry.Block.CalculateHash() != entry.Block.Hash {
			return fmt.Errorf("confirmed block sequence %d has invalid hash integrity", entry.Sequence)
		}
		lastSequence = entry.Sequence
	}
	if bundle.LatestSnapshot != nil {
		if bundle.LatestSnapshot.LastSequence <= 0 {
			return fmt.Errorf("latest snapshot sequence must be positive when present")
		}
		if bundle.LatestSnapshot.LastSequence > lastSequence {
			return fmt.Errorf("latest snapshot sequence %d exceeds latest confirmed block sequence %d", bundle.LatestSnapshot.LastSequence, lastSequence)
		}
	}
	return nil
}

func ImportBundleToPath(targetPath string, bundle *LatticeExportBundle) (*LatticeRestoreResult, error) {
	if targetPath == "" {
		basePath := "data/lattice/lattice.db"
		if bundle != nil && bundle.Path != "" {
			basePath = bundle.Path
		}
		targetPath = defaultRestoreTarget(basePath, "imported-lattice")
	}
	if err := validateExportBundle(bundle); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create import target directory: %w", err)
	}
	if _, err := os.Stat(targetPath); err == nil {
		return nil, fmt.Errorf("import target already exists: %s", targetPath)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to inspect import target %s: %w", targetPath, err)
	}

	config := DefaultSnapshotConfig()
	if bundle != nil {
		if bundle.SnapshotInterval >= 0 {
			config.Interval = bundle.SnapshotInterval
		}
		if bundle.SnapshotRetention > 0 {
			config.Retention = bundle.SnapshotRetention
		}
	}
	store, err := NewLatticeStoreWithConfig(targetPath, config)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = store.Close()
	}()

	tx, err := store.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin lattice import transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, entry := range bundle.ConfirmedBlocks {
		encoded, err := json.Marshal(entry.Block)
		if err != nil {
			return nil, fmt.Errorf("failed to encode confirmed block sequence %d during import: %w", entry.Sequence, err)
		}
		_, err = tx.Exec(
			`INSERT INTO confirmed_blocks (sequence, hash, account, type, height, timestamp, block_json) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			entry.Sequence,
			entry.Block.Hash,
			entry.Block.Account,
			entry.Block.Type,
			entry.Block.Height,
			entry.Block.Timestamp,
			string(encoded),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to import confirmed block sequence %d: %w", entry.Sequence, err)
		}
	}

	if bundle.LatestSnapshot != nil {
		encoded, err := json.Marshal(bundle.LatestSnapshot)
		if err != nil {
			return nil, fmt.Errorf("failed to encode imported snapshot: %w", err)
		}
		_, err = tx.Exec(`INSERT INTO lattice_snapshots (snapshot_sequence, snapshot_json) VALUES (?, ?)`, bundle.LatestSnapshot.LastSequence, string(encoded))
		if err != nil {
			return nil, fmt.Errorf("failed to import snapshot sequence %d: %w", bundle.LatestSnapshot.LastSequence, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit lattice import transaction: %w", err)
	}

	reloaded, err := NewPersistentLattice(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to reopen imported lattice database: %w", err)
	}
	defer func() {
		_ = reloaded.Close()
	}()
	integrity, err := reloaded.VerifyPersistence()
	if err != nil {
		return nil, err
	}
	if !integrity.Healthy {
		return nil, fmt.Errorf("imported lattice database failed integrity verification: %#v", integrity)
	}

	return &LatticeRestoreResult{
		Mode:           "import-bundle",
		TargetPath:     targetPath,
		CreatedAt:      time.Now().UnixMilli(),
		BlockCount:     integrity.BlockCount,
		SnapshotCount:  integrity.SnapshotCount,
		LatestSequence: integrity.LatestBlockSequence,
		Integrity:      integrity,
	}, nil
}

func RestoreBackupToPath(sourcePath, targetPath string) (*LatticeRestoreResult, error) {
	if sourcePath == "" {
		return nil, fmt.Errorf("restore source path required")
	}
	if targetPath == "" {
		targetPath = defaultRestoreTarget(sourcePath, "restored-backup")
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create restore target directory: %w", err)
	}
	if _, err := os.Stat(targetPath); err == nil {
		return nil, fmt.Errorf("restore target already exists: %s", targetPath)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to inspect restore target %s: %w", targetPath, err)
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open restore source %s: %w", sourcePath, err)
	}
	defer func() {
		_ = source.Close()
	}()
	target, err := os.Create(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create restore target %s: %w", targetPath, err)
	}
	defer func() {
		_ = target.Close()
	}()
	if _, err := io.Copy(target, source); err != nil {
		return nil, fmt.Errorf("failed to copy restore source into %s: %w", targetPath, err)
	}
	if err := target.Sync(); err != nil {
		return nil, fmt.Errorf("failed to sync restored database %s: %w", targetPath, err)
	}

	reloaded, err := NewPersistentLattice(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to reopen restored backup database: %w", err)
	}
	defer func() {
		_ = reloaded.Close()
	}()
	integrity, err := reloaded.VerifyPersistence()
	if err != nil {
		return nil, err
	}
	if !integrity.Healthy {
		return nil, fmt.Errorf("restored backup failed integrity verification: %#v", integrity)
	}

	return &LatticeRestoreResult{
		Mode:           "restore-backup",
		SourcePath:     sourcePath,
		TargetPath:     targetPath,
		CreatedAt:      time.Now().UnixMilli(),
		BlockCount:     integrity.BlockCount,
		SnapshotCount:  integrity.SnapshotCount,
		LatestSequence: integrity.LatestBlockSequence,
		Integrity:      integrity,
	}, nil
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
