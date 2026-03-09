package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogging_LogsRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	handler := Logging(logger, inner)

	req := httptest.NewRequest(http.MethodGet, "/v1/shelters/nearest?lat=35&lon=-97", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	logged := buf.String()

	// Verify key fields are logged.
	if !strings.Contains(logged, "GET") {
		t.Error("expected log to contain HTTP method 'GET'")
	}
	if !strings.Contains(logged, "/v1/shelters/nearest") {
		t.Error("expected log to contain request path")
	}
	if !strings.Contains(logged, "200") {
		t.Error("expected log to contain status code 200")
	}
	if !strings.Contains(logged, "duration") {
		t.Error("expected log to contain duration field")
	}
}

func TestLogging_CapturesNonOKStatus(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	handler := Logging(logger, inner)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	logged := buf.String()
	if !strings.Contains(logged, "404") {
		t.Error("expected log to contain status code 404")
	}
}
