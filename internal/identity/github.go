package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type GitHubVerifier struct {
	Client  *http.Client
	BaseURL string
}

func NewGitHubVerifier() *GitHubVerifier {
	return &GitHubVerifier{
		Client:  &http.Client{Timeout: 10 * time.Second},
		BaseURL: "https://api.github.com",
	}
}

// Verify implements the Verifier interface required by VerifierService.
func (v *GitHubVerifier) Verify(ctx context.Context, attr Attestation) (*VerificationResult, error) {
	// Extract the gist ID from the URL. Example: https://gist.github.com/username/abcdef123456
	parts := strings.Split(attr.URL, "/")
	if len(parts) < 1 {
		return &VerificationResult{Success: false, Message: "Invalid Gist URL format.", VerifiedAt: time.Now().UnixMilli(), Kind: attr.Kind, URL: attr.URL, Account: attr.Account}, nil
	}
	gistID := parts[len(parts)-1]

	url := fmt.Sprintf("%s/gists/%s", v.BaseURL, gistID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &VerificationResult{Success: false, Message: fmt.Sprintf("Failed to create GitHub API request: %v", err), VerifiedAt: time.Now().UnixMilli(), Kind: attr.Kind, URL: attr.URL, Account: attr.Account}, nil
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := v.Client.Do(req)
	if err != nil {
		return &VerificationResult{Success: false, Message: fmt.Sprintf("Failed to execute GitHub API request: %v", err), VerifiedAt: time.Now().UnixMilli(), Kind: attr.Kind, URL: attr.URL, Account: attr.Account}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return &VerificationResult{Success: false, Message: fmt.Sprintf("Gist not found: %s", gistID), VerifiedAt: time.Now().UnixMilli(), Kind: attr.Kind, URL: attr.URL, Account: attr.Account}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return &VerificationResult{Success: false, Message: fmt.Sprintf("GitHub API returned unexpected status: %d", resp.StatusCode), VerifiedAt: time.Now().UnixMilli(), Kind: attr.Kind, URL: attr.URL, Account: attr.Account}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &VerificationResult{Success: false, Message: fmt.Sprintf("Failed to read GitHub API response: %v", err), VerifiedAt: time.Now().UnixMilli(), Kind: attr.Kind, URL: attr.URL, Account: attr.Account}, nil
	}

	var gist struct {
		Files map[string]struct {
			Content string `json:"content"`
		} `json:"files"`
	}

	if err := json.Unmarshal(body, &gist); err != nil {
		return &VerificationResult{Success: false, Message: fmt.Sprintf("Failed to parse GitHub JSON response: %v", err), VerifiedAt: time.Now().UnixMilli(), Kind: attr.Kind, URL: attr.URL, Account: attr.Account}, nil
	}

	expectedAttestation := fmt.Sprintf("BOBTORRENT_IDENTITY:%s", attr.Account)

	for _, file := range gist.Files {
		if strings.Contains(file.Content, expectedAttestation) {
			return &VerificationResult{
				Success:    true,
				Message:    "Identity successfully verified via GitHub Gist.",
				VerifiedAt: time.Now().UnixMilli(),
				Kind:       attr.Kind,
				URL:        attr.URL,
				Account:    attr.Account,
			}, nil
		}
	}

	return &VerificationResult{
		Success:    false,
		Message:    fmt.Sprintf("Verification failed: No file in Gist contains the expected attestation %s", expectedAttestation),
		VerifiedAt: time.Now().UnixMilli(),
		Kind:       attr.Kind,
		URL:        attr.URL,
		Account:    attr.Account,
	}, nil
}
