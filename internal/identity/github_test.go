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

func TestGitHubVerifier(t *testing.T) {
	expectedAccount := "test-account-123"
	expectedAttestation := "BOBTORRENT_IDENTITY:" + expectedAccount
	gistID := "abcdef123456"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/gists/"+gistID, r.URL.Path)

		// Mock GitHub Gist API response
		response := `{
			"files": {
				"bobtorrent.txt": {
					"content": "` + expectedAttestation + `"
				}
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, response)
	}))
	defer ts.Close()

	verifier := NewGitHubVerifier()
	verifier.BaseURL = ts.URL

	attr := Attestation{
		Kind:    KindGitHub,
		URL:     "https://gist.github.com/testuser/" + gistID,
		Account: expectedAccount,
	}

	result, err := verifier.Verify(context.Background(), attr)
	require.NoError(t, err)
	assert.True(t, result.Success)
}
