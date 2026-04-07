package identity

import (
	"context"
	"testing"
)

func TestMockVerifier(t *testing.T) {
	v := &MockVerifier{}
	ctx := context.Background()

	// Should succeed for KindMock
	res1, err := v.Verify(ctx, Attestation{Kind: KindMock, URL: "https://example.com", Account: "abc"})
	if err != nil || !res1.Success {
		t.Fatalf("expected success for mock kind, got %v (err: %v)", res1.Success, err)
	}

	// Should succeed if URL contains verify-me
	res2, err := v.Verify(ctx, Attestation{Kind: KindGitHub, URL: "https://github.com/user/verify-me", Account: "abc"})
	if err != nil || !res2.Success {
		t.Fatalf("expected success for verify-me URL, got %v (err: %v)", res2.Success, err)
	}

	// Should fail otherwise
	res3, err := v.Verify(ctx, Attestation{Kind: KindGitHub, URL: "https://github.com/user/other", Account: "abc"})
	if err != nil || res3.Success {
		t.Fatalf("expected failure for other URL, got %v (err: %v)", res3.Success, err)
	}
}

func TestVerifierService(t *testing.T) {
	svc := NewVerifierService()
	mock := &MockVerifier{}
	svc.RegisterVerifier(KindGitHub, mock)

	ctx := context.Background()

	// Registered kind
	res1, err := svc.Verify(ctx, Attestation{Kind: KindGitHub, URL: "verify-me", Account: "abc"})
	if err != nil || !res1.Success {
		t.Fatalf("expected success for registered verifier, got %v", res1.Success)
	}

	// Unregistered kind
	res2, err := svc.Verify(ctx, Attestation{Kind: KindORCID, URL: "verify-me", Account: "abc"})
	if err != nil || res2.Success {
		t.Fatalf("expected failure for unregistered verifier, got %v", res2.Success)
	}
}
