package identity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestORCIDVerifier(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/success", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"bio": "I am a researcher. My key is bobcoin_key_12345"}`))
	})
	mux.HandleFunc("/fail_missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"bio": "I am a researcher."}`))
	})
	mux.HandleFunc("/error_500", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	verifier := NewORCIDVerifier()

	tests := []struct {
		name       string
		attr       Attestation
		wantSuccess bool
		wantMsg    string
	}{
		{
			name: "Not ORCID Host",
			attr: Attestation{
				Kind:    KindORCID,
				URL:     "https://example.com/orcid",
				Account: "bobcoin_key_12345",
			},
			wantSuccess: false,
			wantMsg:     "Verification failed: URL does not appear to be a valid ORCID URL",
		},
		{
			name: "Subdomain spoofing",
			attr: Attestation{
				Kind:    KindORCID,
				URL:     "https://orcid.org.attacker.com/profile",
				Account: "bobcoin_key_12345",
			},
			wantSuccess: false,
			wantMsg:     "Verification failed: URL does not appear to be a valid ORCID URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := verifier.Verify(context.Background(), tt.attr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Success != tt.wantSuccess {
				t.Errorf("got success %v, want %v", res.Success, tt.wantSuccess)
			}
			if res.Message != tt.wantMsg {
				t.Errorf("got message %q, want %q", res.Message, tt.wantMsg)
			}
		})
	}
}
