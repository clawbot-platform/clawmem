package store

import (
	"context"
	"testing"
	"time"

	"clawmem/internal/domain/memory"
)

func TestFileStoreCreateListGet(t *testing.T) {
	t.Parallel()

	fileStore, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	record := memory.MemoryRecord{
		ID:         "mem-test-001",
		MemoryType: memory.MemoryTypeTrustArtifact,
		Scope:      memory.MemoryScopeTrustLab,
		ScenarioID: "starter-mandate-review",
		SourceID:   "trust-artifact-001",
		Summary:    "Stored trust artifact summary.",
		Metadata:   map[string]any{"artifact_family": "mandate"},
		Tags:       []string{"trust"},
		CreatedAt:  time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
	}

	if _, err := fileStore.Create(context.Background(), record); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := fileStore.Get(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != record.ID {
		t.Fatalf("expected id %q, got %q", record.ID, got.ID)
	}

	result, err := fileStore.List(context.Background(), memory.MemoryQuery{
		MemoryType: memory.MemoryTypeTrustArtifact,
		ScenarioID: "starter-mandate-review",
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
}
