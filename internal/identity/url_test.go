package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestURLVerifier(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}

	pubHex := hex.EncodeToString(pub)
	message := "This is a test message to prove identity"
	sig := ed25519.Sign(priv, []byte(message))
	sigHex := hex.EncodeToString(sig)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"` + message + `","signature":"` + sigHex + `"}`))
	}))
	defer ts.Close()

	verifier := NewURLVerifier()
	isValid, err := verifier.Verify(ts.URL, message, pubHex)

	if err != nil {
		t.Fatalf("Verify failed with error: %v", err)
	}

	if !isValid {
		t.Fatalf("Expected signature to be valid, but was not")
	}
}

func TestURLVerifier_InvalidSignature(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}

	pubHex := hex.EncodeToString(pub)
	message := "This is a test message to prove identity"

	// Create a random signature (invalid)
	_, badPriv, _ := ed25519.GenerateKey(rand.Reader)
	badSig := ed25519.Sign(badPriv, []byte(message))
	badSigHex := hex.EncodeToString(badSig)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"` + message + `","signature":"` + badSigHex + `"}`))
	}))
	defer ts.Close()

	verifier := NewURLVerifier()
	isValid, err := verifier.Verify(ts.URL, message, pubHex)

	if err != nil {
		t.Fatalf("Verify shouldn't have failed with error: %v", err)
	}

	if isValid {
		t.Fatalf("Expected signature to be invalid, but was valid")
	}
}
