package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// pgForeignKeyViolation is SQLSTATE 23503 (Class 23 — integrity constraint).
const pgForeignKeyViolation = "23503"

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
