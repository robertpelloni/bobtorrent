package dhtproxy

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(path string) (*Database, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	d := &Database{db: db}
	if err := d.init(); err != nil {
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return d, nil
}

func (d *Database) init() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS torrents (
			info_hash TEXT PRIMARY KEY,
			display_name TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_crawl DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS peers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			info_hash TEXT,
			ip TEXT,
			port INTEGER,
			country_code TEXT,
			latitude REAL,
			longitude REAL,
			last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(info_hash) REFERENCES torrents(info_hash),
			UNIQUE(info_hash, ip, port)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_peers_info_hash ON peers(info_hash)`,
	}

	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

func (d *Database) UpsertTorrent(infoHash, displayName string) error {
	query := `INSERT INTO torrents (info_hash, display_name) VALUES (?, ?)
			  ON CONFLICT(info_hash) DO UPDATE SET display_name = EXCLUDED.display_name`
	_, err := d.db.Exec(query, infoHash, displayName)
	return err
}

func (d *Database) AddPeer(infoHash, ip string, port int, country string, lat, lon float64) error {
	query := `INSERT INTO peers (info_hash, ip, port, country_code, latitude, longitude, last_seen)
			  VALUES (?, ?, ?, ?, ?, ?, ?)
			  ON CONFLICT(info_hash, ip, port) DO UPDATE SET last_seen = EXCLUDED.last_seen`
	_, err := d.db.Exec(query, infoHash, ip, port, country, lat, lon, time.Now())
	return err
}

func (d *Database) GetPeers(infoHash string, limit int) ([]Peer, error) {
	query := `SELECT ip, port, country_code, latitude, longitude FROM peers
			  WHERE info_hash = ? ORDER BY last_seen DESC LIMIT ?`
	rows, err := d.db.Query(query, infoHash, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []Peer
	for rows.Next() {
		var p Peer
		if err := rows.Scan(&p.IP, &p.Port, &p.Country, &p.Latitude, &p.Longitude); err != nil {
			return nil, err
		}
		peers = append(peers, p)
	}

	return peers, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}
