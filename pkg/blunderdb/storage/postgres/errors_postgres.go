package postgres

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// pgForeignKeyViolation is SQLSTATE 23503 (Class 23 — integrity constraint);
// pgUniqueViolation is 23505, the same class.
const (
	pgForeignKeyViolation = "23503"
	pgUniqueViolation     = "23505"
	pgNumericOutOfRange   = "22003"
)

// isUniqueViolation reports whether err is PostgreSQL refusing a row for a
// UNIQUE index — the (tenant_id, zobrist_hash) index on position here.
func isUniqueViolation(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == pgUniqueViolation
}

// referenced maps a FOREIGN KEY violation onto storage.ErrNotFound — the row
// the caller pointed at does not exist, the same thing Get reports for a bad
// id. Mirrors the SQLite backend.
func referenced(err error) error {
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == pgForeignKeyViolation {
		return fmt.Errorf("%w: %w", storage.ErrNotFound, err)
	}
	return outOfRange(err)
}

// outOfRange names a value wider than its column an invalid one: refused, whether
// by the server (22003) or by the driver encoding the parameter of a narrow column.
func outOfRange(err error) error {
	var pe *pgconn.PgError
	if err == nil {
		return nil
	}
	if (errors.As(err, &pe) && pe.Code == pgNumericOutOfRange) ||
		strings.Contains(err.Error(), "than maximum value for int") || strings.Contains(err.Error(), "than minimum value for int") {
		return fmt.Errorf("%w: %w", storage.ErrInvalid, err)
	}
	return err
}

// isForeignKeyViolation reports whether err is PostgreSQL refusing a row for
// the FOREIGN KEY named constraint. The name is part of the test: a table with
// several foreign keys would otherwise have a refusal on one of them taken for
// a refusal on the one the caller expects.
func isForeignKeyViolation(err error, constraint string) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == pgForeignKeyViolation && pe.ConstraintName == constraint
}
