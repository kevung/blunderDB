package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// pgForeignKeyViolation is SQLSTATE 23503 (Class 23 — integrity constraint);
// pgUniqueViolation is 23505, the same class.
const (
	pgForeignKeyViolation = "23503"
	pgUniqueViolation     = "23505"
)

// isUniqueViolation reports whether err is PostgreSQL refusing a row for a
// UNIQUE index — the (tenant_id, zobrist_hash) index on position here.
func isUniqueViolation(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == pgUniqueViolation
}

// referenced maps a FOREIGN KEY violation onto storage.ErrNotFound — the row
// the caller pointed at does not exist, the same thing Get reports for a bad
// id. Mirrors the SQLite backend; any other error passes through unchanged.
func referenced(err error) error {
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == pgForeignKeyViolation {
		return fmt.Errorf("%w: %w", storage.ErrNotFound, err)
	}
	return err
}
