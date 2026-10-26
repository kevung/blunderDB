package sqlshared

import (
	"context"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ReaderFloor returns the recorded storage.MinReaderVersionKey, "" when the
// database predates it. The metadata table is global in both schemas, so the
// floor is one value per database, never per tenant.
func ReaderFloor(ctx context.Context, db Execer) (string, error) {
	var floor string
	err := db.QueryRow(ctx, `SELECT value FROM metadata WHERE key = ?`, storage.MinReaderVersionKey).Scan(&floor)
	if errors.Is(err, ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", errf(db, "read "+storage.MinReaderVersionKey, err)
	}
	return floor, nil
}

// CheckReaderFloor refuses a database whose recorded floor is newer than the
// schema this binary speaks (storage.ErrDatabaseTooNew). Read only.
func CheckReaderFloor(ctx context.Context, db Execer) error {
	floor, err := ReaderFloor(ctx, db)
	if err != nil {
		return err
	}
	return storage.CheckReaderFloor(floor)
}

// RaiseReaderFloor records storage.RaisedReaderFloor on a database open for
// writing. It writes only when the value changes: an open with nothing to
// raise leaves the database as it was.
func RaiseReaderFloor(ctx context.Context, db Execer) error {
	floor, err := ReaderFloor(ctx, db)
	if err != nil {
		return err
	}
	raised := storage.RaisedReaderFloor(floor)
	if raised == floor {
		return nil
	}
	if _, err := db.Exec(ctx, upsertMetadataSQL, storage.MinReaderVersionKey, raised); err != nil {
		return errf(db, "record "+storage.MinReaderVersionKey, err)
	}
	return nil
}
