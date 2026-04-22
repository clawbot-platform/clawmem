package interfaces

import "context"

type CaseHistoryRetriever interface {
    FindSamePairCases(ctx context.Context, tenantID string, screenedFingerprint string, matchedListUID string, limit int) ([]map[string]any, error)
    FindContradictionPatternCases(ctx context.Context, tenantID string, contradictionPattern string, limit int) ([]map[string]any, error)
}
