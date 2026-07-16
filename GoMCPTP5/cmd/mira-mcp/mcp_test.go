package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mira-tp4/internal/apiclient"
	"mira-tp4/internal/core"
)

// fakeMiraAPI is a minimal stand-in for the real mira HTTP API: enough of
// the envelope contract (data/meta/error) for internal/apiclient to talk to,
// so the MCP tools can be exercised end to end without Postgres/docker.
type fakeMiraAPI struct {
	mu     sync.Mutex
	notes  []*core.Note
	nextID int
}

func newFakeMiraAPI() *httptest.Server {
	api := &fakeMiraAPI{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/notes", api.create)
	mux.HandleFunc("GET /api/v1/notes/{id}", api.get)
	mux.HandleFunc("GET /api/v1/notes", api.list)
	mux.HandleFunc("GET /api/v1/search", api.search)
	return httptest.NewServer(mux)
}

func writeEnvelope(w http.ResponseWriter, status int, data any, total int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	env := map[string]any{"data": data}
	if total > 0 || data != nil {
		env["meta"] = map[string]int{"total": total}
	}
	json.NewEncoder(w).Encode(env)
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

func (f *fakeMiraAPI) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title   string   `json:"title"`
		Content string   `json:"content"`
		Tags    []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "bad json")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeAPIError(w, http.StatusBadRequest, "validation_error", "title is required")
		return
	}

	f.mu.Lock()
	f.nextID++
	n := &core.Note{
		ID:               fmt.Sprintf("note-%d", f.nextID),
		Title:            req.Title,
		Content:          req.Content,
		Status:           core.StatusActive,
		Tags:             req.Tags,
		EnrichmentStatus: core.EnrichmentPending,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	f.notes = append(f.notes, n)
	f.mu.Unlock()

	writeEnvelope(w, http.StatusCreated, n, 0)
}

func (f *fakeMiraAPI) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, n := range f.notes {
		if n.ID == id {
			writeEnvelope(w, http.StatusOK, n, 0)
			return
		}
	}
	writeAPIError(w, http.StatusNotFound, "not_found", "note not found")
}

func (f *fakeMiraAPI) list(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 20
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	total := len(f.notes)
	start := min(offset, total)
	end := min(start+limit, total)

	writeEnvelope(w, http.StatusOK, f.notes[start:end], total)
}

func (f *fakeMiraAPI) search(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q"))
	if q == "" {
		writeAPIError(w, http.StatusBadRequest, "validation_error", "q is required")
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var results []*core.Note
	for _, n := range f.notes {
		if strings.Contains(strings.ToLower(n.Title), q) || strings.Contains(strings.ToLower(n.Content), q) {
			results = append(results, n)
		}
	}
	writeEnvelope(w, http.StatusOK, results, len(results))
}

// connectTestClient wires an in-process MCP client to a fresh server backed
// by a fake mira API, using the SDK's in-memory transport pair: no
// subprocess, no real network, deterministic and fast.
func connectTestClient(t *testing.T) (*mcp.ClientSession, *httptest.Server) {
	t.Helper()

	api := newFakeMiraAPI()
	t.Cleanup(api.Close)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := apiclient.New(api.URL)
	server := newServer(client, logger)

	t1, t2 := mcp.NewInMemoryTransports()
	ctx := context.Background()

	if _, err := server.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	cs, err := mcpClient.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	t.Cleanup(func() { cs.Close() })

	return cs, api
}

func callTool(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s) protocol error = %v", name, err)
	}
	return res
}

func decodeStructured[T any](t *testing.T, res *mcp.CallToolResult) T {
	t.Helper()
	var out T
	b, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal structured content: %v (raw=%s)", err, b)
	}
	return out
}

func TestListTools_ExposesAllFour(t *testing.T) {
	cs, _ := connectTestClient(t)

	res, err := cs.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	got := map[string]bool{}
	for _, tool := range res.Tools {
		got[tool.Name] = true
		if tool.Description == "" {
			t.Errorf("tool %q has no description", tool.Name)
		}
	}
	for _, want := range []string{"search_notes", "get_note", "add_note", "list_recent_notes"} {
		if !got[want] {
			t.Errorf("expected tool %q to be registered, got %v", want, got)
		}
	}
}

func TestAddNote_SearchNotes_GetNote_RoundTrip(t *testing.T) {
	cs, _ := connectTestClient(t)

	addRes := callTool(t, cs, "add_note", map[string]any{
		"title":   "Notes sur les channels Go",
		"content": "select, buffered vs unbuffered",
		"tags":    []string{"go"},
	})
	if addRes.IsError {
		t.Fatalf("add_note returned isError: %v", addRes.Content)
	}
	created := decodeStructured[core.Note](t, addRes)
	if created.ID == "" {
		t.Fatalf("add_note: expected a non-empty id")
	}
	if created.EnrichmentStatus != core.EnrichmentPending {
		t.Fatalf("add_note: enrichment_status = %q, want %q", created.EnrichmentStatus, core.EnrichmentPending)
	}

	searchRes := callTool(t, cs, "search_notes", map[string]any{"query": "channels"})
	if searchRes.IsError {
		t.Fatalf("search_notes returned isError: %v", searchRes.Content)
	}
	searchOut := decodeStructured[notesListOutput](t, searchRes)
	if searchOut.Count != 1 || searchOut.Notes[0].ID != created.ID {
		t.Fatalf("search_notes: got %+v, want exactly the created note", searchOut)
	}

	getRes := callTool(t, cs, "get_note", map[string]any{"id": created.ID})
	if getRes.IsError {
		t.Fatalf("get_note returned isError: %v", getRes.Content)
	}
	fetched := decodeStructured[core.Note](t, getRes)
	if fetched.Title != created.Title {
		t.Fatalf("get_note: title = %q, want %q", fetched.Title, created.Title)
	}
}

func TestSearchNotes_LimitIsRespected(t *testing.T) {
	cs, _ := connectTestClient(t)

	for i := 0; i < 5; i++ {
		callTool(t, cs, "add_note", map[string]any{
			"title":   fmt.Sprintf("golang note %d", i),
			"content": "about the go programming language",
		})
	}

	res := callTool(t, cs, "search_notes", map[string]any{"query": "golang", "limit": 2})
	if res.IsError {
		t.Fatalf("search_notes returned isError: %v", res.Content)
	}
	out := decodeStructured[notesListOutput](t, res)
	if out.Count != 2 {
		t.Fatalf("search_notes: count = %d, want 2 (limit not applied)", out.Count)
	}
}

func TestListRecentNotes_DefaultsAndOrdering(t *testing.T) {
	cs, _ := connectTestClient(t)

	var ids []string
	for i := 0; i < 3; i++ {
		res := callTool(t, cs, "add_note", map[string]any{
			"title":   fmt.Sprintf("note %d", i),
			"content": "x",
		})
		ids = append(ids, decodeStructured[core.Note](t, res).ID)
	}

	res := callTool(t, cs, "list_recent_notes", map[string]any{})
	if res.IsError {
		t.Fatalf("list_recent_notes returned isError: %v", res.Content)
	}
	out := decodeStructured[notesListOutput](t, res)
	if out.Count != 3 {
		t.Fatalf("list_recent_notes: count = %d, want 3", out.Count)
	}
	if out.Notes[len(out.Notes)-1].ID != ids[len(ids)-1] {
		t.Fatalf("list_recent_notes: last note id = %q, want most recently created %q", out.Notes[len(out.Notes)-1].ID, ids[len(ids)-1])
	}
}

func TestSearchNotes_EmptyQueryIsCleanToolError(t *testing.T) {
	cs, _ := connectTestClient(t)

	res := callTool(t, cs, "search_notes", map[string]any{"query": "   "})
	if !res.IsError {
		t.Fatalf("expected isError=true for an empty query")
	}
}

func TestSearchNotes_NegativeLimitIsCleanToolError(t *testing.T) {
	cs, _ := connectTestClient(t)

	res := callTool(t, cs, "search_notes", map[string]any{"query": "go", "limit": -1})
	if !res.IsError {
		t.Fatalf("expected isError=true for a negative limit")
	}
}

func TestAddNote_MissingTitleIsCleanToolError(t *testing.T) {
	cs, _ := connectTestClient(t)

	res := callTool(t, cs, "add_note", map[string]any{"title": "  ", "content": "x"})
	if !res.IsError {
		t.Fatalf("expected isError=true for a blank title")
	}
}

func TestGetNote_UnknownIDSurfacesAPIErrorCleanly(t *testing.T) {
	cs, _ := connectTestClient(t)

	res := callTool(t, cs, "get_note", map[string]any{"id": "does-not-exist"})
	if !res.IsError {
		t.Fatalf("expected isError=true for an unknown id")
	}
	found := false
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok && strings.Contains(tc.Text, "not found") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the mira API's not_found message to surface in tool error content, got %v", res.Content)
	}
}

func TestGetNote_EmptyIDIsCleanToolError(t *testing.T) {
	cs, _ := connectTestClient(t)

	res := callTool(t, cs, "get_note", map[string]any{"id": "  "})
	if !res.IsError {
		t.Fatalf("expected isError=true for a blank id")
	}
}
