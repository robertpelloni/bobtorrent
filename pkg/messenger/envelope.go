package messenger

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

type PayloadType string

const (
	TypeChat       PayloadType = "CHAT"
	TypeMarket     PayloadType = "MARKET"
	TypeBlockchain PayloadType = "BLOCKCHAIN"
	TypeSignal     PayloadType = "SIGNAL"
)

// Envelope is the standard metadata-blinded container for all mesh messages.
type Envelope struct {
	ID            string      `json:"id"`            // Unique hash of the payload
	SenderPubkey  string      `json:"sender_pubkey"` // Hex-encoded public key
	Timestamp     int64       `json:"timestamp"`     // Unix timestamp
	Signature     string      `json:"signature"`     // Hex-encoded signature
	PayloadType   PayloadType `json:"payload_type"`
	EncryptedBody []byte      `json:"encrypted_body"` // Blinded payload
}

// NewEnvelope creates and signs a new message container.
func NewEnvelope(senderPub string, privKey ed25519.PrivateKey, pType PayloadType, body []byte) (*Envelope, error) {
	ts := time.Now().Unix()

	// Pre-calculate ID based on body + timestamp
	h := sha256.New()
	h.Write(body)
	h.Write([]byte(fmt.Sprintf("%d", ts)))
	id := fmt.Sprintf("%x", h.Sum(nil))

	// Data to sign: ID|sender|timestamp|type
	toSign := fmt.Sprintf("%s|%s|%d|%s", id, senderPub, ts, pType)
	sig := ed25519.Sign(privKey, []byte(toSign))

	return &Envelope{
		ID:            id,
		SenderPubkey:  senderPub,
		Timestamp:     ts,
		Signature:     hex.EncodeToString(sig),
		PayloadType:   pType,
		EncryptedBody: body,
	}, nil
}

// Verify checks the cryptographic integrity of the envelope.
func (e *Envelope) Verify() bool {
	pubBytes, err := hex.DecodeString(e.SenderPubkey)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return false
	}

	sigBytes, err := hex.DecodeString(e.Signature)
	if err != nil {
		return false
	}

	// Reconstruct the message that was signed
	toSign := fmt.Sprintf("%s|%s|%d|%s", e.ID, e.SenderPubkey, e.Timestamp, e.PayloadType)

	return ed25519.Verify(ed25519.PublicKey(pubBytes), []byte(toSign), sigBytes)
}

// Marshal encodes the envelope to JSON for transport.
func (e *Envelope) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalEnvelope decodes raw bytes into an Envelope.
func UnmarshalEnvelope(data []byte) (*Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return &env, nil
}
