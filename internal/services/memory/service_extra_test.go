package memory

import (
	"context"
	"testing"
	"time"

	domain "clawmem/internal/domain/memory"
	"clawmem/internal/platform/store"
)

func TestServiceCreateSeedListGetAndCount(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
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

	result, err := service.List(context.Background(), domain.MemoryQuery{
		MemoryType: domain.MemoryTypeScenario,
		ScenarioID: "sample-order-review",
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
