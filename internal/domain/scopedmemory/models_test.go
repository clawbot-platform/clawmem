package scopedmemory

import (
	"testing"
	"time"
)

func TestNormalizeNamespaceAndValidate(t *testing.T) {
	t.Parallel()

	ns := NormalizeNamespace(Namespace{
		RepoNamespace:  " ach-trust-lab ",
		RunNamespace:   " weekrun-2026 ",
		CycleNamespace: " day-1 ",
		AgentNamespace: " reviewer ",
	})
	if ns.RepoNamespace != "ach-trust-lab" || ns.RunNamespace != "weekrun-2026" || ns.CycleNamespace != "day-1" || ns.AgentNamespace != "reviewer" {
		t.Fatalf("NormalizeNamespace() unexpected result %#v", ns)
	}
	if err := ns.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if err := (Namespace{RepoNamespace: "repo-only"}).Validate(); err == nil {
		t.Fatal("expected Validate() to reject missing run namespace")
	}
}

func TestNormalizeQueryAndSnapshotQueryDefaults(t *testing.T) {
	t.Parallel()

	query := NormalizeQuery(Query{
		RepoNamespace:          " ach-trust-lab ",
		RunNamespace:           " weekrun-1 ",
		CycleNamespace:         " day-2 ",
		AgentNamespace:         " typologies ",
		MemoryClass:            MemoryClassUnresolvedGapsAlias,
		Status:                 " OPEN ",
		SourceRunID:            " run-1 ",
		SourceCycleID:          " cycle-1 ",
		SourceArtifactID:       " artifact-1 ",
		SourcePolicyDecisionID: " policy-1 ",
		SourceModelProfileID:   " granite-3.3 ",
		Limit:                  999,
		Offset:                 -10,
	})
	if query.RepoNamespace != "ach-trust-lab" || query.RunNamespace != "weekrun-1" || query.CycleNamespace != "day-2" || query.AgentNamespace != "typologies" {
		t.Fatalf("NormalizeQuery() namespace trim failed %#v", query)
	}
	if query.MemoryClass != MemoryClassUnresolvedGaps {
		t.Fatalf("expected unresolved gaps alias normalization, got %q", query.MemoryClass)
	}
	if query.Status != StatusOpen {
		t.Fatalf("expected status normalization to open, got %q", query.Status)
	}
	if query.SourceRunID != "run-1" || query.SourceCycleID != "cycle-1" || query.SourceArtifactID != "artifact-1" || query.SourcePolicyDecisionID != "policy-1" || query.SourceModelProfileID != "granite-3.3" {
		t.Fatalf("expected provenance trims, got %#v", query)
	}
	if query.Limit != MaxPageSize || query.Offset != 0 {
		t.Fatalf("expected pagination normalization, got limit=%d offset=%d", query.Limit, query.Offset)
	}

	defaulted := NormalizeQuery(Query{})
	if defaulted.Limit != DefaultPageSize || defaulted.Offset != 0 {
		t.Fatalf("expected default pagination, got %#v", defaulted)
	}

	snapshot := NormalizeSnapshotQuery(SnapshotQuery{
		RepoNamespace:  " ach-trust-lab ",
		RunNamespace:   " weekrun-1 ",
		CycleNamespace: " day-3 ",
		Limit:          -1,
		Offset:         -5,
	})
	if snapshot.RepoNamespace != "ach-trust-lab" || snapshot.RunNamespace != "weekrun-1" || snapshot.CycleNamespace != "day-3" {
		t.Fatalf("NormalizeSnapshotQuery() trim failed %#v", snapshot)
	}
	if snapshot.Limit != DefaultPageSize || snapshot.Offset != 0 {
		t.Fatalf("expected default snapshot pagination, got %#v", snapshot)
	}

	capped := NormalizeSnapshotQuery(SnapshotQuery{Limit: MaxPageSize + 99})
	if capped.Limit != MaxPageSize {
		t.Fatalf("expected snapshot query cap %d, got %d", MaxPageSize, capped.Limit)
	}
}

func TestRecordValidate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	valid := Record{
		ID:            "smr-001",
		RepoNamespace: "ach-trust-lab",
		RunNamespace:  "weekrun-1",
		MemoryClass:   MemoryClassCarryForwardRisks,
		Status:        StatusOpen,
		ContentText:   "descriptor drift risk",
		CreatedBy:     "runner",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}

	resolvedNoTimestamp := valid
	resolvedNoTimestamp.Status = StatusResolved
	if err := resolvedNoTimestamp.Validate(); err == nil {
		t.Fatal("expected resolved status to require resolved_at")
	}

	emptyContent := valid
	emptyContent.ContentText = "   "
	emptyContent.ContentJSON = nil
	if err := emptyContent.Validate(); err == nil {
		t.Fatal("expected content validation failure")
	}

	missingCreatedBy := valid
	missingCreatedBy.CreatedBy = " "
	if err := missingCreatedBy.Validate(); err == nil {
		t.Fatal("expected created_by validation failure")
	}
}

func TestSnapshotValidateAndNormalization(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	snapshot := Snapshot{
		SnapshotID:               "sms-001",
		RepoNamespace:            "ach-trust-lab",
		RunNamespace:             "weekrun-1",
		CreatedAt:                now,
		CreatedBy:                "runner",
		Summary:                  "checkpoint",
		RecordRefs:               []string{" smr-2 ", "smr-1", "smr-1"},
		ManifestChecksum:         "hash-001",
		PreviousSnapshotChecksum: " hash-000 ",
	}
	if err := snapshot.Validate(); err != nil {
		t.Fatalf("Validate(valid snapshot) error = %v", err)
	}

	invalid := snapshot
	invalid.ManifestChecksum = " "
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected missing checksum validation failure")
	}
}

func TestNormalizeClassStatusAndHelpers(t *testing.T) {
	t.Parallel()

	classCases := map[MemoryClass]MemoryClass{
		MemoryClassCycleSummariesAlias:   MemoryClassPriorCycleSummaries,
		MemoryClassCarryForwardRiskAlias: MemoryClassCarryForwardRisks,
		MemoryClassUnresolvedGapsAlias:   MemoryClassUnresolvedGaps,
		MemoryClassBacklogItemAlias:      MemoryClassBacklogItems,
		MemoryClassReviewerNoteAlias:     MemoryClassReviewerNotes,
		MemoryClassPolicyExceptionAlias:  MemoryClassPolicyExceptions,
		MemoryClass(" custom_class "):    MemoryClass(" custom_class "),
	}
	for in, want := range classCases {
		if got := NormalizeClass(in); got != want {
			t.Fatalf("NormalizeClass(%q) = %q, want %q", in, got, want)
		}
	}

	if got := NormalizeStatus(" Resolved "); got != StatusResolved {
		t.Fatalf("NormalizeStatus(resolved) = %q", got)
	}
	if got := NormalizeStatus(" custom "); got != Status("custom") {
		t.Fatalf("NormalizeStatus(custom) = %q", got)
	}
	if err := ValidateStatus(StatusSuperseded); err != nil {
		t.Fatalf("ValidateStatus(superseded) error = %v", err)
	}
	if err := ValidateStatus("invalid-status"); err == nil {
		t.Fatal("expected invalid status error")
	}

	cleaned := CleanTexts([]string{" risk-b ", "", "risk-a", "risk-b", "risk-c"})
	expected := []string{"risk-a", "risk-b", "risk-c"}
	if len(cleaned) != len(expected) {
		t.Fatalf("CleanTexts() length mismatch: got=%v want=%v", cleaned, expected)
	}
	for i := range expected {
		if cleaned[i] != expected[i] {
			t.Fatalf("CleanTexts() mismatch at %d: got=%v want=%v", i, cleaned, expected)
		}
	}
}
