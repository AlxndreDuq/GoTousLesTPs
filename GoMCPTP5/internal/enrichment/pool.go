package enrichment

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"mira-tp4/internal/core"
)

// Pool is a bounded set of workers consuming jobs from a Dispatcher's
// channel and applying an Enricher, writing results back through
// core.Repository. Each job gets its own timeout, independent of the
// process's shutdown context, so an in-flight job started before shutdown
// can still finish draining rather than being cut off mid-write.
type Pool struct {
	repo     core.Repository
	enricher Enricher
	logger   *slog.Logger
	timeout  time.Duration
}

func NewPool(repo core.Repository, enricher Enricher, logger *slog.Logger, timeout time.Duration) *Pool {
	return &Pool{repo: repo, enricher: enricher, logger: logger, timeout: timeout}
}

// Start launches n worker goroutines reading from jobs until it is closed,
// and returns a WaitGroup callers can Wait() on to know every worker has
// exited (used for graceful shutdown: close the dispatcher, then wait).
func (p *Pool) Start(jobs <-chan Job, workers int) *sync.WaitGroup {
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			p.worker(jobs)
		}()
	}
	return &wg
}

func (p *Pool) worker(jobs <-chan Job) {
	for job := range jobs {
		p.process(job)
	}
}

func (p *Pool) process(job Job) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	note, err := p.repo.Get(ctx, job.NoteID)
	if err != nil {
		p.logger.Error("enrichment_fetch_failed", "note_id", job.NoteID, "error", err)
		return
	}

	result, err := p.enricher.Enrich(ctx, note)
	if err != nil {
		p.logger.Error("enrichment_failed", "note_id", job.NoteID, "error", err)
		result = core.EnrichmentResult{NoteID: note.ID, Status: core.EnrichmentFailed}
	}

	if err := p.repo.SaveEnrichment(ctx, result); err != nil {
		p.logger.Error("enrichment_save_failed", "note_id", job.NoteID, "error", err)
		return
	}

	p.logger.Info("enrichment_done", "note_id", job.NoteID, "status", result.Status)
}
