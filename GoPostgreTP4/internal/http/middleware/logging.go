package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// Logging emits one structured log line per request once it completes,
// including the request id, method, path, final status code and duration.
// It must wrap Recovery so that a recovered panic's 500 status is captured.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := newStatusWriter(w)
			start := time.Now()

			next.ServeHTTP(sw, r)

			logger.Info("http_request",
				"request_id", RequestIDFromContext(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}
