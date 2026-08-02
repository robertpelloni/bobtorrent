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

type ORCIDVerifier struct {
	Client  *http.Client
	BaseURL string
}

func NewORCIDVerifier() *ORCIDVerifier {
	return &ORCIDVerifier{
		Client:  &http.Client{Timeout: 10 * time.Second},
		BaseURL: "https://pub.orcid.org/v3.0",
	}
}

// Verify implements the Verifier interface required by VerifierService.
func (v *ORCIDVerifier) Verify(ctx context.Context, attr Attestation) (*VerificationResult, error) {
	// The URL field should contain the ORCID ID
	orcidID := strings.TrimSpace(attr.URL)

	url := fmt.Sprintf("%s/%s", v.BaseURL, orcidID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &VerificationResult{Success: false, Message: fmt.Sprintf("Failed to create ORCID API request: %v", err), VerifiedAt: time.Now().UnixMilli(), Kind: attr.Kind, URL: attr.URL, Account: attr.Account}, nil
	}

	req.Header.Set("Accept", "application/json")

	resp, err := v.Client.Do(req)
	if err != nil {
		return &VerificationResult{Success: false, Message: fmt.Sprintf("Failed to execute ORCID API request: %v", err), VerifiedAt: time.Now().UnixMilli(), Kind: attr.Kind, URL: attr.URL, Account: attr.Account}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return &VerificationResult{Success: false, Message: fmt.Sprintf("ORCID record not found: %s", orcidID), VerifiedAt: time.Now().UnixMilli(), Kind: attr.Kind, URL: attr.URL, Account: attr.Account}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return &VerificationResult{Success: false, Message: fmt.Sprintf("ORCID API returned unexpected status: %d", resp.StatusCode), VerifiedAt: time.Now().UnixMilli(), Kind: attr.Kind, URL: attr.URL, Account: attr.Account}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &VerificationResult{Success: false, Message: fmt.Sprintf("Failed to read ORCID API response: %v", err), VerifiedAt: time.Now().UnixMilli(), Kind: attr.Kind, URL: attr.URL, Account: attr.Account}, nil
	}

	var record struct {
		Person *struct {
			Biography *struct {
				Content string `json:"content"`
			} `json:"biography"`
		} `json:"person"`
	}

	if err := json.Unmarshal(body, &record); err != nil {
		return &VerificationResult{Success: false, Message: fmt.Sprintf("Failed to parse ORCID JSON response: %v", err), VerifiedAt: time.Now().UnixMilli(), Kind: attr.Kind, URL: attr.URL, Account: attr.Account}, nil
	}

	if record.Person == nil || record.Person.Biography == nil || record.Person.Biography.Content == "" {
		return &VerificationResult{Success: false, Message: fmt.Sprintf("No biography found on ORCID record %s", orcidID), VerifiedAt: time.Now().UnixMilli(), Kind: attr.Kind, URL: attr.URL, Account: attr.Account}, nil
	}

	biography := record.Person.Biography.Content
	expectedAttestation := fmt.Sprintf("BOBTORRENT_IDENTITY:%s", attr.Account)

	if strings.Contains(biography, expectedAttestation) {
		return &VerificationResult{
			Success:    true,
			Message:    "Identity successfully verified via ORCID biography.",
			VerifiedAt: time.Now().UnixMilli(),
			Kind:       attr.Kind,
			URL:        attr.URL,
			Account:    attr.Account,
		}, nil
	}

	return &VerificationResult{
		Success:    false,
		Message:    fmt.Sprintf("Verification failed: Biography does not contain expected attestation %s", expectedAttestation),
		VerifiedAt: time.Now().UnixMilli(),
		Kind:       attr.Kind,
		URL:        attr.URL,
		Account:    attr.Account,
	}, nil
}
