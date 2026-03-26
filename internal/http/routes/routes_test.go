package routes

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"clawmem/internal/http/handlers"
	"clawmem/internal/platform/store"
	memoryservice "clawmem/internal/services/memory"
	opsservice "clawmem/internal/services/ops"
	replayservice "clawmem/internal/services/replay"
	trustservice "clawmem/internal/services/trust"
)

func TestRouterServesSystemAndMemoryEndpoints(t *testing.T) {
	t.Parallel()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	memorySvc := memoryservice.NewService(fileStore)
	opsSvc := opsservice.NewService(memorySvc)
	replaySvc := replayservice.NewService(memorySvc)
	trustSvc := trustservice.NewService(memorySvc)
	systemHandler := handlers.NewSystemHandler(func(ctx context.Context) error {
		_, err := fileStore.Count(ctx)
		return err
	})
	memoryHandler := handlers.NewMemoryHandler(memorySvc, replaySvc, trustSvc)
	opsHandler := handlers.NewOpsHandler(opsSvc)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	router := New(systemHandler, memoryHandler, opsHandler, logger)

	for _, path := range []string{"/healthz", "/readyz", "/version", "/metrics", "/api/v1/memories", "/api/v1/ops/namespaces", "/api/v1/ops/clawbots", "/api/v1/ops/maintenance"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}
