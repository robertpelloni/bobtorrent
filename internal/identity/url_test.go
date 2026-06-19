package identity

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestURLVerifier(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/success", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`This is my website. Verify me: bobcoin_key_12345`))
	})
	mux.HandleFunc("/fail_missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`This is my website. Verify me: `))
	})
	mux.HandleFunc("/error_404", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	verifier := NewURLVerifier()
	// Override transport for testing local server without triggering SSRF blocker
	verifier.client.SetTransport(&http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	})

	tests := []struct {
		name       string
		attr       Attestation
		wantSuccess bool
		wantMsg    string
	}{
		{
			name: "Success",
			attr: Attestation{
				Kind:    KindURL,
				URL:     server.URL + "/success",
				Account: "bobcoin_key_12345",
			},
			wantSuccess: true,
			wantMsg:     "URL identity successfully verified.",
		},
		{
			name: "Missing Key",
			attr: Attestation{
				Kind:    KindURL,
				URL:     server.URL + "/fail_missing",
				Account: "bobcoin_key_12345",
			},
			wantSuccess: false,
			wantMsg:     "Verification failed: Bobcoin account public key not found in URL content.",
		},
		{
			name: "HTTP Error",
			attr: Attestation{
				Kind:    KindURL,
				URL:     server.URL + "/error_404",
				Account: "bobcoin_key_12345",
			},
			wantSuccess: false,
			wantMsg:     "Verification failed: URL returned status 404",
		},
		{
			name: "Invalid Scheme",
			attr: Attestation{
				Kind:    KindURL,
				URL:     "ftp://example.com/verify",
				Account: "bobcoin_key_12345",
			},
			wantSuccess: false,
			wantMsg:     "Verification failed: URL must use http or https scheme",
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
