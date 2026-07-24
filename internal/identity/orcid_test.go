package identity

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestORCIDVerifier_Success(t *testing.T) {
	expectedAccount := "1234567890abcdef"
	expectedAttestation := "BOBTORRENT_IDENTITY:" + expectedAccount

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/0000-0002-1825-0097", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		response := `{
			"person": {
				"biography": {
					"content": "Researcher, Developer. Attestation: ` + expectedAttestation + `."
				}
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, response)
	}))
	defer ts.Close()

	verifier := NewORCIDVerifier()
	verifier.BaseURL = ts.URL

	attr := Attestation{
		Kind:    KindORCID,
		URL:     "0000-0002-1825-0097",
		Account: expectedAccount,
	}

	result, err := verifier.Verify(context.Background(), attr)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestORCIDVerifier_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	verifier := NewORCIDVerifier()
	verifier.BaseURL = ts.URL

	attr := Attestation{
		Kind:    KindORCID,
		URL:     "0000-0000-0000-0000",
		Account: "test-acc",
	}

	result, err := verifier.Verify(context.Background(), attr)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Message, "record not found")
}

func TestORCIDVerifier_NoBiography(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"person": {
				"biography": null
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, response)
	}))
	defer ts.Close()

	verifier := NewORCIDVerifier()
	verifier.BaseURL = ts.URL

	attr := Attestation{
		Kind:    KindORCID,
		URL:     "0000-0002-1825-0097",
		Account: "test-acc",
	}

	result, err := verifier.Verify(context.Background(), attr)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Message, "No biography found")
}
