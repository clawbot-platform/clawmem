package memory

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

type MemoryType string

const (
	MemoryTypeReplayCase    MemoryType = "replay_case"
	MemoryTypeTrustArtifact MemoryType = "trust_artifact"
	MemoryTypeBenchmarkNote MemoryType = "benchmark_note"
	MemoryTypeScenario      MemoryType = "scenario_summary"
)

type MemoryScope string

const (
	MemoryScopeScenario  MemoryScope = "scenario"
	MemoryScopeTrust     MemoryScope = "trust"
	MemoryScopeBenchmark MemoryScope = "benchmark"
	MemoryScopePlatform  MemoryScope = "platform"
)

type RetentionPolicy string

const (
	RetentionPolicyStandard       RetentionPolicy = "standard"
	RetentionPolicySession        RetentionPolicy = "session"
	RetentionPolicyExpiring       RetentionPolicy = "expiring"
	RetentionPolicyPreserved      RetentionPolicy = "preserved"
	RetentionPolicyReplayPreserve RetentionPolicy = "replay_preserve"
)

const (
	DefaultProjectID     = "default-project"
	DefaultEnvironment   = "development"
	DefaultClawbotID     = "shared"
	DefaultImportance    = 50
	DefaultPageSize      = 25
	MaxPageSize          = 100
	DefaultSummaryWindow = 50
)

type MemoryRecord struct {
	ID              string          `json:"id"`
	Namespace       string          `json:"namespace"`
	ProjectID       string          `json:"project_id"`
	Environment     string          `json:"environment"`
	ClawbotID       string          `json:"clawbot_id"`
	SessionID       string          `json:"session_id,omitempty"`
	MemoryType      MemoryType      `json:"memory_type"`
	Scope           MemoryScope     `json:"scope"`
	ScenarioID      string          `json:"scenario_id,omitempty"`
	SourceID        string          `json:"source_id,omitempty"`
	SourceRef       string          `json:"source_ref"`
	Summary         string          `json:"summary"`
	Importance      int             `json:"importance"`
	Pinned          bool            `json:"pinned"`
	ReplayLinked    bool            `json:"replay_linked"`
	StabilityScore  int             `json:"stability_score"`
	RecallCount     int             `json:"recall_count"`
	ReferenceCount  int             `json:"reference_count"`
	RetentionPolicy RetentionPolicy `json:"retention_policy"`
	ExpiresAt       *time.Time      `json:"expires_at,omitempty"`
	DecayEligibleAt *time.Time      `json:"decay_eligible_at,omitempty"`
	LastAccessedAt  *time.Time      `json:"last_accessed_at,omitempty"`
	IdempotencyKey  string          `json:"idempotency_key,omitempty"`
	Metadata        map[string]any  `json:"metadata"`
	Tags            []string        `json:"tags"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type MemoryQuery struct {
	Namespace   string
	ProjectID   string
	Environment string
	ClawbotID   string
	SessionID   string
	MemoryType  MemoryType
	ScenarioID  string
	SourceRef   string
	Limit       int
	Offset      int
}

type MemoryQueryResult struct {
	Records []MemoryRecord `json:"records"`
	Total   int            `json:"total"`
	Limit   int            `json:"limit"`
	Offset  int            `json:"offset"`
	HasMore bool           `json:"has_more"`
}

type Summary struct {
	TotalRecords     int            `json:"total_records"`
	PinnedRecords    int            `json:"pinned_records"`
	ExpiringRecords  int            `json:"expiring_records"`
	ReplayLinked     int            `json:"replay_linked"`
	DecayEligible    int            `json:"decay_eligible"`
	RecordsByType    map[string]int `json:"records_by_type"`
	LastActivityAt   *time.Time     `json:"last_activity_at,omitempty"`
	RecordsByProject map[string]int `json:"records_by_project,omitempty"`
}

type NamespaceSummary struct {
	Namespace         string         `json:"namespace"`
	ProjectID         string         `json:"project_id"`
	Environment       string         `json:"environment"`
	ClawbotID         string         `json:"clawbot_id"`
	SessionID         string         `json:"session_id,omitempty"`
	TotalRecords      int            `json:"total_records"`
	RecordsByType     map[string]int `json:"records_by_type"`
	LastActivityAt    *time.Time     `json:"last_activity_at,omitempty"`
	PinnedCount       int            `json:"pinned_count"`
	ReplayLinkedCount int            `json:"replay_linked_count"`
	DecayEligible     int            `json:"decay_eligible_count"`
	ApproximateBytes  int64          `json:"approximate_bytes"`
	AverageStability  int            `json:"average_stability"`
}

type ClawbotSummary struct {
	ProjectID         string         `json:"project_id"`
	Environment       string         `json:"environment"`
	ClawbotID         string         `json:"clawbot_id"`
	TotalRecords      int            `json:"total_records"`
	RecordsByType     map[string]int `json:"records_by_type"`
	LastActivityAt    *time.Time     `json:"last_activity_at,omitempty"`
	PinnedCount       int            `json:"pinned_count"`
	ReplayLinkedCount int            `json:"replay_linked_count"`
	DecayEligible     int            `json:"decay_eligible_count"`
	ApproximateBytes  int64          `json:"approximate_bytes"`
	AverageStability  int            `json:"average_stability"`
}

type MaintenanceJobType string

const (
	MaintenanceJobDecayUpdate        MaintenanceJobType = "decay_update"
	MaintenanceJobExpiredCleanup     MaintenanceJobType = "expired_cleanup"
	MaintenanceJobStaleCompaction    MaintenanceJobType = "stale_summary_compaction"
	MaintenanceJobReplayPreservation MaintenanceJobType = "replay_preservation_enforcement"
)

type MaintenanceJobStatus struct {
	JobType        MaintenanceJobType `json:"job_type"`
	LastRunAt      *time.Time         `json:"last_run_at,omitempty"`
	LastDurationMS int64              `json:"last_duration_ms"`
	LastResult     string             `json:"last_result"`
	LastError      string             `json:"last_error,omitempty"`
	LastSummary    map[string]int     `json:"last_summary,omitempty"`
}

type MaintenanceOverview struct {
	Jobs                   []MaintenanceJobStatus `json:"jobs"`
	DecayQueueCount        int                    `json:"decay_queue_count"`
	ExpiredCount           int                    `json:"expired_count"`
	ReplayPreservedCount   int                    `json:"replay_preserved_count"`
	StaleSummaryCandidates int                    `json:"stale_summary_candidates"`
	LastUpdatedAt          time.Time              `json:"last_updated_at"`
}

func (r MemoryRecord) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return errors.New("memory record id is required")
	}
	if strings.TrimSpace(string(r.MemoryType)) == "" {
		return errors.New("memory_type is required")
	}
	if strings.TrimSpace(string(r.Scope)) == "" {
		return errors.New("scope is required")
	}
	if strings.TrimSpace(r.SourceRef) == "" && strings.TrimSpace(r.SourceID) == "" {
		return errors.New("source_ref is required")
	}
	if strings.TrimSpace(r.Summary) == "" {
		return errors.New("summary is required")
	}
	if strings.TrimSpace(r.ProjectID) == "" {
		return errors.New("project_id is required")
	}
	if strings.TrimSpace(r.Environment) == "" {
		return errors.New("environment is required")
	}
	if strings.TrimSpace(r.ClawbotID) == "" {
		return errors.New("clawbot_id is required")
	}
	if strings.TrimSpace(r.Namespace) == "" {
		return errors.New("namespace is required")
	}
	if r.Importance < 0 || r.Importance > 100 {
		return fmt.Errorf("importance must be between 0 and 100")
	}
	if r.StabilityScore < 0 || r.StabilityScore > 100 {
		return fmt.Errorf("stability_score must be between 0 and 100")
	}
	if r.RecallCount < 0 {
		return fmt.Errorf("recall_count must be zero or greater")
	}
	if r.ReferenceCount < 0 {
		return fmt.Errorf("reference_count must be zero or greater")
	}
	switch r.RetentionPolicy {
	case RetentionPolicyStandard, RetentionPolicySession, RetentionPolicyExpiring, RetentionPolicyPreserved, RetentionPolicyReplayPreserve:
	case "":
		return errors.New("retention_policy is required")
	default:
		return fmt.Errorf("invalid retention_policy %q", r.RetentionPolicy)
	}
	return nil
}

func ComputeStability(record MemoryRecord) int {
	score := record.Importance
	score += min(record.RecallCount*5, 20)
	score += min(record.ReferenceCount*4, 16)
	if record.Pinned {
		score += 20
	}
	if record.ReplayLinked || record.MemoryType == MemoryTypeReplayCase || record.RetentionPolicy == RetentionPolicyReplayPreserve {
		score += 15
	}
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func RecallRecord(record MemoryRecord, now time.Time) MemoryRecord {
	record.RecallCount++
	record.LastAccessedAt = timePointer(now.UTC())
	record.UpdatedAt = now.UTC()
	record.StabilityScore = ComputeStability(record)
	return record
}

func IsExpired(record MemoryRecord, now time.Time) bool {
	return record.ExpiresAt != nil && !now.Before(*record.ExpiresAt)
}

func IsDecayEligible(record MemoryRecord, now time.Time) bool {
	if record.Pinned {
		return false
	}
	if record.ReplayLinked || record.RetentionPolicy == RetentionPolicyReplayPreserve || record.RetentionPolicy == RetentionPolicyPreserved {
		return false
	}
	if IsExpired(record, now) {
		return true
	}
	lastActivity := record.UpdatedAt
	if record.LastAccessedAt != nil && record.LastAccessedAt.After(lastActivity) {
		lastActivity = *record.LastAccessedAt
	}

	windowHours := 24 + record.Importance/2 + record.StabilityScore/3 + record.RecallCount*4 + record.ReferenceCount*6
	if record.RetentionPolicy == RetentionPolicySession {
		windowHours = 12
	}
	eligibleAt := lastActivity.Add(time.Duration(windowHours) * time.Hour)
	return !now.Before(eligibleAt)
}

func DecayRecord(record MemoryRecord, now time.Time) MemoryRecord {
	record.DecayEligibleAt = timePointer(now.UTC())
	penalty := 10
	if record.Importance < 40 {
		penalty = 15
	}
	if record.StabilityScore > 70 {
		penalty = 5
	}
	record.StabilityScore -= penalty
	if record.StabilityScore < 0 {
		record.StabilityScore = 0
	}
	record.UpdatedAt = now.UTC()
	return record
}

func NormalizeQuery(query MemoryQuery) MemoryQuery {
	query.Namespace = strings.TrimSpace(query.Namespace)
	query.ProjectID = strings.TrimSpace(query.ProjectID)
	query.Environment = strings.TrimSpace(query.Environment)
	query.ClawbotID = strings.TrimSpace(query.ClawbotID)
	query.SessionID = strings.TrimSpace(query.SessionID)
	query.ScenarioID = strings.TrimSpace(query.ScenarioID)
	query.SourceRef = strings.TrimSpace(query.SourceRef)
	if query.Limit <= 0 {
		query.Limit = DefaultPageSize
	}
	if query.Limit > MaxPageSize {
		query.Limit = MaxPageSize
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	return query
}

func BuildNamespace(projectID string, environment string, clawbotID string, sessionID string, memoryType MemoryType) string {
	parts := []string{
		strings.TrimSpace(projectID),
		strings.TrimSpace(environment),
		strings.TrimSpace(clawbotID),
	}
	if strings.TrimSpace(sessionID) != "" {
		parts = append(parts, strings.TrimSpace(sessionID))
	}
	if strings.TrimSpace(string(memoryType)) != "" {
		parts = append(parts, strings.TrimSpace(string(memoryType)))
	}

	filtered := parts[:0]
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, "/")
}

func CleanTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	cleaned := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		cleaned = append(cleaned, trimmed)
	}
	slices.Sort(cleaned)
	return cleaned
}

func timePointer(value time.Time) *time.Time {
	return &value
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
