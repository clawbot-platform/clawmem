package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"clawmem/internal/domain/scopedmemory"
	scopedservice "clawmem/internal/services/scopedmemory"
)

func TestScopedMemoryHandlerContextAndQueryErrorBranches(t *testing.T) {
	t.Parallel()

	handler := NewScopedMemoryHandler(&stubScopedMemoryManager{
		fetchCompactContextFn: func(context.Context, scopedmemory.Namespace) (scopedmemory.CompactContext, error) {
			return scopedmemory.CompactContext{}, errors.New("context fetch failed")
		},
		listRecordsFn: func(context.Context, scopedmemory.Query) (scopedmemory.QueryResult, error) {
			return scopedmemory.QueryResult{}, errors.New("list failed")
		},
		listSnapshotsFn: func(context.Context, scopedmemory.SnapshotQuery) (scopedmemory.SnapshotQueryResult, error) {
			return scopedmemory.SnapshotQueryResult{}, errors.New("snapshot list failed")
		},
	})

	badJSON := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/context", strings.NewReader(`{"namespace":`))
	badJSON.Header.Set("Content-Type", "application/json")
	badJSONRec := httptest.NewRecorder()
	handler.Context(badJSONRec, badJSON)
	if badJSONRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad json, got %d body=%s", badJSONRec.Code, badJSONRec.Body.String())
	}

	invalidNamespace := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/context", strings.NewReader(`{"namespace":{"repo_namespace":"ach-trust-lab"}}`))
	invalidNamespace.Header.Set("Content-Type", "application/json")
	invalidNamespaceRec := httptest.NewRecorder()
	handler.Context(invalidNamespaceRec, invalidNamespace)
	if invalidNamespaceRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid namespace, got %d body=%s", invalidNamespaceRec.Code, invalidNamespaceRec.Body.String())
	}

	serviceErrorReq := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/context", strings.NewReader(`{"namespace":{"repo_namespace":"ach-trust-lab","run_namespace":"weekrun-1"}}`))
	serviceErrorReq.Header.Set("Content-Type", "application/json")
	serviceErrorRec := httptest.NewRecorder()
	handler.Context(serviceErrorRec, serviceErrorReq)
	if serviceErrorRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for context fetch failure, got %d body=%s", serviceErrorRec.Code, serviceErrorRec.Body.String())
	}

	queryBadPagination := httptest.NewRequest(http.MethodGet, "/api/v1/scoped-memory/query?limit=bad", nil)
	queryBadPaginationRec := httptest.NewRecorder()
	handler.Query(queryBadPaginationRec, queryBadPagination)
	if queryBadPaginationRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad pagination, got %d body=%s", queryBadPaginationRec.Code, queryBadPaginationRec.Body.String())
	}

	queryListErr := httptest.NewRequest(http.MethodGet, "/api/v1/scoped-memory/query?repo_namespace=ach-trust-lab&run_namespace=weekrun-1", nil)
	queryListErrRec := httptest.NewRecorder()
	handler.Query(queryListErrRec, queryListErr)
	if queryListErrRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for list records failure, got %d body=%s", queryListErrRec.Code, queryListErrRec.Body.String())
	}

	querySnapshotsErr := httptest.NewRequest(http.MethodGet, "/api/v1/scoped-memory/query?kind=snapshots&repo_namespace=ach-trust-lab&run_namespace=weekrun-1", nil)
	querySnapshotsErrRec := httptest.NewRecorder()
	handler.Query(querySnapshotsErrRec, querySnapshotsErr)
	if querySnapshotsErrRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for list snapshots failure, got %d body=%s", querySnapshotsErrRec.Code, querySnapshotsErrRec.Body.String())
	}
}

func TestScopedMemoryHandlerSnapshotAndStatusValidationBranches(t *testing.T) {
	t.Parallel()

	handler := NewScopedMemoryHandler(&stubScopedMemoryManager{
		createSnapshotFn: func(context.Context, scopedservice.CreateSnapshotInput) (scopedmemory.Snapshot, error) {
			return scopedmemory.Snapshot{}, errors.New("snapshot create failed")
		},
		updateRecordStatusFn: func(context.Context, string, scopedservice.UpdateRecordStatusInput) (scopedmemory.Record, error) {
			return scopedmemory.Record{}, nil
		},
	})

	createBadJSON := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/snapshots", strings.NewReader(`{"namespace":`))
	createBadJSON.Header.Set("Content-Type", "application/json")
	createBadJSONRec := httptest.NewRecorder()
	handler.CreateSnapshot(createBadJSONRec, createBadJSON)
	if createBadJSONRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for snapshot bad json, got %d body=%s", createBadJSONRec.Code, createBadJSONRec.Body.String())
	}

	createServiceErr := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/snapshots", strings.NewReader(`{"namespace":{"repo_namespace":"ach-trust-lab","run_namespace":"weekrun-1"}}`))
	createServiceErr.Header.Set("Content-Type", "application/json")
	createServiceErrRec := httptest.NewRecorder()
	handler.CreateSnapshot(createServiceErrRec, createServiceErr)
	if createServiceErrRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for snapshot service error, got %d body=%s", createServiceErrRec.Code, createServiceErrRec.Body.String())
	}

	getMissingIDReq := httptest.NewRequest(http.MethodGet, "/api/v1/scoped-memory/snapshots/", nil)
	getMissingIDRec := httptest.NewRecorder()
	handler.GetSnapshot(getMissingIDRec, getMissingIDReq)
	if getMissingIDRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when snapshot_id missing, got %d body=%s", getMissingIDRec.Code, getMissingIDRec.Body.String())
	}

	updateMissingIDReq := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/records//status", strings.NewReader(`{"status":"open"}`))
	updateMissingIDReq.Header.Set("Content-Type", "application/json")
	updateMissingIDRec := httptest.NewRecorder()
	handler.UpdateRecordStatus(updateMissingIDRec, updateMissingIDReq)
	if updateMissingIDRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when record_id missing, got %d body=%s", updateMissingIDRec.Code, updateMissingIDRec.Body.String())
	}
}

type stubScopedMemoryManager struct {
	getRecordFn           func(context.Context, string) (scopedmemory.Record, error)
	listRecordsFn         func(context.Context, scopedmemory.Query) (scopedmemory.QueryResult, error)
	fetchCompactContextFn func(context.Context, scopedmemory.Namespace) (scopedmemory.CompactContext, error)
	persistNotesFn        func(context.Context, scopedmemory.Namespace, scopedservice.PersistNotesInput) (scopedservice.PersistNotesResult, error)
	updateRecordStatusFn  func(context.Context, string, scopedservice.UpdateRecordStatusInput) (scopedmemory.Record, error)
	createSnapshotFn      func(context.Context, scopedservice.CreateSnapshotInput) (scopedmemory.Snapshot, error)
	getSnapshotFn         func(context.Context, string) (scopedmemory.Snapshot, error)
	listSnapshotsFn       func(context.Context, scopedmemory.SnapshotQuery) (scopedmemory.SnapshotQueryResult, error)
	exportSnapshotFn      func(context.Context, string) (scopedmemory.SnapshotExport, error)
	exportRunFn           func(context.Context, scopedmemory.Namespace) (scopedmemory.RunMemoryExport, error)
}

func (s *stubScopedMemoryManager) GetRecord(ctx context.Context, id string) (scopedmemory.Record, error) {
	if s.getRecordFn != nil {
		return s.getRecordFn(ctx, id)
	}
	return scopedmemory.Record{}, nil
}

func (s *stubScopedMemoryManager) ListRecords(ctx context.Context, query scopedmemory.Query) (scopedmemory.QueryResult, error) {
	if s.listRecordsFn != nil {
		return s.listRecordsFn(ctx, query)
	}
	return scopedmemory.QueryResult{}, nil
}

func (s *stubScopedMemoryManager) FetchCompactContext(ctx context.Context, ns scopedmemory.Namespace) (scopedmemory.CompactContext, error) {
	if s.fetchCompactContextFn != nil {
		return s.fetchCompactContextFn(ctx, ns)
	}
	return scopedmemory.CompactContext{}, nil
}

func (s *stubScopedMemoryManager) PersistNotes(ctx context.Context, ns scopedmemory.Namespace, input scopedservice.PersistNotesInput) (scopedservice.PersistNotesResult, error) {
	if s.persistNotesFn != nil {
		return s.persistNotesFn(ctx, ns, input)
	}
	return scopedservice.PersistNotesResult{}, nil
}

func (s *stubScopedMemoryManager) UpdateRecordStatus(ctx context.Context, id string, input scopedservice.UpdateRecordStatusInput) (scopedmemory.Record, error) {
	if s.updateRecordStatusFn != nil {
		return s.updateRecordStatusFn(ctx, id, input)
	}
	return scopedmemory.Record{}, nil
}

func (s *stubScopedMemoryManager) CreateSnapshot(ctx context.Context, input scopedservice.CreateSnapshotInput) (scopedmemory.Snapshot, error) {
	if s.createSnapshotFn != nil {
		return s.createSnapshotFn(ctx, input)
	}
	return scopedmemory.Snapshot{}, nil
}

func (s *stubScopedMemoryManager) GetSnapshot(ctx context.Context, id string) (scopedmemory.Snapshot, error) {
	if s.getSnapshotFn != nil {
		return s.getSnapshotFn(ctx, id)
	}
	return scopedmemory.Snapshot{}, nil
}

func (s *stubScopedMemoryManager) ListSnapshots(ctx context.Context, query scopedmemory.SnapshotQuery) (scopedmemory.SnapshotQueryResult, error) {
	if s.listSnapshotsFn != nil {
		return s.listSnapshotsFn(ctx, query)
	}
	return scopedmemory.SnapshotQueryResult{}, nil
}

func (s *stubScopedMemoryManager) ExportSnapshot(ctx context.Context, id string) (scopedmemory.SnapshotExport, error) {
	if s.exportSnapshotFn != nil {
		return s.exportSnapshotFn(ctx, id)
	}
	return scopedmemory.SnapshotExport{}, nil
}

func (s *stubScopedMemoryManager) ExportRun(ctx context.Context, ns scopedmemory.Namespace) (scopedmemory.RunMemoryExport, error) {
	if s.exportRunFn != nil {
		return s.exportRunFn(ctx, ns)
	}
	return scopedmemory.RunMemoryExport{}, nil
}
