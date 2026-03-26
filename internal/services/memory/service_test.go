package memory

import (
	"context"
	"testing"
	"time"

	domain "clawmem/internal/domain/memory"
	"clawmem/internal/platform/store"
)

func TestServiceCreateAndGet(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	service := NewService(fileStore)
	service.now = func() time.Time { return time.Date(2026, 3, 24, 1, 0, 0, 0, time.UTC) }
	service.idGen = func() string { return "mem-service-001" }

	record, err := service.Create(context.Background(), CreateInput{
		MemoryType: domain.MemoryTypeScenario,
		Scope:      domain.MemoryScopePlatform,
		ScenarioID: "sample-order-review",
		SourceID:   "sample-pack-001",
		Summary:    "Scenario summary memory.",
		Metadata:   map[string]any{"pack_id": "sample-pack"},
		Tags:       []string{"scenario"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if record.ID != "mem-service-001" {
		t.Fatalf("expected deterministic id, got %q", record.ID)
	}
	if record.ProjectID != domain.DefaultProjectID || record.Environment != domain.DefaultEnvironment || record.ClawbotID != domain.DefaultClawbotID {
		t.Fatalf("expected default namespace fields, got %#v", record)
	}
	if record.SourceRef != "sample-pack-001" || record.Namespace == "" {
		t.Fatalf("expected normalized source and namespace, got %#v", record)
	}
}

func TestServiceCreateIdempotentAndBatch(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	service := NewService(fileStore)
	service.now = func() time.Time { return time.Date(2026, 3, 24, 1, 0, 0, 0, time.UTC) }
	idValues := []string{"mem-service-001", "mem-service-002"}
	service.idGen = func() string {
		id := idValues[0]
		idValues = idValues[1:]
		return id
	}

	first, err := service.Create(context.Background(), CreateInput{
		ProjectID:      "alpha",
		Environment:    "test",
		ClawbotID:      "bot-a",
		SessionID:      "session-1",
		MemoryType:     domain.MemoryTypeReplayCase,
		Scope:          domain.MemoryScopeScenario,
		SourceRef:      "replay-1",
		Summary:        "Replay summary memory.",
		IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}

	second, err := service.Create(context.Background(), CreateInput{
		ProjectID:      "alpha",
		Environment:    "test",
		ClawbotID:      "bot-a",
		SessionID:      "session-1",
		MemoryType:     domain.MemoryTypeReplayCase,
		Scope:          domain.MemoryScopeScenario,
		SourceRef:      "replay-1",
		Summary:        "Replay summary memory.",
		IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected idempotent create to return same record, got %s and %s", first.ID, second.ID)
	}

	batch, err := service.CreateBatch(context.Background(), []CreateInput{{
		ProjectID:   "alpha",
		Environment: "test",
		ClawbotID:   "bot-a",
		MemoryType:  domain.MemoryTypeTrustArtifact,
		Scope:       domain.MemoryScopeTrust,
		SourceRef:   "trust-1",
		Summary:     "Trust summary.",
	}})
	if err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}
	if len(batch) != 1 || batch[0].ID != "mem-service-002" {
		t.Fatalf("unexpected batch result %#v", batch)
	}
}

func TestServiceSummary(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	service := NewService(fileStore)
	if _, err := service.CreateSeed(context.Background(), domain.MemoryRecord{
		ID:          "mem-seed-001",
		MemoryType:  domain.MemoryTypeScenario,
		Scope:       domain.MemoryScopePlatform,
		ProjectID:   "alpha",
		Environment: "test",
		ClawbotID:   "bot-a",
		SourceRef:   "seed-1",
		Summary:     "Seed summary.",
		Pinned:      true,
		CreatedAt:   time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("CreateSeed() error = %v", err)
	}

	summary, err := service.Summary(context.Background())
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.TotalRecords != 1 || summary.PinnedRecords != 1 {
		t.Fatalf("unexpected summary %#v", summary)
	}
}
