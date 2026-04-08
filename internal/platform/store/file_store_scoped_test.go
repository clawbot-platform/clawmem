package store

import (
	"context"
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
