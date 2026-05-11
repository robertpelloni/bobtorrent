package storage

import (
	"fmt"
	"path/filepath"
	"time"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type Registry struct {
	pool *sqlitex.Pool
}

type AssetRecord struct {
	ID        string
	Filename  string
	Size      int64
	Chunks    int
	CreatedAt time.Time
}

func NewRegistry(dataDir string) (*Registry, error) {
	dbPath := filepath.Join(dataDir, "registry.sqlite")

	pool, err := sqlitex.Open(dbPath, 0, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite registry: %w", err)
	}

	conn := pool.Get(nil)
	defer pool.Put(conn)

	err = sqlitex.ExecuteTransient(conn, `
		CREATE TABLE IF NOT EXISTS assets (
			id TEXT PRIMARY KEY,
			filename TEXT NOT NULL,
			size INTEGER NOT NULL,
			chunks INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize registry schema: %w", err)
	}

	return &Registry{pool: pool}, nil
}

func (r *Registry) RegisterAsset(id, filename string, size int64, chunks int) error {
	conn := r.pool.Get(nil)
	defer r.pool.Put(conn)

	return sqlitex.ExecuteTransient(conn, `
		INSERT OR REPLACE INTO assets (id, filename, size, chunks, created_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, &sqlitex.ExecOptions{
		Args: []interface{}{id, filename, size, chunks},
	})
}

func (r *Registry) ListAssets() ([]AssetRecord, error) {
	conn := r.pool.Get(nil)
	defer r.pool.Put(conn)

	var assets []AssetRecord
	err := sqlitex.ExecuteTransient(conn, `SELECT id, filename, size, chunks, created_at FROM assets ORDER BY created_at DESC`, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			t, err := time.Parse("2006-01-02 15:04:05", stmt.ColumnText(4))
			if err != nil {
				t = time.Now()
			}
			assets = append(assets, AssetRecord{
				ID:        stmt.ColumnText(0),
				Filename:  stmt.ColumnText(1),
				Size:      stmt.ColumnInt64(2),
				Chunks:    stmt.ColumnInt(3),
				CreatedAt: t,
			})
			return nil
		},
	})

	return assets, err
}

func (r *Registry) Close() error {
	return r.pool.Close()
}
