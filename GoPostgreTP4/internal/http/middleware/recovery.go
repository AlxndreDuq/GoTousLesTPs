package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery catches panics from downstream handlers, logs the stack trace and
// responds with a 500 JSON envelope instead of crashing the server.
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic_recovered",
						"request_id", RequestIDFromContext(r.Context()),
						"error", rec,
						"stack", string(debug.Stack()),
					)
					if sw, ok := w.(*statusWriter); ok && sw.Status() != 0 {
						return
					}
					writeJSONError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
