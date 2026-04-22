package interfaces

import "context"

type HistoryRetrievalRequest struct {
    TenantID                  string
    ScreenedEntityFingerprint string
    MatchedListUID            string
    Limit                     int
}

type HistoryRetrievalResult struct {
    Cases []map[string]any
}

type HistoryRetriever interface {
    RetrievePriorCases(ctx context.Context, req HistoryRetrievalRequest) (*HistoryRetrievalResult, error)
}
