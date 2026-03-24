package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"clawmem/internal/domain/memory"
)

type FileStore struct {
	root       string
	recordsDir string
	mu         sync.RWMutex
}

func NewFileStore(root string) (*FileStore, error) {
	cleanRoot := filepath.Clean(root)
	recordsDir := filepath.Join(cleanRoot, "records")
	if err := os.MkdirAll(recordsDir, 0o750); err != nil {
		return nil, fmt.Errorf("create storage directories: %w", err)
	}
	return &FileStore{
		root:       cleanRoot,
		recordsDir: recordsDir,
	}, nil
}

func (s *FileStore) Create(_ context.Context, record memory.MemoryRecord) (memory.MemoryRecord, error) {
	if err := record.Validate(); err != nil {
		return memory.MemoryRecord{}, err
	}
	if err := validateRecordID(record.ID); err != nil {
		return memory.MemoryRecord{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.recordPath(record.ID)
	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return memory.MemoryRecord{}, fmt.Errorf("marshal memory record: %w", err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return memory.MemoryRecord{}, fmt.Errorf("write memory record: %w", err)
	}

	return record, nil
}

func (s *FileStore) List(_ context.Context, query memory.MemoryQuery) (memory.MemoryQueryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records, err := s.loadAll()
	if err != nil {
		return memory.MemoryQueryResult{}, err
	}

	filtered := make([]memory.MemoryRecord, 0, len(records))
	for _, record := range records {
		if query.MemoryType != "" && record.MemoryType != query.MemoryType {
			continue
		}
		if query.ScenarioID != "" && record.ScenarioID != query.ScenarioID {
			continue
		}
		filtered = append(filtered, record)
	}

	slices.SortFunc(filtered, func(a, b memory.MemoryRecord) int {
		if a.CreatedAt.Equal(b.CreatedAt) {
			return strings.Compare(a.ID, b.ID)
		}
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		return 1
	})

	return memory.MemoryQueryResult{
		Records: filtered,
		Total:   len(filtered),
	}, nil
}

func (s *FileStore) Get(_ context.Context, id string) (memory.MemoryRecord, error) {
	if err := validateRecordID(id); err != nil {
		return memory.MemoryRecord{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	payload, err := os.ReadFile(s.recordPath(id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return memory.MemoryRecord{}, ErrNotFound
		}
		return memory.MemoryRecord{}, fmt.Errorf("read memory record: %w", err)
	}

	var record memory.MemoryRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		return memory.MemoryRecord{}, fmt.Errorf("decode memory record: %w", err)
	}

	return record, nil
}

func (s *FileStore) Count(_ context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records, err := s.loadAll()
	if err != nil {
		return 0, err
	}
	return len(records), nil
}

func (s *FileStore) recordPath(id string) string {
	return filepath.Join(s.recordsDir, id+".json")
}

func (s *FileStore) loadAll() ([]memory.MemoryRecord, error) {
	entries, err := os.ReadDir(s.recordsDir)
	if err != nil {
		return nil, fmt.Errorf("read records directory: %w", err)
	}

	records := make([]memory.MemoryRecord, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		payload, err := os.ReadFile(filepath.Join(s.recordsDir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read memory record file: %w", err)
		}
		var record memory.MemoryRecord
		if err := json.Unmarshal(payload, &record); err != nil {
			return nil, fmt.Errorf("decode memory record file: %w", err)
		}
		records = append(records, record)
	}
	return records, nil
}

func validateRecordID(id string) error {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return errors.New("memory record id is required")
	}
	if strings.Contains(trimmed, "..") || strings.ContainsAny(trimmed, `/\`) {
		return errors.New("memory record id contains invalid path characters")
	}
	return nil
}

var ErrNotFound = errors.New("memory record not found")
