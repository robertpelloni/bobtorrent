package identity

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// isPrivateIPHost checks if the given host string resolves to a private, loopback, or unspecified IP address.
var isPrivateIPHost = func(host string) bool {
	hostname := host
	if strings.Contains(host, ":") {
		h, _, err := net.SplitHostPort(host)
		if err == nil {
			hostname = h
		}
	}

	ips, err := net.LookupIP(hostname)
	if err != nil {
		// If we can't resolve it, assume it's unsafe or invalid.
		return true
	}

	for _, ip := range ips {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return true
		}
	}
	return false
}

// URLVerifier validates Bobcoin account ownership via a custom URL.
// It fetches the provided URL and checks if the returned content contains
// the publisher's Bobcoin public key.
type URLVerifier struct {
	client *resty.Client
}

func NewURLVerifier() *URLVerifier {
	return &URLVerifier{
		client: resty.New().
			SetTimeout(10 * time.Second).
			SetHeader("User-Agent", "Bobtorrent-Go-Verifier/1.0"),
	}
}

func (v *URLVerifier) Verify(ctx context.Context, attr Attestation) (*VerificationResult, error) {
	u, err := url.Parse(attr.URL)
	if err != nil {
		return v.fail(attr, fmt.Sprintf("invalid URL: %v", err)), nil
	}

	if u.Scheme != "https" && u.Scheme != "http" {
		return v.fail(attr, "URL must use http or https scheme"), nil
	}

	// Prevent SSRF by rejecting loopback, private, and unspecified addresses
	if isPrivateIPHost(u.Host) {
		return v.fail(attr, "URL host must be a public address"), nil
	}

	resp, err := v.client.R().SetContext(ctx).Get(attr.URL)
	if err != nil {
		return v.fail(attr, fmt.Sprintf("failed to fetch URL content: %v", err)), nil
	}

	if !resp.IsSuccess() {
		return v.fail(attr, fmt.Sprintf("URL returned status %d", resp.StatusCode())), nil
	}

	content := strings.TrimSpace(string(resp.Body()))
	if !strings.Contains(content, attr.Account) {
		return v.fail(attr, "Bobcoin account public key not found in URL content."), nil
	}

	return &VerificationResult{
		Success:    true,
		Message:    "URL identity successfully verified.",
		VerifiedAt: time.Now().UnixMilli(),
		Kind:       attr.Kind,
		URL:        attr.URL,
		Account:    attr.Account,
	}, nil
}

func (v *URLVerifier) fail(attr Attestation, msg string) *VerificationResult {
	return &VerificationResult{
		Success:    false,
		Message:    "Verification failed: " + msg,
		VerifiedAt: time.Now().UnixMilli(),
		Kind:       attr.Kind,
		URL:        attr.URL,
		Account:    attr.Account,
	}
}
