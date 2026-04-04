package publish

import (
	"os"
	"testing"
)

func TestStoreShardAndPublishManifest(t *testing.T) {
	baseDir := t.TempDir()
	registry, err := NewRegistry(baseDir)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	shardData := []byte("hello bobtorrent shard")
	stored, err := registry.StoreShard("", shardData, "http://localhost:8000")
	if err != nil {
		t.Fatalf("StoreShard failed: %v", err)
	}
	if stored.Hash == "" {
		t.Fatal("expected stored shard hash")
	}
	if _, err := os.Stat(stored.Path); err != nil {
		t.Fatalf("expected shard on disk: %v", err)
	}

	manifest := map[string]interface{}{
		"source": map[string]interface{}{
			"name": "demo.txt",
			"size": 123,
		},
		"encryption": map[string]interface{}{
			"ciphertextHash": "manifest-demo-id",
		},
		"erasure": map[string]interface{}{
			"shards": []interface{}{
				map[string]interface{}{"index": 0, "hash": stored.Hash},
			},
		},
	}

	published, err := registry.PublishManifest(manifest, "http://localhost:8000")
	if err != nil {
		t.Fatalf("PublishManifest failed: %v", err)
	}
	if published.ID != "manifest-demo-id" {
		t.Fatalf("unexpected manifest id: %s", published.ID)
	}
	if _, err := os.Stat(published.Path); err != nil {
		t.Fatalf("expected manifest on disk: %v", err)
	}

	shards := manifest["erasure"].(map[string]interface{})["shards"].([]interface{})
	firstShard := shards[0].(map[string]interface{})
	if firstShard["url"] == "" {
		t.Fatal("expected shard URL to be injected into manifest")
	}
}
