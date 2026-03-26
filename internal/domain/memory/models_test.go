package memory

import (
	"testing"
	"time"
)

func TestMemoryRecordValidate(t *testing.T) {
	t.Parallel()

	record := MemoryRecord{
		ID:              "mem-1",
		Namespace:       "project/dev/shared/trust_artifact",
		ProjectID:       "project",
		Environment:     "dev",
		ClawbotID:       "shared",
		MemoryType:      MemoryTypeTrustArtifact,
		Scope:           MemoryScopeTrust,
		SourceRef:       "artifact-1",
		Summary:         "Stored trust summary.",
		Importance:      50,
		RetentionPolicy: RetentionPolicyStandard,
	}
	if err := record.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestMemoryRecordValidateRequiresFields(t *testing.T) {
	t.Parallel()

	record := MemoryRecord{}
	if err := record.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestMemoryRecordValidateRequiredFieldBranches(t *testing.T) {
	t.Parallel()

	base := MemoryRecord{
		ID:              "mem-1",
		Namespace:       "project/dev/shared/trust_artifact",
		ProjectID:       "project",
		Environment:     "dev",
		ClawbotID:       "shared",
		MemoryType:      MemoryTypeTrustArtifact,
		Scope:           MemoryScopeTrust,
		SourceRef:       "artifact-1",
		Summary:         "Stored trust summary.",
		Importance:      50,
		RetentionPolicy: RetentionPolicyStandard,
	}

	cases := []struct {
		name   string
		mutate func(*MemoryRecord)
	}{
		{name: "missing id", mutate: func(record *MemoryRecord) { record.ID = "" }},
		{name: "missing memory type", mutate: func(record *MemoryRecord) { record.MemoryType = "" }},
		{name: "missing scope", mutate: func(record *MemoryRecord) { record.Scope = "" }},
		{name: "missing source", mutate: func(record *MemoryRecord) { record.SourceRef = ""; record.SourceID = "" }},
		{name: "missing summary", mutate: func(record *MemoryRecord) { record.Summary = "" }},
		{name: "missing project", mutate: func(record *MemoryRecord) { record.ProjectID = "" }},
		{name: "missing environment", mutate: func(record *MemoryRecord) { record.Environment = "" }},
		{name: "missing clawbot", mutate: func(record *MemoryRecord) { record.ClawbotID = "" }},
		{name: "missing namespace", mutate: func(record *MemoryRecord) { record.Namespace = "" }},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			record := base
			testCase.mutate(&record)
			if err := record.Validate(); err == nil {
				t.Fatalf("expected validation error for %s", testCase.name)
			}
		})
	}
}

func TestNormalizeQueryAndHelpers(t *testing.T) {
	query := NormalizeQuery(MemoryQuery{Limit: 1000, Offset: -1})
	if query.Limit != MaxPageSize {
		t.Fatalf("expected max page size, got %d", query.Limit)
	}
	if query.Offset != 0 {
		t.Fatalf("expected normalized offset 0, got %d", query.Offset)
	}

	namespace := BuildNamespace("project", "prod", "bot-1", "session-9", MemoryTypeReplayCase)
	if namespace != "project/prod/bot-1/session-9/replay_case" {
		t.Fatalf("unexpected namespace %s", namespace)
	}

	tags := CleanTags([]string{" replay ", "", "replay", "critical"})
	if len(tags) != 2 || tags[0] != "critical" || tags[1] != "replay" {
		t.Fatalf("unexpected cleaned tags %#v", tags)
	}
}

func TestMemoryRecordValidateImportanceBounds(t *testing.T) {
	t.Parallel()

	record := MemoryRecord{
		ID:              "mem-1",
		Namespace:       "project/dev/shared/trust_artifact",
		ProjectID:       "project",
		Environment:     "dev",
		ClawbotID:       "shared",
		MemoryType:      MemoryTypeTrustArtifact,
		Scope:           MemoryScopeTrust,
		SourceRef:       "artifact-1",
		Summary:         "Stored trust summary.",
		Importance:      101,
		RetentionPolicy: RetentionPolicyStandard,
		ExpiresAt:       testTimePointer(time.Now().UTC()),
	}
	if err := record.Validate(); err == nil {
		t.Fatal("expected importance validation error")
	}

	record.Importance = -1
	if err := record.Validate(); err == nil {
		t.Fatal("expected negative importance validation error")
	}
}

func TestMemoryRecordValidateLifecycleBounds(t *testing.T) {
	t.Parallel()

	base := MemoryRecord{
		ID:              "mem-1",
		Namespace:       "project/dev/shared/trust_artifact",
		ProjectID:       "project",
		Environment:     "dev",
		ClawbotID:       "shared",
		MemoryType:      MemoryTypeTrustArtifact,
		Scope:           MemoryScopeTrust,
		SourceRef:       "artifact-1",
		Summary:         "Stored trust summary.",
		Importance:      50,
		RetentionPolicy: RetentionPolicyStandard,
	}

	record := base
	record.StabilityScore = 101
	if err := record.Validate(); err == nil {
		t.Fatal("expected stability validation error")
	}

	record = base
	record.RecallCount = -1
	if err := record.Validate(); err == nil {
		t.Fatal("expected recall validation error")
	}

	record = base
	record.ReferenceCount = -1
	if err := record.Validate(); err == nil {
		t.Fatal("expected reference validation error")
	}

	record = base
	record.RetentionPolicy = "unknown"
	if err := record.Validate(); err == nil {
		t.Fatal("expected retention validation error")
	}

	record = base
	record.RetentionPolicy = ""
	if err := record.Validate(); err == nil {
		t.Fatal("expected missing retention validation error")
	}
}

func TestLifecycleHelpers(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	expiredAt := now.Add(-time.Minute)
	record := MemoryRecord{
		ID:              "mem-1",
		Namespace:       "project/dev/shared/benchmark_note",
		ProjectID:       "project",
		Environment:     "dev",
		ClawbotID:       "shared",
		MemoryType:      MemoryTypeBenchmarkNote,
		Scope:           MemoryScopeBenchmark,
		SourceRef:       "note-1",
		Summary:         "Stored benchmark note.",
		Importance:      40,
		RecallCount:     1,
		ReferenceCount:  2,
		RetentionPolicy: RetentionPolicyStandard,
		CreatedAt:       now.Add(-72 * time.Hour),
		UpdatedAt:       now.Add(-72 * time.Hour),
	}

	if got := ComputeStability(record); got != 53 {
		t.Fatalf("expected stability 53, got %d", got)
	}

	recalled := RecallRecord(record, now)
	if recalled.RecallCount != 2 || recalled.LastAccessedAt == nil || !recalled.LastAccessedAt.Equal(now) {
		t.Fatalf("unexpected recalled record %#v", recalled)
	}
	if recalled.StabilityScore != 58 {
		t.Fatalf("expected recalled stability 58, got %d", recalled.StabilityScore)
	}

	expired := record
	expired.ExpiresAt = &expiredAt
	if !IsExpired(expired, now) {
		t.Fatal("expected record to be expired")
	}

	if IsDecayEligible(MemoryRecord{
		Pinned:          true,
		Importance:      10,
		RetentionPolicy: RetentionPolicyStandard,
		UpdatedAt:       now.Add(-100 * time.Hour),
	}, now) {
		t.Fatal("expected pinned record to be excluded from decay")
	}

	if IsDecayEligible(MemoryRecord{
		ReplayLinked:    true,
		Importance:      10,
		RetentionPolicy: RetentionPolicyReplayPreserve,
		UpdatedAt:       now.Add(-100 * time.Hour),
	}, now) {
		t.Fatal("expected replay-preserved record to be excluded from decay")
	}

	eligible := MemoryRecord{
		Importance:      10,
		StabilityScore:  20,
		RetentionPolicy: RetentionPolicyStandard,
		UpdatedAt:       now.Add(-100 * time.Hour),
	}
	if !IsDecayEligible(eligible, now) {
		t.Fatal("expected low-stability record to be decay eligible")
	}

	sessionScoped := MemoryRecord{
		Importance:      90,
		StabilityScore:  90,
		RetentionPolicy: RetentionPolicySession,
		UpdatedAt:       now.Add(-13 * time.Hour),
	}
	if !IsDecayEligible(sessionScoped, now) {
		t.Fatal("expected session record to use the shorter decay window")
	}

	decayed := DecayRecord(eligible, now)
	if decayed.DecayEligibleAt == nil || !decayed.DecayEligibleAt.Equal(now) {
		t.Fatalf("expected decay timestamp, got %#v", decayed.DecayEligibleAt)
	}
	if decayed.StabilityScore != 5 {
		t.Fatalf("expected low-importance decay penalty, got %d", decayed.StabilityScore)
	}

	highStability := DecayRecord(MemoryRecord{Importance: 70, StabilityScore: 80}, now)
	if highStability.StabilityScore != 75 {
		t.Fatalf("expected reduced penalty for high stability, got %d", highStability.StabilityScore)
	}
}

func TestComputeStabilityCapsAtOneHundred(t *testing.T) {
	t.Parallel()

	record := MemoryRecord{
		Importance:      95,
		RecallCount:     10,
		ReferenceCount:  10,
		Pinned:          true,
		ReplayLinked:    true,
		RetentionPolicy: RetentionPolicyReplayPreserve,
		MemoryType:      MemoryTypeReplayCase,
	}
	if got := ComputeStability(record); got != 100 {
		t.Fatalf("expected capped stability 100, got %d", got)
	}
}

func testTimePointer(value time.Time) *time.Time {
	return &value
}
