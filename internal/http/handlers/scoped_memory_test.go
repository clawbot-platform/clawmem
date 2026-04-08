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
			"snapshot_summary": "day-3 checkpoint"
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

	queryReq := httptest.NewRequest(http.MethodGet, "/api/v1/scoped-memory/query?repo_namespace=ach-trust-lab&run_namespace=weekrun-2026-06-showcase&memory_class=unresolved_gaps&status=open", nil)
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
