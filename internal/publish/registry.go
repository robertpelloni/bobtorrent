package publish

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Registry persists uploaded storage shards and published manifests to disk.
//
// Why this exists:
//   1. The Bobcoin WASM workbench can now perform real browser-side preprocessing.
//   2. The Go supernode needs a durable place to accept those prepared shards.
//   3. We want a simple, local-first publication flow before introducing full
//      lattice anchoring or cross-peer replication.
//
// This registry is intentionally conservative:
//   - shards are content-addressed by SHA-256 hash
//   - manifests are stored as JSON files under a deterministic ID
//   - duplicate uploads are safe and idempotent
//   - no process-global state is required beyond the filesystem paths
//
// Future directions:
//   - replicate manifests across peers
//   - anchor published manifest IDs on the lattice
//   - add signature verification and uploader authorization
//   - expose pagination/listing/search over published assets

type Registry struct {
	baseDir      string
	shardsDir    string
	manifestsDir string
	mu           sync.Mutex
}

type StoredShard struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
	Path string `json:"path"`
	URL  string `json:"url"`
}

type StoredManifest struct {
	ID          string                 `json:"id"`
	Locator     string                 `json:"locator"`
	Path        string                 `json:"path"`
	ManifestURL string                 `json:"manifestUrl"`
	PublishedAt int64                  `json:"publishedAt"`
	Manifest    map[string]interface{} `json:"manifest"`
}

func NewRegistry(baseDir string) (*Registry, error) {
	shardsDir := filepath.Join(baseDir, "shards")
	manifestsDir := filepath.Join(baseDir, "manifests")

	for _, dir := range []string{baseDir, shardsDir, manifestsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create publish directory %s: %w", dir, err)
		}
	}

	return &Registry{
		baseDir:      baseDir,
		shardsDir:    shardsDir,
		manifestsDir: manifestsDir,
	}, nil
}

// StoreShard validates the claimed hash against the provided bytes and stores
// the shard under a content-addressed filename. Duplicate uploads are accepted.
func (r *Registry) StoreShard(hash string, data []byte, publicBaseURL string) (*StoredShard, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	computed := sha256.Sum256(data)
	computedHex := hex.EncodeToString(computed[:])
	if hash == "" {
		hash = computedHex
	}
	if hash != computedHex {
		return nil, fmt.Errorf("shard hash mismatch: claimed %s, computed %s", hash, computedHex)
	}

	path := filepath.Join(r.shardsDir, hash+".bin")
	if _, err := os.Stat(path); err == nil {
		return &StoredShard{
			Hash: hash,
			Size: int64(len(data)),
			Path: path,
			URL:  fmt.Sprintf("%s/shards/%s", publicBaseURL, hash),
		}, nil
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write shard %s: %w", hash, err)
	}

	return &StoredShard{
		Hash: hash,
		Size: int64(len(data)),
		Path: path,
		URL:  fmt.Sprintf("%s/shards/%s", publicBaseURL, hash),
	}, nil
}

// PublishManifest enriches and persists a manifest after all referenced shards
// have been uploaded. The manifest ID defaults to encryption.ciphertextHash if
// available, otherwise it is derived from the manifest JSON bytes.
func (r *Registry) PublishManifest(manifest map[string]interface{}, publicBaseURL string) (*StoredManifest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	manifestID := deriveManifestID(manifest)
	publishedAt := time.Now().UnixMilli()

	manifest["publishedAt"] = publishedAt
	manifest["manifestId"] = manifestID
	manifest["locator"] = fmt.Sprintf("bobtorrent://manifest/%s", manifestID)
	manifest["manifestUrl"] = fmt.Sprintf("%s/manifests/%s", publicBaseURL, manifestID)

	// Enrich shard entries with downloadable URLs when the standard erasure
	// structure is present.
	if erasure, ok := manifest["erasure"].(map[string]interface{}); ok {
		if shards, ok := erasure["shards"].([]interface{}); ok {
			for _, rawShard := range shards {
				shardMap, ok := rawShard.(map[string]interface{})
				if !ok {
					continue
				}
				hash, _ := shardMap["hash"].(string)
				if hash != "" {
					shardMap["url"] = fmt.Sprintf("%s/shards/%s", publicBaseURL, hash)
				}
			}
		}
	}

	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	path := filepath.Join(r.manifestsDir, manifestID+".json")
	if err := os.WriteFile(path, encoded, 0644); err != nil {
		return nil, fmt.Errorf("failed to write manifest %s: %w", manifestID, err)
	}

	return &StoredManifest{
		ID:          manifestID,
		Locator:     fmt.Sprintf("bobtorrent://manifest/%s", manifestID),
		Path:        path,
		ManifestURL: fmt.Sprintf("%s/manifests/%s", publicBaseURL, manifestID),
		PublishedAt: publishedAt,
		Manifest:    manifest,
	}, nil
}

func (r *Registry) ManifestPath(id string) string {
	return filepath.Join(r.manifestsDir, id+".json")
}

func (r *Registry) ShardPath(hash string) string {
	return filepath.Join(r.shardsDir, hash+".bin")
}

func deriveManifestID(manifest map[string]interface{}) string {
	if encryption, ok := manifest["encryption"].(map[string]interface{}); ok {
		if ciphertextHash, ok := encryption["ciphertextHash"].(string); ok && ciphertextHash != "" {
			return ciphertextHash
		}
	}

	encoded, _ := json.Marshal(manifest)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}
