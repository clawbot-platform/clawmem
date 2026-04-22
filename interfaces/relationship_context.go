package interfaces

import "context"

type RelationshipContextRequest struct {
	MatchedListUID string
	Program        string
	EntityType     string
}

type RelationshipContextResult struct {
	MatchedListUID string
	Depth          int
	Summary        []string
	Paths          []string
	Neighborhood   []map[string]any
}

type RelationshipContextRetriever interface {
	RetrieveRelationshipContext(ctx context.Context, req RelationshipContextRequest) (*RelationshipContextResult, error)
}
