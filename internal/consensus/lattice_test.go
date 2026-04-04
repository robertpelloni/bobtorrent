package consensus

import (
	"testing"

	"bobtorrent/pkg/torrent"
)

func TestProcessPublishManifestAnchorsWalletAttributedManifest(t *testing.T) {
	lattice := NewLattice()
	wallet, err := torrent.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair failed: %v", err)
	}

	genesis := torrent.NewBlock("open", wallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	if err := genesis.Sign(wallet.PrivateKey); err != nil {
		t.Fatalf("Sign genesis failed: %v", err)
	}
	if err := lattice.ProcessBlock(genesis); err != nil {
		t.Fatalf("ProcessBlock genesis failed: %v", err)
	}

	proofHash := torrent.HashSHA256("manifest-anchor-proof")
	proofSignature, err := torrent.Sign(proofHash, wallet.PrivateKey)
	if err != nil {
		t.Fatalf("Sign proof failed: %v", err)
	}

	payload := map[string]interface{}{
		"manifestId":     "manifest-123",
		"locator":        "bobtorrent://manifest/manifest-123",
		"manifestUrl":    "http://localhost:8000/manifests/manifest-123",
		"name":           "demo.bin",
		"size":           float64(512),
		"ciphertextHash": "cipher-hash-123",
		"publicationProof": map[string]interface{}{
			"messageHash": proofHash,
			"signature":   proofSignature,
			"publicKey":   wallet.PublicKey,
		},
	}

	anchor := torrent.NewBlock("publish_manifest", wallet.PublicKey, &genesis.Hash, genesis.Balance, genesis.StakedBalance, 1, "manifest-123", nil, payload)
	if err := anchor.Sign(wallet.PrivateKey); err != nil {
		t.Fatalf("Sign anchor failed: %v", err)
	}
	if err := lattice.ProcessBlock(anchor); err != nil {
		t.Fatalf("ProcessBlock anchor failed: %v", err)
	}

	stored, ok := lattice.anchors[anchor.Hash]
	if !ok {
		t.Fatal("expected anchor to be stored")
	}
	if stored.Owner != wallet.PublicKey {
		t.Fatalf("unexpected anchor owner: %s", stored.Owner)
	}
	if stored.ManifestID != "manifest-123" {
		t.Fatalf("unexpected manifest id: %s", stored.ManifestID)
	}
}
