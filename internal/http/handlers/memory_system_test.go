package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"clawmem/internal/domain/memory"
	"clawmem/internal/domain/replay"
	"clawmem/internal/domain/trust"
	"clawmem/internal/platform/store"
	memoryservice "clawmem/internal/services/memory"
	replayservice "clawmem/internal/services/replay"
	trustservice "clawmem/internal/services/trust"
	"clawmem/internal/version"
)

func TestCreateBatchMemoriesValidationAndErrors(t *testing.T) {
	t.Parallel()

	handler := NewMemoryHandler(stubMemoryManager{}, stubReplayManager{}, stubTrustManager{})

	tests := []struct {
		name string
		body string
		code int
	}{
		{name: "malformed json", body: `{"records":`, code: http.StatusBadRequest},
		{name: "empty records", body: `{"records":[]}`, code: http.StatusBadRequest},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/memories/batch", strings.NewReader(testCase.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			handler.CreateBatchMemories(rec, req)
			if rec.Code != testCase.code {
				t.Fatalf("expected %d, got %d body=%s", testCase.code, rec.Code, rec.Body.String())
			}
		})
	}

	errHandler := NewMemoryHandler(stubMemoryManager{createBatchErr: errors.New("batch failed")}, stubReplayManager{}, stubTrustManager{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/memories/batch", strings.NewReader(`{"records":[{"memory_type":"benchmark_note","scope":"benchmark","source_ref":"note-1","summary":"Batch note."}]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	errHandler.CreateBatchMemories(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetMemorySuccessAndErrors(t *testing.T) {
	t.Parallel()

	record := memory.MemoryRecord{ID: "mem-1", Summary: "Stored memory."}
	handler := NewMemoryHandler(stubMemoryManager{getRecord: record}, stubReplayManager{}, stubTrustManager{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/memories/mem-1", nil)
	req.SetPathValue("id", "mem-1")
	rec := httptest.NewRecorder()
	handler.GetMemory(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	notFound := NewMemoryHandler(stubMemoryManager{getErr: store.ErrNotFound}, stubReplayManager{}, stubTrustManager{})
	notFoundRec := httptest.NewRecorder()
	notFound.GetMemory(notFoundRec, req)
	if notFoundRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", notFoundRec.Code, notFoundRec.Body.String())
	}

	badHandler := NewMemoryHandler(stubMemoryManager{getErr: errors.New("bad id")}, stubReplayManager{}, stubTrustManager{})
	badRec := httptest.NewRecorder()
	badHandler.GetMemory(badRec, req)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", badRec.Code, badRec.Body.String())
	}
}

func TestListReplayAndTrustEmptyAndError(t *testing.T) {
	t.Parallel()

	handler := NewMemoryHandler(stubMemoryManager{}, stubReplayManager{}, stubTrustManager{})

	replayReq := httptest.NewRequest(http.MethodGet, "/api/v1/replay?scenario_id=scn-1", nil)
	replayRec := httptest.NewRecorder()
	handler.ListReplay(replayRec, replayReq)
	if replayRec.Code != http.StatusOK || !strings.Contains(replayRec.Body.String(), `"total":0`) {
		t.Fatalf("expected empty replay list, got %d body=%s", replayRec.Code, replayRec.Body.String())
	}

	trustReq := httptest.NewRequest(http.MethodGet, "/api/v1/trust?scenario_id=scn-1", nil)
	trustRec := httptest.NewRecorder()
	handler.ListTrust(trustRec, trustReq)
	if trustRec.Code != http.StatusOK || !strings.Contains(trustRec.Body.String(), `"total":0`) {
		t.Fatalf("expected empty trust list, got %d body=%s", trustRec.Code, trustRec.Body.String())
	}

	replayErrHandler := NewMemoryHandler(stubMemoryManager{}, stubReplayManager{listErr: errors.New("replay unavailable")}, stubTrustManager{})
	replayErrRec := httptest.NewRecorder()
	replayErrHandler.ListReplay(replayErrRec, replayReq)
	if replayErrRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", replayErrRec.Code, replayErrRec.Body.String())
	}

	trustErrHandler := NewMemoryHandler(stubMemoryManager{}, stubReplayManager{}, stubTrustManager{listErr: errors.New("trust unavailable")})
	trustErrRec := httptest.NewRecorder()
	trustErrHandler.ListTrust(trustErrRec, trustReq)
	if trustErrRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", trustErrRec.Code, trustErrRec.Body.String())
	}
}

func TestSystemHandlerReadyzAndVersion(t *testing.T) {
	t.Parallel()

	readyHandler := NewSystemHandler(func(context.Context) error { return nil })
	readyReq := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	readyRec := httptest.NewRecorder()
	readyHandler.Readyz(readyRec, readyReq)
	if readyRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", readyRec.Code, readyRec.Body.String())
	}

	notReadyHandler := NewSystemHandler(func(context.Context) error { return errors.New("store unavailable") })
	notReadyRec := httptest.NewRecorder()
	notReadyHandler.Readyz(notReadyRec, readyReq)
	if notReadyRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", notReadyRec.Code, notReadyRec.Body.String())
	}

	versionReq := httptest.NewRequest(http.MethodGet, "/version", nil)
	versionRec := httptest.NewRecorder()
	readyHandler.Version(versionRec, versionReq)
	if versionRec.Code != http.StatusOK || !strings.Contains(versionRec.Body.String(), version.Get().Version) {
		t.Fatalf("expected version payload, got %d body=%s", versionRec.Code, versionRec.Body.String())
	}
}

type stubMemoryManager struct {
	createBatchErr error
	getRecord      memory.MemoryRecord
	getErr         error
}

func (s stubMemoryManager) Create(context.Context, memoryservice.CreateInput) (memory.MemoryRecord, error) {
	return memory.MemoryRecord{}, nil
}

func (s stubMemoryManager) CreateBatch(_ context.Context, inputs []memoryservice.CreateInput) ([]memory.MemoryRecord, error) {
	if s.createBatchErr != nil {
		return nil, s.createBatchErr
	}
	records := make([]memory.MemoryRecord, 0, len(inputs))
	for index := range inputs {
		records = append(records, memory.MemoryRecord{ID: "mem-batch", Summary: inputs[index].Summary})
	}
	return records, nil
}

func (s stubMemoryManager) List(context.Context, memory.MemoryQuery) (memory.MemoryQueryResult, error) {
	return memory.MemoryQueryResult{}, nil
}

func (s stubMemoryManager) Get(context.Context, string) (memory.MemoryRecord, error) {
	if s.getErr != nil {
		return memory.MemoryRecord{}, s.getErr
	}
	return s.getRecord, nil
}

func (s stubMemoryManager) Summary(context.Context) (memory.Summary, error) {
	return memory.Summary{}, nil
}

type stubReplayManager struct {
	listErr error
}

func (s stubReplayManager) Store(context.Context, replayservice.StoreInput) (replay.ReplayMemoryRecord, error) {
	return replay.ReplayMemoryRecord{}, nil
}

func (s stubReplayManager) List(context.Context, string) ([]replay.ReplayMemoryRecord, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return []replay.ReplayMemoryRecord{}, nil
}

type stubTrustManager struct {
	listErr error
}

func (s stubTrustManager) Store(context.Context, trustservice.StoreInput) (trust.TrustMemoryRecord, error) {
	return trust.TrustMemoryRecord{}, nil
}

func (s stubTrustManager) List(context.Context, string) ([]trust.TrustMemoryRecord, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return []trust.TrustMemoryRecord{}, nil
}
