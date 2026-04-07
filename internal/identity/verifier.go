package identity

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// AttestationKind represents the type of external identity being verified.
type AttestationKind string

const (
	KindGitHub AttestationKind = "github"
	KindORCID  AttestationKind = "orcid"
	KindURL    AttestationKind = "url"
	KindMock   AttestationKind = "mock"
)

// Attestation represents a claim by a publisher that they own an external
// identity or resource.
type Attestation struct {
	Kind    AttestationKind `json:"kind"`
	URL     string          `json:"url"`
	Account string          `json:"account"` // The Bobcoin account public key
}

// VerificationResult provides details on whether an attestation is authentic.
type VerificationResult struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	VerifiedAt int64     `json:"verifiedAt"`
	Kind      AttestationKind `json:"kind"`
	URL       string    `json:"url"`
	Account   string    `json:"account"`
}

// Verifier defines the interface for checking external identity claims.
type Verifier interface {
	Verify(ctx context.Context, attr Attestation) (*VerificationResult, error)
}

// VerifierService orchestrates multiple verifiers based on the attestation kind.
type VerifierService struct {
	verifiers map[AttestationKind]Verifier
}

func NewVerifierService() *VerifierService {
	return &VerifierService{
		verifiers: make(map[AttestationKind]Verifier),
	}
}

func (s *VerifierService) RegisterVerifier(kind AttestationKind, v Verifier) {
	s.verifiers[kind] = v
}

func (s *VerifierService) Verify(ctx context.Context, attr Attestation) (*VerificationResult, error) {
	v, ok := s.verifiers[attr.Kind]
	if !ok {
		return &VerificationResult{
			Success:    false,
			Message:    fmt.Sprintf("unsupported attestation kind: %s", attr.Kind),
			VerifiedAt: time.Now().UnixMilli(),
			Kind:       attr.Kind,
			URL:        attr.URL,
			Account:    attr.Account,
		}, nil
	}

	return v.Verify(ctx, attr)
}

// MockVerifier is used for development and testing. It accepts all attestations
// for the "mock" kind and those containing "verify-me" in the URL.
type MockVerifier struct{}

func (m *MockVerifier) Verify(ctx context.Context, attr Attestation) (*VerificationResult, error) {
	success := attr.Kind == KindMock || strings.Contains(attr.URL, "verify-me")
	msg := "Identity successfully verified (Mock)."
	if !success {
		msg = "Verification failed: Identity resource not found or signature mismatch (Mock)."
	}

	return &VerificationResult{
		Success:    success,
		Message:    msg,
		VerifiedAt: time.Now().UnixMilli(),
		Kind:       attr.Kind,
		URL:        attr.URL,
		Account:    attr.Account,
	}, nil
}
