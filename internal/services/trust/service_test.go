package trust

import (
	"context"
	"testing"

	"clawmem/internal/platform/store"
	memoryservice "clawmem/internal/services/memory"
)

func TestStoreTrustSummary(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	service := NewService(memoryservice.NewService(fileStore))
	record, err := service.Store(context.Background(), StoreInput{
		ScenarioID:     "starter-mandate-review",
		SourceID:       "trust-artifact-001",
		Summary:        "Trust artifact summary.",
		ArtifactFamily: "mandate",
		ArtifactType:   "mandate_artifact",
	})
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	if record.ArtifactFamily != "mandate" {
		t.Fatalf("expected mandate artifact family, got %q", record.ArtifactFamily)
	}
}
