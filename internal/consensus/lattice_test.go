package consensus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
		"publisher": map[string]interface{}{
			"alias":     "CipherArchivist",
			"website":   "https://bob.example",
			"statement": "Preserving sovereign knowledge across the lattice.",
			"avatar":    "https://bob.example/avatar.png",
			"proofs": []interface{}{
				map[string]interface{}{"kind": "github", "label": "GitHub Identity", "issuer": "GitHub", "url": "https://github.com/cipherarchivist"},
				map[string]interface{}{"kind": "orcid", "label": "ORCID Research Profile", "issuer": "ORCID", "url": "https://orcid.org/0000-0000-0000-0000"},
			},
		},
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
	if stored.PublisherAlias != "CipherArchivist" {
		t.Fatalf("unexpected publisher alias: %s", stored.PublisherAlias)
	}
	if stored.PublisherWebsite != "https://bob.example" {
		t.Fatalf("unexpected publisher website: %s", stored.PublisherWebsite)
	}
	if stored.PublisherAvatar != "https://bob.example/avatar.png" {
		t.Fatalf("unexpected publisher avatar: %s", stored.PublisherAvatar)
	}
	if len(stored.PublisherProofs) != 2 {
		t.Fatalf("unexpected publisher proofs length: %d", len(stored.PublisherProofs))
	}
	if len(stored.PublisherProofKinds) != 2 {
		t.Fatalf("unexpected publisher proof kinds length: %d", len(stored.PublisherProofKinds))
	}
	if len(stored.PublisherProofLabels) != 2 {
		t.Fatalf("unexpected publisher proof labels length: %d", len(stored.PublisherProofLabels))
	}
	if len(stored.PublisherProofIssuers) != 2 {
		t.Fatalf("unexpected publisher proof issuers length: %d", len(stored.PublisherProofIssuers))
	}
	if stored.PublisherProofKinds[0] != "github" {
		t.Fatalf("unexpected first proof kind: %s", stored.PublisherProofKinds[0])
	}
	if stored.PublisherProofLabels[0] != "GitHub Identity" {
		t.Fatalf("unexpected first proof label: %s", stored.PublisherProofLabels[0])
	}
	if stored.PublisherProofIssuers[0] != "GitHub" {
		t.Fatalf("unexpected first proof issuer: %s", stored.PublisherProofIssuers[0])
	}
}

func TestPersistentLatticeReplaysConfirmedBlocksOnRestart(t *testing.T) {
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

	anchorPayload := map[string]interface{}{
		"manifestId":     "persistent-manifest-1",
		"locator":        "bobtorrent://manifest/persistent-manifest-1",
		"manifestUrl":    "http://localhost:8000/manifests/persistent-manifest-1",
		"name":           "archive.bin",
		"size":           float64(2048),
		"ciphertextHash": "cipher-persist-1",
		"publisher": map[string]interface{}{
			"alias":   "PersistentArchivist",
			"website": "https://archive.example",
			"proofs": []interface{}{
				map[string]interface{}{"kind": "website", "url": "https://archive.example/about"},
			},
		},
	}

	anchor := torrent.NewBlock("publish_manifest", wallet.PublicKey, &genesis.Hash, genesis.Balance, genesis.StakedBalance, 1, "persistent-manifest-1", nil, anchorPayload)
	if err := anchor.Sign(wallet.PrivateKey); err != nil {
		t.Fatalf("Sign anchor failed: %v", err)
	}
	if err := lattice.ProcessBlock(anchor); err != nil {
		t.Fatalf("ProcessBlock anchor failed: %v", err)
	}

	if err := lattice.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	reloaded, err := NewPersistentLattice(dbPath)
	if err != nil {
		t.Fatalf("NewPersistentLattice reload failed: %v", err)
	}
	defer func() {
		if err := reloaded.Close(); err != nil {
			t.Fatalf("Close reload failed: %v", err)
		}
	}()

	if len(reloaded.chains[wallet.PublicKey]) != 2 {
		t.Fatalf("expected 2 replayed blocks, got %d", len(reloaded.chains[wallet.PublicKey]))
	}
	if len(reloaded.blocks) != 2 {
		t.Fatalf("expected 2 replayed blocks in block index, got %d", len(reloaded.blocks))
	}
	stored, ok := reloaded.anchors[anchor.Hash]
	if !ok {
		t.Fatal("expected replayed anchor to be restored")
	}
	if stored.ManifestID != "persistent-manifest-1" {
		t.Fatalf("unexpected replayed manifest id: %s", stored.ManifestID)
	}
	if len(stored.PublisherProofKinds) != 1 || stored.PublisherProofKinds[0] != "website" {
		t.Fatalf("unexpected replayed proof kinds: %#v", stored.PublisherProofKinds)
	}
	if reloaded.stateHash == "" || reloaded.stateHash == strings.Repeat("0", 64) {
		t.Fatalf("expected replayed state hash to be non-genesis, got %s", reloaded.stateHash)
	}
}

func TestPersistentLatticeRestoresFromSnapshotAndReplaysTail(t *testing.T) {
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
	for i := 0; i < int(defaultLatticeSnapshotInterval)-1; i++ {
		block := torrent.NewBlock("achievement_unlock", wallet.PublicKey, &frontier.Hash, frontier.Balance, frontier.StakedBalance, frontier.Height+1, fmt.Sprintf("achievement-%d", i), nil, map[string]interface{}{"achievement": fmt.Sprintf("A-%d", i)})
		if err := block.Sign(wallet.PrivateKey); err != nil {
			t.Fatalf("Sign achievement block %d failed: %v", i, err)
		}
		if err := lattice.ProcessBlock(block); err != nil {
			t.Fatalf("ProcessBlock achievement block %d failed: %v", i, err)
		}
		frontier = block
	}

	if lattice.snapshotSequence != defaultLatticeSnapshotInterval {
		t.Fatalf("expected snapshot sequence %d, got %d", defaultLatticeSnapshotInterval, lattice.snapshotSequence)
	}

	anchorPayload := map[string]interface{}{
		"manifestId":     "snapshot-tail-manifest",
		"locator":        "bobtorrent://manifest/snapshot-tail-manifest",
		"manifestUrl":    "http://localhost:8000/manifests/snapshot-tail-manifest",
		"name":           "tail.bin",
		"size":           float64(4096),
		"ciphertextHash": "snapshot-tail-ciphertext",
	}
	anchor := torrent.NewBlock("publish_manifest", wallet.PublicKey, &frontier.Hash, frontier.Balance, frontier.StakedBalance, frontier.Height+1, "snapshot-tail-manifest", nil, anchorPayload)
	if err := anchor.Sign(wallet.PrivateKey); err != nil {
		t.Fatalf("Sign tail anchor failed: %v", err)
	}
	if err := lattice.ProcessBlock(anchor); err != nil {
		t.Fatalf("ProcessBlock tail anchor failed: %v", err)
	}

	if err := lattice.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	reloaded, err := NewPersistentLattice(dbPath)
	if err != nil {
		t.Fatalf("NewPersistentLattice reload failed: %v", err)
	}
	defer func() {
		if err := reloaded.Close(); err != nil {
			t.Fatalf("Close reload failed: %v", err)
		}
	}()

	if reloaded.snapshotSequence != defaultLatticeSnapshotInterval {
		t.Fatalf("expected reload snapshot sequence %d, got %d", defaultLatticeSnapshotInterval, reloaded.snapshotSequence)
	}
	if reloaded.persistedSequence != defaultLatticeSnapshotInterval+1 {
		t.Fatalf("expected persisted sequence %d, got %d", defaultLatticeSnapshotInterval+1, reloaded.persistedSequence)
	}
	if len(reloaded.blocks) != int(defaultLatticeSnapshotInterval)+1 {
		t.Fatalf("expected %d total replayed blocks, got %d", defaultLatticeSnapshotInterval+1, len(reloaded.blocks))
	}
	stored, ok := reloaded.anchors[anchor.Hash]
	if !ok {
		t.Fatal("expected tail anchor to be restored after snapshot replay")
	}
	if stored.ManifestID != "snapshot-tail-manifest" {
		t.Fatalf("unexpected tail manifest id: %s", stored.ManifestID)
	}
	if reloaded.GetFrontier(wallet.PublicKey).Hash != anchor.Hash {
		t.Fatalf("expected frontier hash %s, got %s", anchor.Hash, reloaded.GetFrontier(wallet.PublicKey).Hash)
	}
}

func TestPersistentLatticeVerifyAndRepairRebuildsSnapshotLayer(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "lattice.db")
	lattice, err := NewPersistentLattice(dbPath)
	if err != nil {
		t.Fatalf("NewPersistentLattice failed: %v", err)
	}
	defer func() {
		if err := lattice.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

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

	for i := 0; i < int(defaultLatticeSnapshotInterval)-1; i++ {
		frontier := lattice.GetFrontier(wallet.PublicKey)
		block := torrent.NewBlock("achievement_unlock", wallet.PublicKey, &frontier.Hash, frontier.Balance, frontier.StakedBalance, frontier.Height+1, fmt.Sprintf("verify-repair-%d", i), nil, map[string]interface{}{"achievement": fmt.Sprintf("R-%d", i)})
		if err := block.Sign(wallet.PrivateKey); err != nil {
			t.Fatalf("Sign achievement block %d failed: %v", i, err)
		}
		if err := lattice.ProcessBlock(block); err != nil {
			t.Fatalf("ProcessBlock achievement block %d failed: %v", i, err)
		}
	}

	if lattice.snapshotSequence != defaultLatticeSnapshotInterval {
		t.Fatalf("expected snapshot sequence %d, got %d", defaultLatticeSnapshotInterval, lattice.snapshotSequence)
	}

	if _, err := lattice.store.db.Exec(`INSERT INTO lattice_snapshots (snapshot_sequence, snapshot_json) VALUES (?, ?)`, 9999, `{not-valid-json}`); err != nil {
		t.Fatalf("failed to inject corrupt snapshot row: %v", err)
	}

	report, err := lattice.VerifyPersistence()
	if err != nil {
		t.Fatalf("VerifyPersistence failed: %v", err)
	}
	if report.Healthy {
		t.Fatal("expected verification report to be unhealthy after corrupt snapshot injection")
	}
	if !report.Repairable {
		t.Fatal("expected corrupt snapshot layer to remain repairable")
	}
	if len(report.InvalidSnapshotSequences) == 0 {
		t.Fatal("expected corrupt snapshot sequence to be detected")
	}

	repaired, err := lattice.RepairPersistence()
	if err != nil {
		t.Fatalf("RepairPersistence failed: %v", err)
	}
	if !repaired.Healthy {
		t.Fatalf("expected repaired persistence to be healthy, got %#v", repaired)
	}
	if repaired.SnapshotCount != 1 {
		t.Fatalf("expected exactly one rebuilt snapshot, got %d", repaired.SnapshotCount)
	}
	if lattice.snapshotSequence != lattice.persistedSequence {
		t.Fatalf("expected snapshot sequence %d to match persisted sequence after repair, got %d", lattice.persistedSequence, lattice.snapshotSequence)
	}
}

func TestPersistentLatticeRestoresMixedConsensusTransitionsAfterSnapshotTailReplay(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "lattice.db")
	lattice, err := NewPersistentLattice(dbPath)
	if err != nil {
		t.Fatalf("NewPersistentLattice failed: %v", err)
	}
	defer func() {
		if lattice != nil {
			if err := lattice.Close(); err != nil {
				t.Fatalf("Close failed: %v", err)
			}
		}
	}()

	alice, err := torrent.GenerateKeypair()
	if err != nil {
		t.Fatalf("Generate alice keypair failed: %v", err)
	}
	bob, err := torrent.GenerateKeypair()
	if err != nil {
		t.Fatalf("Generate bob keypair failed: %v", err)
	}

	process := func(block *torrent.Block, privateKey string, label string) {
		if err := block.Sign(privateKey); err != nil {
			t.Fatalf("Sign %s failed: %v", label, err)
		}
		if err := lattice.ProcessBlock(block); err != nil {
			t.Fatalf("ProcessBlock %s failed: %v", label, err)
		}
	}

	aliceOpen := torrent.NewBlock("open", alice.PublicKey, nil, 2500, 0, 0, "SYSTEM_GENESIS", nil, nil)
	process(aliceOpen, alice.PrivateKey, "alice open")

	for i := 0; i < int(defaultLatticeSnapshotInterval)-1; i++ {
		frontier := lattice.GetFrontier(alice.PublicKey)
		block := torrent.NewBlock("achievement_unlock", alice.PublicKey, &frontier.Hash, frontier.Balance, frontier.StakedBalance, frontier.Height+1, fmt.Sprintf("prefill-%d", i), nil, map[string]interface{}{"achievement": fmt.Sprintf("P-%d", i)})
		process(block, alice.PrivateKey, fmt.Sprintf("prefill-%d", i))
	}

	if lattice.snapshotSequence != defaultLatticeSnapshotInterval {
		t.Fatalf("expected snapshot sequence %d before mixed tail, got %d", defaultLatticeSnapshotInterval, lattice.snapshotSequence)
	}

	aliceFrontier := lattice.GetFrontier(alice.PublicKey)
	sendOpen := torrent.NewBlock("send", alice.PublicKey, &aliceFrontier.Hash, aliceFrontier.Balance-200, aliceFrontier.StakedBalance, aliceFrontier.Height+1, bob.PublicKey, nil, map[string]interface{}{"reason": "mixed-tail-open"})
	process(sendOpen, alice.PrivateKey, "send open")

	bobOpen := torrent.NewBlock("open", bob.PublicKey, nil, 200, 0, 0, sendOpen.Hash, nil, nil)
	process(bobOpen, bob.PrivateKey, "bob open")

	aliceFrontier = lattice.GetFrontier(alice.PublicKey)
	sendReceive := torrent.NewBlock("send", alice.PublicKey, &aliceFrontier.Hash, aliceFrontier.Balance-150, aliceFrontier.StakedBalance, aliceFrontier.Height+1, bob.PublicKey, nil, map[string]interface{}{"reason": "mixed-tail-receive"})
	process(sendReceive, alice.PrivateKey, "send receive")

	bobFrontier := lattice.GetFrontier(bob.PublicKey)
	receive := torrent.NewBlock("receive", bob.PublicKey, &bobFrontier.Hash, bobFrontier.Balance+150, bobFrontier.StakedBalance, bobFrontier.Height+1, sendReceive.Hash, nil, nil)
	process(receive, bob.PrivateKey, "receive")

	aliceFrontier = lattice.GetFrontier(alice.PublicKey)
	proposal := torrent.NewBlock("proposal", alice.PublicKey, &aliceFrontier.Hash, aliceFrontier.Balance-int64(ProposalCost), aliceFrontier.StakedBalance, aliceFrontier.Height+1, "", nil, map[string]interface{}{"title": "Durable Replay Proposal", "description": "Ensure replay restores governance state."})
	process(proposal, alice.PrivateKey, "proposal")

	bobFrontier = lattice.GetFrontier(bob.PublicKey)
	vote := torrent.NewBlock("vote", bob.PublicKey, &bobFrontier.Hash, bobFrontier.Balance, bobFrontier.StakedBalance, bobFrontier.Height+1, proposal.Hash, nil, map[string]interface{}{"vote": "for"})
	process(vote, bob.PrivateKey, "vote")

	aliceFrontier = lattice.GetFrontier(alice.PublicKey)
	mintNFT := torrent.NewBlock("mint_nft", alice.PublicKey, &aliceFrontier.Hash, aliceFrontier.Balance-int64(NFTMintCost), aliceFrontier.StakedBalance, aliceFrontier.Height+1, "", nil, map[string]interface{}{"name": "Persistent Tail NFT", "magnet": "magnet:?xt=urn:btih:persistenttail", "description": "Replay this ownership transition after restart."})
	process(mintNFT, alice.PrivateKey, "mint nft")

	aliceFrontier = lattice.GetFrontier(alice.PublicKey)
	transferNFT := torrent.NewBlock("transfer_nft", alice.PublicKey, &aliceFrontier.Hash, aliceFrontier.Balance-int64(NFTTransferFee), aliceFrontier.StakedBalance, aliceFrontier.Height+1, mintNFT.Hash, nil, map[string]interface{}{"newOwner": bob.PublicKey})
	process(transferNFT, alice.PrivateKey, "transfer nft")

	bobFrontier = lattice.GetFrontier(bob.PublicKey)
	stake := torrent.NewBlock("stake", bob.PublicKey, &bobFrontier.Hash, bobFrontier.Balance-100, bobFrontier.StakedBalance+100, bobFrontier.Height+1, "", nil, map[string]interface{}{"amount": 100})
	process(stake, bob.PrivateKey, "stake")

	bobFrontier = lattice.GetFrontier(bob.PublicKey)
	unstake := torrent.NewBlock("unstake", bob.PublicKey, &bobFrontier.Hash, bobFrontier.Balance+100, 0, bobFrontier.Height+1, "", nil, map[string]interface{}{"amount": 100})
	process(unstake, bob.PrivateKey, "unstake")

	aliceFrontier = lattice.GetFrontier(alice.PublicKey)
	secret := "persistent-tail-secret"
	hashLock := torrent.HashSHA256(secret)
	initiateSwap := torrent.NewBlock("initiate_swap", alice.PublicKey, &aliceFrontier.Hash, aliceFrontier.Balance-300, aliceFrontier.StakedBalance, aliceFrontier.Height+1, "", nil, map[string]interface{}{"recipient": bob.PublicKey, "hashLock": hashLock, "expiry": float64(time.Now().Add(10 * time.Minute).UnixMilli())})
	process(initiateSwap, alice.PrivateKey, "initiate swap")

	bobFrontier = lattice.GetFrontier(bob.PublicKey)
	claimSwap := torrent.NewBlock("claim_swap", bob.PublicKey, &bobFrontier.Hash, bobFrontier.Balance+300, bobFrontier.StakedBalance, bobFrontier.Height+1, initiateSwap.Hash, nil, map[string]interface{}{"secret": secret})
	process(claimSwap, bob.PrivateKey, "claim swap")

	if err := lattice.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	lattice = nil

	reloaded, err := NewPersistentLattice(dbPath)
	if err != nil {
		t.Fatalf("NewPersistentLattice reload failed: %v", err)
	}
	defer func() {
		if err := reloaded.Close(); err != nil {
			t.Fatalf("Close reload failed: %v", err)
		}
	}()

	if reloaded.snapshotSequence != defaultLatticeSnapshotInterval {
		t.Fatalf("expected snapshot sequence %d after reload, got %d", defaultLatticeSnapshotInterval, reloaded.snapshotSequence)
	}
	if len(reloaded.blocks) != int(defaultLatticeSnapshotInterval)+12 {
		t.Fatalf("expected %d total replayed blocks, got %d", defaultLatticeSnapshotInterval+12, len(reloaded.blocks))
	}
	if frontier := reloaded.GetFrontier(alice.PublicKey); frontier == nil || frontier.Hash != initiateSwap.Hash {
		t.Fatalf("expected alice frontier %s after replay, got %#v", initiateSwap.Hash, frontier)
	}
	if frontier := reloaded.GetFrontier(bob.PublicKey); frontier == nil || frontier.Hash != claimSwap.Hash {
		t.Fatalf("expected bob frontier %s after replay, got %#v", claimSwap.Hash, frontier)
	}
	if pending := reloaded.pending[bob.PublicKey]; len(pending) != 0 {
		t.Fatalf("expected bob pending queue to be empty after replayed open/receive flow, got %#v", pending)
	}
	storedProposal := reloaded.proposals[proposal.Hash]
	if storedProposal == nil || storedProposal.Status != "passed" {
		t.Fatalf("expected replayed proposal to be passed, got %#v", storedProposal)
	}
	if voteMap := reloaded.votes[proposal.Hash]; voteMap == nil || voteMap[bob.PublicKey].Type != "for" {
		t.Fatalf("expected replayed vote map to include bob's vote, got %#v", voteMap)
	}
	storedNFT := reloaded.nfts[mintNFT.Hash]
	if storedNFT == nil || storedNFT.Owner != bob.PublicKey {
		t.Fatalf("expected replayed nft owner %s, got %#v", bob.PublicKey, storedNFT)
	}
	if _, ok := reloaded.stakeInfo[bob.PublicKey]; ok {
		t.Fatalf("expected bob stake info to be cleared after replayed full unstake, got %#v", reloaded.stakeInfo[bob.PublicKey])
	}
	storedSwap := reloaded.swaps[initiateSwap.Hash]
	if storedSwap == nil || storedSwap.Status != "CLAIMED" || storedSwap.Claimer != bob.PublicKey || storedSwap.Secret != secret {
		t.Fatalf("expected replayed swap to be claimed by bob with stored secret, got %#v", storedSwap)
	}
}

func TestPersistentLatticeWithCustomSnapshotConfigHonorsIntervalAndRetention(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "lattice.db")
	lattice, err := NewPersistentLatticeWithConfig(dbPath, SnapshotConfig{Interval: 5, Retention: 1})
	if err != nil {
		t.Fatalf("NewPersistentLatticeWithConfig failed: %v", err)
	}
	defer func() {
		if err := lattice.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

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

	for i := 0; i < 9; i++ {
		frontier := lattice.GetFrontier(wallet.PublicKey)
		block := torrent.NewBlock("achievement_unlock", wallet.PublicKey, &frontier.Hash, frontier.Balance, frontier.StakedBalance, frontier.Height+1, fmt.Sprintf("custom-snapshot-%d", i), nil, map[string]interface{}{"achievement": fmt.Sprintf("CS-%d", i)})
		if err := block.Sign(wallet.PrivateKey); err != nil {
			t.Fatalf("Sign achievement block %d failed: %v", i, err)
		}
		if err := lattice.ProcessBlock(block); err != nil {
			t.Fatalf("ProcessBlock achievement block %d failed: %v", i, err)
		}
	}

	if lattice.store.SnapshotInterval() != 5 {
		t.Fatalf("expected snapshot interval 5, got %d", lattice.store.SnapshotInterval())
	}
	if lattice.store.SnapshotRetention() != 1 {
		t.Fatalf("expected snapshot retention 1, got %d", lattice.store.SnapshotRetention())
	}
	if lattice.snapshotSequence != 10 {
		t.Fatalf("expected latest snapshot sequence 10, got %d", lattice.snapshotSequence)
	}
	snapshotCount, err := lattice.store.CountSnapshots()
	if err != nil {
		t.Fatalf("CountSnapshots failed: %v", err)
	}
	if snapshotCount != 1 {
		t.Fatalf("expected snapshot retention to keep 1 snapshot, got %d", snapshotCount)
	}

	bundle, err := lattice.ExportPersistence()
	if err != nil {
		t.Fatalf("ExportPersistence failed: %v", err)
	}
	if bundle.SnapshotInterval != 5 || bundle.SnapshotRetention != 1 {
		t.Fatalf("expected export bundle snapshot config 5/1, got %d/%d", bundle.SnapshotInterval, bundle.SnapshotRetention)
	}
}

func TestPersistentLatticeExportIncludesDurableHistory(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "lattice.db")
	lattice, err := NewPersistentLattice(dbPath)
	if err != nil {
		t.Fatalf("NewPersistentLattice failed: %v", err)
	}
	defer func() {
		if err := lattice.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

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

	for i := 0; i < int(defaultLatticeSnapshotInterval)-1; i++ {
		frontier := lattice.GetFrontier(wallet.PublicKey)
		block := torrent.NewBlock("achievement_unlock", wallet.PublicKey, &frontier.Hash, frontier.Balance, frontier.StakedBalance, frontier.Height+1, fmt.Sprintf("export-%d", i), nil, map[string]interface{}{"achievement": fmt.Sprintf("E-%d", i)})
		if err := block.Sign(wallet.PrivateKey); err != nil {
			t.Fatalf("Sign achievement block %d failed: %v", i, err)
		}
		if err := lattice.ProcessBlock(block); err != nil {
			t.Fatalf("ProcessBlock achievement block %d failed: %v", i, err)
		}
	}

	bundle, err := lattice.ExportPersistence()
	if err != nil {
		t.Fatalf("ExportPersistence failed: %v", err)
	}
	if bundle.Integrity == nil || !bundle.Integrity.Healthy {
		t.Fatalf("expected healthy export integrity report, got %#v", bundle.Integrity)
	}
	if len(bundle.ConfirmedBlocks) != int(defaultLatticeSnapshotInterval) {
		t.Fatalf("expected %d confirmed blocks in export, got %d", defaultLatticeSnapshotInterval, len(bundle.ConfirmedBlocks))
	}
	if bundle.LatestSnapshot == nil {
		t.Fatal("expected export bundle to include latest snapshot")
	}
	if bundle.LatestSnapshot.LastSequence != defaultLatticeSnapshotInterval {
		t.Fatalf("expected exported snapshot sequence %d, got %d", defaultLatticeSnapshotInterval, bundle.LatestSnapshot.LastSequence)
	}
}

func TestPersistentLatticeBackupCreatesPortableSQLiteCopy(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "lattice.db")
	lattice, err := NewPersistentLattice(dbPath)
	if err != nil {
		t.Fatalf("NewPersistentLattice failed: %v", err)
	}
	defer func() {
		if err := lattice.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

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

	backupPath := filepath.Join(t.TempDir(), "backup", "portable-lattice.db")
	backup, err := lattice.BackupPersistence(backupPath)
	if err != nil {
		t.Fatalf("BackupPersistence failed: %v", err)
	}
	if backup.BackupPath != backupPath {
		t.Fatalf("unexpected backup path: %s", backup.BackupPath)
	}
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("expected backup file to exist: %v", err)
	}

	reloaded, err := NewPersistentLattice(backupPath)
	if err != nil {
		t.Fatalf("NewPersistentLattice from backup failed: %v", err)
	}
	defer func() {
		if err := reloaded.Close(); err != nil {
			t.Fatalf("Close reloaded backup failed: %v", err)
		}
	}()
	if len(reloaded.blocks) != 1 {
		t.Fatalf("expected backup reload to contain 1 block, got %d", len(reloaded.blocks))
	}
	if reloaded.GetFrontier(wallet.PublicKey) == nil {
		t.Fatal("expected backup reload frontier to exist")
	}
}

func TestImportBundleToPathCreatesVerifiedPortableDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "lattice.db")
	lattice, err := NewPersistentLattice(dbPath)
	if err != nil {
		t.Fatalf("NewPersistentLattice failed: %v", err)
	}
	defer func() {
		if err := lattice.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

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

	bundle, err := lattice.ExportPersistence()
	if err != nil {
		t.Fatalf("ExportPersistence failed: %v", err)
	}

	importPath := filepath.Join(t.TempDir(), "imported", "bundle-import.db")
	result, err := ImportBundleToPath(importPath, bundle)
	if err != nil {
		t.Fatalf("ImportBundleToPath failed: %v", err)
	}
	if result.TargetPath != importPath {
		t.Fatalf("unexpected import target path: %s", result.TargetPath)
	}
	if result.BlockCount != 1 {
		t.Fatalf("expected imported block count 1, got %d", result.BlockCount)
	}

	reloaded, err := NewPersistentLattice(importPath)
	if err != nil {
		t.Fatalf("NewPersistentLattice from imported bundle failed: %v", err)
	}
	defer func() {
		if err := reloaded.Close(); err != nil {
			t.Fatalf("Close imported lattice failed: %v", err)
		}
	}()
	if reloaded.GetFrontier(wallet.PublicKey) == nil {
		t.Fatal("expected imported lattice frontier to exist")
	}
}

func TestRestoreBackupToPathCreatesVerifiedPortableDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "lattice.db")
	lattice, err := NewPersistentLattice(dbPath)
	if err != nil {
		t.Fatalf("NewPersistentLattice failed: %v", err)
	}
	defer func() {
		if err := lattice.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

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

	backupPath := filepath.Join(t.TempDir(), "backup", "seed.db")
	backup, err := lattice.BackupPersistence(backupPath)
	if err != nil {
		t.Fatalf("BackupPersistence failed: %v", err)
	}

	restorePath := filepath.Join(t.TempDir(), "restored", "portable-restore.db")
	result, err := RestoreBackupToPath(backup.BackupPath, restorePath)
	if err != nil {
		t.Fatalf("RestoreBackupToPath failed: %v", err)
	}
	if result.TargetPath != restorePath {
		t.Fatalf("unexpected restore target path: %s", result.TargetPath)
	}
	if result.SourcePath != backup.BackupPath {
		t.Fatalf("unexpected restore source path: %s", result.SourcePath)
	}

	reloaded, err := NewPersistentLattice(restorePath)
	if err != nil {
		t.Fatalf("NewPersistentLattice from restored backup failed: %v", err)
	}
	defer func() {
		if err := reloaded.Close(); err != nil {
			t.Fatalf("Close restored lattice failed: %v", err)
		}
	}()
	if reloaded.GetFrontier(wallet.PublicKey) == nil {
		t.Fatal("expected restored lattice frontier to exist")
	}
}

func TestCreateSignedEncryptedBackupBundleRestoresVerifiedPortableDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "lattice.db")
	lattice, err := NewPersistentLattice(dbPath)
	if err != nil {
		t.Fatalf("NewPersistentLattice failed: %v", err)
	}
	defer func() {
		if err := lattice.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

	wallet, err := torrent.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair failed: %v", err)
	}
	operator, err := torrent.GenerateKeypair()
	if err != nil {
		t.Fatalf("Generate operator keypair failed: %v", err)
	}

	genesis := torrent.NewBlock("open", wallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	if err := genesis.Sign(wallet.PrivateKey); err != nil {
		t.Fatalf("Sign genesis failed: %v", err)
	}
	if err := lattice.ProcessBlock(genesis); err != nil {
		t.Fatalf("ProcessBlock genesis failed: %v", err)
	}

	bundlePath := filepath.Join(t.TempDir(), "bundles", "portable.secure-backup.json")
	bundle, err := lattice.CreateSignedEncryptedBackupBundle(bundlePath, "correct horse battery staple", operator.PrivateKey)
	if err != nil {
		t.Fatalf("CreateSignedEncryptedBackupBundle failed: %v", err)
	}
	if !bundle.Signed || bundle.Signature == nil {
		t.Fatalf("expected signed secure backup bundle, got %#v", bundle)
	}
	if bundle.Signature.PublicKey != operator.PublicKey {
		t.Fatalf("unexpected operator signing public key: %s", bundle.Signature.PublicKey)
	}
	if _, err := os.Stat(bundlePath); err != nil {
		t.Fatalf("expected secure bundle file to exist: %v", err)
	}

	restorePath := filepath.Join(t.TempDir(), "restored", "secure-restore.db")
	restored, err := lattice.RestoreSignedEncryptedBackupBundle(bundle.BundlePath, "correct horse battery staple", restorePath, true)
	if err != nil {
		t.Fatalf("RestoreSignedEncryptedBackupBundle failed: %v", err)
	}
	if !restored.SignatureVerified {
		t.Fatalf("expected restore to verify signature, got %#v", restored)
	}
	if restored.Restore.TargetPath != restorePath {
		t.Fatalf("unexpected secure restore target path: %s", restored.Restore.TargetPath)
	}

	reloaded, err := NewPersistentLattice(restorePath)
	if err != nil {
		t.Fatalf("NewPersistentLattice from secure restore failed: %v", err)
	}
	defer func() {
		if err := reloaded.Close(); err != nil {
			t.Fatalf("Close secure restored lattice failed: %v", err)
		}
	}()
	if reloaded.GetFrontier(wallet.PublicKey) == nil {
		t.Fatal("expected secure restored lattice frontier to exist")
	}
}

func TestRestoreSignedEncryptedBackupBundleRejectsTamperedSignature(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "lattice.db")
	lattice, err := NewPersistentLattice(dbPath)
	if err != nil {
		t.Fatalf("NewPersistentLattice failed: %v", err)
	}
	defer func() {
		if err := lattice.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

	wallet, err := torrent.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair failed: %v", err)
	}
	operator, err := torrent.GenerateKeypair()
	if err != nil {
		t.Fatalf("Generate operator keypair failed: %v", err)
	}

	genesis := torrent.NewBlock("open", wallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	if err := genesis.Sign(wallet.PrivateKey); err != nil {
		t.Fatalf("Sign genesis failed: %v", err)
	}
	if err := lattice.ProcessBlock(genesis); err != nil {
		t.Fatalf("ProcessBlock genesis failed: %v", err)
	}

	bundlePath := filepath.Join(t.TempDir(), "bundles", "tampered.secure-backup.json")
	bundle, err := lattice.CreateSignedEncryptedBackupBundle(bundlePath, "bundle-passphrase", operator.PrivateKey)
	if err != nil {
		t.Fatalf("CreateSignedEncryptedBackupBundle failed: %v", err)
	}

	raw, err := os.ReadFile(bundle.BundlePath)
	if err != nil {
		t.Fatalf("Read secure bundle failed: %v", err)
	}
	var parsed SecureLatticeBackupBundle
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("Decode secure bundle failed: %v", err)
	}
	parsed.Signature.Signature = strings.Repeat("x", len(parsed.Signature.Signature))
	mutated, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		t.Fatalf("Re-encode tampered bundle failed: %v", err)
	}
	if err := os.WriteFile(bundle.BundlePath, mutated, 0600); err != nil {
		t.Fatalf("Write tampered bundle failed: %v", err)
	}

	if _, err := lattice.RestoreSignedEncryptedBackupBundle(bundle.BundlePath, "bundle-passphrase", filepath.Join(t.TempDir(), "restored", "should-fail.db"), true); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("expected signature verification failure, got %v", err)
	}
}

func TestRestoreSignedEncryptedBackupBundleRejectsWrongPassphrase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "lattice.db")
	lattice, err := NewPersistentLattice(dbPath)
	if err != nil {
		t.Fatalf("NewPersistentLattice failed: %v", err)
	}
	defer func() {
		if err := lattice.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

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

	bundlePath := filepath.Join(t.TempDir(), "bundles", "wrong-passphrase.secure-backup.json")
	if _, err := lattice.CreateSignedEncryptedBackupBundle(bundlePath, "correct-passphrase", ""); err != nil {
		t.Fatalf("CreateSignedEncryptedBackupBundle failed: %v", err)
	}

	if _, err := lattice.RestoreSignedEncryptedBackupBundle(bundlePath, "incorrect-passphrase", filepath.Join(t.TempDir(), "restored", "wrong-passphrase.db"), false); err == nil || !strings.Contains(err.Error(), "decrypt") {
		t.Fatalf("expected decryption failure for wrong passphrase, got %v", err)
	}
}
