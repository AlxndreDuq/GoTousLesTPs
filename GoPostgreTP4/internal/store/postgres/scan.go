package postgres

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"mira-tp4/internal/core"
)

// querier is satisfied by both *pgxpool.Pool and pgx.Tx, so helpers can run
// either directly against the pool or inside a caller-managed transaction.
type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

const noteColumns = `id, title, content, status, enrichment_status, summary, score, created_at, updated_at`

func scanNote(row pgx.Row) (core.Note, error) {
	var n core.Note
	var score sql.NullFloat64

	err := row.Scan(&n.ID, &n.Title, &n.Content, &n.Status, &n.EnrichmentStatus, &n.Summary, &score, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return core.Note{}, err
	}
	if score.Valid {
		n.Score = &score.Float64
	}
	return n, nil
}

func scanNotes(rows pgx.Rows) ([]core.Note, error) {
	defer rows.Close()

	notes := make([]core.Note, 0)
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func noteIDs(notes []core.Note) []string {
	ids := make([]string, len(notes))
	for i, n := range notes {
		ids[i] = n.ID
	}
	return ids
}
