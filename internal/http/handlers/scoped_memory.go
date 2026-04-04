package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"clawmem/internal/domain/scopedmemory"
	"clawmem/internal/platform/store"
	scopedservice "clawmem/internal/services/scopedmemory"
)

type ScopedMemoryManager interface {
	ListRecords(context.Context, scopedmemory.Query) (scopedmemory.QueryResult, error)
	FetchCompactContext(context.Context, scopedmemory.Namespace) (scopedmemory.CompactContext, error)
	PersistNotes(context.Context, scopedmemory.Namespace, scopedservice.PersistNotesInput) (scopedservice.PersistNotesResult, error)
	CreateSnapshot(context.Context, scopedservice.CreateSnapshotInput) (scopedmemory.Snapshot, error)
	GetSnapshot(context.Context, string) (scopedmemory.Snapshot, error)
	ListSnapshots(context.Context, scopedmemory.SnapshotQuery) (scopedmemory.SnapshotQueryResult, error)
	ExportSnapshot(context.Context, string) (scopedmemory.SnapshotExport, error)
	ExportRun(context.Context, scopedmemory.Namespace) (scopedmemory.RunMemoryExport, error)
}

type ScopedMemoryHandler struct {
	scoped ScopedMemoryManager
}

func NewScopedMemoryHandler(scoped ScopedMemoryManager) *ScopedMemoryHandler {
	return &ScopedMemoryHandler{scoped: scoped}
}

func (h *ScopedMemoryHandler) Context(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Namespace scopedmemory.Namespace `json:"namespace"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	payload.Namespace = scopedmemory.NormalizeNamespace(payload.Namespace)
	if err := payload.Namespace.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, err := h.scoped.FetchCompactContext(r.Context(), payload.Namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": ctx})
}

func (h *ScopedMemoryHandler) Notes(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Namespace scopedmemory.Namespace          `json:"namespace"`
		Input     scopedservice.PersistNotesInput `json:"input"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	payload.Namespace = scopedmemory.NormalizeNamespace(payload.Namespace)
	if err := payload.Namespace.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.scoped.PersistNotes(r.Context(), payload.Namespace, payload.Input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (h *ScopedMemoryHandler) CreateSnapshot(w http.ResponseWriter, r *http.Request) {
	var input scopedservice.CreateSnapshotInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	input.Namespace = scopedmemory.NormalizeNamespace(input.Namespace)
	if err := input.Namespace.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	snapshot, err := h.scoped.CreateSnapshot(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": snapshot})
}

func (h *ScopedMemoryHandler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshotID := strings.TrimSpace(r.PathValue("snapshot_id"))
	if snapshotID == "" {
		writeError(w, http.StatusBadRequest, "snapshot_id is required")
		return
	}

	includeRecords := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("include_records")), "true")
	if includeRecords {
		export, err := h.scoped.ExportSnapshot(r.Context(), snapshotID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": export})
		return
	}

	snapshot, err := h.scoped.GetSnapshot(r.Context(), snapshotID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": snapshot})
}

func (h *ScopedMemoryHandler) Query(w http.ResponseWriter, r *http.Request) {
	query := scopedmemory.Query{
		RepoNamespace:  strings.TrimSpace(r.URL.Query().Get("repo_namespace")),
		RunNamespace:   strings.TrimSpace(r.URL.Query().Get("run_namespace")),
		CycleNamespace: strings.TrimSpace(r.URL.Query().Get("cycle_namespace")),
		AgentNamespace: strings.TrimSpace(r.URL.Query().Get("agent_namespace")),
		MemoryClass:    scopedmemory.MemoryClass(strings.TrimSpace(r.URL.Query().Get("memory_class"))),
		Status:         scopedmemory.Status(strings.TrimSpace(r.URL.Query().Get("status"))),
	}

	limit, offset, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	query.Limit = limit
	query.Offset = offset

	exportMode := strings.TrimSpace(r.URL.Query().Get("export"))
	if strings.EqualFold(exportMode, "run") {
		ns := scopedmemory.Namespace{
			RepoNamespace:  query.RepoNamespace,
			RunNamespace:   query.RunNamespace,
			CycleNamespace: query.CycleNamespace,
		}
		ns = scopedmemory.NormalizeNamespace(ns)
		if err := ns.Validate(); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		export, err := h.scoped.ExportRun(r.Context(), ns)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": export})
		return
	}

	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	if strings.EqualFold(kind, "snapshots") {
		result, err := h.scoped.ListSnapshots(r.Context(), scopedmemory.SnapshotQuery{
			RepoNamespace:  query.RepoNamespace,
			RunNamespace:   query.RunNamespace,
			CycleNamespace: query.CycleNamespace,
			Limit:          query.Limit,
			Offset:         query.Offset,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": result})
		return
	}

	result, err := h.scoped.ListRecords(r.Context(), query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}
