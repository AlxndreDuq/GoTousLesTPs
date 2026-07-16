package http_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apihttp "mira-tp4/internal/http"
)

// These two routes never touch the notes service, so a nil *core.Service is
// enough to exercise them without pulling in a fake repository.
func newTestRouter() http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return apihttp.NewRouter(nil, logger)
}

func TestHealthz_Returns200(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestDocs_ServesSwaggerHTML(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected a non-empty docs HTML body")
	}
}

// The openapi.yaml spec is read from disk relative to the process's working
// directory (documented as requiring the repo root); running from the
// package's own test directory should fail gracefully, not panic.
func TestDocsOpenAPISpec_MissingFileReturns500Gracefully(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/docs/openapi.yaml", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
