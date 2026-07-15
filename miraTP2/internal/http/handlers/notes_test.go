package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"mira-tp2/internal/core"
	apihttp "mira-tp2/internal/http"
	"mira-tp2/internal/store"
)

func newTestRouter() http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := core.NewService(store.NewMemoryStore())
	return apihttp.NewRouter(service, logger)
}

func decodeEnvelope(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()
	var env map[string]any
	if err := json.Unmarshal(body.Bytes(), &env); err != nil {
		t.Fatalf("decode response body: %v (body=%s)", err, body.String())
	}
	return env
}

func TestCreateNote_Success(t *testing.T) {
	router := newTestRouter()

	body := `{"title":"Groceries","content":"Milk, eggs"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusCreated, rec.Body.String())
	}

	env := decodeEnvelope(t, rec.Body)
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object in response, got %#v", env)
	}
	if id, _ := data["id"].(string); id == "" {
		t.Fatalf("expected non-empty id, got %#v", data["id"])
	}
	if title, _ := data["title"].(string); title != "Groceries" {
		t.Fatalf("title = %q, want %q", title, "Groceries")
	}
}

func TestCreateNote_InvalidPayload(t *testing.T) {
	router := newTestRouter()

	body := `{"content":"missing a title"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	env := decodeEnvelope(t, rec.Body)
	errBody, ok := env["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object in response, got %#v", env)
	}
	if code, _ := errBody["code"].(string); code != "validation_error" {
		t.Fatalf("error.code = %q, want %q", code, "validation_error")
	}
}

func TestGetNote_NotFound(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes/does-not-exist", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusNotFound, rec.Body.String())
	}

	env := decodeEnvelope(t, rec.Body)
	errBody, ok := env["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object in response, got %#v", env)
	}
	if code, _ := errBody["code"].(string); code != "not_found" {
		t.Fatalf("error.code = %q, want %q", code, "not_found")
	}
}
