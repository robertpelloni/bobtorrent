package identity

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ORCIDVerifier struct {
	Client  *http.Client
	BaseURL string
}

func NewORCIDVerifier() *ORCIDVerifier {
	return &ORCIDVerifier{
		Client:  &http.Client{},
		BaseURL: "https://pub.orcid.org/v3.0",
	}
}

func (v *ORCIDVerifier) Verify(orcidID, expectedAttestation string) (bool, error) {
	orcidID = strings.TrimSpace(orcidID)

	url := fmt.Sprintf("%s/%s", v.BaseURL, orcidID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create ORCID API request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := v.Client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to execute ORCID API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, fmt.Errorf("ORCID record not found: %s", orcidID)
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("ORCID API returned unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read ORCID API response: %w", err)
	}

	var record struct {
		Person *struct {
			Biography *struct {
				Content string `json:"content"`
			} `json:"biography"`
		} `json:"person"`
	}

	if err := json.Unmarshal(body, &record); err != nil {
		return false, fmt.Errorf("failed to parse ORCID JSON response: %w", err)
	}

	if record.Person == nil || record.Person.Biography == nil || record.Person.Biography.Content == "" {
		return false, fmt.Errorf("no biography found on ORCID record %s", orcidID)
	}

	biography := record.Person.Biography.Content

	if strings.Contains(biography, expectedAttestation) {
		return true, nil
	}

	return false, nil
}
