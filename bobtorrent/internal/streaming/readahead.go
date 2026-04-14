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
	position  int64
	ready     chan struct{}
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
		ready:    make(chan struct{}, 1),
	}
}

func (r *ReadaheadBuffer) StartPrefetch() {
	log.Println("Starting predictive readahead for manifest:", r.manifest.OriginalFilename)
	go r.fetchLoop(r.current)
}

func (r *ReadaheadBuffer) fetchLoop(startIndex int) {
	for i := startIndex; i < len(r.manifest.Chunks); i++ {
		select {
		case <-r.ctx.Done():
			return
		default:
			chunk := r.manifest.Chunks[i]

			plaintext, err := r.store.FetchAndDecryptBlob(chunk)
			if err != nil {
				log.Printf("Failed to fetch chunk %d: %v", i, err)
				continue
			}

			r.mu.Lock()
			if r.current == i {
				r.buffer = bytes.NewReader(plaintext)

				var cumulative int64 = 0
				for j := 0; j < i; j++ {
				    cumulative += r.manifest.Chunks[j].Size
				}
				chunkOffset := r.position - cumulative
				if chunkOffset > 0 {
				    r.buffer.Seek(chunkOffset, io.SeekStart)
				}

				select {
				case r.ready <- struct{}{}:
				default:
				}
			}
			r.mu.Unlock()
			return
		}
	}
}

func (r *ReadaheadBuffer) Read(p []byte) (int, error) {
    if len(p) == 0 {
        return 0, nil
    }

	for {
		r.mu.Lock()
		buf := r.buffer
		r.mu.Unlock()

		if buf == nil {
		    select {
		    case <-r.ready:
		    case <-r.ctx.Done():
			    return 0, io.EOF
		    }
            continue
        }

		r.mu.Lock()
        if r.buffer == nil {
            r.mu.Unlock()
            continue
        }

		n, err := r.buffer.Read(p)
        if n > 0 {
		    r.position += int64(n)
        }

        if err == io.EOF {
            if r.current >= len(r.manifest.Chunks)-1 {
                r.mu.Unlock()
                return n, io.EOF // Stop timeout loops here
            }

            r.current++
            r.buffer = nil
            r.mu.Unlock()

            go r.fetchLoop(r.current)

            if n > 0 {
                return n, nil
            }
            continue
        }

        r.mu.Unlock()
		return n, err
	}
}

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

	if newPosition == r.manifest.FileSize {
		r.position = newPosition
		r.buffer = bytes.NewReader([]byte{})
		return r.position, nil
	}

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

	if targetChunkIndex == -1 {
		return 0, fmt.Errorf("could not resolve seek to chunk")
	}

	if targetChunkIndex != r.current {
		r.current = targetChunkIndex
		r.position = newPosition
		r.buffer = nil
		go r.fetchLoop(r.current)
	} else if r.buffer != nil {
		r.buffer.Seek(chunkOffset, io.SeekStart)
		r.position = newPosition
	} else {
	    r.position = newPosition
	}

	return r.position, nil
}

func (r *ReadaheadBuffer) Close() error {
	r.cancel()
	return nil
}
