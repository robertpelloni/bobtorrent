package consensus

import (
	"fmt"
	"path/filepath"
	"testing"

	"bobtorrent/pkg/torrent"
)

// Disaster-recovery drill at unit level: a chain that has crossed the
// snapshot interval must survive the full operator path — encrypted
// backup bundle -> restore to a fresh path -> cold boot — with every
// block intact. (Live-verified against the real 27-block chain on
// 2026-11-17; this keeps the proof durable.)
func TestSecureBundleRestorePreservesSnapshotChain(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "lattice.db")
	lattice, err := NewPersistentLattice(dbPath)
	if err != nil {
		t.Fatalf("NewPersistentLattice failed: %v", err)
	}

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
	frontier := genesis
	for i := 0; i < int(defaultLatticeSnapshotInterval); i++ {
		block := torrent.NewBlock("achievement_unlock", wallet.PublicKey, &frontier.Hash, frontier.Balance, frontier.StakedBalance, frontier.Height+1, fmt.Sprintf("achievement-%d", i), nil, map[string]interface{}{"achievement": fmt.Sprintf("A-%d", i)})
		if err := block.Sign(wallet.PrivateKey); err != nil {
			t.Fatalf("Sign block %d failed: %v", i, err)
		}
		if err := lattice.ProcessBlock(block); err != nil {
			t.Fatalf("ProcessBlock %d failed: %v", i, err)
		}
		frontier = block
	}
	wantBlocks := int(defaultLatticeSnapshotInterval) + 1
	if len(lattice.blocks) != wantBlocks {
		t.Fatalf("setup: expected %d blocks, got %d", wantBlocks, len(lattice.blocks))
	}

	bundlePath := filepath.Join(t.TempDir(), "bundles", "drill.secure-backup.json")
	bundle, err := lattice.CreateSignedEncryptedBackupBundle(bundlePath, "drill-passphrase", wallet.PrivateKey)
	if err != nil {
		t.Fatalf("CreateSignedEncryptedBackupBundle failed: %v", err)
	}
	if _, err := lattice.RestoreSignedEncryptedBackupBundle(bundle.BundlePath, "wrong-passphrase", filepath.Join(t.TempDir(), "bad.db"), false); err == nil {
		t.Fatal("expected wrong-passphrase restore to fail")
	}

	restorePath := filepath.Join(t.TempDir(), "restored", "drill.db")
	restored, err := lattice.RestoreSignedEncryptedBackupBundle(bundle.BundlePath, "drill-passphrase", restorePath, true)
	if err != nil {
		t.Fatalf("RestoreSignedEncryptedBackupBundle failed: %v", err)
	}
	if restored.Restore.BlockCount != int64(wantBlocks) {
		t.Fatalf("expected restored block count %d, got %d", wantBlocks, restored.Restore.BlockCount)
	}
	if err := lattice.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	reloaded, err := NewPersistentLattice(restorePath)
	if err != nil {
		t.Fatalf("cold boot of restored DB failed: %v", err)
	}
	defer func() { _ = reloaded.Close() }()
	if len(reloaded.blocks) != wantBlocks {
		t.Fatalf("expected %d blocks after cold boot, got %d", wantBlocks, len(reloaded.blocks))
	}
	if reloaded.GetFrontier(wallet.PublicKey).Hash != frontier.Hash {
		t.Fatalf("frontier mismatch after restore: got %s want %s", reloaded.GetFrontier(wallet.PublicKey).Hash, frontier.Hash)
	}
}
