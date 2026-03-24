package replay

import "clawmem/internal/domain/memory"

type ReplayMemoryRecord struct {
	Record         memory.MemoryRecord `json:"record"`
	OutcomeSummary string              `json:"outcome_summary"`
}
