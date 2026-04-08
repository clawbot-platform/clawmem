package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"clawmem/internal/domain/scopedmemory"
)

func TestFileStoreScopedRecordCreateListUpdate(t *testing.T) {
	t.Parallel()

	fileStore, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	record := scopedmemory.Record{
		ID:                     "smr-001",
		RepoNamespace:          "ach-trust-lab",
		RunNamespace:           "weekrun-2026-06-demo",
		CycleNamespace:         "day-1",
		AgentNamespace:         "feature-gap",
		MemoryClass:            scopedmemory.MemoryClassUnresolvedGaps,
		Status:                 scopedmemory.StatusOpen,
		ContentText:            "missing sender diversity signal",
		CreatedBy:              "cycle-runner",
		CreatedAt:              time.Date(2026, 4, 4, 14, 0, 0, 0, time.UTC),
		UpdatedAt:              time.Date(2026, 4, 4, 14, 0, 0, 0, time.UTC),
		SourceRunID:            "run-1",
		SourceCycleID:          "cycle-1",
		SourceArtifactID:       "artifact-1",
		SourcePolicyDecisionID: "policy-1",
		SourceModelProfileID:   "ach-default",
	}

	created, err := fileStore.CreateScopedRecord(context.Background(), record)
	if err != nil {
		t.Fatalf("CreateScopedRecord() error = %v", err)
	}
	if created.ID != record.ID {
		t.Fatalf("expected record id %s, got %s", record.ID, created.ID)
	}

	result, err := fileStore.ListScopedRecords(context.Background(), scopedmemory.Query{
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-2026-06-demo",
		MemoryClass:   scopedmemory.MemoryClassUnresolvedGaps,
		Status:        scopedmemory.StatusOpen,
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("ListScopedRecords() error = %v", err)
	}
	if result.Total != 1 || len(result.Records) != 1 {
		t.Fatalf("expected one record, got %#v", result)
	}
	filteredByProvenance, err := fileStore.ListScopedRecords(context.Background(), scopedmemory.Query{
		RepoNamespace:    "ach-trust-lab",
		RunNamespace:     "weekrun-2026-06-demo",
		SourceArtifactID: "artifact-1",
		Limit:            10,
	})
	if err != nil {
		t.Fatalf("ListScopedRecords(provenance) error = %v", err)
	}
	if filteredByProvenance.Total != 1 {
		t.Fatalf("expected one provenance-filtered record, got %#v", filteredByProvenance)
	}

	stored, err := fileStore.GetScopedRecord(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("GetScopedRecord() error = %v", err)
	}
	stored.Status = scopedmemory.StatusResolved
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)
	stored.ResolvedAt = &now
	stored.UpdatedAt = now
	updated, err := fileStore.UpdateScopedRecord(context.Background(), stored)
	if err != nil {
		t.Fatalf("UpdateScopedRecord() error = %v", err)
	}
	if updated.Status != scopedmemory.StatusResolved || updated.ResolvedAt == nil {
		t.Fatalf("expected resolved record, got %#v", updated)
	}
}

func TestFileStoreScopedSnapshotCreateGetList(t *testing.T) {
	t.Parallel()

	fileStore, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	record := scopedmemory.Record{
		ID:            "smr-010",
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-2026-06-demo",
		MemoryClass:   scopedmemory.MemoryClassReviewerNotes,
		Status:        scopedmemory.StatusOpen,
		ContentText:   "review before approval",
		CreatedBy:     "reviewer",
		CreatedAt:     time.Date(2026, 4, 4, 9, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 4, 4, 9, 0, 0, 0, time.UTC),
	}
	if _, err := fileStore.CreateScopedRecord(context.Background(), record); err != nil {
		t.Fatalf("CreateScopedRecord() error = %v", err)
	}

	snapshot := scopedmemory.Snapshot{
		SnapshotID:       "sms-010",
		RepoNamespace:    "ach-trust-lab",
		RunNamespace:     "weekrun-2026-06-demo",
		CreatedAt:        time.Date(2026, 4, 4, 9, 15, 0, 0, time.UTC),
		CreatedBy:        "cycle-runner",
		Summary:          "cycle snapshot",
		RecordRefs:       []string{"smr-010"},
		QueryCriteria:    scopedmemory.Query{RepoNamespace: "ach-trust-lab", RunNamespace: "weekrun-2026-06-demo"},
		ManifestChecksum: "abc123",
	}
	created, err := fileStore.CreateScopedSnapshot(context.Background(), snapshot)
	if err != nil {
		t.Fatalf("CreateScopedSnapshot() error = %v", err)
	}
	if created.SnapshotID != snapshot.SnapshotID {
		t.Fatalf("expected snapshot id %s, got %s", snapshot.SnapshotID, created.SnapshotID)
	}

	got, err := fileStore.GetScopedSnapshot(context.Background(), snapshot.SnapshotID)
	if err != nil {
		t.Fatalf("GetScopedSnapshot() error = %v", err)
	}
	if got.Summary != "cycle snapshot" {
		t.Fatalf("expected snapshot summary, got %#v", got)
	}

	listed, err := fileStore.ListScopedSnapshots(context.Background(), scopedmemory.SnapshotQuery{
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-2026-06-demo",
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("ListScopedSnapshots() error = %v", err)
	}
	if listed.Total != 1 || len(listed.Snapshots) != 1 {
		t.Fatalf("expected one snapshot, got %#v", listed)
	}
}

func TestFileStoreScopedRecordFiltersAndErrorPaths(t *testing.T) {
	t.Parallel()

	fileStore, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	now := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)
	record := scopedmemory.Record{
		ID:                     "smr-filter-001",
		RepoNamespace:          "ach-trust-lab",
		RunNamespace:           "weekrun-governance",
		CycleNamespace:         "day-2",
		AgentNamespace:         "rule-mapping",
		MemoryClass:            scopedmemory.MemoryClassCarryForwardRisks,
		Status:                 scopedmemory.StatusOpen,
		ContentText:            "risk-alpha",
		CreatedBy:              "runner",
		CreatedAt:              now,
		UpdatedAt:              now,
		SourceRunID:            "run-a",
		SourceCycleID:          "cycle-a",
		SourceArtifactID:       "artifact-a",
		SourcePolicyDecisionID: "policy-a",
		SourceModelProfileID:   "granite-a",
	}
	if _, err := fileStore.CreateScopedRecord(context.Background(), record); err != nil {
		t.Fatalf("CreateScopedRecord() error = %v", err)
	}

	other := record
	other.ID = "smr-filter-002"
	other.RepoNamespace = "other-repo"
	other.RunNamespace = "other-run"
	other.CycleNamespace = "day-9"
	other.AgentNamespace = "other-agent"
	other.MemoryClass = scopedmemory.MemoryClassReviewerNotes
	other.Status = scopedmemory.StatusResolved
	resolvedAt := now.Add(time.Minute)
	other.ResolvedAt = &resolvedAt
	other.CreatedAt = resolvedAt
	other.UpdatedAt = resolvedAt
	other.SourceRunID = "run-b"
	other.SourceCycleID = "cycle-b"
	other.SourceArtifactID = "artifact-b"
	other.SourcePolicyDecisionID = "policy-b"
	other.SourceModelProfileID = "granite-b"
	if _, err := fileStore.CreateScopedRecord(context.Background(), other); err != nil {
		t.Fatalf("CreateScopedRecord(other) error = %v", err)
	}

	tests := []struct {
		name  string
		query scopedmemory.Query
		want  int
	}{
		{name: "repo", query: scopedmemory.Query{RepoNamespace: "ach-trust-lab", Limit: 10}, want: 1},
		{name: "run", query: scopedmemory.Query{RunNamespace: "weekrun-governance", Limit: 10}, want: 1},
		{name: "cycle", query: scopedmemory.Query{CycleNamespace: "day-2", Limit: 10}, want: 1},
		{name: "agent", query: scopedmemory.Query{AgentNamespace: "rule-mapping", Limit: 10}, want: 1},
		{name: "class alias", query: scopedmemory.Query{MemoryClass: scopedmemory.MemoryClassCarryForwardRiskAlias, Limit: 10}, want: 1},
		{name: "status", query: scopedmemory.Query{Status: " open ", Limit: 10}, want: 1},
		{name: "source run", query: scopedmemory.Query{SourceRunID: "run-a", Limit: 10}, want: 1},
		{name: "source cycle", query: scopedmemory.Query{SourceCycleID: "cycle-a", Limit: 10}, want: 1},
		{name: "source artifact", query: scopedmemory.Query{SourceArtifactID: "artifact-a", Limit: 10}, want: 1},
		{name: "source policy decision", query: scopedmemory.Query{SourcePolicyDecisionID: "policy-a", Limit: 10}, want: 1},
		{name: "source model profile", query: scopedmemory.Query{SourceModelProfileID: "granite-a", Limit: 10}, want: 1},
		{name: "empty result", query: scopedmemory.Query{SourceArtifactID: "does-not-exist", Limit: 10}, want: 0},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := fileStore.ListScopedRecords(context.Background(), tc.query)
			if err != nil {
				t.Fatalf("ListScopedRecords() error = %v", err)
			}
			if result.Total != tc.want {
				t.Fatalf("ListScopedRecords() total = %d, want %d", result.Total, tc.want)
			}
		})
	}

	if _, err := fileStore.CreateScopedRecord(context.Background(), record); err == nil {
		t.Fatal("expected duplicate create error")
	}
	if _, err := fileStore.GetScopedRecord(context.Background(), "smr-missing-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound from GetScopedRecord, got %v", err)
	}
	if _, err := fileStore.UpdateScopedRecord(context.Background(), scopedmemory.Record{
		ID:            "smr-missing-002",
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-governance",
		MemoryClass:   scopedmemory.MemoryClassBacklogItems,
		Status:        scopedmemory.StatusOpen,
		ContentText:   "backlog",
		CreatedBy:     "runner",
		CreatedAt:     now,
		UpdatedAt:     now,
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound from UpdateScopedRecord, got %v", err)
	}
}

func TestFileStoreScopedMalformedFiles(t *testing.T) {
	t.Parallel()

	fileStore, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	recordPath := filepath.Join(fileStore.scopedRecordsDir, "bad.json")
	if err := os.WriteFile(recordPath, []byte(`{"id":"smr-bad"`), 0o600); err != nil {
		t.Fatalf("WriteFile(record bad json) error = %v", err)
	}
	if _, err := fileStore.ListScopedRecords(context.Background(), scopedmemory.Query{Limit: 10}); err == nil || !strings.Contains(err.Error(), "decode scoped memory record file") {
		t.Fatalf("expected malformed record decode error, got %v", err)
	}

	snapshotPath := filepath.Join(fileStore.scopedSnapsDir, "bad.json")
	if err := os.WriteFile(snapshotPath, []byte(`{"snapshot_id":"sms-bad"`), 0o600); err != nil {
		t.Fatalf("WriteFile(snapshot bad json) error = %v", err)
	}
	if _, err := fileStore.ListScopedSnapshots(context.Background(), scopedmemory.SnapshotQuery{Limit: 10}); err == nil || !strings.Contains(err.Error(), "decode scoped memory snapshot file") {
		t.Fatalf("expected malformed snapshot decode error, got %v", err)
	}
}

func TestFileStoreScopedSnapshotLegacyChecksumAndNotFound(t *testing.T) {
	t.Parallel()

	fileStore, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	legacySnapshotJSON := `{
		"snapshot_id":"sms-legacy-001",
		"repo_namespace":" ach-trust-lab ",
		"run_namespace":" weekrun-legacy ",
		"created_at":"2026-04-05T11:00:00Z",
		"created_by":" ",
		"summary":"legacy checkpoint",
		"record_refs":["smr-2","smr-1","smr-1"],
		"query_criteria":{"RepoNamespace":" ach-trust-lab ","RunNamespace":" weekrun-legacy "}
	}`
	if err := os.WriteFile(filepath.Join(fileStore.scopedSnapsDir, "sms-legacy-001.json"), []byte(legacySnapshotJSON), 0o600); err != nil {
		t.Fatalf("WriteFile(legacy snapshot) error = %v", err)
	}

	got, err := fileStore.GetScopedSnapshot(context.Background(), "sms-legacy-001")
	if err != nil {
		t.Fatalf("GetScopedSnapshot() error = %v", err)
	}
	if got.ManifestChecksum != "legacy-sms-legacy-001" {
		t.Fatalf("expected legacy checksum fallback, got %q", got.ManifestChecksum)
	}
	if got.CreatedBy != "system" {
		t.Fatalf("expected created_by default system, got %q", got.CreatedBy)
	}
	if len(got.RecordRefs) != 2 || got.RecordRefs[0] != "smr-1" || got.RecordRefs[1] != "smr-2" {
		t.Fatalf("expected normalized record refs, got %#v", got.RecordRefs)
	}
	if got.QueryCriteria.RepoNamespace != "ach-trust-lab" || got.QueryCriteria.RunNamespace != "weekrun-legacy" {
		t.Fatalf("expected normalized query criteria, got %#v", got.QueryCriteria)
	}

	listed, err := fileStore.ListScopedSnapshots(context.Background(), scopedmemory.SnapshotQuery{
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-legacy",
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("ListScopedSnapshots() error = %v", err)
	}
	if listed.Total != 1 {
		t.Fatalf("expected one legacy snapshot in list, got %#v", listed)
	}

	if _, err := fileStore.GetScopedSnapshot(context.Background(), "sms-missing-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing snapshot, got %v", err)
	}
}
