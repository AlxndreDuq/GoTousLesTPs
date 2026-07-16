package apiclient_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"mira-tp4/internal/apiclient"
	"mira-tp4/internal/core"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) *apiclient.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return apiclient.New(srv.URL)
}

func writeEnvelope(t *testing.T, w http.ResponseWriter, status int, data any, total int) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	env := map[string]any{"data": data}
	if total >= 0 {
		env["meta"] = map[string]int{"total": total}
	}
	if err := json.NewEncoder(w).Encode(env); err != nil {
		t.Fatalf("encode envelope: %v", err)
	}
}

func writeAPIErrorEnvelope(t *testing.T, w http.ResponseWriter, status int, code, message string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	env := map[string]any{"error": map[string]string{"code": code, "message": message}}
	if err := json.NewEncoder(w).Encode(env); err != nil {
		t.Fatalf("encode envelope: %v", err)
	}
}

func TestCreateNote_Success(t *testing.T) {
	var gotBody map[string]any
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/notes" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		note := core.Note{ID: "abc", Title: gotBody["title"].(string), EnrichmentStatus: core.EnrichmentPending}
		writeEnvelope(t, w, http.StatusCreated, note, -1)
	})

	note, err := client.CreateNote(t.Context(), "Groceries", "Milk", []string{"maison"})
	if err != nil {
		t.Fatalf("CreateNote() error = %v", err)
	}
	if note.ID != "abc" || note.Title != "Groceries" {
		t.Fatalf("CreateNote() = %+v, unexpected", note)
	}
	if gotBody["title"] != "Groceries" || gotBody["content"] != "Milk" {
		t.Fatalf("request body = %+v, missing title/content", gotBody)
	}
	tags, _ := gotBody["tags"].([]any)
	if len(tags) != 1 || tags[0] != "maison" {
		t.Fatalf("request body tags = %v, want [maison]", gotBody["tags"])
	}
}

func TestCreateNote_ValidationErrorSurfacesAsAPIError(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeAPIErrorEnvelope(t, w, http.StatusBadRequest, "validation_error", "title is required")
	})

	_, err := client.CreateNote(t.Context(), "", "content", nil)
	var apiErr *apiclient.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("CreateNote() error = %v, want *apiclient.APIError", err)
	}
	if apiErr.Status != http.StatusBadRequest || apiErr.Code != "validation_error" {
		t.Fatalf("APIError = %+v, unexpected", apiErr)
	}
}

func TestGetNote_Success(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/notes/abc-123" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeEnvelope(t, w, http.StatusOK, core.Note{ID: "abc-123", Title: "Found"}, -1)
	})

	note, err := client.GetNote(t.Context(), "abc-123")
	if err != nil {
		t.Fatalf("GetNote() error = %v", err)
	}
	if note.ID != "abc-123" || note.Title != "Found" {
		t.Fatalf("GetNote() = %+v, unexpected", note)
	}
}

func TestGetNote_NotFoundSurfacesAsAPIError(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeAPIErrorEnvelope(t, w, http.StatusNotFound, "not_found", "note not found")
	})

	_, err := client.GetNote(t.Context(), "missing")
	var apiErr *apiclient.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("GetNote() error = %v, want *apiclient.APIError", err)
	}
	if apiErr.Status != http.StatusNotFound {
		t.Fatalf("APIError.Status = %d, want 404", apiErr.Status)
	}
}

func TestSearchNotes_Success(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search" || r.URL.Query().Get("q") != "milk" {
			t.Fatalf("unexpected request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		writeEnvelope(t, w, http.StatusOK, []core.Note{{ID: "1", Title: "Milk"}}, 1)
	})

	notes, err := client.SearchNotes(t.Context(), "milk")
	if err != nil {
		t.Fatalf("SearchNotes() error = %v", err)
	}
	if len(notes) != 1 || notes[0].ID != "1" {
		t.Fatalf("SearchNotes() = %+v, unexpected", notes)
	}
}

func TestSearchNotes_EmptyQueryIsAPIError(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeAPIErrorEnvelope(t, w, http.StatusBadRequest, "validation_error", "q is required")
	})

	_, err := client.SearchNotes(t.Context(), "")
	var apiErr *apiclient.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("SearchNotes() error = %v, want *apiclient.APIError", err)
	}
}

// TestListRecent_ReturnsLastNInCreationOrder exercises the two-request logic
// in ListRecent: it first asks for the total count, then fetches the last n
// notes by offset, relying on the API returning notes oldest-first.
func TestListRecent_ReturnsLastNInCreationOrder(t *testing.T) {
	all := make([]core.Note, 10)
	for i := range all {
		all[i] = core.Note{ID: strconv.Itoa(i), Title: "note " + strconv.Itoa(i)}
	}

	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		end := min(offset+limit, len(all))
		start := min(offset, end)
		writeEnvelope(t, w, http.StatusOK, all[start:end], len(all))
	})

	notes, err := client.ListRecent(t.Context(), 3)
	if err != nil {
		t.Fatalf("ListRecent() error = %v", err)
	}
	if len(notes) != 3 {
		t.Fatalf("ListRecent(3) returned %d notes, want 3", len(notes))
	}
	if notes[0].ID != "7" || notes[2].ID != "9" {
		t.Fatalf("ListRecent(3) = %+v, want the last 3 notes (ids 7,8,9)", notes)
	}
}

func TestListRecent_NMoreThanTotalReturnsEverything(t *testing.T) {
	all := []core.Note{{ID: "1"}, {ID: "2"}}

	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		end := min(offset+limit, len(all))
		start := min(offset, end)
		writeEnvelope(t, w, http.StatusOK, all[start:end], len(all))
	})

	notes, err := client.ListRecent(t.Context(), 10)
	if err != nil {
		t.Fatalf("ListRecent() error = %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("ListRecent(10) with only 2 notes total = %d notes, want 2", len(notes))
	}
}

func TestDo_MalformedJSONResponseIsAnError(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	})

	_, err := client.GetNote(t.Context(), "any")
	if err == nil {
		t.Fatalf("GetNote() error = nil, want a decode error for malformed JSON")
	}
}
