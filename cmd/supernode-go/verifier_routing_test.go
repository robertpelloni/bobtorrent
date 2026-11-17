package main

import (
	"context"
	"strings"
	"testing"

	"bobtorrent/internal/identity"
)

// ORCID and URL attestation kinds must route to the real verifiers,
// not the mock. A loopback URL must be refused by the real URLVerifier's
// SSRF guard — the mock would have reported a "(Mock)" verdict instead.
func TestVerifierRoutingUsesRealVerifiers(t *testing.T) {
	initVerifierService()
	if verifierSvc == nil {
		t.Fatal("verifier service not initialized")
	}

	res, err := verifierSvc.Verify(context.Background(), identity.Attestation{
		Kind:    identity.KindURL,
		URL:     "http://127.0.0.1:8002/status",
		Account: "tester",
	})
	if err != nil {
		t.Fatalf("verify error: %v", err)
	}
	if res.Success {
		t.Fatal("loopback URL must not verify")
	}
	if strings.Contains(res.Message, "(Mock)") {
		t.Fatalf("URL kind still routed to mock verifier: %s", res.Message)
	}
}
