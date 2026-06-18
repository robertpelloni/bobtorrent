package identity

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// ORCIDVerifier validates Bobcoin account ownership via ORCID.
// It fetches the provided URL and checks if the returned content contains
// the publisher's Bobcoin public key.
type ORCIDVerifier struct {
	client *resty.Client
}

func NewORCIDVerifier() *ORCIDVerifier {
	return &ORCIDVerifier{
		client: resty.New().
			SetTimeout(10 * time.Second).
			SetHeader("User-Agent", "Bobtorrent-Go-Verifier/1.0").
			SetHeader("Accept", "application/json"),
	}
}

func (v *ORCIDVerifier) Verify(ctx context.Context, attr Attestation) (*VerificationResult, error) {
	u, err := url.Parse(attr.URL)
	if err != nil {
		return v.fail(attr, fmt.Sprintf("invalid ORCID URL: %v", err)), nil
	}

	// Ensure the host is exactly "pub.orcid.org" or "orcid.org"
	if u.Host != "pub.orcid.org" && u.Host != "orcid.org" {
		return v.fail(attr, "URL does not appear to be a valid ORCID URL"), nil
	}

	resp, err := v.client.R().SetContext(ctx).Get(attr.URL)
	if err != nil {
		return v.fail(attr, fmt.Sprintf("failed to fetch ORCID content: %v", err)), nil
	}

	if !resp.IsSuccess() {
		return v.fail(attr, fmt.Sprintf("ORCID returned status %d", resp.StatusCode())), nil
	}

	content := strings.TrimSpace(string(resp.Body()))
	if !strings.Contains(content, attr.Account) {
		return v.fail(attr, "Bobcoin account public key not found in ORCID content."), nil
	}

	return &VerificationResult{
		Success:    true,
		Message:    "ORCID identity successfully verified.",
		VerifiedAt: time.Now().UnixMilli(),
		Kind:       attr.Kind,
		URL:        attr.URL,
		Account:    attr.Account,
	}, nil
}

func (v *ORCIDVerifier) fail(attr Attestation, msg string) *VerificationResult {
	return &VerificationResult{
		Success:    false,
		Message:    "Verification failed: " + msg,
		VerifiedAt: time.Now().UnixMilli(),
		Kind:       attr.Kind,
		URL:        attr.URL,
		Account:    attr.Account,
	}
}
