package routes

import (
	"log/slog"
	"net/http"

	"clawmem/internal/http/handlers"
	"clawmem/internal/http/middleware"
)

func New(system *handlers.SystemHandler, memory *handlers.MemoryHandler, scoped *handlers.ScopedMemoryHandler, ops *handlers.OpsHandler, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", system.Healthz)
	mux.HandleFunc("GET /readyz", system.Readyz)
	mux.HandleFunc("GET /version", system.Version)
	mux.HandleFunc("GET /metrics", memory.Metrics)

	mux.HandleFunc("GET /api/v1/memories", memory.ListMemories)
	mux.HandleFunc("POST /api/v1/memories", memory.CreateMemory)
	mux.HandleFunc("POST /api/v1/memories/batch", memory.CreateBatchMemories)
	mux.HandleFunc("GET /api/v1/memories/{id}", memory.GetMemory)

	mux.HandleFunc("GET /api/v1/replay", memory.ListReplay)
	mux.HandleFunc("POST /api/v1/replay", memory.CreateReplay)

	mux.HandleFunc("GET /api/v1/trust", memory.ListTrust)
	mux.HandleFunc("POST /api/v1/trust", memory.CreateTrust)

	mux.HandleFunc("POST /api/v1/scoped-memory/context", scoped.Context)
	mux.HandleFunc("POST /api/v1/scoped-memory/notes", scoped.Notes)
	mux.HandleFunc("POST /api/v1/scoped-memory/snapshots", scoped.CreateSnapshot)
	mux.HandleFunc("GET /api/v1/scoped-memory/snapshots/{snapshot_id}", scoped.GetSnapshot)
	mux.HandleFunc("GET /api/v1/scoped-memory/query", scoped.Query)

	mux.HandleFunc("GET /api/v1/ops/namespaces", ops.ListNamespaceSummaries)
	mux.HandleFunc("GET /api/v1/ops/clawbots", ops.ListClawbotSummaries)
	mux.HandleFunc("GET /api/v1/ops/maintenance", ops.MaintenanceOverview)
	mux.HandleFunc("POST /api/v1/ops/maintenance/{job}/run", ops.RunMaintenanceJob)

	return middleware.Chain(mux, middleware.Recoverer(logger), middleware.RequestLogger(logger))
}
