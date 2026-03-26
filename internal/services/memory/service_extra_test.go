package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "clawmem/internal/domain/memory"

	storepkg "clawmem/internal/platform/store"
)

func TestServiceCreateSeedListGetAndCount(t *testing.T) {
	t.Parallel()

	fileStore, err := storepkg.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	service := NewService(fileStore)
	seeded, err := service.CreateSeed(context.Background(), domain.MemoryRecord{
		ID:         "mem-seed-001",
		MemoryType: domain.MemoryTypeScenario,
		Scope:      domain.MemoryScopePlatform,
		ScenarioID: "sample-order-review",
		SourceID:   "sample-pack-001",
		Summary:    "Seed summary.",
		CreatedAt:  time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateSeed() error = %v", err)
	}
	if len(seeded.Tags) != 0 {
		t.Fatalf("expected empty tags, got %#v", seeded.Tags)
	}
	if seeded.ProjectID == "" || seeded.Environment == "" || seeded.ClawbotID == "" || seeded.Namespace == "" {
		t.Fatalf("expected normalized seed namespace fields, got %#v", seeded)
	}

	result, err := service.List(context.Background(), domain.MemoryQuery{
		MemoryType: domain.MemoryTypeScenario,
		ScenarioID: "sample-order-review",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}

	got, err := service.Get(context.Background(), seeded.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != seeded.ID {
		t.Fatalf("unexpected Get() result %#v", got)
	}

	count, err := service.Count(context.Background())
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}
}

func TestServiceListAllUpdateDeleteAndRecall(t *testing.T) {
	t.Parallel()

	fileStore, err := storepkg.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	service := NewService(fileStore)
	service.now = func() time.Time { return now }
	service.idGen = func() string { return "mem-service-ops-001" }

	created, err := service.Create(context.Background(), CreateInput{
		ProjectID:       "alpha",
		Environment:     "test",
		ClawbotID:       "bot-a",
		MemoryType:      domain.MemoryTypeReplayCase,
		Scope:           domain.MemoryScopeScenario,
		SourceRef:       "replay-1",
		Summary:         "Replay summary memory.",
		Importance:      60,
		RetentionPolicy: domain.RetentionPolicyReplayPreserve,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !created.ReplayLinked || created.StabilityScore == 0 {
		t.Fatalf("expected normalized replay fields, got %#v", created)
	}

	all, err := service.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	if len(all) != 1 || all[0].ID != created.ID {
		t.Fatalf("unexpected ListAll() result %#v", all)
	}

	got, err := service.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.RecallCount != 1 || got.LastAccessedAt == nil || !got.LastAccessedAt.Equal(now) {
		t.Fatalf("expected recall tracking on get, got %#v", got)
	}

	got.Pinned = true
	got.Summary = "Updated replay summary."
	updated, err := service.UpdateRecord(context.Background(), got)
	if err != nil {
		t.Fatalf("UpdateRecord() error = %v", err)
	}
	if !updated.Pinned || updated.Summary != "Updated replay summary." {
		t.Fatalf("unexpected updated record %#v", updated)
	}

	if err := service.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	all, err = service.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll() after delete error = %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected empty store after delete, got %#v", all)
	}
}

func TestNewServiceAndHelpers(t *testing.T) {
	t.Parallel()

	service := NewService(&stubStore{})
	if service.store == nil || service.now == nil || service.idGen == nil {
		t.Fatalf("expected initialized service helpers, got %#v", service)
	}
	if service.now().IsZero() {
		t.Fatal("expected service clock to return a real timestamp")
	}
	if got := service.idGen(); got == "" || got[:4] != "mem-" {
		t.Fatalf("expected generated id with mem- prefix, got %q", got)
	}
	if firstNonEmpty("", " value ", "") != "value" {
		t.Fatal("expected firstNonEmpty to return trimmed first value")
	}
	if firstNonEmpty("", "") != "" {
		t.Fatal("expected empty fallback from firstNonEmpty")
	}
}

func TestCreateValidationAndPinnedRetention(t *testing.T) {
	t.Parallel()

	fileStore, err := storepkg.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	service := NewService(fileStore)
	service.now = func() time.Time { return time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC) }
	service.idGen = func() string { return "mem-service-pinned-001" }

	cases := []CreateInput{
		{Scope: domain.MemoryScopePlatform, SourceRef: "src-1", Summary: "Missing type."},
		{MemoryType: domain.MemoryTypeScenario, SourceRef: "src-1", Summary: "Missing scope."},
		{MemoryType: domain.MemoryTypeScenario, Scope: domain.MemoryScopePlatform, Summary: "Missing source."},
		{MemoryType: domain.MemoryTypeScenario, Scope: domain.MemoryScopePlatform, SourceRef: "src-1"},
	}
	for _, input := range cases {
		if _, err := service.Create(context.Background(), input); err == nil {
			t.Fatalf("expected create validation error for %#v", input)
		}
	}

	record, err := service.Create(context.Background(), CreateInput{
		ProjectID:  "alpha",
		MemoryType: domain.MemoryTypeBenchmarkNote,
		Scope:      domain.MemoryScopeBenchmark,
		SourceRef:  "note-pinned",
		Summary:    "Pinned note.",
		Pinned:     true,
	})
	if err != nil {
		t.Fatalf("Create(pinned) error = %v", err)
	}
	if record.RetentionPolicy != domain.RetentionPolicyPreserved {
		t.Fatalf("expected preserved retention for pinned record, got %q", record.RetentionPolicy)
	}
}

func TestLookupIdempotentRecordBranches(t *testing.T) {
	t.Parallel()

	existing := domain.MemoryRecord{ID: "mem-1"}
	service := NewService(&stubStore{
		findRecord: existing,
	})

	record, ok, err := service.lookupIdempotentRecord(context.Background(), "")
	if err != nil || ok || record.ID != "" {
		t.Fatalf("expected empty-key no-op, got record=%#v ok=%v err=%v", record, ok, err)
	}

	record, ok, err = service.lookupIdempotentRecord(context.Background(), "idem-1")
	if err != nil || !ok || record.ID != existing.ID {
		t.Fatalf("expected stored idempotent hit, got record=%#v ok=%v err=%v", record, ok, err)
	}

	service.store = &stubStore{findErr: storepkg.ErrNotFound}
	record, ok, err = service.lookupIdempotentRecord(context.Background(), "missing")
	if err != nil || ok || record.ID != "" {
		t.Fatalf("expected not-found idempotent miss, got record=%#v ok=%v err=%v", record, ok, err)
	}

	service.store = &stubStore{findErr: errors.New("boom")}
	if _, _, err := service.lookupIdempotentRecord(context.Background(), "bad"); err == nil {
		t.Fatal("expected lookup error")
	}
}

type stubStore struct {
	findRecord domain.MemoryRecord
	findErr    error
}

func (s *stubStore) Create(context.Context, domain.MemoryRecord) (domain.MemoryRecord, error) {
	return domain.MemoryRecord{}, nil
}

func (s *stubStore) Update(context.Context, domain.MemoryRecord) (domain.MemoryRecord, error) {
	return domain.MemoryRecord{}, nil
}

func (s *stubStore) Delete(context.Context, string) error {
	return nil
}

func (s *stubStore) List(context.Context, domain.MemoryQuery) (domain.MemoryQueryResult, error) {
	return domain.MemoryQueryResult{}, nil
}

func (s *stubStore) ListAll(context.Context) ([]domain.MemoryRecord, error) {
	return nil, nil
}

func (s *stubStore) Get(context.Context, string) (domain.MemoryRecord, error) {
	return domain.MemoryRecord{}, nil
}

func (s *stubStore) Count(context.Context) (int, error) {
	return 0, nil
}

func (s *stubStore) FindByIdempotency(context.Context, string) (domain.MemoryRecord, error) {
	if s.findErr != nil {
		return domain.MemoryRecord{}, s.findErr
	}
	return s.findRecord, nil
}

func (s *stubStore) Summary(context.Context) (domain.Summary, error) {
	return domain.Summary{}, nil
}
