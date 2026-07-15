package middleware

import (
	"net/http"
	"sync"
)

// statusWriter wraps an http.ResponseWriter to capture the status code that
// was written, and to make concurrent/duplicate writes safe. It is shared by
// the logging, recovery and timeout middlewares so they all observe (and can
// safely set) the same final status for a request.
type statusWriter struct {
	http.ResponseWriter
	mu          sync.Mutex
	status      int
	wroteHeader bool
}

func newStatusWriter(w http.ResponseWriter) *statusWriter {
	return &statusWriter{ResponseWriter: w}
}

func (w *statusWriter) WriteHeader(code int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	if !w.wroteHeader {
		w.wroteHeader = true
		w.status = http.StatusOK
		w.ResponseWriter.WriteHeader(http.StatusOK)
	}
	w.mu.Unlock()
	return w.ResponseWriter.Write(b)
}

// Status returns the status code written so far, or 0 if none yet.
func (w *statusWriter) Status() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.status
}

// TryWriteHeader writes the header only if nothing has been written yet.
// It reports whether it actually wrote the header.
func (w *statusWriter) TryWriteHeader(code int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.wroteHeader {
		return false
	}
	w.wroteHeader = true
	w.status = code
	w.ResponseWriter.WriteHeader(code)
	return true
}
