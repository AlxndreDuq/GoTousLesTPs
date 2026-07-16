package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"mira-tp4/internal/core"
)

// Repository is a pgx-backed implementation of core.Repository.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a note and its initial tags in a single transaction: if
// either write fails, neither is kept.
func (r *Repository) Create(ctx context.Context, note core.Note) (core.Note, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return core.Note{}, fmt.Errorf("begin create note: %w", err)
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		INSERT INTO notes (title, content, status, enrichment_status)
		VALUES ($1, $2, $3, $4)
		RETURNING `+noteColumns, note.Title, note.Content, note.Status, core.EnrichmentPending)

	created, err := scanNote(row)
	if err != nil {
		return core.Note{}, fmt.Errorf("insert note: %w", mapErr(err))
	}

	if err := insertTags(ctx, tx, created.ID, note.Tags); err != nil {
		return core.Note{}, fmt.Errorf("insert tags: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return core.Note{}, fmt.Errorf("commit create note: %w", err)
	}

	created.Tags = note.Tags
	return created, nil
}

func (r *Repository) Get(ctx context.Context, id string) (core.Note, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+noteColumns+` FROM notes WHERE id::text = $1`, id)
	note, err := scanNote(row)
	if err != nil {
		return core.Note{}, mapErr(err)
	}

	tags, err := loadTags(ctx, r.pool, []string{note.ID})
	if err != nil {
		return core.Note{}, fmt.Errorf("load tags: %w", err)
	}
	note.Tags = tags[note.ID]
	return note, nil
}

func (r *Repository) List(ctx context.Context, filter core.ListFilter) (core.ListResult, error) {
	where := ""
	args := []any{}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where = "WHERE status = $1"
	}

	var total int
	countQuery := "SELECT count(*) FROM notes " + where
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return core.ListResult{}, fmt.Errorf("count notes: %w", err)
	}

	args = append(args, filter.Limit, filter.Offset)
	listQuery := fmt.Sprintf(`
		SELECT %s FROM notes
		%s
		ORDER BY created_at ASC
		LIMIT $%d OFFSET $%d
	`, noteColumns, where, len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return core.ListResult{}, fmt.Errorf("list notes: %w", err)
	}
	notes, err := scanNotes(rows)
	if err != nil {
		return core.ListResult{}, fmt.Errorf("scan notes: %w", err)
	}

	if err := attachTags(ctx, r.pool, notes); err != nil {
		return core.ListResult{}, err
	}

	return core.ListResult{Notes: notes, Total: total}, nil
}

func (r *Repository) Update(ctx context.Context, id string, patch core.UpdateInput) (core.Note, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return core.Note{}, fmt.Errorf("begin update note: %w", err)
	}
	defer tx.Rollback(ctx)

	sets := []string{"updated_at = now()"}
	args := []any{}
	set := func(column string, val any) {
		args = append(args, val)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if patch.Title != nil {
		set("title", *patch.Title)
	}
	if patch.Content != nil {
		set("content", *patch.Content)
	}
	if patch.Status != nil {
		set("status", *patch.Status)
	}
	if patch.Title != nil || patch.Content != nil {
		// The previous enrichment (tags/summary/score/embedding) no longer
		// reflects the note's text; the service enqueues a fresh job.
		sets = append(sets, "enrichment_status = 'pending'")
	}

	args = append(args, id)
	query := fmt.Sprintf(`UPDATE notes SET %s WHERE id::text = $%d RETURNING `+noteColumns,
		strings.Join(sets, ", "), len(args))

	row := tx.QueryRow(ctx, query, args...)
	updated, err := scanNote(row)
	if err != nil {
		return core.Note{}, mapErr(err)
	}

	if patch.Tags != nil {
		if err := replaceTags(ctx, tx, updated.ID, *patch.Tags); err != nil {
			return core.Note{}, fmt.Errorf("replace tags: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return core.Note{}, fmt.Errorf("commit update note: %w", err)
	}

	tags, err := loadTags(ctx, r.pool, []string{updated.ID})
	if err != nil {
		return core.Note{}, fmt.Errorf("load tags: %w", err)
	}
	updated.Tags = tags[updated.ID]
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM notes WHERE id::text = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return core.ErrNotFound
	}
	return nil
}

// SaveEnrichment persists an enrichment task's outcome: it always updates
// the note's status/summary/score, and on success additionally merges in
// the auto-generated tags (without touching user-supplied ones) and upserts
// the embedding — all in one transaction.
func (r *Repository) SaveEnrichment(ctx context.Context, result core.EnrichmentResult) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save enrichment: %w", err)
	}
	defer tx.Rollback(ctx)

	var scoreArg any
	if result.Status == core.EnrichmentDone {
		scoreArg = result.Score
	}

	if _, err := tx.Exec(ctx, `
		UPDATE notes SET enrichment_status = $1, summary = $2, score = $3, updated_at = now()
		WHERE id::text = $4
	`, result.Status, result.Summary, scoreArg, result.NoteID); err != nil {
		return fmt.Errorf("update note enrichment: %w", mapErr(err))
	}

	if result.Status == core.EnrichmentDone {
		if err := insertTags(ctx, tx, result.NoteID, result.Tags); err != nil {
			return fmt.Errorf("merge enrichment tags: %w", err)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO note_embeddings (note_id, embedding, updated_at)
			VALUES ($1, $2, now())
			ON CONFLICT (note_id) DO UPDATE SET embedding = EXCLUDED.embedding, updated_at = now()
		`, result.NoteID, vectorOf(result.Embedding)); err != nil {
			return fmt.Errorf("upsert embedding: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit save enrichment: %w", err)
	}
	return nil
}

// attachTags loads tags for a batch of notes and sets each note's Tags
// field in place.
func attachTags(ctx context.Context, db querier, notes []core.Note) error {
	tagsByID, err := loadTags(ctx, db, noteIDs(notes))
	if err != nil {
		return fmt.Errorf("load tags: %w", err)
	}
	for i := range notes {
		notes[i].Tags = tagsByID[notes[i].ID]
	}
	return nil
}
