package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"clawmem/internal/domain/scopedmemory"
)

func (s *FileStore) CreateScopedRecord(_ context.Context, record scopedmemory.Record) (scopedmemory.Record, error) {
	record = normalizeScopedRecord(record)
	if err := record.Validate(); err != nil {
		return scopedmemory.Record{}, err
	}
	if err := validateRecordID(record.ID); err != nil {
		return scopedmemory.Record{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.scopedRecordPath(record.ID)
	if _, err := os.Stat(path); err == nil {
		return scopedmemory.Record{}, fmt.Errorf("scoped memory record %q already exists", record.ID)
	} else if err != nil && !os.IsNotExist(err) {
		return scopedmemory.Record{}, fmt.Errorf("stat scoped memory record: %w", err)
	}

	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return scopedmemory.Record{}, fmt.Errorf("marshal scoped memory record: %w", err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return scopedmemory.Record{}, fmt.Errorf("write scoped memory record: %w", err)
	}

	return record, nil
}

func (s *FileStore) UpdateScopedRecord(_ context.Context, record scopedmemory.Record) (scopedmemory.Record, error) {
	record = normalizeScopedRecord(record)
	if err := record.Validate(); err != nil {
		return scopedmemory.Record{}, err
	}
	if err := validateRecordID(record.ID); err != nil {
		return scopedmemory.Record{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.scopedRecordPath(record.ID)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return scopedmemory.Record{}, ErrNotFound
		}
		return scopedmemory.Record{}, fmt.Errorf("stat scoped memory record: %w", err)
	}

	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return scopedmemory.Record{}, fmt.Errorf("marshal scoped memory record: %w", err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return scopedmemory.Record{}, fmt.Errorf("write scoped memory record: %w", err)
	}
	return record, nil
}

func (s *FileStore) GetScopedRecord(_ context.Context, id string) (scopedmemory.Record, error) {
	if err := validateRecordID(id); err != nil {
		return scopedmemory.Record{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	payload, err := os.ReadFile(s.scopedRecordPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return scopedmemory.Record{}, ErrNotFound
		}
		return scopedmemory.Record{}, fmt.Errorf("read scoped memory record: %w", err)
	}

	var record scopedmemory.Record
	if err := json.Unmarshal(payload, &record); err != nil {
		return scopedmemory.Record{}, fmt.Errorf("decode scoped memory record: %w", err)
	}

	return normalizeScopedRecord(record), nil
}

func (s *FileStore) ListScopedRecords(_ context.Context, query scopedmemory.Query) (scopedmemory.QueryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = scopedmemory.NormalizeQuery(query)
	records, err := s.loadAllScopedRecords()
	if err != nil {
		return scopedmemory.QueryResult{}, err
	}

	filtered := make([]scopedmemory.Record, 0, len(records))
	for _, record := range records {
		record = normalizeScopedRecord(record)
		if query.RepoNamespace != "" && record.RepoNamespace != query.RepoNamespace {
			continue
		}
		if query.RunNamespace != "" && record.RunNamespace != query.RunNamespace {
			continue
		}
		if query.CycleNamespace != "" && record.CycleNamespace != query.CycleNamespace {
			continue
		}
		if query.AgentNamespace != "" && record.AgentNamespace != query.AgentNamespace {
			continue
		}
		if query.MemoryClass != "" && scopedmemory.NormalizeClass(record.MemoryClass) != query.MemoryClass {
			continue
		}
		if query.Status != "" && scopedmemory.NormalizeStatus(record.Status) != query.Status {
			continue
		}
		filtered = append(filtered, record)
	}

	slices.SortFunc(filtered, func(a, b scopedmemory.Record) int {
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
	page := append([]scopedmemory.Record(nil), filtered[start:end]...)

	return scopedmemory.QueryResult{
		Records: page,
		Total:   total,
		Limit:   query.Limit,
		Offset:  query.Offset,
		HasMore: end < total,
	}, nil
}

func (s *FileStore) CreateScopedSnapshot(_ context.Context, snapshot scopedmemory.Snapshot) (scopedmemory.Snapshot, error) {
	snapshot = normalizeScopedSnapshot(snapshot)
	if err := snapshot.Validate(); err != nil {
		return scopedmemory.Snapshot{}, err
	}
	if err := validateRecordID(snapshot.SnapshotID); err != nil {
		return scopedmemory.Snapshot{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.scopedSnapshotPath(snapshot.SnapshotID)
	if _, err := os.Stat(path); err == nil {
		return scopedmemory.Snapshot{}, fmt.Errorf("scoped memory snapshot %q already exists", snapshot.SnapshotID)
	} else if err != nil && !os.IsNotExist(err) {
		return scopedmemory.Snapshot{}, fmt.Errorf("stat scoped memory snapshot: %w", err)
	}

	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return scopedmemory.Snapshot{}, fmt.Errorf("marshal scoped memory snapshot: %w", err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return scopedmemory.Snapshot{}, fmt.Errorf("write scoped memory snapshot: %w", err)
	}

	return snapshot, nil
}

func (s *FileStore) GetScopedSnapshot(_ context.Context, snapshotID string) (scopedmemory.Snapshot, error) {
	if err := validateRecordID(snapshotID); err != nil {
		return scopedmemory.Snapshot{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	payload, err := os.ReadFile(s.scopedSnapshotPath(snapshotID))
	if err != nil {
		if os.IsNotExist(err) {
			return scopedmemory.Snapshot{}, ErrNotFound
		}
		return scopedmemory.Snapshot{}, fmt.Errorf("read scoped memory snapshot: %w", err)
	}

	var snapshot scopedmemory.Snapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return scopedmemory.Snapshot{}, fmt.Errorf("decode scoped memory snapshot: %w", err)
	}

	return normalizeScopedSnapshot(snapshot), nil
}

func (s *FileStore) ListScopedSnapshots(_ context.Context, query scopedmemory.SnapshotQuery) (scopedmemory.SnapshotQueryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = scopedmemory.NormalizeSnapshotQuery(query)
	snapshots, err := s.loadAllScopedSnapshots()
	if err != nil {
		return scopedmemory.SnapshotQueryResult{}, err
	}

	filtered := make([]scopedmemory.Snapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		snapshot = normalizeScopedSnapshot(snapshot)
		if query.RepoNamespace != "" && snapshot.RepoNamespace != query.RepoNamespace {
			continue
		}
		if query.RunNamespace != "" && snapshot.RunNamespace != query.RunNamespace {
			continue
		}
		if query.CycleNamespace != "" && snapshot.CycleNamespace != query.CycleNamespace {
			continue
		}
		filtered = append(filtered, snapshot)
	}

	slices.SortFunc(filtered, func(a, b scopedmemory.Snapshot) int {
		if a.CreatedAt.Equal(b.CreatedAt) {
			return strings.Compare(a.SnapshotID, b.SnapshotID)
		}
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		return 1
	})

	total := len(filtered)
	start := min(query.Offset, total)
	end := min(start+query.Limit, total)
	page := append([]scopedmemory.Snapshot(nil), filtered[start:end]...)

	return scopedmemory.SnapshotQueryResult{
		Snapshots: page,
		Total:     total,
		Limit:     query.Limit,
		Offset:    query.Offset,
		HasMore:   end < total,
	}, nil
}

func (s *FileStore) scopedRecordPath(id string) string {
	return filepath.Join(s.scopedRecordsDir, id+".json")
}

func (s *FileStore) scopedSnapshotPath(id string) string {
	return filepath.Join(s.scopedSnapsDir, id+".json")
}

func (s *FileStore) loadAllScopedRecords() ([]scopedmemory.Record, error) {
	entries, err := os.ReadDir(s.scopedRecordsDir)
	if err != nil {
		return nil, fmt.Errorf("read scoped records directory: %w", err)
	}

	records := make([]scopedmemory.Record, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		payload, err := os.ReadFile(filepath.Join(s.scopedRecordsDir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read scoped memory record file: %w", err)
		}
		var record scopedmemory.Record
		if err := json.Unmarshal(payload, &record); err != nil {
			return nil, fmt.Errorf("decode scoped memory record file: %w", err)
		}
		records = append(records, normalizeScopedRecord(record))
	}
	return records, nil
}

func (s *FileStore) loadAllScopedSnapshots() ([]scopedmemory.Snapshot, error) {
	entries, err := os.ReadDir(s.scopedSnapsDir)
	if err != nil {
		return nil, fmt.Errorf("read scoped snapshots directory: %w", err)
	}

	snapshots := make([]scopedmemory.Snapshot, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		payload, err := os.ReadFile(filepath.Join(s.scopedSnapsDir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read scoped memory snapshot file: %w", err)
		}
		var snapshot scopedmemory.Snapshot
		if err := json.Unmarshal(payload, &snapshot); err != nil {
			return nil, fmt.Errorf("decode scoped memory snapshot file: %w", err)
		}
		snapshots = append(snapshots, normalizeScopedSnapshot(snapshot))
	}
	return snapshots, nil
}

func normalizeScopedRecord(record scopedmemory.Record) scopedmemory.Record {
	ns := scopedmemory.NormalizeNamespace(scopedmemory.Namespace{
		RepoNamespace:  record.RepoNamespace,
		RunNamespace:   record.RunNamespace,
		CycleNamespace: record.CycleNamespace,
		AgentNamespace: record.AgentNamespace,
	})
	record.RepoNamespace = ns.RepoNamespace
	record.RunNamespace = ns.RunNamespace
	record.CycleNamespace = ns.CycleNamespace
	record.AgentNamespace = ns.AgentNamespace
	record.MemoryClass = scopedmemory.NormalizeClass(record.MemoryClass)
	if scopedmemory.NormalizeStatus(record.Status) == "" {
		record.Status = scopedmemory.StatusOpen
	} else {
		record.Status = scopedmemory.NormalizeStatus(record.Status)
	}
	if record.MetadataJSON == nil {
		record.MetadataJSON = map[string]any{}
	}
	if record.ContentJSON == nil {
		record.ContentJSON = map[string]any{}
	}
	record.ContentText = strings.TrimSpace(record.ContentText)
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	if record.CreatedBy == "" {
		record.CreatedBy = "system"
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = record.UpdatedAt
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = record.CreatedAt
	}
	return record
}

func normalizeScopedSnapshot(snapshot scopedmemory.Snapshot) scopedmemory.Snapshot {
	ns := scopedmemory.NormalizeNamespace(scopedmemory.Namespace{
		RepoNamespace:  snapshot.RepoNamespace,
		RunNamespace:   snapshot.RunNamespace,
		CycleNamespace: snapshot.CycleNamespace,
	})
	snapshot.RepoNamespace = ns.RepoNamespace
	snapshot.RunNamespace = ns.RunNamespace
	snapshot.CycleNamespace = ns.CycleNamespace
	snapshot.RecordRefs = scopedmemory.CleanTexts(snapshot.RecordRefs)
	snapshot.CreatedBy = strings.TrimSpace(snapshot.CreatedBy)
	if snapshot.CreatedBy == "" {
		snapshot.CreatedBy = "system"
	}
	if snapshot.MetadataJSON == nil {
		snapshot.MetadataJSON = map[string]any{}
	}
	snapshot.QueryCriteria = scopedmemory.NormalizeQuery(snapshot.QueryCriteria)
	return snapshot
}
