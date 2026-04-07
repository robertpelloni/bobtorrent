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
	defer registry.Close()

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

	// Test ListManifests
	list, err := registry.ListManifests(10)
	if err != nil {
		t.Fatalf("ListManifests failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 manifest in list, got %d", len(list))
	}
	if list[0].ID != "manifest-demo-id" {
		t.Fatalf("unexpected manifest id in list: %s", list[0].ID)
	}
}

func TestRegistryDurability(t *testing.T) {
	baseDir := t.TempDir()

	// 1. Create and add data
	func() {
		registry, err := NewRegistry(baseDir)
		if err != nil {
			t.Fatalf("NewRegistry failed: %v", err)
		}
		defer registry.Close()

		manifest := map[string]interface{}{
			"encryption": map[string]interface{}{"ciphertextHash": "durable-id"},
		}
		_, err = registry.PublishManifest(manifest, "http://localhost:8000")
		if err != nil {
			t.Fatalf("PublishManifest failed: %v", err)
		}
	}()

	// 2. Re-open and verify
	registry, err := NewRegistry(baseDir)
	if err != nil {
		t.Fatalf("Second NewRegistry failed: %v", err)
	}
	defer registry.Close()

	list, err := registry.ListManifests(10)
	if err != nil {
		t.Fatalf("Second ListManifests failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 manifest after re-opening, got %d", len(list))
	}
	if list[0].ID != "durable-id" {
		t.Fatalf("unexpected manifest id after re-opening: %s", list[0].ID)
	}
}
