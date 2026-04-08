package consensus

import (
	"testing"
	"time"

	"bobtorrent/pkg/torrent"
)

func TestTransitionSendReceive(t *testing.T) {
	lattice := NewLattice()
	alice := mustGenerateKeypair(t)
	bob := mustGenerateKeypair(t)

	// 1. Alice Opens
	aliceOpen := torrent.NewBlock("open", alice.PublicKey, nil, 2000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, aliceOpen, alice.PrivateKey)
	if err := lattice.ProcessBlock(aliceOpen); err != nil {
		t.Fatalf("Alice open failed: %v", err)
	}

	// 2. Alice sends 500 to Bob
	aliceSend := torrent.NewBlock("send", alice.PublicKey, &aliceOpen.Hash, 1500, 0, 1, bob.PublicKey, nil, nil)
	mustSignBlock(t, aliceSend, alice.PrivateKey)
	if err := lattice.ProcessBlock(aliceSend); err != nil {
		t.Fatalf("Alice send failed: %v", err)
	}

	if lattice.GetBalance(alice.PublicKey) != 1500 {
		t.Fatalf("expected Alice balance 1500, got %d", lattice.GetBalance(alice.PublicKey))
	}
	if len(lattice.pending[bob.PublicKey]) != 1 {
		t.Fatalf("expected 1 pending for Bob, got %d", len(lattice.pending[bob.PublicKey]))
	}

	// 3. Bob Opens (consumes pending)
	bobOpen := torrent.NewBlock("open", bob.PublicKey, nil, 500, 0, 0, aliceSend.Hash, nil, nil)
	mustSignBlock(t, bobOpen, bob.PrivateKey)
	if err := lattice.ProcessBlock(bobOpen); err != nil {
		t.Fatalf("Bob open failed: %v", err)
	}

	if lattice.GetBalance(bob.PublicKey) != 500 {
		t.Fatalf("expected Bob balance 500, got %d", lattice.GetBalance(bob.PublicKey))
	}
	if len(lattice.pending[bob.PublicKey]) != 0 {
		t.Fatalf("expected 0 pending for Bob, got %d", len(lattice.pending[bob.PublicKey]))
	}
}

func TestTransitionNFTLifeCycle(t *testing.T) {
	lattice := NewLattice()
	alice := mustGenerateKeypair(t)
	bob := mustGenerateKeypair(t)

	// Alice Opens
	aliceOpen := torrent.NewBlock("open", alice.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, aliceOpen, alice.PrivateKey)
	lattice.ProcessBlock(aliceOpen)

	// 1. Mint NFT (Costs 50)
	mintPayload := map[string]interface{}{"name": "Lattice Art", "magnet": "magnet:?xt=urn:btih:nft123"}
	mintNFT := torrent.NewBlock("mint_nft", alice.PublicKey, &aliceOpen.Hash, 950, 0, 1, "", nil, mintPayload)
	mustSignBlock(t, mintNFT, alice.PrivateKey)
	if err := lattice.ProcessBlock(mintNFT); err != nil {
		t.Fatalf("Mint NFT failed: %v", err)
	}

	nft, ok := lattice.nfts[mintNFT.Hash]
	if !ok || nft.Owner != alice.PublicKey {
		t.Fatal("NFT not registered to Alice")
	}

	// 2. Transfer NFT (Costs 1)
	transferPayload := map[string]interface{}{"newOwner": bob.PublicKey}
	transferNFT := torrent.NewBlock("transfer_nft", alice.PublicKey, &mintNFT.Hash, 949, 0, 2, mintNFT.Hash, nil, transferPayload)
	mustSignBlock(t, transferNFT, alice.PrivateKey)
	if err := lattice.ProcessBlock(transferNFT); err != nil {
		t.Fatalf("Transfer NFT failed: %v", err)
	}

	if nft.Owner != bob.PublicKey {
		t.Fatalf("expected NFT owner Bob, got %s", nft.Owner)
	}
	if lattice.GetBalance(alice.PublicKey) != 949 {
		t.Fatalf("expected Alice balance 949, got %d", lattice.GetBalance(alice.PublicKey))
	}
}

func TestTransitionStaking(t *testing.T) {
	lattice := NewLattice()
	alice := mustGenerateKeypair(t)

	// Alice Opens
	aliceOpen := torrent.NewBlock("open", alice.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, aliceOpen, alice.PrivateKey)
	lattice.ProcessBlock(aliceOpen)

	// 1. Stake 400
	stakePayload := map[string]interface{}{"amount": float64(400)}
	stake := torrent.NewBlock("stake", alice.PublicKey, &aliceOpen.Hash, 600, 400, 1, "", nil, stakePayload)
	mustSignBlock(t, stake, alice.PrivateKey)
	if err := lattice.ProcessBlock(stake); err != nil {
		t.Fatalf("Stake failed: %v", err)
	}

	if lattice.GetStakedBalance(alice.PublicKey) != 400 {
		t.Fatalf("expected staked balance 400, got %d", lattice.GetStakedBalance(alice.PublicKey))
	}

	// 2. Unstake (with immediate yield)
	// Note: in a real ledger time passes, but here we test the transition logic.
	unstakePayload := map[string]interface{}{"amount": float64(400)}
	unstake := torrent.NewBlock("unstake", alice.PublicKey, &stake.Hash, 1000, 0, 2, "", nil, unstakePayload)
	mustSignBlock(t, unstake, alice.PrivateKey)
	if err := lattice.ProcessBlock(unstake); err != nil {
		t.Fatalf("Unstake failed: %v", err)
	}

	if lattice.GetStakedBalance(alice.PublicKey) != 0 {
		t.Fatalf("expected staked balance 0, got %d", lattice.GetStakedBalance(alice.PublicKey))
	}
	if lattice.GetFrontier(alice.PublicKey).Balance != 1000 {
		t.Fatalf("expected balance 1000, got %d", lattice.GetFrontier(alice.PublicKey).Balance)
	}
}

func TestTransitionHTLCSwap(t *testing.T) {
	lattice := NewLattice()
	alice := mustGenerateKeypair(t)
	bob := mustGenerateKeypair(t)

	// Alice Opens
	aliceOpen := torrent.NewBlock("open", alice.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, aliceOpen, alice.PrivateKey)
	lattice.ProcessBlock(aliceOpen)

	// Bob Opens
	aliceSendBob := torrent.NewBlock("send", alice.PublicKey, &aliceOpen.Hash, 900, 0, 1, bob.PublicKey, nil, nil)
	mustSignBlock(t, aliceSendBob, alice.PrivateKey)
	lattice.ProcessBlock(aliceSendBob)
	bobOpen := torrent.NewBlock("open", bob.PublicKey, nil, 100, 0, 0, aliceSendBob.Hash, nil, nil)
	mustSignBlock(t, bobOpen, bob.PrivateKey)
	lattice.ProcessBlock(bobOpen)

	// 1. Alice Initiates Swap (locks 500)
	secret := "top-secret-123"
	hashLock := torrent.HashSHA256(secret)
	initiatePayload := map[string]interface{}{
		"recipient": bob.PublicKey,
		"hashLock": hashLock,
		"expiry": float64(time.Now().Add(time.Hour).UnixMilli()),
	}
	initiate := torrent.NewBlock("initiate_swap", alice.PublicKey, &aliceSendBob.Hash, 400, 0, 2, "", nil, initiatePayload)
	mustSignBlock(t, initiate, alice.PrivateKey)
	if err := lattice.ProcessBlock(initiate); err != nil {
		t.Fatalf("Initiate swap failed: %v", err)
	}

	swap, ok := lattice.swaps[initiate.Hash]
	if !ok || swap.Status != "LOCKED" {
		t.Fatal("Swap not locked")
	}

	// 2. Bob Claims Swap
	claimPayload := map[string]interface{}{"secret": secret}
	bobFrontier := lattice.GetFrontier(bob.PublicKey)
	claim := torrent.NewBlock("claim_swap", bob.PublicKey, &bobFrontier.Hash, 600, 0, 1, initiate.Hash, nil, claimPayload)
	mustSignBlock(t, claim, bob.PrivateKey)
	if err := lattice.ProcessBlock(claim); err != nil {
		t.Fatalf("Claim swap failed: %v", err)
	}

	if lattice.GetBalance(bob.PublicKey) != 600 {
		t.Fatalf("expected Bob balance 600, got %d", lattice.GetBalance(bob.PublicKey))
	}
	if swap.Status != "CLAIMED" {
		t.Fatalf("expected swap status CLAIMED, got %s", swap.Status)
	}
}

func TestTransitionHTLCRefund(t *testing.T) {
	lattice := NewLattice()
	alice := mustGenerateKeypair(t)
	bob := mustGenerateKeypair(t)

	// Alice Opens
	aliceOpen := torrent.NewBlock("open", alice.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, aliceOpen, alice.PrivateKey)
	lattice.ProcessBlock(aliceOpen)

	// 1. Alice Initiates Swap (locks 500)
	secret := "top-secret-123"
	hashLock := torrent.HashSHA256(secret)
	now := time.Now().UnixMilli()
	expiry := now + 10000 // Expires in 10 seconds
	initiatePayload := map[string]interface{}{
		"recipient": bob.PublicKey,
		"hashLock":  hashLock,
		"expiry":    float64(expiry),
	}
	initiate := torrent.NewBlock("initiate_swap", alice.PublicKey, &aliceOpen.Hash, 500, 0, 1, "", nil, initiatePayload)
	initiate.Timestamp = now
	mustSignBlock(t, initiate, alice.PrivateKey)
	if err := lattice.ProcessBlock(initiate); err != nil {
		t.Fatalf("Initiate swap failed: %v", err)
	}

	swap, ok := lattice.swaps[initiate.Hash]
	if !ok || swap.Status != "LOCKED" {
		t.Fatal("Swap not locked")
	}

	// 2. Alice Refunds Swap (after expiry)
	refund := torrent.NewBlock("refund_swap", alice.PublicKey, &initiate.Hash, 1000, 0, 2, initiate.Hash, nil, nil)
	refund.Timestamp = expiry + 1000 // 1 second after expiry
	mustSignBlock(t, refund, alice.PrivateKey)
	if err := lattice.ProcessBlock(refund); err != nil {
		t.Fatalf("Refund swap failed: %v", err)
	}

	if lattice.GetBalance(alice.PublicKey) != 1000 {
		t.Fatalf("expected Alice balance 1000, got %d", lattice.GetBalance(alice.PublicKey))
	}
	if swap.Status != "REFUNDED" {
		t.Fatalf("expected swap status REFUNDED, got %s", swap.Status)
	}
}

func TestTransitionGovernance(t *testing.T) {
	lattice := NewLattice()
	alice := mustGenerateKeypair(t)

	// Alice Opens (High balance for voting power)
	aliceOpen := torrent.NewBlock("open", alice.PublicKey, nil, 10000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, aliceOpen, alice.PrivateKey)
	lattice.ProcessBlock(aliceOpen)

	// 1. Alice proposes (Costs 10)
	propPayload := map[string]interface{}{"title": "Build More Supernodes", "description": "Expand the network."}
	proposal := torrent.NewBlock("proposal", alice.PublicKey, &aliceOpen.Hash, 9990, 0, 1, "", nil, propPayload)
	mustSignBlock(t, proposal, alice.PrivateKey)
	if err := lattice.ProcessBlock(proposal); err != nil {
		t.Fatalf("Proposal failed: %v", err)
	}

	p, ok := lattice.proposals[proposal.Hash]
	if !ok || p.Status != "active" {
		t.Fatal("Proposal not active")
	}

	// 2. Alice votes FOR
	aliceFrontier := lattice.GetFrontier(alice.PublicKey)
	vote := torrent.NewBlock("vote", alice.PublicKey, &aliceFrontier.Hash, 9990, 0, 2, proposal.Hash, nil, map[string]interface{}{"vote": "for"})
	mustSignBlock(t, vote, alice.PrivateKey)
	if err := lattice.ProcessBlock(vote); err != nil {
		t.Fatalf("Vote failed: %v", err)
	}

	if p.VotesFor <= 0 {
		t.Fatal("expected positive votes for")
	}
}

func TestTransitionStorageMarket(t *testing.T) {
	lattice := NewLattice()
	alice := mustGenerateKeypair(t)
	bob := mustGenerateKeypair(t)

	// Alice Opens
	aliceOpen := torrent.NewBlock("open", alice.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, aliceOpen, alice.PrivateKey)
	lattice.ProcessBlock(aliceOpen)

	// Bob Opens
	bobOpen := torrent.NewBlock("open", bob.PublicKey, nil, 0, 0, 0, "", nil, nil)
	// Bob needs funds to open, but let's assume we use the SYSTEM_GENESIS bypass for simplicity in this test
	// or Alice sends to Bob. Let's do Alice sends to Bob.
	aliceSendBob := torrent.NewBlock("send", alice.PublicKey, &aliceOpen.Hash, 900, 0, 1, bob.PublicKey, nil, nil)
	mustSignBlock(t, aliceSendBob, alice.PrivateKey)
	lattice.ProcessBlock(aliceSendBob)

	bobOpen.Link = aliceSendBob.Hash
	bobOpen.Balance = 100
	mustSignBlock(t, bobOpen, bob.PrivateKey)
	lattice.ProcessBlock(bobOpen)

	// 1. Alice Creates Market Bid (Costs 100)
	bidPayload := map[string]interface{}{"magnet": "magnet:?xt=urn:btih:market-test"}
	bid := torrent.NewBlock("market_bid", alice.PublicKey, &aliceSendBob.Hash, 800, 0, 2, "", nil, bidPayload)
	mustSignBlock(t, bid, alice.PrivateKey)
	if err := lattice.ProcessBlock(bid); err != nil {
		t.Fatalf("Market bid failed: %v", err)
	}

	b, ok := lattice.marketBids[bid.Hash]
	if !ok || b.Status != "OPEN" {
		t.Fatal("Market bid not open")
	}

	// 2. Bob Accepts Bid (Receives 100)
	accept := torrent.NewBlock("accept_bid", bob.PublicKey, &bobOpen.Hash, 200, 0, 1, bid.Hash, nil, nil)
	mustSignBlock(t, accept, bob.PrivateKey)
	if err := lattice.ProcessBlock(accept); err != nil {
		t.Fatalf("Accept bid failed: %v", err)
	}

	if lattice.GetBalance(bob.PublicKey) != 200 {
		t.Fatalf("expected Bob balance 200, got %d", lattice.GetBalance(bob.PublicKey))
	}
	if b.Status != "ACCEPTED" || b.AcceptedBy != bob.PublicKey {
		t.Fatalf("expected bid status ACCEPTED by Bob, got %s by %s", b.Status, b.AcceptedBy)
	}
}
