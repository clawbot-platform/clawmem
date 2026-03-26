package memory

import "testing"

func TestMemoryRecordValidate(t *testing.T) {
	t.Parallel()

	record := MemoryRecord{
		ID:         "mem-1",
		MemoryType: MemoryTypeTrustArtifact,
		Scope:      MemoryScopeTrust,
		SourceID:   "artifact-1",
		Summary:    "Stored trust summary.",
	}
	if err := record.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestMemoryRecordValidateRequiresFields(t *testing.T) {
	t.Parallel()

	record := MemoryRecord{}
	if err := record.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
