package identity

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// isPrivateIP checks if an IP is private, loopback, or unspecified.
func isPrivateIP(ip net.IP) bool {
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

// safeDialContext creates a dialer that prevents SSRF via TOCTOU/DNS rebinding
// by checking the resolved IP address before making the connection.
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}

	var safeIP net.IP
	for _, ip := range ips {
		if !isPrivateIP(ip) {
			safeIP = ip
			break
		}
	}

	if safeIP == nil {
		return nil, errors.New("URL host resolves to private or unsafe IP address")
	}

	safeAddr := net.JoinHostPort(safeIP.String(), port)
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return dialer.DialContext(ctx, network, safeAddr)
}

// URLVerifier validates Bobcoin account ownership via a custom URL.
// It fetches the provided URL and checks if the returned content contains
// the publisher's Bobcoin public key.
type URLVerifier struct {
	client *resty.Client
}

func NewURLVerifier() *URLVerifier {
	transport := &http.Transport{
		DialContext: safeDialContext,
	}

	return &URLVerifier{
		client: resty.New().
			SetTransport(transport).
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
