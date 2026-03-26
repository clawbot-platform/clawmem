package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"clawmem/internal/domain/memory"
	"clawmem/internal/domain/replay"
	"clawmem/internal/domain/trust"
	"clawmem/internal/platform/store"
	memoryservice "clawmem/internal/services/memory"
	replayservice "clawmem/internal/services/replay"
	trustservice "clawmem/internal/services/trust"
)

type MemoryManager interface {
	Create(context.Context, memoryservice.CreateInput) (memory.MemoryRecord, error)
	CreateBatch(context.Context, []memoryservice.CreateInput) ([]memory.MemoryRecord, error)
	List(context.Context, memory.MemoryQuery) (memory.MemoryQueryResult, error)
	Get(context.Context, string) (memory.MemoryRecord, error)
	Summary(context.Context) (memory.Summary, error)
}

type ReplayManager interface {
	Store(context.Context, replayservice.StoreInput) (replay.ReplayMemoryRecord, error)
	List(context.Context, string) ([]replay.ReplayMemoryRecord, error)
}

type TrustManager interface {
	Store(context.Context, trustservice.StoreInput) (trust.TrustMemoryRecord, error)
	List(context.Context, string) ([]trust.TrustMemoryRecord, error)
}

type MemoryHandler struct {
	memory MemoryManager
	replay ReplayManager
	trust  TrustManager
}

func NewMemoryHandler(memory MemoryManager, replay ReplayManager, trust TrustManager) *MemoryHandler {
	return &MemoryHandler{
		memory: memory,
		replay: replay,
		trust:  trust,
	}
}

func (h *MemoryHandler) ListMemories(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.memory.List(r.Context(), memory.MemoryQuery{
		Namespace:   strings.TrimSpace(r.URL.Query().Get("namespace")),
		ProjectID:   strings.TrimSpace(r.URL.Query().Get("project_id")),
		Environment: strings.TrimSpace(r.URL.Query().Get("environment")),
		ClawbotID:   strings.TrimSpace(r.URL.Query().Get("clawbot_id")),
		SessionID:   strings.TrimSpace(r.URL.Query().Get("session_id")),
		MemoryType:  memory.MemoryType(strings.TrimSpace(r.URL.Query().Get("memory_type"))),
		ScenarioID:  strings.TrimSpace(r.URL.Query().Get("scenario_id")),
		SourceRef:   strings.TrimSpace(r.URL.Query().Get("source_ref")),
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *MemoryHandler) CreateMemory(w http.ResponseWriter, r *http.Request) {
	var input memoryservice.CreateInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(input.IdempotencyKey) == "" {
		input.IdempotencyKey = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	}

	record, err := h.memory.Create(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, record)
}

func (h *MemoryHandler) CreateBatchMemories(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Records []memoryservice.CreateInput `json:"records"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(payload.Records) == 0 {
		writeError(w, http.StatusBadRequest, "records is required")
		return
	}

	records, err := h.memory.CreateBatch(r.Context(), payload.Records)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"records": records,
		"total":   len(records),
	})
}

func (h *MemoryHandler) GetMemory(w http.ResponseWriter, r *http.Request) {
	record, err := h.memory.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (h *MemoryHandler) ListReplay(w http.ResponseWriter, r *http.Request) {
	records, err := h.replay.List(r.Context(), strings.TrimSpace(r.URL.Query().Get("scenario_id")))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"records": records,
		"total":   len(records),
	})
}

func (h *MemoryHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	summary, err := h.memory.Summary(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var builder strings.Builder
	builder.WriteString("# TYPE clawmem_records_total gauge\n")
	builder.WriteString("clawmem_records_total " + strconv.Itoa(summary.TotalRecords) + "\n")
	builder.WriteString("# TYPE clawmem_records_pinned_total gauge\n")
	builder.WriteString("clawmem_records_pinned_total " + strconv.Itoa(summary.PinnedRecords) + "\n")
	builder.WriteString("# TYPE clawmem_records_expiring_total gauge\n")
	builder.WriteString("clawmem_records_expiring_total " + strconv.Itoa(summary.ExpiringRecords) + "\n")
	for memoryType, count := range summary.RecordsByType {
		builder.WriteString(`clawmem_records_by_type{memory_type="` + memoryType + `"}` + " " + strconv.Itoa(count) + "\n")
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(builder.String()))
}

func (h *MemoryHandler) CreateReplay(w http.ResponseWriter, r *http.Request) {
	var input replayservice.StoreInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	record, err := h.replay.Store(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, record)
}

func (h *MemoryHandler) ListTrust(w http.ResponseWriter, r *http.Request) {
	records, err := h.trust.List(r.Context(), strings.TrimSpace(r.URL.Query().Get("scenario_id")))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"records": records,
		"total":   len(records),
	})
}

func (h *MemoryHandler) CreateTrust(w http.ResponseWriter, r *http.Request) {
	var input trustservice.StoreInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	record, err := h.trust.Store(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, record)
}
