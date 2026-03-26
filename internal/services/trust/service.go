package trust

import (
	"context"
	"errors"
	"strings"
	"time"

	"clawmem/internal/domain/memory"
	"clawmem/internal/domain/trust"
	memoryservice "clawmem/internal/services/memory"
)

type Service struct {
	memory *memoryservice.Service
}

type StoreInput struct {
	ProjectID       string                 `json:"project_id,omitempty"`
	Environment     string                 `json:"environment,omitempty"`
	ClawbotID       string                 `json:"clawbot_id,omitempty"`
	SessionID       string                 `json:"session_id,omitempty"`
	ScenarioID      string                 `json:"scenario_id"`
	SourceID        string                 `json:"source_id,omitempty"`
	SourceRef       string                 `json:"source_ref,omitempty"`
	Summary         string                 `json:"summary"`
	ArtifactFamily  string                 `json:"artifact_family"`
	ArtifactType    string                 `json:"artifact_type"`
	Importance      int                    `json:"importance,omitempty"`
	Pinned          bool                   `json:"pinned,omitempty"`
	RetentionPolicy memory.RetentionPolicy `json:"retention_policy,omitempty"`
	ExpiresAt       *time.Time             `json:"expires_at,omitempty"`
	Metadata        map[string]any         `json:"metadata"`
	Tags            []string               `json:"tags"`
}

func NewService(memory *memoryservice.Service) *Service {
	return &Service{memory: memory}
}

func (s *Service) Store(ctx context.Context, input StoreInput) (trust.TrustMemoryRecord, error) {
	if strings.TrimSpace(input.SourceRef) == "" && strings.TrimSpace(input.SourceID) == "" {
		return trust.TrustMemoryRecord{}, errors.New("source_ref is required")
	}
	if strings.TrimSpace(input.Summary) == "" {
		return trust.TrustMemoryRecord{}, errors.New("summary is required")
	}
	if strings.TrimSpace(input.ArtifactFamily) == "" {
		return trust.TrustMemoryRecord{}, errors.New("artifact_family is required")
	}
	if strings.TrimSpace(input.ArtifactType) == "" {
		return trust.TrustMemoryRecord{}, errors.New("artifact_type is required")
	}

	metadata := cloneMap(input.Metadata)
	metadata["artifact_family"] = input.ArtifactFamily
	metadata["artifact_type"] = input.ArtifactType

	record, err := s.memory.Create(ctx, memoryservice.CreateInput{
		ProjectID:       input.ProjectID,
		Environment:     input.Environment,
		ClawbotID:       input.ClawbotID,
		SessionID:       input.SessionID,
		MemoryType:      memory.MemoryTypeTrustArtifact,
		Scope:           memory.MemoryScopeTrust,
		ScenarioID:      input.ScenarioID,
		SourceID:        input.SourceID,
		SourceRef:       input.SourceRef,
		Summary:         input.Summary,
		Importance:      input.Importance,
		Pinned:          input.Pinned,
		RetentionPolicy: input.RetentionPolicy,
		ExpiresAt:       input.ExpiresAt,
		Metadata:        metadata,
		Tags:            input.Tags,
	})
	if err != nil {
		return trust.TrustMemoryRecord{}, err
	}

	return trust.TrustMemoryRecord{
		Record:         record,
		ArtifactFamily: input.ArtifactFamily,
		ArtifactType:   input.ArtifactType,
	}, nil
}

func (s *Service) List(ctx context.Context, scenarioID string) ([]trust.TrustMemoryRecord, error) {
	result, err := s.memory.List(ctx, memory.MemoryQuery{
		MemoryType: memory.MemoryTypeTrustArtifact,
		ScenarioID: strings.TrimSpace(scenarioID),
	})
	if err != nil {
		return nil, err
	}

	records := make([]trust.TrustMemoryRecord, 0, len(result.Records))
	for _, record := range result.Records {
		artifactFamily, _ := record.Metadata["artifact_family"].(string)
		artifactType, _ := record.Metadata["artifact_type"].(string)
		records = append(records, trust.TrustMemoryRecord{
			Record:         record,
			ArtifactFamily: artifactFamily,
			ArtifactType:   artifactType,
		})
	}
	return records, nil
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}
