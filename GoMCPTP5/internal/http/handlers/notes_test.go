package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"mira-tp4/internal/core"
	apihttp "mira-tp4/internal/http"
)

// fakeRepo is a minimal in-memory core.Repository, used only to exercise
// the HTTP layer (routing, middlewares, envelopes) without a real Postgres
// instance.
type fakeRepo struct {
	mu     sync.Mutex
	notes  map[string]core.Note
	nextID int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{notes: make(map[string]core.Note)}
}

func (r *fakeRepo) Create(_ context.Context, note core.Note) (core.Note, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	note.ID = strconv.Itoa(r.nextID)
	note.CreatedAt = time.Now()
	note.UpdatedAt = note.CreatedAt
	r.notes[note.ID] = note
	return note, nil
}

func (r *fakeRepo) Get(_ context.Context, id string) (core.Note, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	note, ok := r.notes[id]
	if !ok {
		return core.Note{}, core.ErrNotFound
	}
	return note, nil
}

func (r *fakeRepo) List(_ context.Context, filter core.ListFilter) (core.ListResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	notes := make([]core.Note, 0, len(r.notes))
	for _, n := range r.notes {
		notes = append(notes, n)
	}
	return core.ListResult{Notes: notes, Total: len(notes)}, nil
}

func (r *fakeRepo) Update(_ context.Context, id string, patch core.UpdateInput) (core.Note, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	note, ok := r.notes[id]
	if !ok {
		return core.Note{}, core.ErrNotFound
	}
	if patch.Title != nil {
		note.Title = *patch.Title
	}
	if patch.Content != nil {
		note.Content = *patch.Content
	}
	if patch.Status != nil {
		note.Status = *patch.Status
	}
	if patch.Tags != nil {
		note.Tags = *patch.Tags
	}
	r.notes[id] = note
	return note, nil
}

func (r *fakeRepo) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.notes[id]; !ok {
		return core.ErrNotFound
	}
	delete(r.notes, id)
	return nil
}

func (r *fakeRepo) Search(_ context.Context, _ string) (core.ListResult, error) {
	return core.ListResult{}, nil
}

func (r *fakeRepo) SaveEnrichment(_ context.Context, _ core.EnrichmentResult) error {
	return nil
}

func newTestRouter() http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := core.NewService(newFakeRepo(), nil)
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
	if status, _ := data["enrichment_status"].(string); status != core.EnrichmentPending {
		t.Fatalf("enrichment_status = %q, want %q", status, core.EnrichmentPending)
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
