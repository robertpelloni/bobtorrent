package identity

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// GitHubVerifier validates Bobcoin account ownership via GitHub Gists.
// It expects a Gist URL and verifies that the raw content of the Gist
// contains the publisher's Bobcoin public key.
type GitHubVerifier struct {
	client *resty.Client
}

func NewGitHubVerifier() *GitHubVerifier {
	return &GitHubVerifier{
		client: resty.New().
			SetTimeout(10 * time.Second).
			SetHeader("User-Agent", "Bobtorrent-Go-Verifier/1.0"),
	}
}

func (v *GitHubVerifier) Verify(ctx context.Context, attr Attestation) (*VerificationResult, error) {
	rawURL, err := v.getRawGistURL(attr.URL)
	if err != nil {
		return v.fail(attr, fmt.Sprintf("invalid GitHub Gist URL: %v", err)), nil
	}

	resp, err := v.client.R().SetContext(ctx).Get(rawURL)
	if err != nil {
		return v.fail(attr, fmt.Sprintf("failed to fetch Gist content: %v", err)), nil
	}

	if !resp.IsSuccess() {
		return v.fail(attr, fmt.Sprintf("GitHub returned status %d", resp.StatusCode())), nil
	}

	content := strings.TrimSpace(string(resp.Body()))
	if !strings.Contains(content, attr.Account) {
		return v.fail(attr, "Bobcoin account public key not found in Gist content."), nil
	}

	return &VerificationResult{
		Success:    true,
		Message:    "GitHub identity successfully verified via Gist.",
		VerifiedAt: time.Now().UnixMilli(),
		Kind:       attr.Kind,
		URL:        attr.URL,
		Account:    attr.Account,
	}, nil
}

func (v *GitHubVerifier) getRawGistURL(inputURL string) (string, error) {
	u, err := url.Parse(inputURL)
	if err != nil {
		return "", err
	}

	// If it's already a raw URL or a local test URL, return as is
	if u.Host == "gist.githubusercontent.com" || strings.HasPrefix(u.Host, "127.0.0.1") || strings.HasPrefix(u.Host, "localhost") {
		return inputURL, nil
	}

	// Transform standard gist URL: https://gist.github.com/{user}/{id}
	// to raw URL: https://gist.github.com/{user}/{id}/raw
	if u.Host == "gist.github.com" {
		pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(pathParts) < 2 {
			return "", fmt.Errorf("invalid gist path structure")
		}
		return fmt.Sprintf("https://gist.github.com/%s/%s/raw", pathParts[0], pathParts[1]), nil
	}

	// For any other host, return as is (could be a custom raw URL)
	return inputURL, nil
}

func (v *GitHubVerifier) fail(attr Attestation, msg string) *VerificationResult {
	return &VerificationResult{
		Success:    false,
		Message:    "Verification failed: " + msg,
		VerifiedAt: time.Now().UnixMilli(),
		Kind:       attr.Kind,
		URL:        attr.URL,
		Account:    attr.Account,
	}
}
