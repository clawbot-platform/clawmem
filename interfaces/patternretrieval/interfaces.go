package patternretrieval

import "context"

type Query struct {
	TenantID string
	Pattern  string
	Limit    int
}

type Match struct {
	Pattern       string
	DecisionLabel string
	Count         int
}

type Retriever interface {
	FindContradictionPatterns(context.Context, Query) ([]Match, error)
}
