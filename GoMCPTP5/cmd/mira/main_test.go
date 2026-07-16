package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"mira-tp4/internal/apiclient"
	"mira-tp4/internal/core"
)

func writeEnvelope(t *testing.T, w http.ResponseWriter, status int, data any, total int) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	env := map[string]any{"data": data}
	if total >= 0 {
		env["meta"] = map[string]int{"total": total}
	}
	json.NewEncoder(w).Encode(env)
}

func TestRunAdd_PrintsCreatedNote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(t, w, http.StatusCreated, core.Note{ID: "abc", EnrichmentStatus: core.EnrichmentPending}, -1)
	}))
	defer srv.Close()
	client := apiclient.New(srv.URL)

	var out bytes.Buffer
	if err := runAdd(t.Context(), client, &out, []string{"Titre", "Contenu"}); err != nil {
		t.Fatalf("runAdd() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "abc") || !strings.Contains(got, core.EnrichmentPending) {
		t.Fatalf("runAdd() output = %q, want it to mention the note id and enrichment status", got)
	}
}

func TestRunAdd_MissingArgsIsUsageError(t *testing.T) {
	client := apiclient.New("http://unused.invalid")
	var out bytes.Buffer

	err := runAdd(t.Context(), client, &out, []string{"only-title"})
	if err == nil {
		t.Fatal("runAdd() error = nil, want a usage error for missing content")
	}
}

func TestRunList_PrintsNotes(t *testing.T) {
	all := []core.Note{
		{ID: "1", Title: "First", Content: "a", CreatedAt: time.Now()},
		{ID: "2", Title: "Second", Content: "b", Tags: []string{"go"}, CreatedAt: time.Now()},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		end := min(offset+limit, len(all))
		start := min(offset, end)
		writeEnvelope(t, w, http.StatusOK, all[start:end], len(all))
	}))
	defer srv.Close()
	client := apiclient.New(srv.URL)

	var out bytes.Buffer
	if err := runList(t.Context(), client, &out); err != nil {
		t.Fatalf("runList() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "First") || !strings.Contains(got, "Second") || !strings.Contains(got, "tags: go") {
		t.Fatalf("runList() output = %q, want both notes and the tags line", got)
	}
}

func TestRunList_EmptyPrintsPlaceholder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(t, w, http.StatusOK, []core.Note{}, 0)
	}))
	defer srv.Close()
	client := apiclient.New(srv.URL)

	var out bytes.Buffer
	if err := runList(t.Context(), client, &out); err != nil {
		t.Fatalf("runList() error = %v", err)
	}
	if !strings.Contains(out.String(), "aucune note") {
		t.Fatalf("runList() output = %q, want the empty-list placeholder", out.String())
	}
}

func TestRunSearch_JoinsArgsAndPrintsResults(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("q")
		writeEnvelope(t, w, http.StatusOK, []core.Note{{ID: "1", Title: "Found", CreatedAt: time.Now()}}, 1)
	}))
	defer srv.Close()
	client := apiclient.New(srv.URL)

	var out bytes.Buffer
	if err := runSearch(t.Context(), client, &out, []string{"go", "channels"}); err != nil {
		t.Fatalf("runSearch() error = %v", err)
	}

	if gotQuery != "go channels" {
		t.Fatalf("search query sent = %q, want %q", gotQuery, "go channels")
	}
	if !strings.Contains(out.String(), "Found") {
		t.Fatalf("runSearch() output = %q, want it to mention the result", out.String())
	}
}

func TestRunSearch_NoArgsIsUsageError(t *testing.T) {
	client := apiclient.New("http://unused.invalid")
	var out bytes.Buffer

	if err := runSearch(t.Context(), client, &out, nil); err == nil {
		t.Fatal("runSearch() error = nil, want a usage error when no query is given")
	}
}
