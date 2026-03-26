package replay

import (
	"context"
	"errors"
	"strings"
	"time"

	"clawmem/internal/domain/memory"
	"clawmem/internal/domain/replay"
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

func (s *Service) Store(ctx context.Context, input StoreInput) (replay.ReplayMemoryRecord, error) {
	if strings.TrimSpace(input.ScenarioID) == "" {
		return replay.ReplayMemoryRecord{}, errors.New("scenario_id is required")
	}
	if strings.TrimSpace(input.SourceRef) == "" && strings.TrimSpace(input.SourceID) == "" {
		return replay.ReplayMemoryRecord{}, errors.New("source_ref is required")
	}
	if strings.TrimSpace(input.Summary) == "" {
		return replay.ReplayMemoryRecord{}, errors.New("summary is required")
	}

	record, err := s.memory.Create(ctx, memoryservice.CreateInput{
		ProjectID:       input.ProjectID,
		Environment:     input.Environment,
		ClawbotID:       input.ClawbotID,
		SessionID:       input.SessionID,
		MemoryType:      memory.MemoryTypeReplayCase,
		Scope:           memory.MemoryScopeScenario,
		ScenarioID:      input.ScenarioID,
		SourceID:        input.SourceID,
		SourceRef:       input.SourceRef,
		Summary:         input.Summary,
		Importance:      input.Importance,
		Pinned:          input.Pinned,
		RetentionPolicy: input.RetentionPolicy,
		ExpiresAt:       input.ExpiresAt,
		Metadata:        input.Metadata,
		Tags:            input.Tags,
	})
	if err != nil {
		return replay.ReplayMemoryRecord{}, err
	}

	return replay.ReplayMemoryRecord{
		Record:         record,
		OutcomeSummary: record.Summary,
	}, nil
}

func (s *Service) List(ctx context.Context, scenarioID string) ([]replay.ReplayMemoryRecord, error) {
	result, err := s.memory.List(ctx, memory.MemoryQuery{
		MemoryType: memory.MemoryTypeReplayCase,
		ScenarioID: strings.TrimSpace(scenarioID),
	})
	if err != nil {
		return nil, err
	}

	records := make([]replay.ReplayMemoryRecord, 0, len(result.Records))
	for _, record := range result.Records {
		records = append(records, replay.ReplayMemoryRecord{
			Record:         record,
			OutcomeSummary: record.Summary,
		})
	}
	return records, nil
}
