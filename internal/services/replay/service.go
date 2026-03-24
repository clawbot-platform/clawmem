package replay

import (
	"context"
	"errors"
	"strings"

	"clawmem/internal/domain/memory"
	"clawmem/internal/domain/replay"
	memoryservice "clawmem/internal/services/memory"
)

type Service struct {
	memory *memoryservice.Service
}

type StoreInput struct {
	ScenarioID string         `json:"scenario_id"`
	SourceID   string         `json:"source_id"`
	Summary    string         `json:"summary"`
	Metadata   map[string]any `json:"metadata"`
	Tags       []string       `json:"tags"`
}

func NewService(memory *memoryservice.Service) *Service {
	return &Service{memory: memory}
}

func (s *Service) Store(ctx context.Context, input StoreInput) (replay.ReplayMemoryRecord, error) {
	if strings.TrimSpace(input.ScenarioID) == "" {
		return replay.ReplayMemoryRecord{}, errors.New("scenario_id is required")
	}
	if strings.TrimSpace(input.SourceID) == "" {
		return replay.ReplayMemoryRecord{}, errors.New("source_id is required")
	}
	if strings.TrimSpace(input.Summary) == "" {
		return replay.ReplayMemoryRecord{}, errors.New("summary is required")
	}

	record, err := s.memory.Create(ctx, memoryservice.CreateInput{
		MemoryType: memory.MemoryTypeReplayCase,
		Scope:      memory.MemoryScopeScenario,
		ScenarioID: input.ScenarioID,
		SourceID:   input.SourceID,
		Summary:    input.Summary,
		Metadata:   input.Metadata,
		Tags:       input.Tags,
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
