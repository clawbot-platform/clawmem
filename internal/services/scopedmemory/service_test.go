package scopedmemory

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	domain "clawmem/internal/domain/scopedmemory"
	"clawmem/internal/platform/store"
)

func TestPersistNotesStoresProvenanceAndSupportsNamespaceSafeQuery(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	svc := NewService(fileStore)
	svc.now = func() time.Time { return time.Date(2026, 4, 8, 9, 0, 0, 0, time.UTC) }
	svc.recordIDGen = sequenceID("smr-prov")
	svc.snapshotIDGen = sequenceID("sms-prov")

	ns := domain.Namespace{
		RepoNamespace:  "ach-trust-lab",
		RunNamespace:   "weekrun-2026-06-governance",
		CycleNamespace: "day-1",
		AgentNamespace: "feature-gap",
	}
	if _, err := svc.PersistNotes(context.Background(), ns, PersistNotesInput{
		CreatedBy:              "week-runner",
		UnresolvedGaps:         []string{"missing sender diversity"},
		SourceRunID:            "run-123",
		SourceCycleID:          "cycle-001",
		SourceArtifactID:       "artifact-abc",
		SourcePolicyDecisionID: "policy-55",
		SourceModelProfileID:   "ach-default",
		SnapshotSummary:        "day-1 snapshot",
	}); err != nil {
		t.Fatalf("PersistNotes() error = %v", err)
	}

	query, err := svc.ListRecords(context.Background(), domain.Query{
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-2026-06-governance",
		MemoryClass:   domain.MemoryClassUnresolvedGaps,
		Limit:         20,
	})
	if err != nil {
		t.Fatalf("ListRecords() error = %v", err)
	}
	if query.Total != 1 || len(query.Records) != 1 {
		t.Fatalf("expected one unresolved gap record, got %#v", query)
	}
	record := query.Records[0]
	if record.SourceRunID != "run-123" || record.SourceCycleID != "cycle-001" || record.SourceArtifactID != "artifact-abc" || record.SourcePolicyDecisionID != "policy-55" || record.SourceModelProfileID != "ach-default" {
		t.Fatalf("expected provenance fields on record, got %#v", record)
	}

	provenanceQuery, err := svc.ListRecords(context.Background(), domain.Query{
		RepoNamespace:    "ach-trust-lab",
		RunNamespace:     "weekrun-2026-06-governance",
		SourceArtifactID: "artifact-abc",
		Limit:            20,
	})
	if err != nil {
		t.Fatalf("ListRecords(provenance) error = %v", err)
	}
	if provenanceQuery.Total == 0 {
		t.Fatalf("expected records for artifact provenance query, got %#v", provenanceQuery)
	}

	isolated, err := svc.ListRecords(context.Background(), domain.Query{
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-other",
		MemoryClass:   domain.MemoryClassUnresolvedGaps,
		Limit:         20,
	})
	if err != nil {
		t.Fatalf("ListRecords(namespace-safe) error = %v", err)
	}
	if isolated.Total != 0 {
		t.Fatalf("expected namespace-safe isolation, got %#v", isolated)
	}
}

func TestActionableStatusTransitions(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	svc := NewService(fileStore)
	now := time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	svc.recordIDGen = sequenceID("smr-status")
	svc.snapshotIDGen = sequenceID("sms-status")

	ns := domain.Namespace{RepoNamespace: "ach-trust-lab", RunNamespace: "weekrun-status", CycleNamespace: "day-2", AgentNamespace: "reviewer"}
	if _, err := svc.PersistNotes(context.Background(), ns, PersistNotesInput{
		CreatedBy:      "reviewer",
		ReviewerNotes:  []string{"note-a"},
		UnresolvedGaps: []string{"gap-a"},
		Note:           "working-context-entry",
	}); err != nil {
		t.Fatalf("PersistNotes() error = %v", err)
	}

	gaps, err := svc.ListRecords(context.Background(), domain.Query{
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-status",
		MemoryClass:   domain.MemoryClassUnresolvedGaps,
		Status:        domain.StatusOpen,
		Limit:         20,
	})
	if err != nil {
		t.Fatalf("ListRecords() error = %v", err)
	}
	if gaps.Total != 1 {
		t.Fatalf("expected one open gap, got %#v", gaps)
	}

	updated, err := svc.UpdateRecordStatus(context.Background(), gaps.Records[0].ID, UpdateRecordStatusInput{
		Status:    domain.StatusResolved,
		UpdatedBy: "reviewer",
		Reason:    "validated by reviewer",
	})
	if err != nil {
		t.Fatalf("UpdateRecordStatus(resolve) error = %v", err)
	}
	if updated.Status != domain.StatusResolved || updated.ResolvedAt == nil {
		t.Fatalf("expected resolved record, got %#v", updated)
	}

	if _, err := svc.UpdateRecordStatus(context.Background(), updated.ID, UpdateRecordStatusInput{Status: domain.StatusOpen}); err == nil {
		t.Fatal("expected invalid transition from resolved to open")
	}

	working, err := svc.ListRecords(context.Background(), domain.Query{
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-status",
		MemoryClass:   domain.MemoryClassWorkingContext,
		Limit:         20,
	})
	if err != nil {
		t.Fatalf("ListRecords(working_context) error = %v", err)
	}
	if len(working.Records) == 0 {
		t.Fatalf("expected working_context record, got %#v", working)
	}
	if _, err := svc.UpdateRecordStatus(context.Background(), working.Records[0].ID, UpdateRecordStatusInput{Status: domain.StatusResolved}); err == nil {
		t.Fatal("expected actionable-class status transition guard")
	}
}

func TestCompactContextAssemblyOrderingAndCaps(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	svc := NewService(fileStore)
	current := time.Date(2026, 4, 8, 11, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return current }
	svc.recordIDGen = sequenceID("smr-compact")
	svc.snapshotIDGen = sequenceID("sms-compact")

	ns := domain.Namespace{RepoNamespace: "ach-trust-lab", RunNamespace: "weekrun-compact", CycleNamespace: "day-1", AgentNamespace: "daily-summary"}
	for _, risk := range []string{"risk-a", "risk-b", "risk-c", "risk-d", "risk-e"} {
		if _, err := svc.PersistNotes(context.Background(), ns, PersistNotesInput{CreatedBy: "runner", CarryForwardRisks: []string{risk}}); err != nil {
			t.Fatalf("PersistNotes(%s) error = %v", risk, err)
		}
		current = current.Add(time.Minute)
	}

	risks, err := svc.ListRecords(context.Background(), domain.Query{
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-compact",
		MemoryClass:   domain.MemoryClassCarryForwardRisks,
		Status:        domain.StatusOpen,
		Limit:         20,
	})
	if err != nil {
		t.Fatalf("ListRecords(risks) error = %v", err)
	}
	riskE := findRecordByText(risks.Records, "risk-e")
	if riskE.ID == "" {
		t.Fatalf("expected to find risk-e in %#v", risks)
	}
	if _, err := svc.UpdateRecordStatus(context.Background(), riskE.ID, UpdateRecordStatusInput{Status: domain.StatusResolved, UpdatedBy: "runner", Reason: "closed for cycle"}); err != nil {
		t.Fatalf("UpdateRecordStatus(risk-e) error = %v", err)
	}

	for index := 1; index <= 9; index++ {
		nsSummary := domain.Namespace{RepoNamespace: "ach-trust-lab", RunNamespace: "weekrun-compact", CycleNamespace: fmt.Sprintf("day-%d", index), AgentNamespace: "daily-summary"}
		if _, err := svc.PersistNotes(context.Background(), nsSummary, PersistNotesInput{CreatedBy: "runner", PriorCycleSummaries: []string{fmt.Sprintf("summary-%d", index)}}); err != nil {
			t.Fatalf("PersistNotes(summary-%d) error = %v", index, err)
		}
		current = current.Add(time.Minute)
	}

	compact, err := svc.FetchCompactContext(context.Background(), domain.Namespace{
		RepoNamespace:  "ach-trust-lab",
		RunNamespace:   "weekrun-compact",
		CycleNamespace: "day-10",
		AgentNamespace: "policy-tuning",
	})
	if err != nil {
		t.Fatalf("FetchCompactContext() error = %v", err)
	}

	expectedRiskOrder := []string{"risk-d", "risk-c", "risk-b", "risk-a", "risk-e"}
	if len(compact.CarryForwardRisks) != len(expectedRiskOrder) {
		t.Fatalf("expected %d risk items, got %#v", len(expectedRiskOrder), compact.CarryForwardRisks)
	}
	for index, expected := range expectedRiskOrder {
		if compact.CarryForwardRisks[index] != expected {
			t.Fatalf("expected risk order %v, got %v", expectedRiskOrder, compact.CarryForwardRisks)
		}
	}

	if len(compact.PriorCycleSummaries) != 7 {
		t.Fatalf("expected prior summary cap of 7, got %#v", compact.PriorCycleSummaries)
	}
	expectedSummaryHead := []string{"summary-9", "summary-8", "summary-7"}
	for index, expected := range expectedSummaryHead {
		if compact.PriorCycleSummaries[index] != expected {
			t.Fatalf("expected newest-first summaries starting with %v, got %v", expectedSummaryHead, compact.PriorCycleSummaries)
		}
	}
}

func TestSnapshotManifestChecksumsAndExport(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	svc := NewService(fileStore)
	now := time.Date(2026, 4, 8, 15, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	svc.recordIDGen = sequenceID("smr-snap")
	svc.snapshotIDGen = sequenceID("sms-snap")

	ns := domain.Namespace{RepoNamespace: "ach-trust-lab", RunNamespace: "weekrun-snapshot", CycleNamespace: "day-5", AgentNamespace: "ops-playbooks"}
	first, err := svc.PersistNotes(context.Background(), ns, PersistNotesInput{
		CreatedBy:       "runner",
		BacklogItems:    []string{"publish playbook"},
		SnapshotSummary: "snapshot-one",
	})
	if err != nil {
		t.Fatalf("PersistNotes(first) error = %v", err)
	}
	now = now.Add(2 * time.Minute)
	second, err := svc.PersistNotes(context.Background(), ns, PersistNotesInput{
		CreatedBy:       "runner",
		ReviewerNotes:   []string{"approve with edits"},
		SnapshotSummary: "snapshot-two",
	})
	if err != nil {
		t.Fatalf("PersistNotes(second) error = %v", err)
	}

	hexPattern := regexp.MustCompile(`^[a-f0-9]{64}$`)
	if !hexPattern.MatchString(first.Snapshot.ManifestChecksum) {
		t.Fatalf("expected sha256 manifest checksum, got %q", first.Snapshot.ManifestChecksum)
	}
	if !hexPattern.MatchString(second.Snapshot.ManifestChecksum) {
		t.Fatalf("expected sha256 manifest checksum, got %q", second.Snapshot.ManifestChecksum)
	}
	if second.Snapshot.PreviousSnapshotChecksum != first.Snapshot.ManifestChecksum {
		t.Fatalf("expected checksum chaining, got prev=%q first=%q", second.Snapshot.PreviousSnapshotChecksum, first.Snapshot.ManifestChecksum)
	}

	snapshotExport, err := svc.ExportSnapshot(context.Background(), second.SnapshotRef)
	if err != nil {
		t.Fatalf("ExportSnapshot() error = %v", err)
	}
	if snapshotExport.Snapshot.ManifestChecksum == "" {
		t.Fatalf("expected checksum in snapshot export, got %#v", snapshotExport.Snapshot)
	}
	if len(snapshotExport.Records) == 0 {
		t.Fatalf("expected snapshot export records, got %#v", snapshotExport)
	}

	runExport, err := svc.ExportRun(context.Background(), domain.Namespace{RepoNamespace: "ach-trust-lab", RunNamespace: "weekrun-snapshot"})
	if err != nil {
		t.Fatalf("ExportRun() error = %v", err)
	}
	latest, _ := runExport.Manifest["latest_snapshot_checksum"].(string)
	if latest != second.Snapshot.ManifestChecksum {
		t.Fatalf("expected latest snapshot checksum %q, got %#v", second.Snapshot.ManifestChecksum, runExport.Manifest)
	}
}

func sequenceID(prefix string) func() string {
	counter := 0
	return func() string {
		counter++
		return fmt.Sprintf("%s-%03d", prefix, counter)
	}
}

func findRecordByText(records []domain.Record, text string) domain.Record {
	for _, record := range records {
		if record.ContentText == text {
			return record
		}
	}
	return domain.Record{}
}
