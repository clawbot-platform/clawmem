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
		ScenarioID: "starter-mandate-review",
		SourceID:   "scenario-pack-001",
		Summary:    "Scenario summary memory.",
		Metadata:   map[string]any{"pack_id": "starter-pack"},
		Tags:       []string{"scenario"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if record.ID != "mem-service-001" {
		t.Fatalf("expected deterministic id, got %q", record.ID)
	}
}
