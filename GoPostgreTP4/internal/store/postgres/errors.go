package postgres

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"mira-tp4/internal/core"
)

// invalidTextRepresentation is Postgres's error code for a malformed input
// value (e.g. an id that isn't a valid UUID) — treated as "not found" since
// the caller can't tell the difference from the outside.
const invalidTextRepresentation = "22P02"

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		return core.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation {
		return core.ErrNotFound
	}
	return err
}
