package scopedmemory

import (
	"context"
	"fmt"
	"testing"
	"time"

	domain "clawmem/internal/domain/scopedmemory"
	"clawmem/internal/platform/store"
)

func TestPersistNotesAndFetchCompactContext(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	svc := NewService(fileStore)
	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	ids := []string{"smr-100", "smr-101", "smr-102", "smr-103", "smr-104", "smr-105", "smr-106", "smr-107"}
	svc.recordIDGen = func() string {
		id := ids[0]
		ids = ids[1:]
		return id
	}
	svc.snapshotIDGen = func() string { return "sms-100" }

	ns := domain.Namespace{
		RepoNamespace:  "ach-trust-lab",
		RunNamespace:   "weekrun-2026-06-demo",
		CycleNamespace: "day-1",
		AgentNamespace: "daily-summary",
	}
	result, err := svc.PersistNotes(context.Background(), ns, PersistNotesInput{
		CreatedBy:           "week-runner",
		PriorCycleSummaries: []string{"cycle summary day-1"},
		CarryForwardRisks:   []string{"descriptor drift"},
		UnresolvedGaps:      []string{"missing sender diversity"},
		BacklogItems:        []string{"implement sender diversity feature"},
		ReviewerNotes:       []string{"review with fraud ops"},
		Note:                "cycle checkpoint",
		SnapshotSummary:     "day-1 checkpoint",
	})
	if err != nil {
		t.Fatalf("PersistNotes() error = %v", err)
	}
	if result.SnapshotRef != "sms-100" {
		t.Fatalf("expected snapshot ref sms-100, got %s", result.SnapshotRef)
	}
	if result.RecordsWritten < 6 {
		t.Fatalf("expected multiple written records, got %#v", result)
	}

	compact, err := svc.FetchCompactContext(context.Background(), domain.Namespace{
		RepoNamespace:  "ach-trust-lab",
		RunNamespace:   "weekrun-2026-06-demo",
		CycleNamespace: "day-2",
		AgentNamespace: "policy-tuning",
	})
	if err != nil {
		t.Fatalf("FetchCompactContext() error = %v", err)
	}
	if len(compact.PriorCycleSummaries) != 1 || compact.PriorCycleSummaries[0] != "cycle summary day-1" {
		t.Fatalf("unexpected prior cycle summaries %#v", compact.PriorCycleSummaries)
	}
	if len(compact.CarryForwardRisks) != 1 || compact.CarryForwardRisks[0] != "descriptor drift" {
		t.Fatalf("unexpected carry forward risks %#v", compact.CarryForwardRisks)
	}
	if len(compact.UnresolvedGaps) != 1 || compact.UnresolvedGaps[0] != "missing sender diversity" {
		t.Fatalf("unexpected unresolved gaps %#v", compact.UnresolvedGaps)
	}
	if len(compact.BacklogItems) != 1 || compact.BacklogItems[0] != "implement sender diversity feature" {
		t.Fatalf("unexpected backlog items %#v", compact.BacklogItems)
	}
	if len(compact.ReviewerNotes) != 1 || compact.ReviewerNotes[0] != "review with fraud ops" {
		t.Fatalf("unexpected reviewer notes %#v", compact.ReviewerNotes)
	}
}

func TestUnresolvedGapUpsertAndResolve(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	svc := NewService(fileStore)
	now := time.Date(2026, 4, 4, 13, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	idCounter := 0
	svc.recordIDGen = func() string {
		idCounter++
		return "smr-upsert-" + string(rune('0'+idCounter))
	}
	snapshotCounter := 0
	svc.snapshotIDGen = func() string {
		snapshotCounter++
		return fmt.Sprintf("sms-upsert-%d", snapshotCounter)
	}

	ns := domain.Namespace{RepoNamespace: "ach-trust-lab", RunNamespace: "weekrun-upsert", CycleNamespace: "day-3", AgentNamespace: "feature-gap"}
	if _, err := svc.PersistNotes(context.Background(), ns, PersistNotesInput{
		CreatedBy:      "runner",
		UnresolvedGaps: []string{"gap-a"},
	}); err != nil {
		t.Fatalf("PersistNotes(first) error = %v", err)
	}
	if _, err := svc.PersistNotes(context.Background(), ns, PersistNotesInput{
		CreatedBy:      "runner",
		UnresolvedGaps: []string{"gap-a"},
	}); err != nil {
		t.Fatalf("PersistNotes(second) error = %v", err)
	}

	query, err := svc.ListRecords(context.Background(), domain.Query{
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-upsert",
		MemoryClass:   domain.MemoryClassUnresolvedGaps,
		Status:        domain.StatusOpen,
		Limit:         50,
	})
	if err != nil {
		t.Fatalf("ListRecords() error = %v", err)
	}
	if query.Total != 1 {
		t.Fatalf("expected upserted single unresolved gap, got %#v", query)
	}

	if _, err := svc.PersistNotes(context.Background(), ns, PersistNotesInput{
		CreatedBy:             "runner",
		ResolveUnresolvedGaps: []string{"gap-a"},
	}); err != nil {
		t.Fatalf("PersistNotes(resolve) error = %v", err)
	}

	openResult, err := svc.ListRecords(context.Background(), domain.Query{
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-upsert",
		MemoryClass:   domain.MemoryClassUnresolvedGaps,
		Status:        domain.StatusOpen,
		Limit:         50,
	})
	if err != nil {
		t.Fatalf("ListRecords(open) error = %v", err)
	}
	if openResult.Total != 0 {
		t.Fatalf("expected no open unresolved gaps after resolve, got %#v", openResult)
	}
}

func TestExportRunAndSnapshot(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	svc := NewService(fileStore)
	now := time.Date(2026, 4, 4, 16, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	recordIDs := []string{"smr-201", "smr-202", "smr-203", "smr-204"}
	svc.recordIDGen = func() string {
		id := recordIDs[0]
		recordIDs = recordIDs[1:]
		return id
	}
	svc.snapshotIDGen = func() string { return "sms-201" }

	ns := domain.Namespace{RepoNamespace: "ach-trust-lab", RunNamespace: "weekrun-export", CycleNamespace: "day-4", AgentNamespace: "ops-playbooks"}
	persist, err := svc.PersistNotes(context.Background(), ns, PersistNotesInput{
		CreatedBy:           "week-runner",
		BacklogItems:        []string{"publish ops playbook"},
		ReviewerNotes:       []string{"approve for dry run"},
		SnapshotSummary:     "day-4 snapshot",
		SnapshotManifestRef: "artifact://memory/day-4.json",
	})
	if err != nil {
		t.Fatalf("PersistNotes() error = %v", err)
	}
	if persist.SnapshotRef == "" {
		t.Fatalf("expected snapshot ref, got %#v", persist)
	}

	export, err := svc.ExportRun(context.Background(), domain.Namespace{RepoNamespace: "ach-trust-lab", RunNamespace: "weekrun-export"})
	if err != nil {
		t.Fatalf("ExportRun() error = %v", err)
	}
	if export.Manifest["record_count"] == nil {
		t.Fatalf("expected manifest record_count, got %#v", export.Manifest)
	}
	if export.ClassCounts[string(domain.MemoryClassBacklogItems)] == 0 {
		t.Fatalf("expected backlog class count in export, got %#v", export.ClassCounts)
	}

	snapshotExport, err := svc.ExportSnapshot(context.Background(), persist.SnapshotRef)
	if err != nil {
		t.Fatalf("ExportSnapshot() error = %v", err)
	}
	if snapshotExport.Snapshot.SnapshotID != persist.SnapshotRef {
		t.Fatalf("expected snapshot id %s, got %#v", persist.SnapshotRef, snapshotExport.Snapshot)
	}
	if len(snapshotExport.Records) == 0 {
		t.Fatalf("expected snapshot records, got %#v", snapshotExport)
	}
}
