package interfaces

import "context"

type ExampleQuery struct {
	CaseType      string
	DecisionLabel string
	Program       string
}

type ExampleResult struct {
	ExampleIDs []string
	Summaries  []string
}

type ExampleMemory interface {
	RetrieveAcceptedExamples(ctx context.Context, query ExampleQuery) (ExampleResult, error)
}
