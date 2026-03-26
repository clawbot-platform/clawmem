package memory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domain "clawmem/internal/domain/memory"
	storepkg "clawmem/internal/platform/store"
)

type Store interface {
	Create(context.Context, domain.MemoryRecord) (domain.MemoryRecord, error)
	Update(context.Context, domain.MemoryRecord) (domain.MemoryRecord, error)
	Delete(context.Context, string) error
	List(context.Context, domain.MemoryQuery) (domain.MemoryQueryResult, error)
	ListAll(context.Context) ([]domain.MemoryRecord, error)
	Get(context.Context, string) (domain.MemoryRecord, error)
	Count(context.Context) (int, error)
	FindByIdempotency(context.Context, string) (domain.MemoryRecord, error)
	Summary(context.Context) (domain.Summary, error)
}

type Service struct {
	store Store
	now   func() time.Time
	idGen func() string
}

type CreateInput struct {
	ProjectID       string                 `json:"project_id,omitempty"`
	Environment     string                 `json:"environment,omitempty"`
	ClawbotID       string                 `json:"clawbot_id,omitempty"`
	SessionID       string                 `json:"session_id,omitempty"`
	MemoryType      domain.MemoryType      `json:"memory_type"`
	Scope           domain.MemoryScope     `json:"scope"`
	ScenarioID      string                 `json:"scenario_id,omitempty"`
	SourceID        string                 `json:"source_id,omitempty"`
	SourceRef       string                 `json:"source_ref,omitempty"`
	Summary         string                 `json:"summary"`
	Importance      int                    `json:"importance,omitempty"`
	Pinned          bool                   `json:"pinned,omitempty"`
	RetentionPolicy domain.RetentionPolicy `json:"retention_policy,omitempty"`
	ExpiresAt       *time.Time             `json:"expires_at,omitempty"`
	IdempotencyKey  string                 `json:"idempotency_key,omitempty"`
	Metadata        map[string]any         `json:"metadata"`
	Tags            []string               `json:"tags"`
}

func NewService(store Store) *Service {
	return &Service{
		store: store,
		now:   func() time.Time { return time.Now().UTC() },
		idGen: func() string { return fmt.Sprintf("mem-%d", time.Now().UTC().UnixNano()) },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.MemoryRecord, error) {
	if existing, ok, err := s.lookupIdempotentRecord(ctx, strings.TrimSpace(input.IdempotencyKey)); err != nil {
		return domain.MemoryRecord{}, err
	} else if ok {
		return existing, nil
	}

	now := s.now()
	record, err := normalizeRecord(s.idGen(), now, input)
	if err != nil {
		return domain.MemoryRecord{}, err
	}

	return s.store.Create(ctx, record)
}

func (s *Service) CreateBatch(ctx context.Context, inputs []CreateInput) ([]domain.MemoryRecord, error) {
	records := make([]domain.MemoryRecord, 0, len(inputs))
	for _, input := range inputs {
		record, err := s.Create(ctx, input)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
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
	record = normalizeSeedRecord(record)
	return s.store.Create(ctx, record)
}

func (s *Service) List(ctx context.Context, query domain.MemoryQuery) (domain.MemoryQueryResult, error) {
	return s.store.List(ctx, domain.NormalizeQuery(query))
}

func (s *Service) Get(ctx context.Context, id string) (domain.MemoryRecord, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.MemoryRecord{}, err
	}
	now := s.now()
	record = domain.RecallRecord(record, now)
	return s.store.Update(ctx, record)
}

func (s *Service) Count(ctx context.Context) (int, error) {
	return s.store.Count(ctx)
}

func (s *Service) Summary(ctx context.Context) (domain.Summary, error) {
	return s.store.Summary(ctx)
}

func (s *Service) ListAll(ctx context.Context) ([]domain.MemoryRecord, error) {
	return s.store.ListAll(ctx)
}

func (s *Service) UpdateRecord(ctx context.Context, record domain.MemoryRecord) (domain.MemoryRecord, error) {
	return s.store.Update(ctx, record)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
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

func normalizeRecord(id string, now time.Time, input CreateInput) (domain.MemoryRecord, error) {
	if strings.TrimSpace(string(input.MemoryType)) == "" {
		return domain.MemoryRecord{}, errors.New("memory_type is required")
	}
	if strings.TrimSpace(string(input.Scope)) == "" {
		return domain.MemoryRecord{}, errors.New("scope is required")
	}
	if strings.TrimSpace(input.SourceRef) == "" && strings.TrimSpace(input.SourceID) == "" {
		return domain.MemoryRecord{}, errors.New("source_ref is required")
	}
	if strings.TrimSpace(input.Summary) == "" {
		return domain.MemoryRecord{}, errors.New("summary is required")
	}

	projectID := firstNonEmpty(strings.TrimSpace(input.ProjectID), domain.DefaultProjectID)
	environment := firstNonEmpty(strings.TrimSpace(input.Environment), domain.DefaultEnvironment)
	clawbotID := firstNonEmpty(strings.TrimSpace(input.ClawbotID), domain.DefaultClawbotID)
	sessionID := strings.TrimSpace(input.SessionID)
	sourceRef := firstNonEmpty(strings.TrimSpace(input.SourceRef), strings.TrimSpace(input.SourceID))
	importance := input.Importance
	if importance == 0 {
		importance = domain.DefaultImportance
	}

	retentionPolicy := input.RetentionPolicy
	if retentionPolicy == "" {
		retentionPolicy = defaultRetentionPolicy(input.MemoryType, input.Pinned)
	}

	record := domain.MemoryRecord{
		ID:              id,
		Namespace:       domain.BuildNamespace(projectID, environment, clawbotID, sessionID, input.MemoryType),
		ProjectID:       projectID,
		Environment:     environment,
		ClawbotID:       clawbotID,
		SessionID:       sessionID,
		MemoryType:      input.MemoryType,
		Scope:           input.Scope,
		ScenarioID:      strings.TrimSpace(input.ScenarioID),
		SourceID:        sourceRef,
		SourceRef:       sourceRef,
		Summary:         strings.TrimSpace(input.Summary),
		Importance:      importance,
		Pinned:          input.Pinned,
		ReplayLinked:    input.MemoryType == domain.MemoryTypeReplayCase || retentionPolicy == domain.RetentionPolicyReplayPreserve,
		RetentionPolicy: retentionPolicy,
		ExpiresAt:       input.ExpiresAt,
		IdempotencyKey:  strings.TrimSpace(input.IdempotencyKey),
		Metadata:        cloneMap(input.Metadata),
		Tags:            domain.CleanTags(input.Tags),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	record.StabilityScore = domain.ComputeStability(record)

	return record, record.Validate()
}

func normalizeSeedRecord(record domain.MemoryRecord) domain.MemoryRecord {
	projectID := firstNonEmpty(strings.TrimSpace(record.ProjectID), domain.DefaultProjectID)
	environment := firstNonEmpty(strings.TrimSpace(record.Environment), domain.DefaultEnvironment)
	clawbotID := firstNonEmpty(strings.TrimSpace(record.ClawbotID), domain.DefaultClawbotID)
	sourceRef := firstNonEmpty(strings.TrimSpace(record.SourceRef), strings.TrimSpace(record.SourceID))
	importance := record.Importance
	if importance == 0 {
		importance = domain.DefaultImportance
	}
	retentionPolicy := record.RetentionPolicy
	if retentionPolicy == "" {
		retentionPolicy = defaultRetentionPolicy(record.MemoryType, record.Pinned)
	}

	record.ProjectID = projectID
	record.Environment = environment
	record.ClawbotID = clawbotID
	record.SessionID = strings.TrimSpace(record.SessionID)
	record.SourceRef = sourceRef
	record.SourceID = sourceRef
	record.Importance = importance
	record.RetentionPolicy = retentionPolicy
	record.ReplayLinked = record.ReplayLinked || record.MemoryType == domain.MemoryTypeReplayCase || record.RetentionPolicy == domain.RetentionPolicyReplayPreserve
	record.Namespace = firstNonEmpty(strings.TrimSpace(record.Namespace), domain.BuildNamespace(projectID, environment, clawbotID, record.SessionID, record.MemoryType))
	record.Tags = domain.CleanTags(record.Tags)
	record.StabilityScore = domain.ComputeStability(record)
	return record
}

func defaultRetentionPolicy(memoryType domain.MemoryType, pinned bool) domain.RetentionPolicy {
	if pinned {
		return domain.RetentionPolicyPreserved
	}
	if memoryType == domain.MemoryTypeReplayCase {
		return domain.RetentionPolicyReplayPreserve
	}
	return domain.RetentionPolicyStandard
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (s *Service) lookupIdempotentRecord(ctx context.Context, key string) (domain.MemoryRecord, bool, error) {
	if key == "" {
		return domain.MemoryRecord{}, false, nil
	}
	record, err := s.store.FindByIdempotency(ctx, key)
	if err != nil {
		if errors.Is(err, storepkg.ErrNotFound) {
			return domain.MemoryRecord{}, false, nil
		}
		return domain.MemoryRecord{}, false, err
	}
	return record, true, nil
}
