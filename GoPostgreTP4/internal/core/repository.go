package core

import "context"

// Repository is the persistence contract the service depends on.
// It is implemented by internal/store/postgres.
type Repository interface {
	Create(ctx context.Context, note Note) (Note, error)
	Get(ctx context.Context, id string) (Note, error)
	List(ctx context.Context, filter ListFilter) (ListResult, error)
	Update(ctx context.Context, id string, patch UpdateInput) (Note, error)
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, query string) (ListResult, error)

	// SaveEnrichment persists the outcome of an asynchronous enrichment task
	// (tags, summary, score, embedding) produced by internal/enrichment.
	SaveEnrichment(ctx context.Context, result EnrichmentResult) error
}

// EnrichmentQueue publishes enrichment jobs to the internal channel-backed
// pipeline (internal/enrichment). It is implemented by enrichment.Dispatcher.
// Enqueue must not block the caller (the HTTP write path): a saturated queue
// drops the job rather than stalling the response.
type EnrichmentQueue interface {
	Enqueue(noteID string)
}
