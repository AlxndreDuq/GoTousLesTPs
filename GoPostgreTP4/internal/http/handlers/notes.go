package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"mira-tp4/internal/core"
)

// Handlers wires HTTP requests to the core notes service.
type Handlers struct {
	service *core.Service
	logger  *slog.Logger
}

func New(service *core.Service, logger *slog.Logger) *Handlers {
	return &Handlers{service: service, logger: logger}
}

type createNoteRequest struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Status  string   `json:"status"`
	Tags    []string `json:"tags"`
}

type updateNoteRequest struct {
	Title   *string   `json:"title"`
	Content *string   `json:"content"`
	Status  *string   `json:"status"`
	Tags    *[]string `json:"tags"`
}

func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req createNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	note, err := h.service.CreateNote(r.Context(), core.CreateInput{
		Title:   req.Title,
		Content: req.Content,
		Status:  req.Status,
		Tags:    req.Tags,
	})
	if err != nil {
		handleServiceError(w, h.logger, err)
		return
	}

	w.Header().Set("Location", "/api/v1/notes/"+note.ID)
	writeData(w, http.StatusCreated, note)
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	note, err := h.service.GetNote(r.Context(), id)
	if err != nil {
		handleServiceError(w, h.logger, err)
		return
	}
	writeData(w, http.StatusOK, note)
}

func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit, err := parseNonNegativeIntParam(query, "limit")
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	offset, err := parseNonNegativeIntParam(query, "offset")
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	result, err := h.service.ListNotes(r.Context(), core.ListFilter{
		Status: query.Get("status"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		handleServiceError(w, h.logger, err)
		return
	}

	writeList(w, http.StatusOK, result.Notes, Meta{Total: result.Total, Limit: limit, Offset: offset})
}

func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req updateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	note, err := h.service.UpdateNote(r.Context(), id, core.UpdateInput{
		Title:   req.Title,
		Content: req.Content,
		Status:  req.Status,
		Tags:    req.Tags,
	})
	if err != nil {
		handleServiceError(w, h.logger, err)
		return
	}
	writeData(w, http.StatusOK, note)
}

func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.service.DeleteNote(r.Context(), id); err != nil {
		handleServiceError(w, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parseNonNegativeIntParam parses an optional integer query parameter,
// returning 0 if absent and an error if present but invalid or negative.
func parseNonNegativeIntParam(query map[string][]string, name string) (int, error) {
	raw := ""
	if v, ok := query[name]; ok && len(v) > 0 {
		raw = v[0]
	}
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, &paramError{param: name}
	}
	return n, nil
}

type paramError struct{ param string }

func (e *paramError) Error() string {
	return e.param + " must be a non-negative integer"
}
