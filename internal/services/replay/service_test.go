package replay

import (
	"context"
	"testing"

	"clawmem/internal/platform/store"
	memoryservice "clawmem/internal/services/memory"
)

func TestStoreReplaySummary(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	service := NewService(memoryservice.NewService(fileStore))
	record, err := service.Store(context.Background(), StoreInput{
		ScenarioID: "starter-mandate-review",
		SourceID:   "replay-case-001",
		Summary:    "Replay remained stable.",
	})
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	if record.Record.MemoryType != "replay_case" {
		t.Fatalf("expected replay_case memory type, got %q", record.Record.MemoryType)
	}
}
