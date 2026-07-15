package middleware

import (
	"bytes"
	"context"
	"net/http"
	"time"
)

// bufferedWriter captures a handler's response in memory instead of writing
// straight to the client. Timeout uses it so an abandoned (timed-out)
// handler goroutine can keep writing into it harmlessly, without racing with
// the timeout response on the real connection.
type bufferedWriter struct {
	header      http.Header
	body        bytes.Buffer
	status      int
	wroteHeader bool
}

func newBufferedWriter() *bufferedWriter {
	return &bufferedWriter{header: make(http.Header)}
}

func (b *bufferedWriter) Header() http.Header { return b.header }

func (b *bufferedWriter) WriteHeader(code int) {
	if b.wroteHeader {
		return
	}
	b.wroteHeader = true
	b.status = code
}

func (b *bufferedWriter) Write(p []byte) (int, error) {
	if !b.wroteHeader {
		b.WriteHeader(http.StatusOK)
	}
	return b.body.Write(p)
}

// Timeout bounds request handling to d. If the handler doesn't finish in
// time, it responds with a 504 JSON envelope; the handler goroutine is left
// to finish writing into a discarded buffer rather than being killed
// outright (Go has no safe way to preempt a running goroutine).
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			r = r.WithContext(ctx)

			pw := newBufferedWriter()
			done := make(chan struct{})
			go func() {
				defer close(done)
				next.ServeHTTP(pw, r)
			}()

			select {
			case <-done:
				for k, v := range pw.header {
					w.Header()[k] = v
				}
				status := pw.status
				if status == 0 {
					status = http.StatusOK
				}
				w.WriteHeader(status)
				_, _ = w.Write(pw.body.Bytes())
			case <-ctx.Done():
				writeJSONError(w, http.StatusGatewayTimeout, "timeout", "request timed out")
			}
		})
	}
}
