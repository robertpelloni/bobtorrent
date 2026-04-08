package identity

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitHubVerifier(t *testing.T) {
	account := "abc-123-bobcoin-pubkey"

	// Mock GitHub Gist Server
	mux := http.NewServeMux()
	mux.HandleFunc("/user/gist1/raw", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "This is my Bobcoin key: %s", account)
	})
	mux.HandleFunc("/user/gist2/raw", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "No key here.")
	})
	mux.HandleFunc("/user/gist-error/raw", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	verifier := NewGitHubVerifier()
	ctx := context.Background()

	// 1. Successful verification
	res1, err := verifier.Verify(ctx, Attestation{
		Kind:    KindGitHub,
		URL:     server.URL + "/user/gist1/raw",
		Account: account,
	})
	if err != nil || !res1.Success {
		t.Fatalf("expected success for valid gist, got %v (err: %v)", res1.Success, err)
	}

	// 2. Failed verification (missing key)
	res2, err := verifier.Verify(ctx, Attestation{
		Kind:    KindGitHub,
		URL:     server.URL + "/user/gist2/raw",
		Account: account,
	})
	if err != nil || res2.Success {
		t.Fatalf("expected failure for gist without key, got %v", res2.Success)
	}

	// 3. Failed verification (404)
	res3, err := verifier.Verify(ctx, Attestation{
		Kind:    KindGitHub,
		URL:     server.URL + "/user/gist-error/raw",
		Account: account,
	})
	if err != nil || res3.Success {
		t.Fatalf("expected failure for 404 response, got %v", res3.Success)
	}

	// 4. Invalid URL
	res4, err := verifier.Verify(ctx, Attestation{
		Kind:    KindGitHub,
		URL:     "not-a-url",
		Account: account,
	})
	if err != nil || res4.Success {
		t.Fatalf("expected failure for invalid URL, got %v", res4.Success)
	}
}
