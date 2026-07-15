package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"mira-tp2/internal/core"
)

// Meta carries pagination metadata alongside list-shaped responses.
type Meta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	Data  any        `json:"data,omitempty"`
	Meta  *Meta      `json:"meta,omitempty"`
	Error *errorBody `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, env envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(env)
}

func writeData(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, envelope{Data: data})
}

func writeList(w http.ResponseWriter, status int, data any, meta Meta) {
	writeJSON(w, status, envelope{Data: data, Meta: &meta})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, envelope{Error: &errorBody{Code: code, Message: message}})
}

// handleServiceError maps a core service error to an HTTP response, logging
// unexpected errors server-side without leaking their detail to the client.
func handleServiceError(w http.ResponseWriter, logger *slog.Logger, err error) {
	switch {
	case errors.Is(err, core.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, core.ErrValidation):
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		logger.Error("unexpected_error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
