package handlers

import (
	"context"
	"errors"
	"net/http"
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
	List(context.Context, memory.MemoryQuery) (memory.MemoryQueryResult, error)
	Get(context.Context, string) (memory.MemoryRecord, error)
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
	result, err := h.memory.List(r.Context(), memory.MemoryQuery{
		MemoryType: memory.MemoryType(strings.TrimSpace(r.URL.Query().Get("memory_type"))),
		ScenarioID: strings.TrimSpace(r.URL.Query().Get("scenario_id")),
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

	record, err := h.memory.Create(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, record)
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
