package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// /subscribe must reject malformed publisher keys; previously any
// non-empty string (e.g. "zz") was stored as a channel subscription.
func TestHandleSubscribeValidatesPublicKey(t *testing.T) {
	for _, body := range []string{`{"publicKey":"zz"}`, `{"publicKey":"abcd"}`, `{"publicKey":""}`, `{}`} {
		rec := httptest.NewRecorder()
		handleSubscribe(rec, httptest.NewRequest(http.MethodPost, "/subscribe", strings.NewReader(body)))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %s: expected 400, got %d", body, rec.Code)
		}
	}

	valid := `{"publicKey":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}`
	rec := httptest.NewRecorder()
	handleSubscribe(rec, httptest.NewRequest(http.MethodPost, "/subscribe", strings.NewReader(valid)))
	if rec.Code != http.StatusOK {
		t.Fatalf("valid key: expected 200, got %d", rec.Code)
	}
}
