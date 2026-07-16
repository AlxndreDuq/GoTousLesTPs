package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mira-tp4/internal/http/middleware"
)

func TestTimeout_HandlerFinishesInTimePassesThrough(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "yes")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("done"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	middleware.Timeout(time.Second)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if rec.Body.String() != "done" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "done")
	}
	if rec.Header().Get("X-Custom") != "yes" {
		t.Fatalf("expected handler-set header to be propagated through the timeout wrapper")
	}
}

func TestTimeout_SlowHandlerReturns504(t *testing.T) {
	blockUntilDone := make(chan struct{})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
		close(blockUntilDone)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	middleware.Timeout(20*time.Millisecond)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusGatewayTimeout)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v (body=%s)", err, rec.Body.String())
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok || errObj["code"] != "timeout" {
		t.Fatalf("expected error.code = %q, got %#v", "timeout", body)
	}

	// Let the abandoned handler goroutine observe context cancellation and
	// exit, so it doesn't leak past the test.
	<-blockUntilDone
}
