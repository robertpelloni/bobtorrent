package identity

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type URLVerifier struct {
	Client *http.Client
}

func NewURLVerifier() *URLVerifier {
	return &URLVerifier{
		Client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (v *URLVerifier) isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 10:
			return true
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return true
		case ip4[0] == 192 && ip4[1] == 168:
			return true
		}
	}
	return false
}

// Verify implements the Verifier interface required by VerifierService.
func (v *URLVerifier) Verify(ctx context.Context, attr Attestation) (*VerificationResult, error) {
	// Security check to prevent SSRF
	host := attr.URL
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		// Basic naive extraction for SSRF check, in production use net/url
		parts := strings.Split(strings.TrimPrefix(strings.TrimPrefix(host, "http://"), "https://"), "/")
		host = parts[0]
	}
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}

	ips, err := net.LookupIP(host)
	if err == nil {
		for _, ip := range ips {
			if v.isPrivateIP(ip) {
				return &VerificationResult{
					Success:    false,
					Message:    fmt.Sprintf("verification to private/loopback IP %s is not permitted (SSRF protection)", ip.String()),
					VerifiedAt: time.Now().UnixMilli(),
					Kind:       attr.Kind,
					URL:        attr.URL,
					Account:    attr.Account,
				}, nil
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", attr.URL, nil)
	if err != nil {
		return &VerificationResult{
			Success:    false,
			Message:    fmt.Sprintf("Failed to create request: %v", err),
			VerifiedAt: time.Now().UnixMilli(),
			Kind:       attr.Kind,
			URL:        attr.URL,
			Account:    attr.Account,
		}, nil
	}

	resp, err := v.Client.Do(req)
	if err != nil {
		return &VerificationResult{
			Success:    false,
			Message:    fmt.Sprintf("Failed to fetch URL: %v", err),
			VerifiedAt: time.Now().UnixMilli(),
			Kind:       attr.Kind,
			URL:        attr.URL,
			Account:    attr.Account,
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &VerificationResult{
			Success:    false,
			Message:    fmt.Sprintf("URL returned unexpected status: %d", resp.StatusCode),
			VerifiedAt: time.Now().UnixMilli(),
			Kind:       attr.Kind,
			URL:        attr.URL,
			Account:    attr.Account,
		}, nil
	}

	// Limit read to prevent memory exhaustion
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return &VerificationResult{
			Success:    false,
			Message:    fmt.Sprintf("Failed to read URL response: %v", err),
			VerifiedAt: time.Now().UnixMilli(),
			Kind:       attr.Kind,
			URL:        attr.URL,
			Account:    attr.Account,
		}, nil
	}

	content := string(body)
	expectedAttestation := fmt.Sprintf("BOBTORRENT_IDENTITY:%s", attr.Account)

	if strings.Contains(content, expectedAttestation) {
		return &VerificationResult{
			Success:    true,
			Message:    "Identity successfully verified via signed URL.",
			VerifiedAt: time.Now().UnixMilli(),
			Kind:       attr.Kind,
			URL:        attr.URL,
			Account:    attr.Account,
		}, nil
	}

	return &VerificationResult{
		Success:    false,
		Message:    fmt.Sprintf("Verification failed: Content does not contain expected attestation %s", expectedAttestation),
		VerifiedAt: time.Now().UnixMilli(),
		Kind:       attr.Kind,
		URL:        attr.URL,
		Account:    attr.Account,
	}, nil
}
