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
	"time"

	"clawmem/internal/domain/memory"
)

type FileStore struct {
	root             string
	recordsDir       string
	scopedRecordsDir string
	scopedSnapsDir   string
	mu               sync.RWMutex
}

func NewFileStore(root string) (*FileStore, error) {
	cleanRoot := filepath.Clean(root)
	recordsDir := filepath.Join(cleanRoot, "records")
	scopedRecordsDir := filepath.Join(cleanRoot, "scoped-records")
	scopedSnapsDir := filepath.Join(cleanRoot, "scoped-snapshots")
	if err := os.MkdirAll(recordsDir, 0o750); err != nil {
		return nil, fmt.Errorf("create storage directories: %w", err)
	}
	if err := os.MkdirAll(scopedRecordsDir, 0o750); err != nil {
		return nil, fmt.Errorf("create scoped record storage directory: %w", err)
	}
	if err := os.MkdirAll(scopedSnapsDir, 0o750); err != nil {
		return nil, fmt.Errorf("create scoped snapshot storage directory: %w", err)
	}
	return &FileStore{
		root:             cleanRoot,
		recordsDir:       recordsDir,
		scopedRecordsDir: scopedRecordsDir,
		scopedSnapsDir:   scopedSnapsDir,
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
	if _, err := os.Stat(path); err == nil {
		return memory.MemoryRecord{}, fmt.Errorf("memory record %q already exists", record.ID)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return memory.MemoryRecord{}, fmt.Errorf("stat memory record: %w", err)
	}

	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return memory.MemoryRecord{}, fmt.Errorf("marshal memory record: %w", err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return memory.MemoryRecord{}, fmt.Errorf("write memory record: %w", err)
	}

	return record, nil
}

func (s *FileStore) Update(_ context.Context, record memory.MemoryRecord) (memory.MemoryRecord, error) {
	if err := record.Validate(); err != nil {
		return memory.MemoryRecord{}, err
	}
	if err := validateRecordID(record.ID); err != nil {
		return memory.MemoryRecord{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.recordPath(record.ID)
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return memory.MemoryRecord{}, ErrNotFound
		}
		return memory.MemoryRecord{}, fmt.Errorf("stat memory record: %w", err)
	}

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

	query = memory.NormalizeQuery(query)
	records, err := s.loadAll()
	if err != nil {
		return memory.MemoryQueryResult{}, err
	}

	filtered := make([]memory.MemoryRecord, 0, len(records))
	for _, record := range records {
		if query.Namespace != "" && record.Namespace != query.Namespace {
			continue
		}
		if query.ProjectID != "" && record.ProjectID != query.ProjectID {
			continue
		}
		if query.Environment != "" && record.Environment != query.Environment {
			continue
		}
		if query.ClawbotID != "" && record.ClawbotID != query.ClawbotID {
			continue
		}
		if query.SessionID != "" && record.SessionID != query.SessionID {
			continue
		}
		if query.MemoryType != "" && record.MemoryType != query.MemoryType {
			continue
		}
		if query.ScenarioID != "" && record.ScenarioID != query.ScenarioID {
			continue
		}
		if query.SourceRef != "" && record.SourceRef != query.SourceRef {
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

	total := len(filtered)
	start := min(query.Offset, total)
	end := min(start+query.Limit, total)
	page := append([]memory.MemoryRecord(nil), filtered[start:end]...)

	return memory.MemoryQueryResult{
		Records: page,
		Total:   total,
		Limit:   query.Limit,
		Offset:  query.Offset,
		HasMore: end < total,
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

func (s *FileStore) Delete(_ context.Context, id string) error {
	if err := validateRecordID(id); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.Remove(s.recordPath(id)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNotFound
		}
		return fmt.Errorf("delete memory record: %w", err)
	}
	return nil
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

func (s *FileStore) ListAll(_ context.Context) ([]memory.MemoryRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.loadAll()
}

func (s *FileStore) FindByIdempotency(_ context.Context, key string) (memory.MemoryRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records, err := s.loadAll()
	if err != nil {
		return memory.MemoryRecord{}, err
	}
	for _, record := range records {
		if strings.TrimSpace(record.IdempotencyKey) == strings.TrimSpace(key) && strings.TrimSpace(key) != "" {
			return record, nil
		}
	}
	return memory.MemoryRecord{}, ErrNotFound
}

func (s *FileStore) Summary(_ context.Context) (memory.Summary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records, err := s.loadAll()
	if err != nil {
		return memory.Summary{}, err
	}

	summary := memory.Summary{
		RecordsByType:    map[string]int{},
		RecordsByProject: map[string]int{},
	}
	now := time.Now().UTC()
	for _, record := range records {
		summary.TotalRecords++
		if record.Pinned {
			summary.PinnedRecords++
		}
		if record.ExpiresAt != nil {
			summary.ExpiringRecords++
		}
		if record.ReplayLinked || record.MemoryType == memory.MemoryTypeReplayCase {
			summary.ReplayLinked++
		}
		if memory.IsDecayEligible(record, now) {
			summary.DecayEligible++
		}
		summary.RecordsByType[string(record.MemoryType)]++
		summary.RecordsByProject[record.ProjectID]++
		if summary.LastActivityAt == nil || record.UpdatedAt.After(*summary.LastActivityAt) {
			last := record.UpdatedAt
			summary.LastActivityAt = &last
		}
	}
	return summary, nil
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
		record = normalizeLegacyRecord(record)
		records = append(records, record)
	}
	return records, nil
}

func normalizeLegacyRecord(record memory.MemoryRecord) memory.MemoryRecord {
	if strings.TrimSpace(record.ProjectID) == "" {
		record.ProjectID = memory.DefaultProjectID
	}
	if strings.TrimSpace(record.Environment) == "" {
		record.Environment = memory.DefaultEnvironment
	}
	if strings.TrimSpace(record.ClawbotID) == "" {
		record.ClawbotID = memory.DefaultClawbotID
	}
	if strings.TrimSpace(record.SourceRef) == "" {
		record.SourceRef = strings.TrimSpace(record.SourceID)
	}
	if strings.TrimSpace(record.SourceID) == "" {
		record.SourceID = strings.TrimSpace(record.SourceRef)
	}
	if record.Importance == 0 {
		record.Importance = memory.DefaultImportance
	}
	if record.ReplayLinked || record.MemoryType == memory.MemoryTypeReplayCase || record.RetentionPolicy == memory.RetentionPolicyReplayPreserve {
		record.ReplayLinked = true
	}
	if record.RetentionPolicy == "" {
		if record.MemoryType == memory.MemoryTypeReplayCase {
			record.RetentionPolicy = memory.RetentionPolicyReplayPreserve
		} else {
			record.RetentionPolicy = memory.RetentionPolicyStandard
		}
	}
	if strings.TrimSpace(record.Namespace) == "" {
		record.Namespace = memory.BuildNamespace(record.ProjectID, record.Environment, record.ClawbotID, record.SessionID, record.MemoryType)
	}
	if record.StabilityScore == 0 {
		record.StabilityScore = memory.ComputeStability(record)
	}
	record.Tags = memory.CleanTags(record.Tags)
	if record.Metadata == nil {
		record.Metadata = map[string]any{}
	}
	return record
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

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
