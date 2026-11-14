package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The /status endpoint must report the build-time Version (injected via
// -ldflags "-X main.Version=..."), never a hardcoded stale string.
func TestHandleServiceStatusReportsBuildVersion(t *testing.T) {
	old := Version
	Version = "9.9.9-test"
	defer func() { Version = old }()

	rec := httptest.NewRecorder()
	handleServiceStatus(rec, httptest.NewRequest(http.MethodGet, "/status", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["version"] != "9.9.9-test" {
		t.Fatalf("expected version 9.9.9-test, got %#v", body["version"])
	}
}
