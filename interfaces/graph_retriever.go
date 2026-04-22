package interfaces

import "context"

type RelationshipContextQuery = RelationshipContextRequest

type GraphRetriever interface {
	RetrieveRelationshipContext(ctx context.Context, query RelationshipContextQuery) (RelationshipContextResult, error)
}
