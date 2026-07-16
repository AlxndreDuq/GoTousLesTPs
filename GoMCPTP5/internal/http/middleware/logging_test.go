package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"mira-tp4/internal/http/middleware"
)

func TestLogging_EmitsOneLineWithStatusAndRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	chain := middleware.RequestID(middleware.Logging(logger)(next))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil)
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	var line map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &line); err != nil {
		t.Fatalf("decode log line: %v (raw=%s)", err, buf.String())
	}

	if line["msg"] != "http_request" {
		t.Fatalf("msg = %v, want %q", line["msg"], "http_request")
	}
	if line["method"] != http.MethodGet {
		t.Fatalf("method = %v, want %q", line["method"], http.MethodGet)
	}
	if line["path"] != "/api/v1/notes" {
		t.Fatalf("path = %v, want %q", line["path"], "/api/v1/notes")
	}
	status, ok := line["status"].(float64)
	if !ok || int(status) != http.StatusTeapot {
		t.Fatalf("status = %v, want %d", line["status"], http.StatusTeapot)
	}
	if rid, _ := line["request_id"].(string); rid == "" {
		t.Fatalf("expected a non-empty request_id field, got %v", line["request_id"])
	}
}
