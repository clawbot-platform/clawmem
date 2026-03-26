package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"clawmem/internal/domain/memory"
	opsservice "clawmem/internal/services/ops"
)

type OpsManager interface {
	NamespaceSummaries(context.Context, memory.MemoryQuery) ([]memory.NamespaceSummary, error)
	ClawbotSummaries(context.Context, memory.MemoryQuery) ([]memory.ClawbotSummary, error)
	MaintenanceOverview(context.Context) (memory.MaintenanceOverview, error)
	RunJob(context.Context, memory.MaintenanceJobType) (memory.MaintenanceJobStatus, error)
}

type OpsHandler struct {
	ops OpsManager
}

func NewOpsHandler(ops OpsManager) *OpsHandler {
	return &OpsHandler{ops: ops}
}

func (h *OpsHandler) ListNamespaceSummaries(w http.ResponseWriter, r *http.Request) {
	records, err := h.ops.NamespaceSummaries(r.Context(), parseMemoryQuery(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"records": records,
		"total":   len(records),
	})
}

func (h *OpsHandler) ListClawbotSummaries(w http.ResponseWriter, r *http.Request) {
	records, err := h.ops.ClawbotSummaries(r.Context(), parseMemoryQuery(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"records": records,
		"total":   len(records),
	})
}

func (h *OpsHandler) MaintenanceOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := h.ops.MaintenanceOverview(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

func (h *OpsHandler) RunMaintenanceJob(w http.ResponseWriter, r *http.Request) {
	jobType := memory.MaintenanceJobType(strings.TrimSpace(r.PathValue("job")))
	status, err := h.ops.RunJob(r.Context(), jobType)
	if err != nil {
		if errors.Is(err, opsservice.ErrUnsupportedJob()) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func parseMemoryQuery(r *http.Request) memory.MemoryQuery {
	return memory.MemoryQuery{
		Namespace:   strings.TrimSpace(r.URL.Query().Get("namespace")),
		ProjectID:   strings.TrimSpace(r.URL.Query().Get("project_id")),
		Environment: strings.TrimSpace(r.URL.Query().Get("environment")),
		ClawbotID:   strings.TrimSpace(r.URL.Query().Get("clawbot_id")),
		SessionID:   strings.TrimSpace(r.URL.Query().Get("session_id")),
		MemoryType:  memory.MemoryType(strings.TrimSpace(r.URL.Query().Get("memory_type"))),
		ScenarioID:  strings.TrimSpace(r.URL.Query().Get("scenario_id")),
		SourceRef:   strings.TrimSpace(r.URL.Query().Get("source_ref")),
	}
}
