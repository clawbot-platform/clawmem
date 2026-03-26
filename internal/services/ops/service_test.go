package ops

import (
	"context"
	"errors"
	"testing"
	"time"

	"clawmem/internal/domain/memory"
)

func TestNamespaceAndClawbotSummaries(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 26, 15, 0, 0, 0, time.UTC)
	lastAccessed := now.Add(-30 * time.Minute)
	service := NewService(&stubMemoryService{records: []memory.MemoryRecord{
		{
			ID:             "mem-1",
			Namespace:      "alpha/test/bot-a/scenario_summary",
			ProjectID:      "alpha",
			Environment:    "test",
			ClawbotID:      "bot-a",
			MemoryType:     memory.MemoryTypeScenario,
			Scope:          memory.MemoryScopePlatform,
			SourceRef:      "scenario-a",
			Summary:        "Scenario context.",
			Importance:     80,
			Pinned:         true,
			ReplayLinked:   false,
			StabilityScore: 90,
			CreatedAt:      now.Add(-48 * time.Hour),
			UpdatedAt:      now.Add(-2 * time.Hour),
		},
		{
			ID:              "mem-2",
			Namespace:       "alpha/test/bot-a/replay_case",
			ProjectID:       "alpha",
			Environment:     "test",
			ClawbotID:       "bot-a",
			MemoryType:      memory.MemoryTypeReplayCase,
			Scope:           memory.MemoryScopeScenario,
			SourceRef:       "replay-a",
			Summary:         "Replay case.",
			Importance:      55,
			ReplayLinked:    true,
			StabilityScore:  70,
			LastAccessedAt:  &lastAccessed,
			CreatedAt:       now.Add(-24 * time.Hour),
			UpdatedAt:       now.Add(-3 * time.Hour),
			RetentionPolicy: memory.RetentionPolicyReplayPreserve,
		},
		{
			ID:              "mem-3",
			Namespace:       "beta/test/bot-b/benchmark_note",
			ProjectID:       "beta",
			Environment:     "test",
			ClawbotID:       "bot-b",
			SessionID:       "session-1",
			MemoryType:      memory.MemoryTypeBenchmarkNote,
			Scope:           memory.MemoryScopeBenchmark,
			SourceRef:       "note-b",
			Summary:         "Benchmark note.",
			Importance:      10,
			StabilityScore:  20,
			CreatedAt:       now.Add(-72 * time.Hour),
			UpdatedAt:       now.Add(-72 * time.Hour),
			RetentionPolicy: memory.RetentionPolicyStandard,
		},
	}})
	service.now = func() time.Time { return now }

	namespaceSummaries, err := service.NamespaceSummaries(context.Background(), memory.MemoryQuery{})
	if err != nil {
		t.Fatalf("NamespaceSummaries() error = %v", err)
	}
	if len(namespaceSummaries) != 3 {
		t.Fatalf("expected 3 namespace summaries, got %d", len(namespaceSummaries))
	}
	if namespaceSummaries[0].Namespace != "alpha/test/bot-a/replay_case" {
		t.Fatalf("expected namespace sorting, got %#v", namespaceSummaries)
	}
	if namespaceSummaries[0].ReplayLinkedCount != 1 || namespaceSummaries[0].TotalRecords != 1 {
		t.Fatalf("unexpected replay summary %#v", namespaceSummaries[0])
	}
	if namespaceSummaries[1].PinnedCount != 1 || namespaceSummaries[1].AverageStability != 90 {
		t.Fatalf("unexpected scenario summary %#v", namespaceSummaries[1])
	}

	clawbotSummaries, err := service.ClawbotSummaries(context.Background(), memory.MemoryQuery{ProjectID: "alpha"})
	if err != nil {
		t.Fatalf("ClawbotSummaries() error = %v", err)
	}
	if len(clawbotSummaries) != 1 {
		t.Fatalf("expected 1 clawbot summary, got %d", len(clawbotSummaries))
	}
	summary := clawbotSummaries[0]
	if summary.TotalRecords != 2 || summary.ReplayLinkedCount != 1 || summary.PinnedCount != 1 {
		t.Fatalf("unexpected clawbot summary %#v", summary)
	}
	if summary.AverageStability != 80 {
		t.Fatalf("expected average stability 80, got %d", summary.AverageStability)
	}
	if summary.LastActivityAt == nil || !summary.LastActivityAt.Equal(lastAccessed) {
		t.Fatalf("expected last activity from recall timestamp, got %#v", summary.LastActivityAt)
	}
}

func TestNamespaceAndClawbotSummariesEmptyAndFiltered(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 26, 15, 0, 0, 0, time.UTC)
	service := NewService(&stubMemoryService{records: []memory.MemoryRecord{
		{
			ID:              "mem-1",
			Namespace:       "alpha/test/bot-a/scenario_summary",
			ProjectID:       "alpha",
			Environment:     "test",
			ClawbotID:       "bot-a",
			SessionID:       "session-1",
			MemoryType:      memory.MemoryTypeScenario,
			Scope:           memory.MemoryScopePlatform,
			ScenarioID:      "scenario-a",
			SourceRef:       "source-a",
			Summary:         "Scenario context.",
			Importance:      50,
			StabilityScore:  60,
			RetentionPolicy: memory.RetentionPolicyStandard,
			CreatedAt:       now.Add(-2 * time.Hour),
			UpdatedAt:       now.Add(-2 * time.Hour),
		},
	}})
	service.now = func() time.Time { return now }

	filteredNamespaces, err := service.NamespaceSummaries(context.Background(), memory.MemoryQuery{
		ProjectID:  "alpha",
		ScenarioID: "scenario-a",
		SourceRef:  "source-a",
	})
	if err != nil {
		t.Fatalf("NamespaceSummaries(filtered) error = %v", err)
	}
	if len(filteredNamespaces) != 1 {
		t.Fatalf("expected filtered namespace summary, got %#v", filteredNamespaces)
	}

	filteredClawbots, err := service.ClawbotSummaries(context.Background(), memory.MemoryQuery{
		ProjectID:  "alpha",
		SessionID:  "session-1",
		ScenarioID: "missing",
	})
	if err != nil {
		t.Fatalf("ClawbotSummaries(filtered) error = %v", err)
	}
	if len(filteredClawbots) != 0 {
		t.Fatalf("expected no clawbot summaries for mismatched filter, got %#v", filteredClawbots)
	}

	emptyService := NewService(&stubMemoryService{})
	emptyService.now = func() time.Time { return now }
	emptyOverview, err := emptyService.MaintenanceOverview(context.Background())
	if err != nil {
		t.Fatalf("MaintenanceOverview(empty) error = %v", err)
	}
	if len(emptyOverview.Jobs) != 4 || emptyOverview.DecayQueueCount != 0 || emptyOverview.ExpiredCount != 0 {
		t.Fatalf("unexpected empty overview %#v", emptyOverview)
	}
}

func TestNewServiceInitialJobsAndHelpers(t *testing.T) {
	t.Parallel()

	service := NewService(&stubMemoryService{})
	if len(service.jobs) != 4 {
		t.Fatalf("expected 4 initialized jobs, got %d", len(service.jobs))
	}

	sorted := sortClawbotSummaries(map[string]*memory.ClawbotSummary{
		"z": {ProjectID: "zeta", Environment: "test", ClawbotID: "bot-z", TotalRecords: 2, AverageStability: 140},
		"a": {ProjectID: "alpha", Environment: "test", ClawbotID: "bot-a", TotalRecords: 1, AverageStability: 40},
	})
	if len(sorted) != 2 || sorted[0].ProjectID != "alpha" || sorted[1].AverageStability != 70 {
		t.Fatalf("unexpected sorted clawbot summaries %#v", sorted)
	}

	record := memory.MemoryRecord{Summary: "fallback", Metadata: map[string]any{"bad": make(chan int)}}
	if got := approximateRecordBytes(record); got != int64(len(record.Summary)) {
		t.Fatalf("expected fallback bytes %d, got %d", len(record.Summary), got)
	}

	original := memory.MaintenanceJobStatus{
		JobType:     memory.MaintenanceJobDecayUpdate,
		LastSummary: map[string]int{"updated": 2},
		LastRunAt:   timePointer(time.Date(2026, 3, 26, 15, 0, 0, 0, time.UTC)),
	}
	cloned := cloneJobStatus(original)
	cloned.LastSummary["updated"] = 99
	if original.LastSummary["updated"] != 2 {
		t.Fatalf("expected cloned summary map to be isolated, got %#v", original.LastSummary)
	}
	if cloned.LastRunAt == nil {
		t.Fatal("expected copied run timestamp")
	}
	if cloneJobStatus(memory.MaintenanceJobStatus{}).LastSummary == nil {
		t.Fatal("expected empty summary map to be initialized")
	}
}

func TestMatchesQueryBranches(t *testing.T) {
	t.Parallel()

	record := memory.MemoryRecord{
		Namespace:   "alpha/test/bot-a/session-1/scenario_summary",
		ProjectID:   "alpha",
		Environment: "test",
		ClawbotID:   "bot-a",
		SessionID:   "session-1",
		MemoryType:  memory.MemoryTypeScenario,
		ScenarioID:  "scenario-a",
		SourceRef:   "source-a",
	}

	cases := []memory.MemoryQuery{
		{Namespace: "wrong"},
		{ProjectID: "wrong"},
		{Environment: "wrong"},
		{ClawbotID: "wrong"},
		{SessionID: "wrong"},
		{MemoryType: memory.MemoryTypeReplayCase},
		{ScenarioID: "wrong"},
		{SourceRef: "wrong"},
	}
	for _, query := range cases {
		if matchesQuery(record, query) {
			t.Fatalf("expected mismatch for query %#v", query)
		}
	}
	if !matchesQuery(record, memory.MemoryQuery{
		Namespace:   record.Namespace,
		ProjectID:   record.ProjectID,
		Environment: record.Environment,
		ClawbotID:   record.ClawbotID,
		SessionID:   record.SessionID,
		MemoryType:  record.MemoryType,
		ScenarioID:  record.ScenarioID,
		SourceRef:   record.SourceRef,
	}) {
		t.Fatal("expected full query match")
	}
}

func TestMaintenanceOverview(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 26, 15, 0, 0, 0, time.UTC)
	expiredAt := now.Add(-time.Hour)
	service := NewService(&stubMemoryService{records: []memory.MemoryRecord{
		{
			ID:              "mem-expired",
			Namespace:       "alpha/test/bot-a/benchmark_note",
			ProjectID:       "alpha",
			Environment:     "test",
			ClawbotID:       "bot-a",
			MemoryType:      memory.MemoryTypeBenchmarkNote,
			Scope:           memory.MemoryScopeBenchmark,
			SourceRef:       "note-1",
			Summary:         "Expired note.",
			Importance:      15,
			StabilityScore:  10,
			RetentionPolicy: memory.RetentionPolicyExpiring,
			ExpiresAt:       &expiredAt,
			CreatedAt:       now.Add(-72 * time.Hour),
			UpdatedAt:       now.Add(-72 * time.Hour),
		},
		{
			ID:              "mem-replay",
			Namespace:       "alpha/test/bot-a/replay_case",
			ProjectID:       "alpha",
			Environment:     "test",
			ClawbotID:       "bot-a",
			MemoryType:      memory.MemoryTypeReplayCase,
			Scope:           memory.MemoryScopeScenario,
			SourceRef:       "replay-1",
			Summary:         "Replay note.",
			Importance:      55,
			ReplayLinked:    true,
			StabilityScore:  66,
			RetentionPolicy: memory.RetentionPolicyReplayPreserve,
			CreatedAt:       now.Add(-24 * time.Hour),
			UpdatedAt:       now.Add(-24 * time.Hour),
		},
		{
			ID:              "mem-duplicate",
			Namespace:       "alpha/test/bot-a/scenario_summary",
			ProjectID:       "alpha",
			Environment:     "test",
			ClawbotID:       "bot-a",
			MemoryType:      memory.MemoryTypeScenario,
			Scope:           memory.MemoryScopePlatform,
			SourceRef:       "scenario-1",
			Summary:         "Duplicate summary.",
			Importance:      20,
			StabilityScore:  25,
			RetentionPolicy: memory.RetentionPolicyStandard,
			CreatedAt:       now.Add(-50 * time.Hour),
			UpdatedAt:       now.Add(-50 * time.Hour),
		},
		{
			ID:              "mem-duplicate-2",
			Namespace:       "alpha/test/bot-a/scenario_summary",
			ProjectID:       "alpha",
			Environment:     "test",
			ClawbotID:       "bot-a",
			MemoryType:      memory.MemoryTypeScenario,
			Scope:           memory.MemoryScopePlatform,
			SourceRef:       "scenario-1",
			Summary:         "Duplicate summary newer.",
			Importance:      20,
			StabilityScore:  25,
			RetentionPolicy: memory.RetentionPolicyStandard,
			CreatedAt:       now.Add(-49 * time.Hour),
			UpdatedAt:       now.Add(-49 * time.Hour),
		},
	}})
	service.now = func() time.Time { return now }

	overview, err := service.MaintenanceOverview(context.Background())
	if err != nil {
		t.Fatalf("MaintenanceOverview() error = %v", err)
	}
	if overview.DecayQueueCount != 3 {
		t.Fatalf("expected decay queue count 3, got %d", overview.DecayQueueCount)
	}
	if overview.ExpiredCount != 1 {
		t.Fatalf("expected expired count 1, got %d", overview.ExpiredCount)
	}
	if overview.ReplayPreservedCount != 1 {
		t.Fatalf("expected replay-preserved count 1, got %d", overview.ReplayPreservedCount)
	}
	if overview.StaleSummaryCandidates != 1 {
		t.Fatalf("expected stale summary candidate 1, got %d", overview.StaleSummaryCandidates)
	}
	if len(overview.Jobs) != 4 {
		t.Fatalf("expected 4 maintenance jobs, got %d", len(overview.Jobs))
	}
}

func TestRunMaintenanceJobs(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 26, 15, 0, 0, 0, time.UTC)
	expiredAt := now.Add(-time.Hour)
	store := &stubMemoryService{records: []memory.MemoryRecord{
		{
			ID:              "mem-decay",
			Namespace:       "alpha/test/bot-a/benchmark_note",
			ProjectID:       "alpha",
			Environment:     "test",
			ClawbotID:       "bot-a",
			MemoryType:      memory.MemoryTypeBenchmarkNote,
			Scope:           memory.MemoryScopeBenchmark,
			SourceRef:       "note-decay",
			Summary:         "Decay me.",
			Importance:      10,
			StabilityScore:  20,
			RetentionPolicy: memory.RetentionPolicyStandard,
			CreatedAt:       now.Add(-72 * time.Hour),
			UpdatedAt:       now.Add(-72 * time.Hour),
		},
		{
			ID:              "mem-expired",
			Namespace:       "alpha/test/bot-a/benchmark_note",
			ProjectID:       "alpha",
			Environment:     "test",
			ClawbotID:       "bot-a",
			MemoryType:      memory.MemoryTypeBenchmarkNote,
			Scope:           memory.MemoryScopeBenchmark,
			SourceRef:       "note-expired",
			Summary:         "Delete me.",
			Importance:      15,
			StabilityScore:  20,
			RetentionPolicy: memory.RetentionPolicyExpiring,
			ExpiresAt:       &expiredAt,
			CreatedAt:       now.Add(-72 * time.Hour),
			UpdatedAt:       now.Add(-72 * time.Hour),
		},
		{
			ID:              "mem-duplicate-old",
			Namespace:       "alpha/test/bot-a/scenario_summary",
			ProjectID:       "alpha",
			Environment:     "test",
			ClawbotID:       "bot-a",
			MemoryType:      memory.MemoryTypeScenario,
			Scope:           memory.MemoryScopePlatform,
			SourceRef:       "scenario-a",
			Summary:         "Old duplicate.",
			Importance:      30,
			StabilityScore:  30,
			RetentionPolicy: memory.RetentionPolicyStandard,
			CreatedAt:       now.Add(-50 * time.Hour),
			UpdatedAt:       now.Add(-50 * time.Hour),
		},
		{
			ID:              "mem-duplicate-new",
			Namespace:       "alpha/test/bot-a/scenario_summary",
			ProjectID:       "alpha",
			Environment:     "test",
			ClawbotID:       "bot-a",
			MemoryType:      memory.MemoryTypeScenario,
			Scope:           memory.MemoryScopePlatform,
			SourceRef:       "scenario-a",
			Summary:         "New duplicate.",
			Importance:      35,
			StabilityScore:  35,
			RetentionPolicy: memory.RetentionPolicyStandard,
			CreatedAt:       now.Add(-4 * time.Hour),
			UpdatedAt:       now.Add(-4 * time.Hour),
		},
		{
			ID:             "mem-replay",
			Namespace:      "alpha/test/bot-a/benchmark_note",
			ProjectID:      "alpha",
			Environment:    "test",
			ClawbotID:      "bot-a",
			MemoryType:     memory.MemoryTypeBenchmarkNote,
			Scope:          memory.MemoryScopeBenchmark,
			SourceRef:      "note-replay",
			Summary:        "Promoted note.",
			Importance:     40,
			StabilityScore: 20,
			Tags:           []string{"replay"},
			CreatedAt:      now.Add(-4 * time.Hour),
			UpdatedAt:      now.Add(-4 * time.Hour),
		},
	}}
	service := NewService(store)
	service.now = func() time.Time { return now }

	decayStatus, err := service.RunJob(context.Background(), memory.MaintenanceJobDecayUpdate)
	if err != nil {
		t.Fatalf("RunJob(decay) error = %v", err)
	}
	if decayStatus.LastResult != "completed" || decayStatus.LastSummary["updated"] != 2 {
		t.Fatalf("unexpected decay status %#v", decayStatus)
	}

	cleanupStatus, err := service.RunJob(context.Background(), memory.MaintenanceJobExpiredCleanup)
	if err != nil {
		t.Fatalf("RunJob(cleanup) error = %v", err)
	}
	if cleanupStatus.LastSummary["deleted"] != 1 {
		t.Fatalf("unexpected cleanup status %#v", cleanupStatus)
	}

	compactionStatus, err := service.RunJob(context.Background(), memory.MaintenanceJobStaleCompaction)
	if err != nil {
		t.Fatalf("RunJob(compaction) error = %v", err)
	}
	if compactionStatus.LastSummary["deleted"] != 1 {
		t.Fatalf("unexpected compaction status %#v", compactionStatus)
	}

	replayStatus, err := service.RunJob(context.Background(), memory.MaintenanceJobReplayPreservation)
	if err != nil {
		t.Fatalf("RunJob(replay) error = %v", err)
	}
	if replayStatus.LastSummary["updated"] != 1 {
		t.Fatalf("unexpected replay status %#v", replayStatus)
	}

	replayRecord := store.recordByID("mem-replay")
	if !replayRecord.ReplayLinked || replayRecord.RetentionPolicy != memory.RetentionPolicyReplayPreserve || replayRecord.StabilityScore < 70 {
		t.Fatalf("expected replay preservation update, got %#v", replayRecord)
	}
}

func TestRunJobErrors(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 26, 15, 0, 0, 0, time.UTC)

	listFailure := NewService(&stubMemoryService{listErr: errors.New("list failed")})
	listFailure.now = func() time.Time { return now }
	if _, err := listFailure.RunJob(context.Background(), memory.MaintenanceJobDecayUpdate); err == nil {
		t.Fatal("expected list failure")
	}

	updateFailure := NewService(&stubMemoryService{
		updateErr: errors.New("update failed"),
		records: []memory.MemoryRecord{{
			ID:              "mem-decay",
			Namespace:       "alpha/test/bot-a/benchmark_note",
			ProjectID:       "alpha",
			Environment:     "test",
			ClawbotID:       "bot-a",
			MemoryType:      memory.MemoryTypeBenchmarkNote,
			Scope:           memory.MemoryScopeBenchmark,
			SourceRef:       "note-decay",
			Summary:         "Decay me.",
			Importance:      10,
			StabilityScore:  20,
			RetentionPolicy: memory.RetentionPolicyStandard,
			CreatedAt:       now.Add(-72 * time.Hour),
			UpdatedAt:       now.Add(-72 * time.Hour),
		}},
	})
	updateFailure.now = func() time.Time { return now }
	if _, err := updateFailure.RunJob(context.Background(), memory.MaintenanceJobDecayUpdate); err == nil {
		t.Fatal("expected update failure")
	}

	unsupported := NewService(&stubMemoryService{})
	unsupported.now = func() time.Time { return now }
	if _, err := unsupported.RunJob(context.Background(), memory.MaintenanceJobType("unknown")); !errors.Is(err, ErrUnsupportedJob()) {
		t.Fatalf("expected unsupported job error, got %v", err)
	}
}

func TestRunMaintenanceJobsPreservePinnedAndReplayRecords(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 26, 15, 0, 0, 0, time.UTC)
	expiredAt := now.Add(-time.Hour)
	store := &stubMemoryService{records: []memory.MemoryRecord{
		{
			ID:              "mem-pinned",
			Namespace:       "alpha/test/bot-a/benchmark_note",
			ProjectID:       "alpha",
			Environment:     "test",
			ClawbotID:       "bot-a",
			MemoryType:      memory.MemoryTypeBenchmarkNote,
			Scope:           memory.MemoryScopeBenchmark,
			SourceRef:       "note-pinned",
			Summary:         "Pinned and expired.",
			Pinned:          true,
			Importance:      10,
			StabilityScore:  10,
			RetentionPolicy: memory.RetentionPolicyExpiring,
			ExpiresAt:       &expiredAt,
			CreatedAt:       now.Add(-72 * time.Hour),
			UpdatedAt:       now.Add(-72 * time.Hour),
		},
		{
			ID:              "mem-replay",
			Namespace:       "alpha/test/bot-a/benchmark_note",
			ProjectID:       "alpha",
			Environment:     "test",
			ClawbotID:       "bot-a",
			MemoryType:      memory.MemoryTypeReplayCase,
			Scope:           memory.MemoryScopeScenario,
			SourceRef:       "note-replay",
			Summary:         "Replay preserved.",
			ReplayLinked:    true,
			Importance:      10,
			StabilityScore:  10,
			RetentionPolicy: memory.RetentionPolicyReplayPreserve,
			ExpiresAt:       &expiredAt,
			CreatedAt:       now.Add(-72 * time.Hour),
			UpdatedAt:       now.Add(-72 * time.Hour),
		},
	}}
	service := NewService(store)
	service.now = func() time.Time { return now }

	status, err := service.RunJob(context.Background(), memory.MaintenanceJobExpiredCleanup)
	if err != nil {
		t.Fatalf("RunJob(expired_cleanup) error = %v", err)
	}
	if status.LastSummary["deleted"] != 0 {
		t.Fatalf("expected pinned and replay records to survive cleanup, got %#v", status)
	}
}

type stubMemoryService struct {
	records    []memory.MemoryRecord
	listErr    error
	updateErr  error
	deleteErr  error
	updated    []memory.MemoryRecord
	deletedIDs []string
}

func (s *stubMemoryService) ListAll(context.Context) ([]memory.MemoryRecord, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	records := make([]memory.MemoryRecord, len(s.records))
	copy(records, s.records)
	return records, nil
}

func (s *stubMemoryService) UpdateRecord(_ context.Context, record memory.MemoryRecord) (memory.MemoryRecord, error) {
	if s.updateErr != nil {
		return memory.MemoryRecord{}, s.updateErr
	}
	s.updated = append(s.updated, record)
	for index := range s.records {
		if s.records[index].ID == record.ID {
			s.records[index] = record
			return record, nil
		}
	}
	s.records = append(s.records, record)
	return record, nil
}

func (s *stubMemoryService) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedIDs = append(s.deletedIDs, id)
	filtered := s.records[:0]
	for _, record := range s.records {
		if record.ID != id {
			filtered = append(filtered, record)
		}
	}
	s.records = filtered
	return nil
}

func (s *stubMemoryService) recordByID(id string) memory.MemoryRecord {
	for _, record := range s.records {
		if record.ID == id {
			return record
		}
	}
	return memory.MemoryRecord{}
}
