// Package apiclient is a small HTTP client for the mira API, used by
// cmd/mira. Recreating the TP1 CLI on top of it (instead of reading/writing
// the local JSONL file directly) is what guarantees every note created or
// modified goes through the API and therefore triggers automatic
// enrichment.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"mira-tp4/internal/core"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// APIError wraps a non-2xx API response, preserving the status and the
// error envelope's code/message so callers can react to it if needed.
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s (%s, status %d)", e.Message, e.Code, e.Status)
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type meta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type envelope struct {
	Data  json.RawMessage `json:"data"`
	Meta  *meta           `json:"meta"`
	Error *errorBody      `json:"error"`
}

func (c *Client) do(ctx context.Context, method, path string, body any) (envelope, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return envelope{}, err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return envelope{}, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return envelope{}, fmt.Errorf("mira api: %w", err)
	}
	defer resp.Body.Close()

	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return envelope{}, fmt.Errorf("mira api: decode response: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{Status: resp.StatusCode}
		if env.Error != nil {
			apiErr.Code = env.Error.Code
			apiErr.Message = env.Error.Message
		}
		return envelope{}, apiErr
	}

	return env, nil
}

type createNoteRequest struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags,omitempty"`
}

func (c *Client) CreateNote(ctx context.Context, title, content string, tags []string) (core.Note, error) {
	env, err := c.do(ctx, http.MethodPost, "/api/v1/notes", createNoteRequest{Title: title, Content: content, Tags: tags})
	if err != nil {
		return core.Note{}, err
	}
	var note core.Note
	if err := json.Unmarshal(env.Data, &note); err != nil {
		return core.Note{}, fmt.Errorf("mira api: decode note: %w", err)
	}
	return note, nil
}

// GetNote fetches a single note by id. A non-existent or malformed id
// surfaces as an *APIError with status 404 (mirroring the API's contract).
func (c *Client) GetNote(ctx context.Context, id string) (core.Note, error) {
	env, err := c.do(ctx, http.MethodGet, "/api/v1/notes/"+url.PathEscape(id), nil)
	if err != nil {
		return core.Note{}, err
	}
	var note core.Note
	if err := json.Unmarshal(env.Data, &note); err != nil {
		return core.Note{}, fmt.Errorf("mira api: decode note: %w", err)
	}
	return note, nil
}

func (c *Client) listPage(ctx context.Context, limit, offset int) ([]core.Note, int, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))

	env, err := c.do(ctx, http.MethodGet, "/api/v1/notes?"+q.Encode(), nil)
	if err != nil {
		return nil, 0, err
	}
	var notes []core.Note
	if err := json.Unmarshal(env.Data, &notes); err != nil {
		return nil, 0, fmt.Errorf("mira api: decode notes: %w", err)
	}
	total := 0
	if env.Meta != nil {
		total = env.Meta.Total
	}
	return notes, total, nil
}

// ListRecent returns the n most recently created notes, oldest first (same
// order the API returns), mirroring the original mira CLI's `list` command.
func (c *Client) ListRecent(ctx context.Context, n int) ([]core.Note, error) {
	_, total, err := c.listPage(ctx, 1, 0)
	if err != nil {
		return nil, err
	}

	offset := total - n
	if offset < 0 {
		offset = 0
	}

	notes, _, err := c.listPage(ctx, n, offset)
	return notes, err
}

func (c *Client) SearchNotes(ctx context.Context, query string) ([]core.Note, error) {
	q := url.Values{}
	q.Set("q", query)

	env, err := c.do(ctx, http.MethodGet, "/api/v1/search?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	var notes []core.Note
	if err := json.Unmarshal(env.Data, &notes); err != nil {
		return nil, fmt.Errorf("mira api: decode notes: %w", err)
	}
	return notes, nil
}
