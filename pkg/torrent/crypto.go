package torrent

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/mr-tron/base58/base58"
)

// Keypair represents an Ed25519 keypair encoded in Base58.
type Keypair struct {
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

// GenerateKeypair creates a new Ed25519 keypair and encodes it in Base58.
func GenerateKeypair() (*Keypair, error) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}

	return &Keypair{
		PublicKey:  base58.Encode(pub),
		PrivateKey: base58.Encode(priv),
	}, nil
}

// HashSHA256 returns the hex-encoded SHA-256 hash of the input string.
func HashSHA256(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Sign signs the provided message (usually a hash) with the Base58-encoded private key.
func Sign(message string, privateKeyBase58 string) (string, error) {
	privBytes, err := base58.Decode(privateKeyBase58)
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(privBytes) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("invalid private key size: %d", len(privBytes))
	}

	priv := ed25519.PrivateKey(privBytes)
	sig := ed25519.Sign(priv, []byte(message))

	return base58.Encode(sig), nil
}

// Verify checks if the signature is valid for the message and public key.
func Verify(message string, signatureBase58 string, publicKeyBase58 string) bool {
	sigBytes, err := base58.Decode(signatureBase58)
	if err != nil {
		return false
	}

	pubBytes, err := base58.Decode(publicKeyBase58)
	if err != nil {
		return false
	}

	if len(pubBytes) != ed25519.PublicKeySize {
		return false
	}

	pub := ed25519.PublicKey(pubBytes)
	return ed25519.Verify(pub, []byte(message), sigBytes)
}
