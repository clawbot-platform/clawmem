package memory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domain "clawmem/internal/domain/memory"
)

type Store interface {
	Create(context.Context, domain.MemoryRecord) (domain.MemoryRecord, error)
	List(context.Context, domain.MemoryQuery) (domain.MemoryQueryResult, error)
	Get(context.Context, string) (domain.MemoryRecord, error)
	Count(context.Context) (int, error)
}

type Service struct {
	store Store
	now   func() time.Time
	idGen func() string
}

type CreateInput struct {
	MemoryType domain.MemoryType  `json:"memory_type"`
	Scope      domain.MemoryScope `json:"scope"`
	ScenarioID string             `json:"scenario_id,omitempty"`
	SourceID   string             `json:"source_id"`
	Summary    string             `json:"summary"`
	Metadata   map[string]any     `json:"metadata"`
	Tags       []string           `json:"tags"`
}

func NewService(store Store) *Service {
	return &Service{
		store: store,
		now:   func() time.Time { return time.Now().UTC() },
		idGen: func() string { return fmt.Sprintf("mem-%d", time.Now().UTC().UnixNano()) },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.MemoryRecord, error) {
	if strings.TrimSpace(string(input.MemoryType)) == "" {
		return domain.MemoryRecord{}, errors.New("memory_type is required")
	}
	if strings.TrimSpace(string(input.Scope)) == "" {
		return domain.MemoryRecord{}, errors.New("scope is required")
	}
	if strings.TrimSpace(input.SourceID) == "" {
		return domain.MemoryRecord{}, errors.New("source_id is required")
	}
	if strings.TrimSpace(input.Summary) == "" {
		return domain.MemoryRecord{}, errors.New("summary is required")
	}

	now := s.now()
	record := domain.MemoryRecord{
		ID:         s.idGen(),
		MemoryType: input.MemoryType,
		Scope:      input.Scope,
		ScenarioID: strings.TrimSpace(input.ScenarioID),
		SourceID:   strings.TrimSpace(input.SourceID),
		Summary:    strings.TrimSpace(input.Summary),
		Metadata:   cloneMap(input.Metadata),
		Tags:       append([]string(nil), input.Tags...),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	return s.store.Create(ctx, record)
}

func (s *Service) CreateSeed(ctx context.Context, record domain.MemoryRecord) (domain.MemoryRecord, error) {
	if record.Metadata == nil {
		record.Metadata = map[string]any{}
	}
	if record.Tags == nil {
		record.Tags = []string{}
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = s.now()
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = record.CreatedAt
	}
	return s.store.Create(ctx, record)
}

func (s *Service) List(ctx context.Context, query domain.MemoryQuery) (domain.MemoryQueryResult, error) {
	return s.store.List(ctx, query)
}

func (s *Service) Get(ctx context.Context, id string) (domain.MemoryRecord, error) {
	return s.store.Get(ctx, id)
}

func (s *Service) Count(ctx context.Context) (int, error) {
	return s.store.Count(ctx)
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
