package replay

import (
	"context"
	"testing"
	"time"

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
		ScenarioID: "sample-order-review",
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

func TestStoreReplaySummaryWithSourceRefAndLifecycleFields(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	service := NewService(memoryservice.NewService(fileStore))
	expiresAt := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)
	record, err := service.Store(context.Background(), StoreInput{
		ProjectID:       "sample-project",
		Environment:     "test",
		ClawbotID:       "bot-a",
		SessionID:       "session-1",
		ScenarioID:      "sample-order-review",
		SourceRef:       "replay-case-002",
		Summary:         "Replay preserved.",
		Importance:      75,
		Pinned:          true,
		RetentionPolicy: "replay_preserve",
		ExpiresAt:       &expiresAt,
		Metadata:        map[string]any{"round_ref": "round-1"},
		Tags:            []string{" replay ", "replay"},
	})
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	if record.Record.SourceRef != "replay-case-002" || record.Record.ProjectID != "sample-project" {
		t.Fatalf("unexpected stored replay record %#v", record.Record)
	}
	if !record.Record.Pinned || !record.Record.ReplayLinked {
		t.Fatalf("expected replay lifecycle flags, got %#v", record.Record)
	}
}

func TestStoreReplaySummaryValidation(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	service := NewService(memoryservice.NewService(fileStore))
	cases := []StoreInput{
		{SourceID: "replay-case-001", Summary: "Replay remained stable."},
		{ScenarioID: "sample-order-review", Summary: "Replay remained stable."},
		{ScenarioID: "sample-order-review", SourceID: "replay-case-001"},
	}
	for _, input := range cases {
		if _, err := service.Store(context.Background(), input); err == nil {
			t.Fatalf("expected validation error for %#v", input)
		}
	}
}
