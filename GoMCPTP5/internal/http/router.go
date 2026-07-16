package http

import (
	"log/slog"
	"net/http"
	"time"

	"mira-tp4/internal/core"
	"mira-tp4/internal/http/handlers"
	"mira-tp4/internal/http/middleware"
)

const requestTimeout = 5 * time.Second

// NewRouter builds the full HTTP handler for the API: the /api/v1/notes and
// /api/v1/search routes wrapped in the request id, logging, recovery and
// timeout middlewares.
func NewRouter(service *core.Service, logger *slog.Logger) http.Handler {
	h := handlers.New(service, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/notes", h.Create)
	mux.HandleFunc("GET /api/v1/notes", h.List)
	mux.HandleFunc("GET /api/v1/notes/{id}", h.Get)
	mux.HandleFunc("PATCH /api/v1/notes/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/notes/{id}", h.Delete)
	mux.HandleFunc("GET /api/v1/search", h.Search)

	mux.HandleFunc("GET /healthz", serveHealthz)
	mux.HandleFunc("GET /docs", serveDocsHTML)
	mux.HandleFunc("GET /docs/openapi.yaml", serveOpenAPISpec)

	var handler http.Handler = mux
	handler = middleware.Timeout(requestTimeout)(handler)
	handler = middleware.Recovery(logger)(handler)
	handler = middleware.Logging(logger)(handler)
	handler = middleware.RequestID(handler)

	return handler
}

func serveHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
