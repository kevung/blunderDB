package sqlite

import (
	"errors"
	"fmt"

	sqlite3 "modernc.org/sqlite"
	sqlite3lib "modernc.org/sqlite/lib"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// referenced maps a FOREIGN KEY violation onto storage.ErrNotFound: the row
// the caller pointed at (a position, a collection) does not exist, which is
// the same situation Load/Get report for a bad id. Left unmapped the driver
// error surfaces as ErrInternal and every API turns it into a 500 for what
// is a client mistake. Any other error passes through unchanged.
func referenced(err error) error {
	var se *sqlite3.Error
	if errors.As(err, &se) && se.Code() == sqlite3lib.SQLITE_CONSTRAINT_FOREIGNKEY {
		return fmt.Errorf("%w: %w", storage.ErrNotFound, err)
	}
	return err
}

// isUniqueViolation reports whether err is SQLite refusing a row for a UNIQUE
// index — the Zobrist index on position, for what this package uses it for.
func isUniqueViolation(err error) bool {
	var se *sqlite3.Error
	return errors.As(err, &se) && se.Code() == sqlite3lib.SQLITE_CONSTRAINT_UNIQUE
}
