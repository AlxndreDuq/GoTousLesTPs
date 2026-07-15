package postgres

import (
	"context"
	"fmt"

	"github.com/pgvector/pgvector-go"

	"mira-tp4/internal/core"
	"mira-tp4/internal/embedding"
)

// searchLimit bounds how many results a search returns; unlike List, Search
// has no pagination in the API contract (mirrors TP2's simple search).
const searchLimit = 50

// vectorSimilarityFloor discards vector matches too weak to be meaningful
// noise from the deterministic (non-semantic) embedding used here — a real
// embedding model would make a lower/zero floor meaningful too.
const vectorSimilarityFloor = 0.15

// fulltextWeight/vectorWeight combine two differently-scaled signals
// (ts_rank is open-ended, roughly 0..0.6 in practice; cosine similarity is
// bounded to [-1,1]) into a single ranking score. This is a simple fixed
// heuristic, not a calibrated fusion — good enough to demonstrate a hybrid
// query end to end.
const (
	fulltextWeight = 0.6
	vectorWeight   = 0.4
)

var hybridSearchQuery = fmt.Sprintf(`
	WITH scored AS (
		SELECT
			%s,
			COALESCE(ts_rank(n.search_vector, plainto_tsquery('french', $1)), 0) AS fts_rank,
			CASE WHEN e.embedding IS NULL THEN 0
			     ELSE GREATEST(0, 1 - (e.embedding <=> $2))
			END AS vec_sim
		FROM notes n
		LEFT JOIN note_embeddings e ON e.note_id = n.id
	)
	SELECT %s
	FROM scored
	WHERE fts_rank > 0 OR vec_sim > $3
	ORDER BY (%f * fts_rank) + (%f * vec_sim) DESC, created_at DESC
	LIMIT $4
`, qualifiedNoteColumns("n"), noteColumns, fulltextWeight, vectorWeight)

func qualifiedNoteColumns(alias string) string {
	return alias + ".id, " + alias + ".title, " + alias + ".content, " + alias + ".status, " +
		alias + ".enrichment_status, " + alias + ".summary, " + alias + ".score, " +
		alias + ".created_at, " + alias + ".updated_at"
}

// Search runs a hybrid query: full-text (GIN index on notes.search_vector)
// combined with cosine similarity against note_embeddings (HNSW index),
// produced by the enrichment pipeline. Notes not enriched yet still surface
// through the full-text half of the query.
func (r *Repository) Search(ctx context.Context, query string) (core.ListResult, error) {
	queryVector := pgvector.NewVector(embedding.Embed(query))

	rows, err := r.pool.Query(ctx, hybridSearchQuery, query, queryVector, vectorSimilarityFloor, searchLimit)
	if err != nil {
		return core.ListResult{}, fmt.Errorf("hybrid search: %w", err)
	}
	notes, err := scanNotes(rows)
	if err != nil {
		return core.ListResult{}, fmt.Errorf("scan search results: %w", err)
	}

	if err := attachTags(ctx, r.pool, notes); err != nil {
		return core.ListResult{}, err
	}

	return core.ListResult{Notes: notes, Total: len(notes)}, nil
}

func vectorOf(vec []float32) pgvector.Vector {
	return pgvector.NewVector(vec)
}
