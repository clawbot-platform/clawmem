package caseretrieval

import "context"

type Query struct {
	TenantID      string
	AlertID       string
	ScreenedKey   string
	MatchedListUID string
	Program       string
	Limit         int
}

type RetrievedCase struct {
	CaseID         string
	AlertID        string
	DecisionLabel  string
	Reason         string
	OccurredAt     string
}

type Retriever interface {
	FindPriorCases(context.Context, Query) ([]RetrievedCase, error)
}
