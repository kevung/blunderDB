package sqlite

import (
	"context"
	"database/sql"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// CheckReaderFloor refuses a database whose recorded floor is newer than the
// schema this binary speaks (storage.ErrDatabaseTooNew). Read only.
func CheckReaderFloor(ctx context.Context, db *sql.DB) error {
	return sqlshared.CheckReaderFloor(ctx, binder{db: db}.shared())
}

// RaiseReaderFloor records storage.RaisedReaderFloor on a database open for
// writing. It writes only when the value changes: an open with nothing to
// raise leaves the file as it was.
func RaiseReaderFloor(ctx context.Context, db *sql.DB) error {
	return sqlshared.RaiseReaderFloor(ctx, binder{db: db}.shared())
}
