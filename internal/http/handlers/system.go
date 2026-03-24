package handlers

import (
	"context"
	"net/http"
	"time"

	"clawmem/internal/version"
)

type SystemHandler struct {
	ready func(context.Context) error
}

func NewSystemHandler(ready func(context.Context) error) *SystemHandler {
	return &SystemHandler{ready: ready}
}

func (h *SystemHandler) Healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *SystemHandler) Readyz(w http.ResponseWriter, _ *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := h.ready(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *SystemHandler) Version(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, version.Get())
}
