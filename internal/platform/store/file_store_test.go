package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
		ID:              "mem-test-001",
		Namespace:       "sample-project/test/shared/trust_artifact",
		ProjectID:       "sample-project",
		Environment:     "test",
		ClawbotID:       "shared",
		MemoryType:      memory.MemoryTypeTrustArtifact,
		Scope:           memory.MemoryScopeTrust,
		ScenarioID:      "sample-order-review",
		SourceID:        "trust-artifact-001",
		SourceRef:       "trust-artifact-001",
		Summary:         "Stored trust artifact summary.",
		Importance:      70,
		RetentionPolicy: memory.RetentionPolicyPreserved,
		Metadata:        map[string]any{"artifact_family": "mandate"},
		Tags:            []string{"trust"},
		CreatedAt:       time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
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
		ProjectID:  "sample-project",
		MemoryType: memory.MemoryTypeTrustArtifact,
		ScenarioID: "sample-order-review",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}

	count, err := fileStore.Count(context.Background())
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}

	found, err := fileStore.FindByIdempotency(context.Background(), "missing")
	if err == nil || found.ID != "" {
		t.Fatalf("expected missing idempotency lookup error, got %#v %v", found, err)
	}
}

func TestFileStorePaginationSummaryAndIdempotency(t *testing.T) {
	t.Parallel()

	fileStore, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	baseTime := time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC)
	for index := 0; index < 3; index++ {
		record := memory.MemoryRecord{
			ID:              "mem-test-00" + string(rune('1'+index)),
			Namespace:       "sample-project/test/shared/replay_case",
			ProjectID:       "sample-project",
			Environment:     "test",
			ClawbotID:       "shared",
			MemoryType:      memory.MemoryTypeReplayCase,
			Scope:           memory.MemoryScopeScenario,
			ScenarioID:      "sample-order-review",
			SourceID:        "source-" + string(rune('1'+index)),
			SourceRef:       "source-" + string(rune('1'+index)),
			Summary:         "Replay summary.",
			Importance:      60,
			Pinned:          index == 0,
			RetentionPolicy: memory.RetentionPolicyReplayPreserve,
			IdempotencyKey:  "idem-" + string(rune('1'+index)),
			CreatedAt:       baseTime.Add(time.Duration(index) * time.Hour),
			UpdatedAt:       baseTime.Add(time.Duration(index) * time.Hour),
		}
		if _, err := fileStore.Create(context.Background(), record); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	result, err := fileStore.List(context.Background(), memory.MemoryQuery{
		ProjectID:  "sample-project",
		MemoryType: memory.MemoryTypeReplayCase,
		Limit:      2,
		Offset:     1,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Total != 3 || len(result.Records) != 2 || result.HasMore {
		t.Fatalf("unexpected paginated result %#v", result)
	}

	found, err := fileStore.FindByIdempotency(context.Background(), "idem-2")
	if err != nil {
		t.Fatalf("FindByIdempotency() error = %v", err)
	}
	if found.ID == "" {
		t.Fatalf("expected found record, got %#v", found)
	}

	summary, err := fileStore.Summary(context.Background())
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.TotalRecords != 3 || summary.PinnedRecords != 1 {
		t.Fatalf("unexpected summary %#v", summary)
	}
}

func TestFileStoreUpdateDeleteAndListAll(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fileStore, err := NewFileStore(root)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	record := memory.MemoryRecord{
		ID:              "mem-update-001",
		Namespace:       "sample-project/test/shared/benchmark_note",
		ProjectID:       "sample-project",
		Environment:     "test",
		ClawbotID:       "shared",
		MemoryType:      memory.MemoryTypeBenchmarkNote,
		Scope:           memory.MemoryScopeBenchmark,
		SourceID:        "note-001",
		SourceRef:       "note-001",
		Summary:         "Original summary.",
		Importance:      50,
		RetentionPolicy: memory.RetentionPolicyStandard,
		CreatedAt:       time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
	}
	if _, err := fileStore.Create(context.Background(), record); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	record.Summary = "Updated summary."
	record.UpdatedAt = record.UpdatedAt.Add(time.Hour)
	updated, err := fileStore.Update(context.Background(), record)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Summary != "Updated summary." {
		t.Fatalf("unexpected updated record %#v", updated)
	}

	all, err := fileStore.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	if len(all) != 1 || all[0].Summary != "Updated summary." {
		t.Fatalf("unexpected ListAll() result %#v", all)
	}

	payload, err := os.ReadFile(filepath.Join(root, "records", "mem-update-001.json"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(payload) == 0 {
		t.Fatal("expected persisted updated payload")
	}

	if err := fileStore.Delete(context.Background(), record.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	all, err = fileStore.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll() after delete error = %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected empty store after delete, got %#v", all)
	}
}

func TestFileStoreUpdateDeleteAndListAllErrors(t *testing.T) {
	t.Parallel()

	fileStore, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	record := memory.MemoryRecord{
		ID:              "mem-missing-001",
		Namespace:       "sample-project/test/shared/benchmark_note",
		ProjectID:       "sample-project",
		Environment:     "test",
		ClawbotID:       "shared",
		MemoryType:      memory.MemoryTypeBenchmarkNote,
		Scope:           memory.MemoryScopeBenchmark,
		SourceID:        "note-001",
		SourceRef:       "note-001",
		Summary:         "Missing record.",
		Importance:      50,
		RetentionPolicy: memory.RetentionPolicyStandard,
		CreatedAt:       time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
	}

	if _, err := fileStore.Update(context.Background(), record); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound from Update(), got %v", err)
	}
	if err := fileStore.Delete(context.Background(), record.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound from Delete(), got %v", err)
	}

	if _, err := fileStore.ListAll(context.Background()); err != nil {
		t.Fatalf("ListAll() on empty store error = %v", err)
	}
}
