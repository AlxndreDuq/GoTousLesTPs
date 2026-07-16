package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"mira-tp4/internal/http/middleware"
)

func TestRequestID_GeneratesWhenAbsent(t *testing.T) {
	var seenInContext string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenInContext = middleware.RequestIDFromContext(r.Context())
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	middleware.RequestID(next).ServeHTTP(rec, req)

	header := rec.Header().Get("X-Request-ID")
	if header == "" {
		t.Fatal("expected a generated X-Request-ID response header")
	}
	if seenInContext != header {
		t.Fatalf("context request id = %q, want it to match the response header %q", seenInContext, header)
	}
}

func TestRequestID_ReusesInboundHeader(t *testing.T) {
	var seenInContext string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenInContext = middleware.RequestIDFromContext(r.Context())
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "caller-supplied-id")
	rec := httptest.NewRecorder()

	middleware.RequestID(next).ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-ID"); got != "caller-supplied-id" {
		t.Fatalf("X-Request-ID = %q, want the inbound value to be echoed back", got)
	}
	if seenInContext != "caller-supplied-id" {
		t.Fatalf("context request id = %q, want %q", seenInContext, "caller-supplied-id")
	}
}

func TestRequestIDFromContext_EmptyWhenUnset(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := middleware.RequestIDFromContext(req.Context()); got != "" {
		t.Fatalf("RequestIDFromContext() = %q, want empty string when never set", got)
	}
}
