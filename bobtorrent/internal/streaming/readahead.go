package streaming

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/bobtorrent/bobtorrent/pkg/storage"
	"github.com/edsrzf/mmap-go"
)

type ReadaheadBuffer struct {
	manifest    *storage.Manifest
	store       BlobFetcher
	current     int
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.Mutex
	position    int64
	cond        *sync.Cond
	chunkStatus []bool  // true if chunk is downloaded and written to mmap
	chunkError  []error // error if chunk fetch failed
	tempFile    *os.File
	mmap        mmap.MMap
	closed      bool
}

type BlobFetcher interface {
	FetchAndDecryptBlob(chunk storage.Chunk) ([]byte, error)
}

func NewReadaheadBuffer(manifest *storage.Manifest, store BlobFetcher) *ReadaheadBuffer {
	ctx, cancel := context.WithCancel(context.Background())

	// Create a temporary file to back the mmap
	f, err := os.CreateTemp("", "bobtorrent-readahead-*")
	if err != nil {
		log.Printf("Failed to create temp file for readahead: %v", err)
		cancel()
		return nil
	}

	// Pre-allocate file size
	if err := f.Truncate(manifest.FileSize); err != nil {
		log.Printf("Failed to truncate temp file: %v", err)
		f.Close()
		os.Remove(f.Name())
		cancel()
		return nil
	}

	m, err := mmap.Map(f, mmap.RDWR, 0)
	if err != nil {
		log.Printf("Failed to mmap temp file: %v", err)
		f.Close()
		os.Remove(f.Name())
		cancel()
		return nil
	}

	rb := &ReadaheadBuffer{
		manifest:    manifest,
		store:       store,
		current:     0,
		ctx:         ctx,
		cancel:      cancel,
		position:    0,
		chunkStatus: make([]bool, len(manifest.Chunks)),
		chunkError:  make([]error, len(manifest.Chunks)),
		tempFile:    f,
		mmap:        m,
	}
	rb.cond = sync.NewCond(&rb.mu)
	return rb
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
			r.mu.Lock()
			if r.chunkStatus[i] {
				r.mu.Unlock()
				continue
			}
			r.mu.Unlock()

			chunk := r.manifest.Chunks[i]
			plaintext, err := r.store.FetchAndDecryptBlob(chunk)
			if err != nil {
				log.Printf("Failed to fetch chunk %d: %v", i, err)
				r.mu.Lock()
				r.chunkError[i] = err
				r.cond.Broadcast()
				r.mu.Unlock()
				return
			}

			r.mu.Lock()
			// Calculate offset for this chunk
			var offset int64 = 0
			for j := 0; j < i; j++ {
				offset += r.manifest.Chunks[j].Size
			}

			// Copy plaintext to mmap
			copy(r.mmap[offset:], plaintext)
			r.chunkStatus[i] = true
			r.cond.Broadcast()
			r.mu.Unlock()
		}
	}
}

func (r *ReadaheadBuffer) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for {
		if r.closed {
			return 0, io.ErrClosedPipe
		}

		if r.position >= r.manifest.FileSize {
			return 0, io.EOF
		}

		// Check if the chunk containing the current position is ready
		var cumulative int64 = 0
		chunkIndex := -1
		for i, chunk := range r.manifest.Chunks {
			if r.position >= cumulative && r.position < cumulative+chunk.Size {
				chunkIndex = i
				break
			}
			cumulative += chunk.Size
		}

		if chunkIndex == -1 {
			return 0, io.EOF
		}

		if r.chunkError[chunkIndex] != nil {
			return 0, fmt.Errorf("chunk %d fetch failed: %w", chunkIndex, r.chunkError[chunkIndex])
		}

		if r.chunkStatus[chunkIndex] {
			// Data is ready, read from mmap
			remainingInFile := r.manifest.FileSize - r.position
			toRead := int64(len(p))
			if toRead > remainingInFile {
				toRead = remainingInFile
			}

			// How much can we read from the currently ready chunks?
			// For simplicity, we just read as much as requested if it's within the file bounds,
			// but we might need to block again if we cross into a non-ready chunk.

			// Let's see how much continuous data is ready from r.position
			readyCumulative := cumulative + r.manifest.Chunks[chunkIndex].Size
			for i := chunkIndex + 1; i < len(r.manifest.Chunks); i++ {
				if r.chunkStatus[i] {
					readyCumulative += r.manifest.Chunks[i].Size
				} else {
					break
				}
			}

			available := readyCumulative - r.position
			if available < toRead {
				toRead = available
			}

			if toRead > 0 {
				n := copy(p, r.mmap[r.position:r.position+toRead])
				r.position += int64(n)
				return n, nil
			}
			// If toRead is 0 because only exactly up to r.position is ready, we wait.
		}

		// Not ready, wait for fetchLoop to signal
		// If we are waiting for a chunk that is not the one currently being fetched by fetchLoop,
		// we might want to restart fetchLoop from here.
		if chunkIndex > r.current {
			r.current = chunkIndex
			go r.fetchLoop(r.current)
		}

		r.cond.Wait()
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

	r.position = newPosition

	// Find which chunk we are in now
	var cumulative int64 = 0
	targetChunkIndex := -1
	for i, chunk := range r.manifest.Chunks {
		if newPosition >= cumulative && newPosition < cumulative+chunk.Size {
			targetChunkIndex = i
			break
		}
		cumulative += chunk.Size
	}

	if targetChunkIndex != -1 && targetChunkIndex != r.current {
		if !r.chunkStatus[targetChunkIndex] {
			r.current = targetChunkIndex
			go r.fetchLoop(r.current)
		}
	}

	return r.position, nil
}

func (r *ReadaheadBuffer) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.closed = true
	r.cancel()
	r.cond.Broadcast() // Wake up any waiting Read calls

	if r.mmap != nil {
		r.mmap.Unmap()
	}
	if r.tempFile != nil {
		r.tempFile.Close()
		os.Remove(r.tempFile.Name())
	}
	return nil
}
