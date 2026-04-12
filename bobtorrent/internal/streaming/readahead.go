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
}

func NewReadaheadBuffer(manifest *storage.Manifest, store *storage.BlobStore) *ReadaheadBuffer {
	ctx, cancel := context.WithCancel(context.Background())
	return &ReadaheadBuffer{
		manifest: manifest,
		store:    store,
		current:  0,
		ctx:      ctx,
		cancel:   cancel,
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

			// Actively fetch and decrypt using the storage package
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
	if err == io.EOF {
		r.current++
		r.buffer = nil // Require next fetch to populate
	}
	return n, err
}

func (r *ReadaheadBuffer) Close() error {
	r.cancel()
	return nil
}
