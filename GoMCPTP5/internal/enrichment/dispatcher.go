package enrichment

import (
	"log/slog"
	"sync"
)

// Dispatcher owns the internal channel enrichment jobs travel through. It
// implements core.EnrichmentQueue so the HTTP write path (POST/PATCH) can
// publish a job without importing this package or knowing about workers.
type Dispatcher struct {
	jobs   chan Job
	logger *slog.Logger
	once   sync.Once
}

func NewDispatcher(bufferSize int, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{
		jobs:   make(chan Job, bufferSize),
		logger: logger,
	}
}

// Enqueue publishes a job without blocking: if the buffered channel is full
// (the worker pool can't keep up), the job is dropped and logged rather than
// stalling the HTTP response that triggered it. The note simply stays in
// "pending" enrichment status.
func (d *Dispatcher) Enqueue(noteID string) {
	select {
	case d.jobs <- Job{NoteID: noteID}:
	default:
		d.logger.Warn("enrichment_queue_full", "note_id", noteID)
	}
}

// Jobs exposes the read side of the channel for the worker pool.
func (d *Dispatcher) Jobs() <-chan Job {
	return d.jobs
}

// Close stops accepting new jobs conceptually and lets workers drain the
// buffer before exiting. Callers must guarantee no Enqueue call happens
// after Close (main does this by shutting down the HTTP server first).
func (d *Dispatcher) Close() {
	d.once.Do(func() { close(d.jobs) })
}
