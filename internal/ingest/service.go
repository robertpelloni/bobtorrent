package ingest

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"bobtorrent/pkg/storage"
	"github.com/edsrzf/mmap-go"
)

// IngestionStatus tracks the progress of a file being processed.
type IngestionStatus struct {
	ID        string  `json:"id"`
	Filename  string  `json:"filename"`
	Progress  float64 `json:"progress"`
	Status    string  `json:"status"`
	TotalSize int64   `json:"totalSize"`
	StartedAt int64   `json:"startedAt"`
}

// IngestionService manages the processing of large game assets into encrypted shards.
type IngestionService struct {
	dataDir    string
	storage    *storage.Storage
	mu         sync.RWMutex
	jobs       map[string]*IngestionStatus
	registry   *storage.Registry
}

func NewIngestionService(dataDir string, registry *storage.Registry) (*IngestionService, error) {
	s, err := storage.NewStorage(filepath.Join(dataDir, "blobs"), 4, 2)
	if err != nil {
		return nil, err
	}

	return &IngestionService{
		dataDir:  dataDir,
		storage:  s,
		jobs:     make(map[string]*IngestionStatus),
		registry: registry,
	}, nil
}

// ProcessFile streams a file from disk, encrypts and shards it, then registers it in the asset registry.
func (s *IngestionService) ProcessFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", err
	}

	jobID := fmt.Sprintf("ingest_%d", time.Now().UnixNano())
	job := &IngestionStatus{
		ID:        jobID,
		Filename:  filepath.Base(filePath),
		Status:    "Processing",
		TotalSize: stat.Size(),
		StartedAt: time.Now().Unix(),
	}

	s.mu.Lock()
	s.jobs[jobID] = job
	s.mu.Unlock()

	go s.runJob(job, filePath)

	return jobID, nil
}

// ProcessLargeFile uses mmap to efficiently ingest multi-gigabyte assets.
func (s *IngestionService) ProcessLargeFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", err
	}

	// For very large files, we use the mmap path
	if stat.Size() > 100*1024*1024 { // > 100MB
		jobID := fmt.Sprintf("ingest_mmap_%d", time.Now().UnixNano())
		job := &IngestionStatus{
			ID:        jobID,
			Filename:  filepath.Base(filePath),
			Status:    "Processing (Mmap)",
			TotalSize: stat.Size(),
			StartedAt: time.Now().Unix(),
		}
		s.mu.Lock()
		s.jobs[jobID] = job
		s.mu.Unlock()
		go s.runMmapJob(job, filePath)
		return jobID, nil
	}

	return s.ProcessFile(filePath)
}

func (s *IngestionService) runJob(job *IngestionStatus, inputPath string) {
	file, err := os.Open(inputPath)
	if err != nil {
		s.updateJobStatus(job.ID, "Failed: "+err.Error(), 0)
		return
	}
	defer file.Close()

	entry := &storage.FileEntry{
		Name: job.Filename,
		Size: job.TotalSize,
	}

	buffer := make([]byte, storage.ChunkSize)
	var processed int64

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			chunkData := buffer[:n]
			_, key, nonce, err := s.storage.EncryptChunk(chunkData)
			if err != nil {
				s.updateJobStatus(job.ID, "Encryption Failed", 0)
				return
			}

			chunkMeta := storage.ChunkMeta{
				Key:      hex.EncodeToString(key),
				Nonce:    hex.EncodeToString(nonce),
				RealSize: int64(n),
			}

			processed += int64(n)
			progress := float64(processed) / float64(job.TotalSize)
			s.updateJobStatus(job.ID, "Processing", progress)
			entry.Chunks = append(entry.Chunks, chunkMeta)
		}
		if err == io.EOF { break }
		if err != nil {
			s.updateJobStatus(job.ID, "Read Error", 0)
			return
		}
	}

	if s.registry != nil {
		_ = s.registry.RegisterAsset(job.ID, job.Filename, job.TotalSize, len(entry.Chunks))
	}
	s.updateJobStatus(job.ID, "Completed", 1.0)
}

func (s *IngestionService) runMmapJob(job *IngestionStatus, inputPath string) {
	f, err := os.Open(inputPath)
	if err != nil {
		s.updateJobStatus(job.ID, "Mmap Open Failed", 0)
		return
	}
	defer f.Close()

	m, err := mmap.Map(f, mmap.RDONLY, 0)
	if err != nil {
		s.updateJobStatus(job.ID, "Mmap Failed", 0)
		return
	}
	defer m.Unmap()

	entry := &storage.FileEntry{
		Name: job.Filename,
		Size: job.TotalSize,
	}

	var processed int64
	for processed < job.TotalSize {
		end := processed + storage.ChunkSize
		if end > job.TotalSize { end = job.TotalSize }

		chunkData := m[processed:end]
		_, key, nonce, err := s.storage.EncryptChunk(chunkData)
		if err != nil {
			s.updateJobStatus(job.ID, "Mmap Encryption Failed", 0)
			return
		}

		chunkMeta := storage.ChunkMeta{
			Key:      hex.EncodeToString(key),
			Nonce:    hex.EncodeToString(nonce),
			RealSize: int64(end - processed),
		}

		processed = end
		s.updateJobStatus(job.ID, "Processing (Mmap)", float64(processed)/float64(job.TotalSize))
		entry.Chunks = append(entry.Chunks, chunkMeta)
	}

	if s.registry != nil {
		_ = s.registry.RegisterAsset(job.ID, job.Filename, job.TotalSize, len(entry.Chunks))
	}
	s.updateJobStatus(job.ID, "Completed", 1.0)
}

func (s *IngestionService) updateJobStatus(id, status string, progress float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, ok := s.jobs[id]; ok {
		job.Status = status
		job.Progress = progress
	}
}

func (s *IngestionService) GetStatus(id string) (*IngestionStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	return job, ok
}
