package interfaces

import "context"

type RelationshipContextQuery struct {
	MatchedListUID string
	Program        string
	EntityType     string
}

type RelationshipContextResult struct {
	Summary      []string
	Paths        []string
	Neighborhood []map[string]any
}

type GraphRetriever interface {
	RetrieveRelationshipContext(ctx context.Context, query RelationshipContextQuery) (RelationshipContextResult, error)
}
