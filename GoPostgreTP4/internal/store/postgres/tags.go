package postgres

import "context"

// loadTags fetches the tags for a batch of note ids in one round trip,
// returning them grouped by note id (missing/untagged ids simply absent
// from the map).
func loadTags(ctx context.Context, db querier, ids []string) (map[string][]string, error) {
	if len(ids) == 0 {
		return map[string][]string{}, nil
	}

	rows, err := db.Query(ctx, `
		SELECT note_id, tag FROM note_tags
		WHERE note_id::text = ANY($1)
		ORDER BY tag
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]string, len(ids))
	for rows.Next() {
		var noteID, tag string
		if err := rows.Scan(&noteID, &tag); err != nil {
			return nil, err
		}
		result[noteID] = append(result[noteID], tag)
	}
	return result, rows.Err()
}

// insertTags adds tags to a note, skipping ones already present. It is
// additive by design: both the create-transaction (user-supplied tags) and
// the enrichment worker (auto-generated tags) call it, and neither should
// wipe out tags the other one added.
func insertTags(ctx context.Context, db querier, noteID string, tags []string) error {
	for _, tag := range tags {
		if _, err := db.Exec(ctx, `
			INSERT INTO note_tags (note_id, tag) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, noteID, tag); err != nil {
			return err
		}
	}
	return nil
}

// replaceTags fully overwrites a note's tag set — used when a client PATCHes
// the tags field explicitly, as opposed to the enrichment worker's additive
// insertTags.
func replaceTags(ctx context.Context, db querier, noteID string, tags []string) error {
	if _, err := db.Exec(ctx, `DELETE FROM note_tags WHERE note_id = $1`, noteID); err != nil {
		return err
	}
	return insertTags(ctx, db, noteID, tags)
}
