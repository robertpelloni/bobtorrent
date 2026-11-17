package consensus

import (
	"strings"
	"testing"

	"bobtorrent/pkg/torrent"
)

// Opt-in signature enforcement: with RequireSignatures on, live unsigned
// blocks are rejected while properly signed ones are accepted. Default
// (off) keeps accepting unsigned blocks for backward compatibility.
func TestRequireSignaturesRejectsUnsignedAcceptsSigned(t *testing.T) {
	wallet, err := torrent.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair failed: %v", err)
	}

	lattice := NewLattice()
	lattice.RequireSignatures = true

	unsigned := torrent.NewBlock("open", wallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	if err := lattice.ProcessBlock(unsigned); err == nil || !strings.Contains(err.Error(), "signature required") {
		t.Fatalf("expected signature-required rejection, got %v", err)
	}

	signed := torrent.NewBlock("open", wallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	if err := signed.Sign(wallet.PrivateKey); err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if err := lattice.ProcessBlock(signed); err != nil {
		t.Fatalf("signed genesis rejected: %v", err)
	}

	// Default lattice remains permissive.
	permissive := NewLattice()
	unsigned2 := torrent.NewBlock("open", wallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	if err := permissive.ProcessBlock(unsigned2); err != nil {
		t.Fatalf("default lattice should accept unsigned blocks: %v", err)
	}
}
