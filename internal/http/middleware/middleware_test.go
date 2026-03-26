package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChainAppliesMiddlewaresInOrder(t *testing.T) {
	t.Parallel()

	var order []string
	handler := Chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusNoContent)
	}),
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "first-before")
				next.ServeHTTP(w, r)
				order = append(order, "first-after")
			})
		},
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "second-before")
				next.ServeHTTP(w, r)
				order = append(order, "second-after")
			})
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	got := strings.Join(order, ",")
	want := "first-before,second-before,handler,second-after,first-after"
	if got != want {
		t.Fatalf("unexpected order %q", got)
	}
}

func TestRequestLoggerLogsRequest(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	handler := RequestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/memories", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := buf.String()
	if !strings.Contains(body, "http request") || !strings.Contains(body, "method=POST") || !strings.Contains(body, "path=/api/v1/memories") {
		t.Fatalf("unexpected log output %q", body)
	}
}

func TestRecovererHandlesPanic(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := Recoverer(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
