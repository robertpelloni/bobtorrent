package streaming

import (
	"bytes"
	"io"
	"testing"

	"github.com/bobtorrent/bobtorrent/pkg/storage"
	"github.com/stretchr/testify/assert"
)

// MockStore implements the interface or structure expected by ReadaheadBuffer
// Since ReadaheadBuffer expects *storage.BlobStore, and it uses FetchAndDecryptBlob
// We can just construct a small functional test by either mocking the store or writing to a temp dir.
// To keep it simple, we'll initialize a real store with mock data.
func TestReadaheadSeekEOF(t *testing.T) {
	manifest := &storage.Manifest{
		FileSize: 100,
		Chunks: []storage.Chunk{
			{Size: 50, BlobID: "chunk1"},
			{Size: 50, BlobID: "chunk2"},
		},
	}

	rb := NewReadaheadBuffer(manifest, nil)

	// Fast-forward position to EOF (100)
	rb.position = 100
	rb.buffer = bytes.NewReader([]byte{})
	rb.current = 1

	// Test Read at EOF
	p := make([]byte, 10)
	n, err := rb.Read(p)

	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err)

	// Test Seek to EOF exactly
	pos, err := rb.Seek(0, io.SeekEnd)
	assert.NoError(t, err)
	assert.Equal(t, int64(100), pos)

	// Test Read after Seek to EOF
	n, err = rb.Read(p)
	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err)

	// Test Seek beyond EOF
	pos, err = rb.Seek(1, io.SeekEnd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "seek position out of bounds")
}
