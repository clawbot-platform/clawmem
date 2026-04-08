package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestScopedMemoryContextAndNotesFlow(t *testing.T) {
	router := newRouter(t)

	notesReq := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/notes", strings.NewReader(`{
		"namespace": {
			"repo_namespace":"ach-trust-lab",
			"run_namespace":"weekrun-2026-06-demo",
			"cycle_namespace":"day-1",
			"agent_namespace":"daily-summary"
		},
		"input": {
			"created_by": "week-runner",
			"prior_cycle_summaries": ["day-1 summary"],
			"carry_forward_risks": ["descriptor drift risk"],
			"unresolved_gaps": ["missing inbound burst feature"],
			"backlog_items": ["add descriptor anomaly detector"],
			"reviewer_notes": ["verify payroll rationale"],
			"note": "cycle checkpoint"
		}
	}`))
	notesReq.Header.Set("Content-Type", "application/json")
	notesRec := httptest.NewRecorder()
	router.ServeHTTP(notesRec, notesReq)
	if notesRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", notesRec.Code, notesRec.Body.String())
	}
	if !strings.Contains(notesRec.Body.String(), `"snapshot_ref":"sms-`) {
		t.Fatalf("expected snapshot_ref in response body=%s", notesRec.Body.String())
	}
	if !strings.Contains(notesRec.Body.String(), `"manifest_checksum":"`) {
		t.Fatalf("expected snapshot checksum in notes response body=%s", notesRec.Body.String())
	}

	ctxReq := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/context", strings.NewReader(`{
		"namespace": {
			"repo_namespace":"ach-trust-lab",
			"run_namespace":"weekrun-2026-06-demo",
			"cycle_namespace":"day-2",
			"agent_namespace":"policy-tuning"
		}
	}`))
	ctxReq.Header.Set("Content-Type", "application/json")
	ctxRec := httptest.NewRecorder()
	router.ServeHTTP(ctxRec, ctxReq)
	if ctxRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", ctxRec.Code, ctxRec.Body.String())
	}

	body := ctxRec.Body.String()
	for _, expected := range []string{
		`"prior_cycle_summaries":["day-1 summary"]`,
		`"carry_forward_risks":["descriptor drift risk"]`,
		`"unresolved_gaps":["missing inbound burst feature"]`,
		`"backlog_items":["add descriptor anomaly detector"]`,
		`"reviewer_notes":["verify payroll rationale"]`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %s in response body=%s", expected, body)
		}
	}
}

func TestScopedMemorySnapshotAndQueryEndpoints(t *testing.T) {
	router := newRouter(t)

	createNotesReq := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/notes", strings.NewReader(`{
		"namespace": {
			"repo_namespace":"ach-trust-lab",
			"run_namespace":"weekrun-2026-06-showcase",
			"cycle_namespace":"day-3",
			"agent_namespace":"feature-gap"
		},
		"input": {
			"created_by": "cycle-runner",
			"unresolved_gaps": ["missing sender diversity signal"],
			"reviewer_notes": ["confirm with fraud ops"],
			"snapshot_summary": "day-3 checkpoint",
			"source_artifact_id": "artifact-ach-001"
		}
	}`))
	createNotesReq.Header.Set("Content-Type", "application/json")
	createNotesRec := httptest.NewRecorder()
	router.ServeHTTP(createNotesRec, createNotesReq)
	if createNotesRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", createNotesRec.Code, createNotesRec.Body.String())
	}

	var decoded map[string]any
	if err := json.Unmarshal(createNotesRec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	data, _ := decoded["data"].(map[string]any)
	snapshotRef, _ := data["snapshot_ref"].(string)
	if strings.TrimSpace(snapshotRef) == "" {
		t.Fatalf("expected snapshot_ref in response body=%s", createNotesRec.Body.String())
	}

	getSnapReq := httptest.NewRequest(http.MethodGet, "/api/v1/scoped-memory/snapshots/"+snapshotRef, nil)
	getSnapRec := httptest.NewRecorder()
	router.ServeHTTP(getSnapRec, getSnapReq)
	if getSnapRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", getSnapRec.Code, getSnapRec.Body.String())
	}
	if !strings.Contains(getSnapRec.Body.String(), snapshotRef) {
		t.Fatalf("expected snapshot id in response body=%s", getSnapRec.Body.String())
	}

	exportSnapReq := httptest.NewRequest(http.MethodGet, "/api/v1/scoped-memory/snapshots/"+snapshotRef+"?include_records=true", nil)
	exportSnapRec := httptest.NewRecorder()
	router.ServeHTTP(exportSnapRec, exportSnapReq)
	if exportSnapRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", exportSnapRec.Code, exportSnapRec.Body.String())
	}
	if !strings.Contains(exportSnapRec.Body.String(), `"records"`) {
		t.Fatalf("expected records in snapshot export body=%s", exportSnapRec.Body.String())
	}

	queryReq := httptest.NewRequest(http.MethodGet, "/api/v1/scoped-memory/query?repo_namespace=ach-trust-lab&run_namespace=weekrun-2026-06-showcase&memory_class=unresolved_gaps&status=open&source_artifact_id=artifact-ach-001", nil)
	queryRec := httptest.NewRecorder()
	router.ServeHTTP(queryRec, queryReq)
	if queryRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", queryRec.Code, queryRec.Body.String())
	}
	if !strings.Contains(queryRec.Body.String(), `"missing sender diversity signal"`) {
		t.Fatalf("expected unresolved gap in query response body=%s", queryRec.Body.String())
	}

	var listPayload map[string]any
	if err := json.Unmarshal(queryRec.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("unmarshal query response: %v", err)
	}
	listData := listPayload["data"].(map[string]any)
	records := listData["records"].([]any)
	firstRecord := records[0].(map[string]any)
	recordID := firstRecord["id"].(string)

	updateReq := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/records/"+recordID+"/status", strings.NewReader(`{
		"status":"resolved",
		"updated_by":"reviewer",
		"reason":"validated"
	}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for status update, got %d body=%s", updateRec.Code, updateRec.Body.String())
	}
	if !strings.Contains(updateRec.Body.String(), `"status":"resolved"`) {
		t.Fatalf("expected resolved status in response body=%s", updateRec.Body.String())
	}

	exportRunReq := httptest.NewRequest(http.MethodGet, "/api/v1/scoped-memory/query?repo_namespace=ach-trust-lab&run_namespace=weekrun-2026-06-showcase&export=run", nil)
	exportRunRec := httptest.NewRecorder()
	router.ServeHTTP(exportRunRec, exportRunReq)
	if exportRunRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", exportRunRec.Code, exportRunRec.Body.String())
	}
	if !strings.Contains(exportRunRec.Body.String(), `"manifest"`) {
		t.Fatalf("expected run export manifest body=%s", exportRunRec.Body.String())
	}

	listSnapshotsReq := httptest.NewRequest(http.MethodGet, "/api/v1/scoped-memory/query?kind=snapshots&repo_namespace=ach-trust-lab&run_namespace=weekrun-2026-06-showcase", nil)
	listSnapshotsRec := httptest.NewRecorder()
	router.ServeHTTP(listSnapshotsRec, listSnapshotsReq)
	if listSnapshotsRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", listSnapshotsRec.Code, listSnapshotsRec.Body.String())
	}
	if !strings.Contains(listSnapshotsRec.Body.String(), snapshotRef) {
		t.Fatalf("expected snapshot id in query snapshots body=%s", listSnapshotsRec.Body.String())
	}
}

func TestScopedMemoryNotesValidationAndBadJSON(t *testing.T) {
	router := newRouter(t)

	badJSONReq := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/notes", strings.NewReader(`{"namespace":`))
	badJSONReq.Header.Set("Content-Type", "application/json")
	badJSONRec := httptest.NewRecorder()
	router.ServeHTTP(badJSONRec, badJSONReq)
	if badJSONRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad json, got %d body=%s", badJSONRec.Code, badJSONRec.Body.String())
	}

	missingNamespaceReq := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/notes", strings.NewReader(`{
		"namespace":{"repo_namespace":"ach-trust-lab"},
		"input":{"created_by":"runner","note":"checkpoint"}
	}`))
	missingNamespaceReq.Header.Set("Content-Type", "application/json")
	missingNamespaceRec := httptest.NewRecorder()
	router.ServeHTTP(missingNamespaceRec, missingNamespaceReq)
	if missingNamespaceRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing namespace fields, got %d body=%s", missingNamespaceRec.Code, missingNamespaceRec.Body.String())
	}

	missingContentTypeReq := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/notes", strings.NewReader(`{}`))
	missingContentTypeRec := httptest.NewRecorder()
	router.ServeHTTP(missingContentTypeRec, missingContentTypeReq)
	if missingContentTypeRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing content type, got %d body=%s", missingContentTypeRec.Code, missingContentTypeRec.Body.String())
	}
}

func TestScopedMemorySnapshotAndStatusErrorMappings(t *testing.T) {
	router := newRouter(t)

	createNoteReq := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/notes", strings.NewReader(`{
		"namespace":{"repo_namespace":"ach-trust-lab","run_namespace":"weekrun-errors","cycle_namespace":"day-1","agent_namespace":"feature-gap"},
		"input":{"created_by":"runner","unresolved_gaps":["gap-to-resolve"]}
	}`))
	createNoteReq.Header.Set("Content-Type", "application/json")
	createNoteRec := httptest.NewRecorder()
	router.ServeHTTP(createNoteRec, createNoteReq)
	if createNoteRec.Code != http.StatusOK {
		t.Fatalf("expected notes setup 200, got %d body=%s", createNoteRec.Code, createNoteRec.Body.String())
	}

	queryReq := httptest.NewRequest(http.MethodGet, "/api/v1/scoped-memory/query?repo_namespace=ach-trust-lab&run_namespace=weekrun-errors&memory_class=unresolved_gaps&status=open", nil)
	queryRec := httptest.NewRecorder()
	router.ServeHTTP(queryRec, queryReq)
	if queryRec.Code != http.StatusOK {
		t.Fatalf("expected query 200, got %d body=%s", queryRec.Code, queryRec.Body.String())
	}
	var queryPayload map[string]any
	if err := json.Unmarshal(queryRec.Body.Bytes(), &queryPayload); err != nil {
		t.Fatalf("query response decode error: %v", err)
	}
	records := queryPayload["data"].(map[string]any)["records"].([]any)
	recordID := records[0].(map[string]any)["id"].(string)

	invalidStatusReq := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/records/"+recordID+"/status", strings.NewReader(`{"status":"invalid-status","updated_by":"reviewer","reason":"bad"}`))
	invalidStatusReq.Header.Set("Content-Type", "application/json")
	invalidStatusRec := httptest.NewRecorder()
	router.ServeHTTP(invalidStatusRec, invalidStatusReq)
	if invalidStatusRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid status, got %d body=%s", invalidStatusRec.Code, invalidStatusRec.Body.String())
	}

	missingRecordReq := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/records/smr-missing/status", strings.NewReader(`{"status":"resolved","updated_by":"reviewer","reason":"missing"}`))
	missingRecordReq.Header.Set("Content-Type", "application/json")
	missingRecordRec := httptest.NewRecorder()
	router.ServeHTTP(missingRecordRec, missingRecordReq)
	if missingRecordRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing record status update, got %d body=%s", missingRecordRec.Code, missingRecordRec.Body.String())
	}

	getMissingSnapshotReq := httptest.NewRequest(http.MethodGet, "/api/v1/scoped-memory/snapshots/sms-missing-001", nil)
	getMissingSnapshotRec := httptest.NewRecorder()
	router.ServeHTTP(getMissingSnapshotRec, getMissingSnapshotReq)
	if getMissingSnapshotRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing snapshot get, got %d body=%s", getMissingSnapshotRec.Code, getMissingSnapshotRec.Body.String())
	}

	exportMissingSnapshotReq := httptest.NewRequest(http.MethodGet, "/api/v1/scoped-memory/snapshots/sms-missing-001?include_records=true", nil)
	exportMissingSnapshotRec := httptest.NewRecorder()
	router.ServeHTTP(exportMissingSnapshotRec, exportMissingSnapshotReq)
	if exportMissingSnapshotRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing snapshot export, got %d body=%s", exportMissingSnapshotRec.Code, exportMissingSnapshotRec.Body.String())
	}
}

func TestScopedMemoryCreateSnapshotAndQueryValidation(t *testing.T) {
	router := newRouter(t)

	seedReq := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/notes", strings.NewReader(`{
		"namespace":{"repo_namespace":"ach-trust-lab","run_namespace":"weekrun-manual-snapshot","cycle_namespace":"day-3","agent_namespace":"typologies"},
		"input":{"created_by":"runner","reviewer_notes":["seed note"]}
	}`))
	seedReq.Header.Set("Content-Type", "application/json")
	seedRec := httptest.NewRecorder()
	router.ServeHTTP(seedRec, seedReq)
	if seedRec.Code != http.StatusOK {
		t.Fatalf("expected setup notes 200, got %d body=%s", seedRec.Code, seedRec.Body.String())
	}

	createSnapshotReq := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/snapshots", strings.NewReader(`{
		"namespace":{"repo_namespace":"ach-trust-lab","run_namespace":"weekrun-manual-snapshot","cycle_namespace":"day-3"},
		"created_by":"runner",
		"summary":"manual checkpoint"
	}`))
	createSnapshotReq.Header.Set("Content-Type", "application/json")
	createSnapshotRec := httptest.NewRecorder()
	router.ServeHTTP(createSnapshotRec, createSnapshotReq)
	if createSnapshotRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 snapshot create, got %d body=%s", createSnapshotRec.Code, createSnapshotRec.Body.String())
	}
	if !strings.Contains(createSnapshotRec.Body.String(), `"manifest_checksum":"`) {
		t.Fatalf("expected checksum in snapshot create body=%s", createSnapshotRec.Body.String())
	}

	invalidSnapshotJSONReq := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/snapshots", strings.NewReader(`{"namespace":`))
	invalidSnapshotJSONReq.Header.Set("Content-Type", "application/json")
	invalidSnapshotJSONRec := httptest.NewRecorder()
	router.ServeHTTP(invalidSnapshotJSONRec, invalidSnapshotJSONReq)
	if invalidSnapshotJSONRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid snapshot json, got %d body=%s", invalidSnapshotJSONRec.Code, invalidSnapshotJSONRec.Body.String())
	}

	invalidNamespaceReq := httptest.NewRequest(http.MethodPost, "/api/v1/scoped-memory/snapshots", strings.NewReader(`{
		"namespace":{"repo_namespace":"ach-trust-lab"},
		"created_by":"runner",
		"summary":"bad"
	}`))
	invalidNamespaceReq.Header.Set("Content-Type", "application/json")
	invalidNamespaceRec := httptest.NewRecorder()
	router.ServeHTTP(invalidNamespaceRec, invalidNamespaceReq)
	if invalidNamespaceRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid namespace in snapshot create, got %d body=%s", invalidNamespaceRec.Code, invalidNamespaceRec.Body.String())
	}

	badPaginationReq := httptest.NewRequest(http.MethodGet, "/api/v1/scoped-memory/query?repo_namespace=ach-trust-lab&run_namespace=weekrun-manual-snapshot&limit=bad", nil)
	badPaginationRec := httptest.NewRecorder()
	router.ServeHTTP(badPaginationRec, badPaginationReq)
	if badPaginationRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid pagination, got %d body=%s", badPaginationRec.Code, badPaginationRec.Body.String())
	}

	badExportReq := httptest.NewRequest(http.MethodGet, "/api/v1/scoped-memory/query?repo_namespace=ach-trust-lab&export=run", nil)
	badExportRec := httptest.NewRecorder()
	router.ServeHTTP(badExportRec, badExportReq)
	if badExportRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid export namespace, got %d body=%s", badExportRec.Code, badExportRec.Body.String())
	}
}
