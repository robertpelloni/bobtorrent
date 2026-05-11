package identity

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestORCIDVerifier_Success(t *testing.T) {
	expectedAttestation := "BOBTORRENT_IDENTITY:1234567890abcdef"

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

	valid, err := verifier.Verify("0000-0002-1825-0097", expectedAttestation)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestORCIDVerifier_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	verifier := NewORCIDVerifier()
	verifier.BaseURL = ts.URL

	valid, err := verifier.Verify("0000-0000-0000-0000", "attest")
	require.Error(t, err)
	assert.False(t, valid)
	assert.Contains(t, err.Error(), "record not found")
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

	valid, err := verifier.Verify("0000-0002-1825-0097", "attest")
	require.Error(t, err)
	assert.False(t, valid)
	assert.Contains(t, err.Error(), "no biography found")
}
