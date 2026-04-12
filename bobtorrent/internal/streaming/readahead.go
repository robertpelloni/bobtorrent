package streaming

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/bobtorrent/bobtorrent/pkg/storage"
)

type ReadaheadBuffer struct {
	manifest  *storage.Manifest
	store     *storage.BlobStore
	current   int
	buffer    *bytes.Reader
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	position  int64 // Track absolute offset in the virtual file
}

func NewReadaheadBuffer(manifest *storage.Manifest, store *storage.BlobStore) *ReadaheadBuffer {
	ctx, cancel := context.WithCancel(context.Background())
	return &ReadaheadBuffer{
		manifest: manifest,
		store:    store,
		current:  0,
		ctx:      ctx,
		cancel:   cancel,
		position: 0,
	}
}

func (r *ReadaheadBuffer) StartPrefetch() {
	log.Println("Starting predictive readahead for manifest:", r.manifest.OriginalFilename)
	go r.fetchLoop()
}

func (r *ReadaheadBuffer) fetchLoop() {
	for i := r.current; i < len(r.manifest.Chunks); i++ {
		select {
		case <-r.ctx.Done():
			return
		default:
			chunk := r.manifest.Chunks[i]
			log.Printf("Prefetching chunk %d (BlobID: %s)", chunk.Order, chunk.BlobID)

			plaintext, err := r.store.FetchAndDecryptBlob(chunk)
			if err != nil {
				log.Printf("Failed to fetch chunk %d: %v", i, err)
				continue
			}

			r.mu.Lock()
			if r.current == i {
				r.buffer = bytes.NewReader(plaintext)
			}
			r.mu.Unlock()
		}
	}
}

func (r *ReadaheadBuffer) Read(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.buffer == nil {
		return 0, fmt.Errorf("buffer not ready (waiting for fetch)")
	}

	n, err = r.buffer.Read(p)
	if n > 0 {
		r.position += int64(n)
	}

	if err == io.EOF {
		r.current++
		r.buffer = nil
	}
	return n, err
}

// Seek implements io.Seeker to support HTTP Range requests for video players
func (r *ReadaheadBuffer) Seek(offset int64, whence int) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var newPosition int64
	switch whence {
	case io.SeekStart:
		newPosition = offset
	case io.SeekCurrent:
		newPosition = r.position + offset
	case io.SeekEnd:
		newPosition = r.manifest.FileSize + offset
	default:
		return 0, fmt.Errorf("invalid seek whence")
	}

	if newPosition < 0 || newPosition > r.manifest.FileSize {
		return 0, fmt.Errorf("seek position out of bounds")
	}

	// Calculate which chunk the absolute byte offset falls into
	var cumulative int64 = 0
	targetChunkIndex := -1
	var chunkOffset int64 = 0

	for i, chunk := range r.manifest.Chunks {
		if newPosition >= cumulative && newPosition < cumulative+chunk.Size {
			targetChunkIndex = i
			chunkOffset = newPosition - cumulative
			break
		}
		cumulative += chunk.Size
	}

	// If seeking beyond the last chunk boundary but exactly to FileSize
	if newPosition == r.manifest.FileSize {
		r.position = newPosition
		r.buffer = bytes.NewReader([]byte{})
		return r.position, nil
	}

	if targetChunkIndex == -1 {
		return 0, fmt.Errorf("could not resolve seek to chunk")
	}

	// If we are jumping to a new chunk, force a re-fetch
	if targetChunkIndex != r.current {
		log.Printf("Seeking to chunk %d, byte offset %d", targetChunkIndex, chunkOffset)
		r.current = targetChunkIndex
		r.buffer = nil // Buffer will be populated by the fetch loop
		// We would ideally restart the fetchLoop here to prioritize the new index.
	} else if r.buffer != nil {
		// We are in the same chunk, just adjust the internal reader
		r.buffer.Seek(chunkOffset, io.SeekStart)
	}

	r.position = newPosition
	return r.position, nil
}

func (r *ReadaheadBuffer) Close() error {
	r.cancel()
	return nil
}
