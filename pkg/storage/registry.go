package storage
import ("database/sql"; "fmt"; "path/filepath"; "time"; _ "modernc.org/sqlite")
type Registry struct { db *sql.DB }
type AssetRecord struct { ID string; Filename string; Size int64; Chunks int; CreatedAt time.Time }
func NewRegistry(dataDir string) (*Registry, error) {
	dbPath := filepath.Join(dataDir, "registry.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil { return nil, fmt.Errorf("failed to open sqlite registry: %w", err) }
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS assets (id TEXT PRIMARY KEY, filename TEXT NOT NULL, size INTEGER NOT NULL, chunks INTEGER NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`)
	if err != nil { db.Close(); return nil, fmt.Errorf("failed to initialize registry schema: %w", err) }
	return &Registry{db: db}, nil
}
func (r *Registry) RegisterAsset(id, filename string, size int64, chunks int) error {
	_, err := r.db.Exec(`INSERT OR REPLACE INTO assets (id, filename, size, chunks, created_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`, id, filename, size, chunks); return err
}
func (r *Registry) ListAssets() ([]AssetRecord, error) {
	rows, err := r.db.Query(`SELECT id, filename, size, chunks, created_at FROM assets ORDER BY created_at DESC`)
	if err != nil { return nil, err }
	defer rows.Close()
	var assets []AssetRecord
	for rows.Next() {
		var a AssetRecord; var createdAt string
		if err := rows.Scan(&a.ID, &a.Filename, &a.Size, &a.Chunks, &createdAt); err != nil { return nil, err }
		t, err := time.Parse("2006-01-02 15:04:05", createdAt); if err != nil { t = time.Now() }
		a.CreatedAt = t; assets = append(assets, a)
	}
	return assets, nil
}
func (r *Registry) Close() error { return r.db.Close() }
