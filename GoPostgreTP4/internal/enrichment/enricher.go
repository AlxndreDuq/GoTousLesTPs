package enrichment

import (
	"context"

	"mira-tp4/internal/core"
)

// Enricher computes the derived fields (tags, summary, score, embedding) for
// a note. NaiveEnricher is the only implementation today; the interface
// exists so a real model-backed implementation can be swapped in later
// without touching the dispatcher/worker plumbing.
type Enricher interface {
	Enrich(ctx context.Context, note core.Note) (core.EnrichmentResult, error)
}
