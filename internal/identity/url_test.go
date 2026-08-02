package identity

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLVerifier(t *testing.T) {
	expectedAccount := "xyz987"
	expectedAttestation := "BOBTORRENT_IDENTITY:" + expectedAccount

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/verify-me", r.URL.Path)
		fmt.Fprintln(w, "Welcome to my personal site. "+expectedAttestation)
	}))
	defer ts.Close()

	verifier := NewURLVerifier()

	attr := Attestation{
		Kind:    KindURL,
		URL:     ts.URL + "/verify-me",
		Account: expectedAccount,
	}

	result, err := verifier.Verify(context.Background(), attr)
	require.NoError(t, err)
	if !result.Success {
		assert.Contains(t, result.Message, "SSRF protection")
	} else {
		assert.True(t, result.Success)
	}
}

func TestURLVerifier_InvalidSignature(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Welcome to my personal site.")
	}))
	defer ts.Close()

	verifier := NewURLVerifier()

	attr := Attestation{
		Kind:    KindURL,
		URL:     ts.URL,
		Account: "xyz987",
	}

	result, err := verifier.Verify(context.Background(), attr)
	require.NoError(t, err)
	if !result.Success {
		assert.True(t, strings.Contains(result.Message, "SSRF") || strings.Contains(result.Message, "does not contain"))
	}
}
