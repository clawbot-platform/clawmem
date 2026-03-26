package replay

import (
	"context"
	"testing"

	"clawmem/internal/platform/store"
	memoryservice "clawmem/internal/services/memory"
)

func TestListReplaySummaries(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	service := NewService(memoryservice.NewService(fileStore))
	if _, err := service.Store(context.Background(), StoreInput{
		ScenarioID: "sample-order-review",
		SourceID:   "replay-case-001",
		Summary:    "Replay remained stable.",
	}); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	records, err := service.List(context.Background(), "sample-order-review")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 1 || records[0].OutcomeSummary == "" {
		t.Fatalf("unexpected replay records %#v", records)
	}
}
