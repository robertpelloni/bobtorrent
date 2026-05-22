package streaming

import (
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
	defer rb.Close()

	// Fast-forward position to EOF (100)
	rb.mu.Lock()
	rb.position = 100
	rb.current = 1
	rb.mu.Unlock()

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

type MockBlobFetcher struct {
	fetchCount int
	data       map[string][]byte
}

func (m *MockBlobFetcher) FetchAndDecryptBlob(chunk storage.Chunk) ([]byte, error) {
	m.fetchCount++
	return m.data[chunk.BlobID], nil
}

func TestReadaheadCaching(t *testing.T) {
	data1 := []byte("first fifty bytes of data for the first chunk.....")
	data2 := []byte("second fifty bytes of data for the second chunk...")
	manifest := &storage.Manifest{
		FileSize: 100,
		Chunks: []storage.Chunk{
			{Size: 50, BlobID: "chunk1"},
			{Size: 50, BlobID: "chunk2"},
		},
	}

	fetcher := &MockBlobFetcher{
		data: map[string][]byte{
			"chunk1": data1,
			"chunk2": data2,
		},
	}

	rb := NewReadaheadBuffer(manifest, fetcher)
	defer rb.Close()

	rb.StartPrefetch()

	// Read first chunk
	p := make([]byte, 50)
	n, err := io.ReadFull(rb, p)
	assert.NoError(t, err)
	assert.Equal(t, 50, n)
	assert.Equal(t, data1, p)

	// Seek back to start
	_, err = rb.Seek(0, io.SeekStart)
	assert.NoError(t, err)

	// Read again
	p2 := make([]byte, 50)
	n, err = io.ReadFull(rb, p2)
	assert.NoError(t, err)
	assert.Equal(t, 50, n)
	assert.Equal(t, data1, p2)

	// Verify fetchCount is only 2 (once for each chunk, prefetch should have got both eventually)
	// Actually prefetch starts from 0, so it should fetch 1 and 2.
	// Read might trigger re-fetch if not careful, but our mmap logic checks chunkStatus.
	assert.LessOrEqual(t, fetcher.fetchCount, 2)
}
