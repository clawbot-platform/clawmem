package trust

import "clawmem/internal/domain/memory"

type TrustMemoryRecord struct {
	Record         memory.MemoryRecord `json:"record"`
	ArtifactFamily string              `json:"artifact_family"`
	ArtifactType   string              `json:"artifact_type"`
}
