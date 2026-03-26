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
	opsservice "clawmem/internal/services/ops"
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
		"project_id":"sample-project",
		"environment":"test",
		"clawbot_id":"shared",
		"scenario_id":"sample-order-review",
		"source_ref":"sample-pack-001",
		"summary":"Scenario summary memory.",
		"metadata":{"pack_id":"sample-pack"},
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

func TestListMemoriesPaginationAndBatchCreate(t *testing.T) {
	router := newRouter(t)

	batchReq := httptest.NewRequest(http.MethodPost, "/api/v1/memories/batch", strings.NewReader(`{
		"records":[
			{
				"project_id":"sample-project",
				"environment":"test",
				"clawbot_id":"shared",
				"memory_type":"scenario_summary",
				"scope":"platform",
				"source_ref":"source-1",
				"summary":"Summary one."
			},
			{
				"project_id":"sample-project",
				"environment":"test",
				"clawbot_id":"shared",
				"memory_type":"scenario_summary",
				"scope":"platform",
				"source_ref":"source-2",
				"summary":"Summary two."
			}
		]
	}`))
	batchReq.Header.Set("Content-Type", "application/json")
	batchRec := httptest.NewRecorder()
	router.ServeHTTP(batchRec, batchReq)
	if batchRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", batchRec.Code, batchRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/memories?project_id=sample-project&limit=1&offset=1", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", listRec.Code, listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), `"has_more":false`) {
		t.Fatalf("expected paginated response body=%s", listRec.Body.String())
	}
}

func TestCreateMemorySupportsIdempotencyKeyAndMetrics(t *testing.T) {
	router := newRouter(t)

	for index := 0; index < 2; index++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/memories", strings.NewReader(`{
			"project_id":"sample-project",
			"environment":"test",
			"clawbot_id":"shared",
			"memory_type":"benchmark_note",
			"scope":"benchmark",
			"source_ref":"note-1",
			"summary":"Idempotent note."
		}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "idem-note-1")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
		}
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/memories?project_id=sample-project", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if strings.Count(listRec.Body.String(), `"summary":"Idempotent note."`) != 1 {
		t.Fatalf("expected one idempotent record, got %s", listRec.Body.String())
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	router.ServeHTTP(metricsRec, metricsReq)
	if metricsRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", metricsRec.Code)
	}
	if !strings.Contains(metricsRec.Body.String(), "clawmem_records_total") {
		t.Fatalf("expected metrics output, got %s", metricsRec.Body.String())
	}
}

func TestListMemoriesRejectsInvalidPagination(t *testing.T) {
	router := newRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/memories?limit=bad", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCreateTrust(t *testing.T) {
	router := newRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trust", strings.NewReader(`{
		"scenario_id":"sample-order-review",
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

func TestOpsEndpoints(t *testing.T) {
	router := newRouter(t)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/memories", strings.NewReader(`{
		"project_id":"sample-project",
		"environment":"test",
		"clawbot_id":"bot-a",
		"session_id":"session-1",
		"memory_type":"benchmark_note",
		"scope":"benchmark",
		"source_ref":"note-ops-1",
		"summary":"Ops note.",
		"importance":15
	}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createRec.Code, createRec.Body.String())
	}

	for _, path := range []string{
		"/api/v1/ops/namespaces?project_id=sample-project",
		"/api/v1/ops/clawbots?project_id=sample-project",
		"/api/v1/ops/maintenance",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestRunMaintenanceJobEndpoint(t *testing.T) {
	router := newRouter(t)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/memories", strings.NewReader(`{
		"project_id":"sample-project",
		"environment":"test",
		"clawbot_id":"bot-a",
		"memory_type":"benchmark_note",
		"scope":"benchmark",
		"source_ref":"note-decay",
		"summary":"Eligible for decay.",
		"importance":10
	}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createRec.Code, createRec.Body.String())
	}

	runReq := httptest.NewRequest(http.MethodPost, "/api/v1/ops/maintenance/decay_update/run", nil)
	runRec := httptest.NewRecorder()
	router.ServeHTTP(runRec, runReq)
	if runRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", runRec.Code, runRec.Body.String())
	}
	if !strings.Contains(runRec.Body.String(), `"job_type":"decay_update"`) {
		t.Fatalf("expected decay response, got %s", runRec.Body.String())
	}

	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/ops/maintenance/unknown/run", nil)
	badRec := httptest.NewRecorder()
	router.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", badRec.Code, badRec.Body.String())
	}
}

func newRouter(t *testing.T) http.Handler {
	t.Helper()

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
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	return routes.New(systemHandler, memoryHandler, opsHandler, logger)
}
