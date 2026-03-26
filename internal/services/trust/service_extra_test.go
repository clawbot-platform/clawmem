package trust

import (
	"context"
	"testing"

	"clawmem/internal/platform/store"
	memoryservice "clawmem/internal/services/memory"
)

func TestListTrustSummaries(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	service := NewService(memoryservice.NewService(fileStore))
	if _, err := service.Store(context.Background(), StoreInput{
		ScenarioID:     "sample-order-review",
		SourceID:       "trust-artifact-001",
		Summary:        "Trust artifact summary.",
		ArtifactFamily: "mandate",
		ArtifactType:   "mandate_artifact",
	}); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	records, err := service.List(context.Background(), "sample-order-review")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 1 || records[0].ArtifactType != "mandate_artifact" {
		t.Fatalf("unexpected trust records %#v", records)
	}
}
