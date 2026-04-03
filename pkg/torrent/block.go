package torrent

import (
	"encoding/json"
	"fmt"
	"time"
)

// Block represents a block in the Bobcoin asynchronous block lattice.
type Block struct {
	Type          string      `json:"type"`
	Account       string      `json:"account"`
	Previous      *string     `json:"previous"`
	Balance       int64       `json:"balance"`
	StakedBalance int64       `json:"staked_balance"`
	Height        int         `json:"height"`
	Link          string      `json:"link"`
	Spora         interface{} `json:"spora"`
	Payload       interface{} `json:"payload"`
	Timestamp     int64       `json:"timestamp"`
	Hash          string      `json:"hash"`
	Signature     string      `json:"signature"`
}

// NewBlock creates a new block and calculates its hash.
func NewBlock(t, account string, previous *string, balance, staked int64, height int, link string, spora, payload interface{}) *Block {
	b := &Block{
		Type:          t,
		Account:       account,
		Previous:      previous,
		Balance:       balance,
		StakedBalance: staked,
		Height:        height,
		Link:          link,
		Spora:         spora,
		Payload:       payload,
		Timestamp:     time.Now().UnixMilli(),
	}
	b.Hash = b.CalculateHash()
	return b
}

// CalculateHash generates the SHA-256 hash of the block contents.
func (b *Block) CalculateHash() string {
	prev := ""
	if b.Previous != nil {
		prev = *b.Previous
	}

	sporaStr := ""
	if b.Spora != nil {
		data, _ := json.Marshal(b.Spora)
		sporaStr = string(data)
	}

	payloadStr := ""
	if b.Payload != nil {
		data, _ := json.Marshal(b.Payload)
		payloadStr = string(data)
	}

	raw := b.Type + b.Account + prev +
		fmt.Sprintf("%d", b.Balance) +
		fmt.Sprintf("%d", b.StakedBalance) +
		fmt.Sprintf("%d", b.Height) +
		b.Link + sporaStr + payloadStr

	return HashSHA256(raw)
}

// Sign signs the block hash with the provided private key.
func (b *Block) Sign(privateKeyBase58 string) error {
	sig, err := Sign(b.Hash, privateKeyBase58)
	if err != nil {
		return err
	}
	b.Signature = sig
	return nil
}
