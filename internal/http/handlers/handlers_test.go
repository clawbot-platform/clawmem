package handlers_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"clawmem/internal/http/handlers"
	"clawmem/internal/http/routes"
	"clawmem/internal/platform/store"
	memoryservice "clawmem/internal/services/memory"
	replayservice "clawmem/internal/services/replay"
	trustservice "clawmem/internal/services/trust"
)

func TestHealthz(t *testing.T) {
	router := newRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCreateAndGetMemory(t *testing.T) {
	router := newRouter(t)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/memories", strings.NewReader(`{
		"memory_type":"scenario_summary",
		"scope":"platform",
		"scenario_id":"starter-mandate-review",
		"source_id":"scenario-pack-001",
		"summary":"Scenario summary memory.",
		"metadata":{"pack_id":"starter-pack"},
		"tags":["scenario"]
	}`))
	createReq.Header.Set("Content-Type", "application/json")

	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createRec.Code, createRec.Body.String())
	}

	body := createRec.Body.String()
	start := strings.Index(body, `"id":"`)
	if start == -1 {
		t.Fatalf("expected id in response body: %s", body)
	}
	id := body[start+6:]
	id = id[:strings.Index(id, `"`)]

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/memories/"+id, nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", getRec.Code, getRec.Body.String())
	}
}

func TestCreateReplayRejectsBadPayload(t *testing.T) {
	router := newRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/replay", strings.NewReader(`{"scenario_id":""}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateTrust(t *testing.T) {
	router := newRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trust", strings.NewReader(`{
		"scenario_id":"starter-mandate-review",
		"source_id":"trust-artifact-002",
		"summary":"Trust summary.",
		"artifact_family":"mandate",
		"artifact_type":"mandate_artifact"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func newRouter(t *testing.T) http.Handler {
	t.Helper()

	fileStore, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	memorySvc := memoryservice.NewService(fileStore)
	replaySvc := replayservice.NewService(memorySvc)
	trustSvc := trustservice.NewService(memorySvc)

	systemHandler := handlers.NewSystemHandler(func(ctx context.Context) error {
		_, err := fileStore.Count(ctx)
		return err
	})
	memoryHandler := handlers.NewMemoryHandler(memorySvc, replaySvc, trustSvc)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	return routes.New(systemHandler, memoryHandler, logger)
}
