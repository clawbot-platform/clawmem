package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"clawmem/internal/domain/memory"
	opsservice "clawmem/internal/services/ops"
)

func TestRunMaintenanceJobBadRequestAndInternalError(t *testing.T) {
	t.Parallel()

	handler := NewOpsHandler(stubOpsManager{})

	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/ops/maintenance/unknown/run", nil)
	badReq.SetPathValue("job", "unknown")
	badRec := httptest.NewRecorder()
	handler.RunMaintenanceJob(badRec, badReq)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", badRec.Code, badRec.Body.String())
	}

	errReq := httptest.NewRequest(http.MethodPost, "/api/v1/ops/maintenance/decay_update/run", nil)
	errReq.SetPathValue("job", string(memory.MaintenanceJobDecayUpdate))
	errRec := httptest.NewRecorder()
	NewOpsHandler(stubOpsManager{runErr: errors.New("ops unavailable")}).RunMaintenanceJob(errRec, errReq)
	if errRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", errRec.Code, errRec.Body.String())
	}
}

func TestListNamespaceSummariesInternalError(t *testing.T) {
	t.Parallel()

	handler := NewOpsHandler(stubOpsManager{
		namespaceErr: errors.New("list failed"),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/namespaces", nil)
	rec := httptest.NewRecorder()
	handler.ListNamespaceSummaries(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestParseMemoryQuery(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/namespaces?project_id=alpha&environment=test&clawbot_id=bot-a&session_id=s1&memory_type=replay_case&scenario_id=scn-1&source_ref=src-1", nil)
	query := parseMemoryQuery(req)

	if query.ProjectID != "alpha" || query.Environment != "test" || query.ClawbotID != "bot-a" || query.SessionID != "s1" {
		t.Fatalf("unexpected namespace fields %#v", query)
	}
	if query.MemoryType != memory.MemoryTypeReplayCase || query.ScenarioID != "scn-1" || query.SourceRef != "src-1" {
		t.Fatalf("unexpected typed fields %#v", query)
	}
}

type stubOpsManager struct {
	namespaceErr error
	clawbotErr   error
	overviewErr  error
	runErr       error
}

func (s stubOpsManager) NamespaceSummaries(context.Context, memory.MemoryQuery) ([]memory.NamespaceSummary, error) {
	return nil, s.namespaceErr
}

func (s stubOpsManager) ClawbotSummaries(context.Context, memory.MemoryQuery) ([]memory.ClawbotSummary, error) {
	return []memory.ClawbotSummary{{ProjectID: "alpha", Environment: "test", ClawbotID: "bot-a"}}, s.clawbotErr
}

func (s stubOpsManager) MaintenanceOverview(context.Context) (memory.MaintenanceOverview, error) {
	return memory.MaintenanceOverview{LastUpdatedAt: time.Unix(0, 0).UTC()}, s.overviewErr
}

func (s stubOpsManager) RunJob(_ context.Context, jobType memory.MaintenanceJobType) (memory.MaintenanceJobStatus, error) {
	if s.runErr != nil {
		return memory.MaintenanceJobStatus{}, s.runErr
	}
	switch jobType {
	case memory.MaintenanceJobDecayUpdate, memory.MaintenanceJobExpiredCleanup, memory.MaintenanceJobStaleCompaction, memory.MaintenanceJobReplayPreservation:
	default:
		return memory.MaintenanceJobStatus{}, fmt.Errorf("%w %q", opsservice.ErrUnsupportedJob(), jobType)
	}
	return memory.MaintenanceJobStatus{
		JobType:     jobType,
		LastResult:  "completed",
		LastSummary: map[string]int{"updated": 1},
	}, nil
}

func TestRunMaintenanceJobSuccess(t *testing.T) {
	t.Parallel()

	handler := NewOpsHandler(stubOpsManager{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/maintenance/decay_update/run", nil)
	req.SetPathValue("job", string(memory.MaintenanceJobDecayUpdate))
	rec := httptest.NewRecorder()
	handler.RunMaintenanceJob(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"updated":1`) {
		t.Fatalf("expected summary body, got %s", rec.Body.String())
	}
}

func TestListClawbotSummariesAndOverview(t *testing.T) {
	t.Parallel()

	clawbotHandler := NewOpsHandler(stubOpsManager{})
	clawbotReq := httptest.NewRequest(http.MethodGet, "/api/v1/ops/clawbots", nil)
	clawbotRec := httptest.NewRecorder()
	clawbotHandler.ListClawbotSummaries(clawbotRec, clawbotReq)
	if clawbotRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", clawbotRec.Code, clawbotRec.Body.String())
	}
	if !strings.Contains(clawbotRec.Body.String(), `"total":1`) {
		t.Fatalf("expected clawbot payload, got %s", clawbotRec.Body.String())
	}

	overviewReq := httptest.NewRequest(http.MethodGet, "/api/v1/ops/maintenance", nil)
	overviewRec := httptest.NewRecorder()
	clawbotHandler.MaintenanceOverview(overviewRec, overviewReq)
	if overviewRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", overviewRec.Code, overviewRec.Body.String())
	}

	errorHandler := NewOpsHandler(stubOpsManager{
		clawbotErr:  errors.New("clawbot list failed"),
		overviewErr: errors.New("overview failed"),
	})

	clawbotErrRec := httptest.NewRecorder()
	errorHandler.ListClawbotSummaries(clawbotErrRec, clawbotReq)
	if clawbotErrRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", clawbotErrRec.Code, clawbotErrRec.Body.String())
	}

	overviewErrRec := httptest.NewRecorder()
	errorHandler.MaintenanceOverview(overviewErrRec, overviewReq)
	if overviewErrRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", overviewErrRec.Code, overviewErrRec.Body.String())
	}
}
