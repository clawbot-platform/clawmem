package scopedmemory

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
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

func TestGetRecordListSnapshotsAndSnapshotFallbackExport(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	svc := NewService(fileStore)
	now := time.Date(2026, 4, 8, 16, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	svc.recordIDGen = sequenceID("smr-list")
	svc.snapshotIDGen = sequenceID("sms-list")

	ns := domain.Namespace{RepoNamespace: "ach-trust-lab", RunNamespace: "weekrun-list", CycleNamespace: "day-4", AgentNamespace: "rule-mapping"}
	result, err := svc.PersistNotes(context.Background(), ns, PersistNotesInput{
		CreatedBy:       "runner",
		UnresolvedGaps:  []string{"gap-list-a"},
		ReviewerNotes:   []string{"review-list-a"},
		SnapshotSummary: "list snapshot 1",
	})
	if err != nil {
		t.Fatalf("PersistNotes() error = %v", err)
	}
	if len(result.RecordIDs) == 0 {
		t.Fatalf("expected record ids from PersistNotes, got %#v", result)
	}

	got, err := svc.GetRecord(context.Background(), result.RecordIDs[0])
	if err != nil {
		t.Fatalf("GetRecord() error = %v", err)
	}
	if got.ID != result.RecordIDs[0] {
		t.Fatalf("GetRecord() unexpected id %q", got.ID)
	}

	now = now.Add(time.Minute)
	second, err := svc.CreateSnapshot(context.Background(), CreateSnapshotInput{
		Namespace: ns,
		CreatedBy: "runner",
		Summary:   "list snapshot 2",
	})
	if err != nil {
		t.Fatalf("CreateSnapshot() error = %v", err)
	}
	if len(second.RecordRefs) == 0 {
		t.Fatalf("expected inferred record refs in snapshot %#v", second)
	}

	snapshots, err := svc.ListSnapshots(context.Background(), domain.SnapshotQuery{
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-list",
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("ListSnapshots() error = %v", err)
	}
	if snapshots.Total < 2 {
		t.Fatalf("expected at least 2 snapshots, got %#v", snapshots)
	}

	paged, err := svc.listAllSnapshots(context.Background(), domain.SnapshotQuery{
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-list",
		Limit:         1,
	})
	if err != nil {
		t.Fatalf("listAllSnapshots() error = %v", err)
	}
	if len(paged) < 2 {
		t.Fatalf("expected paginated listAllSnapshots to collect all snapshots, got %#v", paged)
	}

	now = now.Add(time.Minute)
	fallbackSnapshot, err := svc.CreateSnapshot(context.Background(), CreateSnapshotInput{
		Namespace:  ns,
		CreatedBy:  "runner",
		Summary:    "fallback export",
		RecordRefs: []string{"smr-missing-001"},
		QueryCriteria: &domain.Query{
			RepoNamespace: "ach-trust-lab",
			RunNamespace:  "weekrun-list",
			Limit:         10,
		},
	})
	if err != nil {
		t.Fatalf("CreateSnapshot(fallback) error = %v", err)
	}

	exported, err := svc.ExportSnapshot(context.Background(), fallbackSnapshot.SnapshotID)
	if err != nil {
		t.Fatalf("ExportSnapshot(fallback) error = %v", err)
	}
	if len(exported.Records) == 0 {
		t.Fatalf("expected fallback export records, got %#v", exported)
	}
}

func TestPersistNotesResolveByIDAndByTextBranches(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	svc := NewService(fileStore)
	now := time.Date(2026, 4, 8, 17, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	svc.recordIDGen = sequenceID("smr-resolve")
	svc.snapshotIDGen = sequenceID("sms-resolve")

	ns := domain.Namespace{RepoNamespace: "ach-trust-lab", RunNamespace: "weekrun-resolve", CycleNamespace: "day-2", AgentNamespace: "policy-tuning"}
	_, err = svc.PersistNotes(context.Background(), ns, PersistNotesInput{
		CreatedBy:         "runner",
		Note:              "working seed",
		CarryForwardRisks: []string{"risk-a"},
		UnresolvedGaps:    []string{"gap-a", "gap-b"},
		BacklogItems:      []string{"backlog-a"},
		ReviewerNotes:     []string{"reviewer-a"},
		PolicyExceptions:  []string{"policy-a"},
	})
	if err != nil {
		t.Fatalf("PersistNotes(seed) error = %v", err)
	}

	getClassRecord := func(class domain.MemoryClass, text string) domain.Record {
		t.Helper()
		result, err := svc.ListRecords(context.Background(), domain.Query{
			RepoNamespace: "ach-trust-lab",
			RunNamespace:  "weekrun-resolve",
			MemoryClass:   class,
			Limit:         20,
		})
		if err != nil {
			t.Fatalf("ListRecords(%s) error = %v", class, err)
		}
		return findRecordByText(result.Records, text)
	}

	risk := getClassRecord(domain.MemoryClassCarryForwardRisks, "risk-a")
	backlog := getClassRecord(domain.MemoryClassBacklogItems, "backlog-a")
	reviewer := getClassRecord(domain.MemoryClassReviewerNotes, "reviewer-a")
	policy := getClassRecord(domain.MemoryClassPolicyExceptions, "policy-a")
	gapA := getClassRecord(domain.MemoryClassUnresolvedGaps, "gap-a")
	gapB := getClassRecord(domain.MemoryClassUnresolvedGaps, "gap-b")
	working := getClassRecord(domain.MemoryClassWorkingContext, "working seed")
	if risk.ID == "" || backlog.ID == "" || reviewer.ID == "" || policy.ID == "" || gapA.ID == "" || gapB.ID == "" || working.ID == "" {
		t.Fatalf("expected all seed records to exist")
	}

	if _, err := svc.UpdateRecordStatus(context.Background(), risk.ID, UpdateRecordStatusInput{
		Status:    domain.StatusResolved,
		UpdatedBy: "runner",
		Reason:    "seed-resolved",
	}); err != nil {
		t.Fatalf("UpdateRecordStatus(risk) error = %v", err)
	}

	now = now.Add(2 * time.Minute)
	resolved, err := svc.PersistNotes(context.Background(), ns, PersistNotesInput{
		CreatedBy:               "runner",
		ResolvedGapIDs:          []string{"smr-does-not-exist", working.ID},
		ResolvedRiskIDs:         []string{risk.ID},
		ResolvedBacklogItemIDs:  []string{backlog.ID},
		ResolvedReviewerNoteIDs: []string{reviewer.ID},
		ResolvedPolicyExceptionIDs: []string{
			policy.ID,
		},
		ResolveUnresolvedGaps: []string{"gap-b"},
		SnapshotSummary:       "resolution checkpoint",
	})
	if err != nil {
		t.Fatalf("PersistNotes(resolve) error = %v", err)
	}

	expectedResolved := []string{risk.ID, backlog.ID, reviewer.ID, policy.ID, gapB.ID}
	for _, id := range expectedResolved {
		if !slices.Contains(resolved.ResolvedRecordIDs, id) {
			t.Fatalf("expected resolved id %q in %#v", id, resolved.ResolvedRecordIDs)
		}
	}
	if slices.Contains(resolved.ResolvedRecordIDs, working.ID) {
		t.Fatalf("did not expect non-actionable record to be resolved, got %#v", resolved.ResolvedRecordIDs)
	}

	gapsResolved, err := svc.ListRecords(context.Background(), domain.Query{
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-resolve",
		MemoryClass:   domain.MemoryClassUnresolvedGaps,
		Status:        domain.StatusResolved,
		Limit:         20,
	})
	if err != nil {
		t.Fatalf("ListRecords(resolved gaps) error = %v", err)
	}
	if findRecordByText(gapsResolved.Records, "gap-b").ID == "" {
		t.Fatalf("expected gap-b to be resolved, got %#v", gapsResolved.Records)
	}
}

func TestCompactHelpersAndTransitionMatrix(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 8, 18, 0, 0, 0, time.UTC)
	ns := domain.Namespace{
		RepoNamespace:  "ach-trust-lab",
		RunNamespace:   "weekrun-helpers",
		CycleNamespace: "day-2",
		AgentNamespace: "daily-summary",
	}
	records := []domain.Record{
		{
			ID:             "r1",
			CycleNamespace: "day-2",
			AgentNamespace: "daily-summary",
			Status:         domain.StatusOpen,
			ContentText:    "current-cycle-note",
			UpdatedAt:      now.Add(5 * time.Minute),
			CreatedAt:      now.Add(5 * time.Minute),
		},
		{
			ID:             "r2",
			CycleNamespace: "day-1",
			Status:         domain.StatusResolved,
			ContentText:    "resolved-prior",
			UpdatedAt:      now.Add(4 * time.Minute),
			CreatedAt:      now.Add(4 * time.Minute),
		},
		{
			ID:          "r3",
			Status:      domain.StatusOpen,
			ContentText: "run-scope-open",
			UpdatedAt:   now.Add(3 * time.Minute),
			CreatedAt:   now.Add(3 * time.Minute),
		},
		{
			ID:          "r4",
			Status:      domain.StatusArchived,
			ContentText: "archived-drop",
			UpdatedAt:   now.Add(2 * time.Minute),
			CreatedAt:   now.Add(2 * time.Minute),
		},
	}

	context := compactTexts(records, compactRule{
		Namespace:           ns,
		MaxItems:            3,
		ExcludeCurrentCycle: true,
		IncludeResolved:     true,
	})
	if len(context) != 2 || context[0] != "run-scope-open" || context[1] != "resolved-prior" {
		t.Fatalf("unexpected compactTexts result %#v", context)
	}

	if statusPriority(domain.StatusOpen) >= statusPriority(domain.StatusResolved) {
		t.Fatalf("expected open status priority ahead of resolved")
	}
	if statusPriority("unknown") != 4 {
		t.Fatalf("expected unknown status priority to default to 4")
	}

	if scopePriority(records[0], ns) != 0 || scopePriority(records[2], ns) != 2 {
		t.Fatalf("unexpected scope priorities current=%d run=%d", scopePriority(records[0], ns), scopePriority(records[2], ns))
	}

	allowed := []struct {
		current domain.Status
		next    domain.Status
		ok      bool
	}{
		{domain.StatusOpen, domain.StatusResolved, true},
		{domain.StatusOpen, domain.StatusArchived, true},
		{domain.StatusResolved, domain.StatusOpen, false},
		{domain.StatusSuperseded, domain.StatusArchived, true},
		{domain.StatusArchived, domain.StatusResolved, false},
	}
	for _, tc := range allowed {
		if got := isAllowedStatusTransition(tc.current, tc.next); got != tc.ok {
			t.Fatalf("isAllowedStatusTransition(%s,%s)=%v want %v", tc.current, tc.next, got, tc.ok)
		}
	}

	if got := valueFromMap(map[string]any{"source_cycle_id": 42}, "source_cycle_id"); got != "42" {
		t.Fatalf("expected numeric valueFromMap conversion, got %q", got)
	}
}

func TestFetchPriorCycleSummariesExcludesCurrentCycleAndValidation(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	svc := NewService(fileStore)
	now := time.Date(2026, 4, 8, 19, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	svc.recordIDGen = sequenceID("smr-prior")
	svc.snapshotIDGen = sequenceID("sms-prior")

	for _, cycle := range []string{"day-1", "day-2", "day-3"} {
		if _, err := svc.PersistNotes(context.Background(), domain.Namespace{
			RepoNamespace:  "ach-trust-lab",
			RunNamespace:   "weekrun-prior",
			CycleNamespace: cycle,
			AgentNamespace: "daily-summary",
		}, PersistNotesInput{
			CreatedBy:           "runner",
			PriorCycleSummaries: []string{"summary-" + cycle},
		}); err != nil {
			t.Fatalf("PersistNotes(%s) error = %v", cycle, err)
		}
		now = now.Add(time.Minute)
	}

	prior, err := svc.FetchPriorCycleSummaries(context.Background(), domain.Namespace{
		RepoNamespace:  "ach-trust-lab",
		RunNamespace:   "weekrun-prior",
		CycleNamespace: "day-3",
		AgentNamespace: "policy-tuning",
	})
	if err != nil {
		t.Fatalf("FetchPriorCycleSummaries() error = %v", err)
	}
	if slices.Contains(prior, "summary-day-3") {
		t.Fatalf("expected current cycle summary exclusion, got %#v", prior)
	}
	if len(prior) == 0 || prior[0] != "summary-day-2" {
		t.Fatalf("expected newest prior summary first, got %#v", prior)
	}

	if _, err := svc.FetchCompactContext(context.Background(), domain.Namespace{RepoNamespace: "ach-trust-lab"}); err == nil {
		t.Fatal("expected FetchCompactContext() namespace validation error")
	}
}

func TestPersistNotesUpsertsExistingGapAndNewScopeSeparately(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	svc := NewService(fileStore)
	now := time.Date(2026, 4, 8, 20, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	svc.recordIDGen = sequenceID("smr-upsert")
	svc.snapshotIDGen = sequenceID("sms-upsert")

	ns := domain.Namespace{RepoNamespace: "ach-trust-lab", RunNamespace: "weekrun-upsert", CycleNamespace: "day-1", AgentNamespace: "feature-gap"}
	if _, err := svc.PersistNotes(context.Background(), ns, PersistNotesInput{
		CreatedBy:      "runner",
		UnresolvedGaps: []string{"missing-gap-a"},
	}); err != nil {
		t.Fatalf("PersistNotes(seed) error = %v", err)
	}
	firstOpen, err := svc.ListRecords(context.Background(), domain.Query{
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-upsert",
		MemoryClass:   domain.MemoryClassUnresolvedGaps,
		Status:        domain.StatusOpen,
		Limit:         20,
	})
	if err != nil {
		t.Fatalf("ListRecords(open) error = %v", err)
	}
	if firstOpen.Total != 1 {
		t.Fatalf("expected one initial unresolved gap, got %#v", firstOpen)
	}
	initialID := firstOpen.Records[0].ID

	now = now.Add(time.Minute)
	if _, err := svc.PersistNotes(context.Background(), ns, PersistNotesInput{
		CreatedBy:      "runner",
		UnresolvedGaps: []string{"missing-gap-a"},
		MetadataJSON: map[string]any{
			"review_cycle": "r1",
		},
		SourceRunID: "source-run-1",
	}); err != nil {
		t.Fatalf("PersistNotes(upsert same scope) error = %v", err)
	}
	sameScope, err := svc.ListRecords(context.Background(), domain.Query{
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-upsert",
		MemoryClass:   domain.MemoryClassUnresolvedGaps,
		Status:        domain.StatusOpen,
		Limit:         20,
	})
	if err != nil {
		t.Fatalf("ListRecords(same scope) error = %v", err)
	}
	if sameScope.Total != 1 || sameScope.Records[0].ID != initialID {
		t.Fatalf("expected unresolved gap upsert to update existing record, got %#v", sameScope)
	}
	if sameScope.Records[0].SourceRunID != "source-run-1" {
		t.Fatalf("expected source provenance backfill, got %#v", sameScope.Records[0])
	}

	now = now.Add(time.Minute)
	otherScope := ns
	otherScope.CycleNamespace = "day-2"
	otherScope.AgentNamespace = "typologies"
	if _, err := svc.PersistNotes(context.Background(), otherScope, PersistNotesInput{
		CreatedBy:      "runner",
		UnresolvedGaps: []string{"missing-gap-a"},
	}); err != nil {
		t.Fatalf("PersistNotes(other scope) error = %v", err)
	}
	all, err := svc.ListRecords(context.Background(), domain.Query{
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-upsert",
		MemoryClass:   domain.MemoryClassUnresolvedGaps,
		Status:        domain.StatusOpen,
		Limit:         20,
	})
	if err != nil {
		t.Fatalf("ListRecords(all scopes) error = %v", err)
	}
	if all.Total != 2 {
		t.Fatalf("expected separate unresolved gap records by scope, got %#v", all)
	}
}

func TestServiceConstructorAndCompactContextErrorPath(t *testing.T) {
	t.Parallel()

	svc := NewService(&stubStore{
		listScopedRecordsFn: func(context.Context, domain.Query) (domain.QueryResult, error) {
			return domain.QueryResult{}, errors.New("list failed")
		},
	})
	if !strings.HasPrefix(svc.recordIDGen(), "smr-") {
		t.Fatalf("expected default record id prefix, got %q", svc.recordIDGen())
	}
	if !strings.HasPrefix(svc.snapshotIDGen(), "sms-") {
		t.Fatalf("expected default snapshot id prefix, got %q", svc.snapshotIDGen())
	}

	_, err := svc.FetchCompactContext(context.Background(), domain.Namespace{
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-errors",
	})
	if err == nil || !strings.Contains(err.Error(), "list failed") {
		t.Fatalf("expected propagated list error, got %v", err)
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

type stubStore struct {
	createScopedRecordFn   func(context.Context, domain.Record) (domain.Record, error)
	updateScopedRecordFn   func(context.Context, domain.Record) (domain.Record, error)
	getScopedRecordFn      func(context.Context, string) (domain.Record, error)
	listScopedRecordsFn    func(context.Context, domain.Query) (domain.QueryResult, error)
	createScopedSnapshotFn func(context.Context, domain.Snapshot) (domain.Snapshot, error)
	getScopedSnapshotFn    func(context.Context, string) (domain.Snapshot, error)
	listScopedSnapshotsFn  func(context.Context, domain.SnapshotQuery) (domain.SnapshotQueryResult, error)
}

func (s *stubStore) CreateScopedRecord(ctx context.Context, record domain.Record) (domain.Record, error) {
	if s.createScopedRecordFn != nil {
		return s.createScopedRecordFn(ctx, record)
	}
	return record, nil
}

func (s *stubStore) UpdateScopedRecord(ctx context.Context, record domain.Record) (domain.Record, error) {
	if s.updateScopedRecordFn != nil {
		return s.updateScopedRecordFn(ctx, record)
	}
	return record, nil
}

func (s *stubStore) GetScopedRecord(ctx context.Context, id string) (domain.Record, error) {
	if s.getScopedRecordFn != nil {
		return s.getScopedRecordFn(ctx, id)
	}
	return domain.Record{}, store.ErrNotFound
}

func (s *stubStore) ListScopedRecords(ctx context.Context, query domain.Query) (domain.QueryResult, error) {
	if s.listScopedRecordsFn != nil {
		return s.listScopedRecordsFn(ctx, query)
	}
	return domain.QueryResult{Records: []domain.Record{}, Limit: query.Limit, Offset: query.Offset}, nil
}

func (s *stubStore) CreateScopedSnapshot(ctx context.Context, snapshot domain.Snapshot) (domain.Snapshot, error) {
	if s.createScopedSnapshotFn != nil {
		return s.createScopedSnapshotFn(ctx, snapshot)
	}
	return snapshot, nil
}

func (s *stubStore) GetScopedSnapshot(ctx context.Context, id string) (domain.Snapshot, error) {
	if s.getScopedSnapshotFn != nil {
		return s.getScopedSnapshotFn(ctx, id)
	}
	return domain.Snapshot{}, store.ErrNotFound
}

func (s *stubStore) ListScopedSnapshots(ctx context.Context, query domain.SnapshotQuery) (domain.SnapshotQueryResult, error) {
	if s.listScopedSnapshotsFn != nil {
		return s.listScopedSnapshotsFn(ctx, query)
	}
	return domain.SnapshotQueryResult{Snapshots: []domain.Snapshot{}, Limit: query.Limit, Offset: query.Offset}, nil
}
