package scopedmemory

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

type MemoryClass string

type Status string

const (
	MemoryClassPriorCycleSummaries   MemoryClass = "prior_cycle_summaries"
	MemoryClassCarryForwardRisks     MemoryClass = "carry_forward_risks"
	MemoryClassUnresolvedGaps        MemoryClass = "unresolved_gaps"
	MemoryClassBacklogItems          MemoryClass = "backlog_items"
	MemoryClassReviewerNotes         MemoryClass = "reviewer_notes"
	MemoryClassWorkingContext        MemoryClass = "working_context"
	MemoryClassSnapshotReference     MemoryClass = "memory_snapshot_reference"
	MemoryClassCycleSummariesAlias   MemoryClass = "cycle_summaries"
	MemoryClassUnresolvedGapsAlias   MemoryClass = "unresolved_gap"
	MemoryClassCarryForwardRiskAlias MemoryClass = "carry_forward_risk"
)

const (
	StatusOpen       Status = "open"
	StatusResolved   Status = "resolved"
	StatusSuperseded Status = "superseded"
	StatusArchived   Status = "archived"
)

const (
	DefaultPageSize = 50
	MaxPageSize     = 500
)

type Namespace struct {
	RepoNamespace  string `json:"repo_namespace"`
	RunNamespace   string `json:"run_namespace"`
	CycleNamespace string `json:"cycle_namespace,omitempty"`
	AgentNamespace string `json:"agent_namespace,omitempty"`
}

type Record struct {
	ID             string         `json:"id"`
	RepoNamespace  string         `json:"repo_namespace"`
	RunNamespace   string         `json:"run_namespace"`
	CycleNamespace string         `json:"cycle_namespace,omitempty"`
	AgentNamespace string         `json:"agent_namespace,omitempty"`
	MemoryClass    MemoryClass    `json:"memory_class"`
	Status         Status         `json:"status"`
	ContentText    string         `json:"content_text"`
	ContentJSON    map[string]any `json:"content_json,omitempty"`
	MetadataJSON   map[string]any `json:"metadata_json,omitempty"`
	CreatedBy      string         `json:"created_by"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	ResolvedAt     *time.Time     `json:"resolved_at,omitempty"`
	ExpiresAt      *time.Time     `json:"expires_at,omitempty"`
}

type Query struct {
	RepoNamespace  string
	RunNamespace   string
	CycleNamespace string
	AgentNamespace string
	MemoryClass    MemoryClass
	Status         Status
	Limit          int
	Offset         int
}

type QueryResult struct {
	Records []Record `json:"records"`
	Total   int      `json:"total"`
	Limit   int      `json:"limit"`
	Offset  int      `json:"offset"`
	HasMore bool     `json:"has_more"`
}

type Snapshot struct {
	SnapshotID     string         `json:"snapshot_id"`
	RepoNamespace  string         `json:"repo_namespace"`
	RunNamespace   string         `json:"run_namespace"`
	CycleNamespace string         `json:"cycle_namespace,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	CreatedBy      string         `json:"created_by"`
	Summary        string         `json:"summary"`
	RecordRefs     []string       `json:"record_refs"`
	QueryCriteria  Query          `json:"query_criteria"`
	ManifestRef    string         `json:"manifest_ref,omitempty"`
	MetadataJSON   map[string]any `json:"metadata_json,omitempty"`
}

type SnapshotQuery struct {
	RepoNamespace  string
	RunNamespace   string
	CycleNamespace string
	Limit          int
	Offset         int
}

type SnapshotQueryResult struct {
	Snapshots []Snapshot `json:"snapshots"`
	Total     int        `json:"total"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
	HasMore   bool       `json:"has_more"`
}

type SnapshotExport struct {
	Snapshot Snapshot `json:"snapshot"`
	Records  []Record `json:"records"`
}

type CompactContext struct {
	PriorCycleSummaries []string `json:"prior_cycle_summaries"`
	CarryForwardRisks   []string `json:"carry_forward_risks"`
	UnresolvedGaps      []string `json:"unresolved_gaps"`
	BacklogItems        []string `json:"backlog_items"`
	ReviewerNotes       []string `json:"reviewer_notes"`
}

type RunMemoryExport struct {
	RepoNamespace  string         `json:"repo_namespace"`
	RunNamespace   string         `json:"run_namespace"`
	CycleNamespace string         `json:"cycle_namespace,omitempty"`
	GeneratedAt    time.Time      `json:"generated_at"`
	Records        []Record       `json:"records"`
	Snapshots      []Snapshot     `json:"snapshots"`
	ClassCounts    map[string]int `json:"class_counts"`
	StatusCounts   map[string]int `json:"status_counts"`
	Manifest       map[string]any `json:"manifest"`
}

func (n Namespace) Validate() error {
	if strings.TrimSpace(n.RepoNamespace) == "" {
		return errors.New("repo_namespace is required")
	}
	if strings.TrimSpace(n.RunNamespace) == "" {
		return errors.New("run_namespace is required")
	}
	return nil
}

func NormalizeNamespace(n Namespace) Namespace {
	n.RepoNamespace = strings.TrimSpace(n.RepoNamespace)
	n.RunNamespace = strings.TrimSpace(n.RunNamespace)
	n.CycleNamespace = strings.TrimSpace(n.CycleNamespace)
	n.AgentNamespace = strings.TrimSpace(n.AgentNamespace)
	return n
}

func NormalizeQuery(query Query) Query {
	query.RepoNamespace = strings.TrimSpace(query.RepoNamespace)
	query.RunNamespace = strings.TrimSpace(query.RunNamespace)
	query.CycleNamespace = strings.TrimSpace(query.CycleNamespace)
	query.AgentNamespace = strings.TrimSpace(query.AgentNamespace)
	query.MemoryClass = normalizeClass(query.MemoryClass)
	query.Status = normalizeStatus(query.Status)
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

func NormalizeSnapshotQuery(query SnapshotQuery) SnapshotQuery {
	query.RepoNamespace = strings.TrimSpace(query.RepoNamespace)
	query.RunNamespace = strings.TrimSpace(query.RunNamespace)
	query.CycleNamespace = strings.TrimSpace(query.CycleNamespace)
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

func (r Record) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return errors.New("id is required")
	}
	n := NormalizeNamespace(Namespace{
		RepoNamespace:  r.RepoNamespace,
		RunNamespace:   r.RunNamespace,
		CycleNamespace: r.CycleNamespace,
		AgentNamespace: r.AgentNamespace,
	})
	if err := n.Validate(); err != nil {
		return err
	}
	if normalizeClass(r.MemoryClass) == "" {
		return errors.New("memory_class is required")
	}
	if normalizeStatus(r.Status) == "" {
		return errors.New("status is required")
	}
	if normalizeStatus(r.Status) == StatusResolved && r.ResolvedAt == nil {
		return errors.New("resolved_at is required when status is resolved")
	}
	if strings.TrimSpace(r.ContentText) == "" && len(r.ContentJSON) == 0 {
		return errors.New("content_text or content_json is required")
	}
	if strings.TrimSpace(r.CreatedBy) == "" {
		return errors.New("created_by is required")
	}
	if r.CreatedAt.IsZero() || r.UpdatedAt.IsZero() {
		return errors.New("created_at and updated_at are required")
	}
	return nil
}

func (s Snapshot) Validate() error {
	if strings.TrimSpace(s.SnapshotID) == "" {
		return errors.New("snapshot_id is required")
	}
	n := NormalizeNamespace(Namespace{
		RepoNamespace:  s.RepoNamespace,
		RunNamespace:   s.RunNamespace,
		CycleNamespace: s.CycleNamespace,
	})
	if err := n.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(s.CreatedBy) == "" {
		return errors.New("created_by is required")
	}
	if s.CreatedAt.IsZero() {
		return errors.New("created_at is required")
	}
	if strings.TrimSpace(s.Summary) == "" {
		return errors.New("summary is required")
	}
	return nil
}

func NormalizeClass(class MemoryClass) MemoryClass {
	return normalizeClass(class)
}

func normalizeClass(class MemoryClass) MemoryClass {
	switch strings.TrimSpace(strings.ToLower(string(class))) {
	case string(MemoryClassPriorCycleSummaries), string(MemoryClassCycleSummariesAlias):
		return MemoryClassPriorCycleSummaries
	case string(MemoryClassCarryForwardRisks), string(MemoryClassCarryForwardRiskAlias):
		return MemoryClassCarryForwardRisks
	case string(MemoryClassUnresolvedGaps), string(MemoryClassUnresolvedGapsAlias):
		return MemoryClassUnresolvedGaps
	case string(MemoryClassBacklogItems):
		return MemoryClassBacklogItems
	case string(MemoryClassReviewerNotes):
		return MemoryClassReviewerNotes
	case string(MemoryClassWorkingContext):
		return MemoryClassWorkingContext
	case string(MemoryClassSnapshotReference):
		return MemoryClassSnapshotReference
	case "":
		return ""
	default:
		return class
	}
}

func normalizeStatus(status Status) Status {
	trimmed := strings.TrimSpace(strings.ToLower(string(status)))
	switch trimmed {
	case string(StatusOpen):
		return StatusOpen
	case string(StatusResolved):
		return StatusResolved
	case string(StatusSuperseded):
		return StatusSuperseded
	case string(StatusArchived):
		return StatusArchived
	case "":
		return ""
	default:
		return Status(trimmed)
	}
}

func ValidateStatus(status Status) error {
	switch normalizeStatus(status) {
	case StatusOpen, StatusResolved, StatusSuperseded, StatusArchived:
		return nil
	default:
		return fmt.Errorf("invalid status %q", status)
	}
}

func NormalizeStatus(status Status) Status {
	return normalizeStatus(status)
}

func CleanTexts(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
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
