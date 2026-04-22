package interfaces

import "context"

type RelationshipContextRequest struct {
	MatchedListUID string
	Depth          int
}

type RelationshipContextRetriever interface {
	RetrieveRelationshipContext(ctx context.Context, req RelationshipContextRequest) (*RelationshipContextResult, error)
}
