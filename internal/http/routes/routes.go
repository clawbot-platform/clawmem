package routes

import (
	"log/slog"
	"net/http"

	"clawmem/internal/http/handlers"
	"clawmem/internal/http/middleware"
)

func New(system *handlers.SystemHandler, memory *handlers.MemoryHandler, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", system.Healthz)
	mux.HandleFunc("GET /readyz", system.Readyz)
	mux.HandleFunc("GET /version", system.Version)

	mux.HandleFunc("GET /api/v1/memories", memory.ListMemories)
	mux.HandleFunc("POST /api/v1/memories", memory.CreateMemory)
	mux.HandleFunc("GET /api/v1/memories/{id}", memory.GetMemory)

	mux.HandleFunc("GET /api/v1/replay", memory.ListReplay)
	mux.HandleFunc("POST /api/v1/replay", memory.CreateReplay)

	mux.HandleFunc("GET /api/v1/trust", memory.ListTrust)
	mux.HandleFunc("POST /api/v1/trust", memory.CreateTrust)

	return middleware.Chain(mux, middleware.Recoverer(logger), middleware.RequestLogger(logger))
}
