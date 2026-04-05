package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleFHEOracleRequiresCiphertext(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/fhe-oracle", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	handleFHEOracle(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["success"] != false {
		t.Fatalf("expected success=false, got %#v", body["success"])
	}
	if body["error"] != "Encrypted payload missing" {
		t.Fatalf("expected missing payload error, got %#v", body["error"])
	}
}

func TestHandleFHEOracleReturnsComputedCiphertext(t *testing.T) {
	original := fheOracleRunner
	defer func() { fheOracleRunner = original }()

	fheOracleRunner = func(_ context.Context, cipherText string) (string, error) {
		if cipherText != "cipher_in" {
			t.Fatalf("expected cipher_in, got %q", cipherText)
		}
		return "cipher_out", nil
	}

	req := httptest.NewRequest(http.MethodPost, "/fhe-oracle", strings.NewReader(`{"cipherText":"cipher_in"}`))
	rec := httptest.NewRecorder()

	handleFHEOracle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["success"] != true {
		t.Fatalf("expected success=true, got %#v", body["success"])
	}
	if body["resultCipher"] != "cipher_out" {
		t.Fatalf("expected cipher_out, got %#v", body["resultCipher"])
	}
}

func TestHandleFHEOracleSurfacesComputationFailure(t *testing.T) {
	original := fheOracleRunner
	defer func() { fheOracleRunner = original }()

	fheOracleRunner = func(_ context.Context, cipherText string) (string, error) {
		return "", context.DeadlineExceeded
	}

	req := httptest.NewRequest(http.MethodPost, "/fhe-oracle", strings.NewReader(`{"cipherText":"cipher_in"}`))
	rec := httptest.NewRecorder()

	handleFHEOracle(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["success"] != false {
		t.Fatalf("expected success=false, got %#v", body["success"])
	}
	if body["error"] != "Homomorphic computation failed." {
		t.Fatalf("expected homomorphic failure error, got %#v", body["error"])
	}
}
