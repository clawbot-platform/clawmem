package memory

import (
	"errors"
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

type MemoryRecord struct {
	ID         string         `json:"id"`
	MemoryType MemoryType     `json:"memory_type"`
	Scope      MemoryScope    `json:"scope"`
	ScenarioID string         `json:"scenario_id,omitempty"`
	SourceID   string         `json:"source_id"`
	Summary    string         `json:"summary"`
	Metadata   map[string]any `json:"metadata"`
	Tags       []string       `json:"tags"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type MemoryQuery struct {
	MemoryType MemoryType
	ScenarioID string
}

type MemoryQueryResult struct {
	Records []MemoryRecord `json:"records"`
	Total   int            `json:"total"`
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
	if strings.TrimSpace(r.SourceID) == "" {
		return errors.New("source_id is required")
	}
	if strings.TrimSpace(r.Summary) == "" {
		return errors.New("summary is required")
	}
	return nil
}
