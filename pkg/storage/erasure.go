package storage

import (
	"bytes"
	"fmt"

	"github.com/klauspost/reedsolomon"
)

// ErasureCoder provides encoding and decoding of data using Reed-Solomon erasure coding.
type ErasureCoder struct {
	dataShards   int
	parityShards int
	enc          reedsolomon.Encoder
}

// NewErasureCoder creates a new ErasureCoder with the specified number of data and parity shards.
func NewErasureCoder(data, parity int) (*ErasureCoder, error) {
	enc, err := reedsolomon.New(data, parity)
	if err != nil {
		return nil, fmt.Errorf("failed to create Reed-Solomon encoder: %w", err)
	}
	return &ErasureCoder{
		dataShards:   data,
		parityShards: parity,
		enc:          enc,
	}, nil
}

// Encode splits data into data shards and generates parity shards.
func (ec *ErasureCoder) Encode(data []byte) ([][]byte, error) {
	// Reedsolomon splits into shards of equal size
	shards, err := ec.enc.Split(data)
	if err != nil {
		return nil, fmt.Errorf("failed to split data into shards: %w", err)
	}

	// Generate parity shards
	if err := ec.enc.Encode(shards); err != nil {
		return nil, fmt.Errorf("failed to encode parity shards: %w", err)
	}

	return shards, nil
}

// Reconstruct attempts to reconstruct missing shards.
func (ec *ErasureCoder) Reconstruct(shards [][]byte) error {
	if err := ec.enc.Reconstruct(shards); err != nil {
		return fmt.Errorf("failed to reconstruct shards: %w", err)
	}

	// Verify reconstruction
	ok, err := ec.enc.Verify(shards)
	if err != nil {
		return fmt.Errorf("failed to verify reconstructed shards: %w", err)
	}
	if !ok {
		return fmt.Errorf("shards failed verification after reconstruction")
	}

	return nil
}

// Join merges data shards back into a single byte slice.
func (ec *ErasureCoder) Join(shards [][]byte, totalSize int) ([]byte, error) {
	// Reed-Solomon Join writes into an io.Writer, so we buffer the output.
	var buf bytes.Buffer
	if err := ec.enc.Join(&buf, shards, totalSize); err != nil {
		return nil, fmt.Errorf("failed to join shards: %w", err)
	}
	return buf.Bytes(), nil
}
