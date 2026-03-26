package trust

import (
	"context"
	"testing"
	"time"

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
		ScenarioID:     "sample-order-review",
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

func TestStoreTrustSummaryWithSourceRefAndMetadata(t *testing.T) {
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
		SourceRef:       "trust-artifact-002",
		Summary:         "Trust artifact summary.",
		ArtifactFamily:  "approval",
		ArtifactType:    "approval_record",
		Importance:      80,
		Pinned:          true,
		RetentionPolicy: "preserved",
		ExpiresAt:       &expiresAt,
		Metadata:        map[string]any{"source_flow": "example"},
		Tags:            []string{" trust ", "trust"},
	})
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	if record.Record.SourceRef != "trust-artifact-002" || record.Record.ProjectID != "sample-project" {
		t.Fatalf("unexpected stored trust record %#v", record.Record)
	}
	if record.Record.Metadata["artifact_family"] != "approval" || record.Record.Metadata["artifact_type"] != "approval_record" {
		t.Fatalf("expected artifact metadata to be preserved, got %#v", record.Record.Metadata)
	}
}

func TestStoreTrustSummaryValidation(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	service := NewService(memoryservice.NewService(fileStore))
	cases := []StoreInput{
		{Summary: "Trust summary.", ArtifactFamily: "mandate", ArtifactType: "mandate_artifact"},
		{SourceID: "trust-artifact-001", ArtifactFamily: "mandate", ArtifactType: "mandate_artifact"},
		{SourceID: "trust-artifact-001", Summary: "Trust summary.", ArtifactType: "mandate_artifact"},
		{SourceID: "trust-artifact-001", Summary: "Trust summary.", ArtifactFamily: "mandate"},
	}
	for _, input := range cases {
		if _, err := service.Store(context.Background(), input); err == nil {
			t.Fatalf("expected validation error for %#v", input)
		}
	}
}
