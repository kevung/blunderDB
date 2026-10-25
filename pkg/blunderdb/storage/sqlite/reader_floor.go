package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// readerFloor returns the recorded storage.MinReaderVersionKey, "" when the
// database predates it.
func readerFloor(ctx context.Context, db *sql.DB) (string, error) {
	var floor string
	err := db.QueryRowContext(ctx, `SELECT value FROM metadata WHERE key = ?`, storage.MinReaderVersionKey).Scan(&floor)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("sqlite: read %s: %w", storage.MinReaderVersionKey, err)
	}
	return floor, nil
}

// CheckReaderFloor refuses a database whose recorded floor is newer than the
// schema this binary speaks (storage.ErrDatabaseTooNew). Read only.
func CheckReaderFloor(ctx context.Context, db *sql.DB) error {
	floor, err := readerFloor(ctx, db)
	if err != nil {
		return err
	}
	return storage.CheckReaderFloor(floor)
}

// RaiseReaderFloor records storage.RaisedReaderFloor on a database open for
// writing. It writes only when the value changes: an open with nothing to
// raise leaves the file as it was.
func RaiseReaderFloor(ctx context.Context, db *sql.DB) error {
	floor, err := readerFloor(ctx, db)
	if err != nil {
		return err
	}
	raised := storage.RaisedReaderFloor(floor)
	if raised == floor {
		return nil
	}
	if _, err := db.ExecContext(ctx,
		`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, storage.MinReaderVersionKey, raised); err != nil {
		return fmt.Errorf("sqlite: record %s: %w", storage.MinReaderVersionKey, err)
	}
	return nil
}
