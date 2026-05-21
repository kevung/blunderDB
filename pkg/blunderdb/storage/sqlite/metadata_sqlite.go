package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type metadataStore struct{ db execer }

var _ storage.MetadataStore = (*metadataStore)(nil)

// metadataVersionKey is the metadata row that records the schema version.
const metadataVersionKey = "database_version"

// Version returns the recorded schema version, or ErrNotFound when the row is
// absent.
func (s *metadataStore) Version(ctx context.Context, scope string) (string, error) {
	var v string
	err := s.db.QueryRowContext(ctx,
		`SELECT value FROM metadata WHERE key = ?`, metadataVersionKey).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("sqlite: database version: %w", storage.ErrNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("sqlite: database version: %w", err)
	}
	return v, nil
}

// SetVersion records the schema version.
func (s *metadataStore) SetVersion(ctx context.Context, scope string, version string) error {
	if _, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO metadata (key, value) VALUES (?,?)`,
		metadataVersionKey, version); err != nil {
		return fmt.Errorf("sqlite: set database version: %w", err)
	}
	return nil
}

// Load returns every metadata key/value pair.
func (s *metadataStore) Load(ctx context.Context, scope string) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, COALESCE(value,'') FROM metadata`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: load metadata: %w", err)
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("sqlite: load metadata: %w", err)
		}
		out[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: load metadata: %w", err)
	}
	return out, nil
}

// Save writes the given metadata key/value pairs, replacing existing keys.
func (s *metadataStore) Save(ctx context.Context, scope string, metadata map[string]string) error {
	err := withTx(ctx, s.db, func(tx execer) error {
		for key, value := range metadata {
			if _, err := tx.ExecContext(ctx,
				`INSERT OR REPLACE INTO metadata (key, value) VALUES (?,?)`,
				key, value); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sqlite: save metadata: %w", err)
	}
	return nil
}

// Counts returns the headline row counts of the database.
func (s *metadataStore) Counts(ctx context.Context, scope string) (storage.Counts, error) {
	var c storage.Counts
	err := s.db.QueryRowContext(ctx, `SELECT
		(SELECT COUNT(*) FROM position),
		(SELECT COUNT(*) FROM analysis),
		(SELECT COUNT(*) FROM match),
		(SELECT COUNT(*) FROM game),
		(SELECT COUNT(*) FROM move)`).
		Scan(&c.Positions, &c.Analyses, &c.Matches, &c.Games, &c.Moves)
	if err != nil {
		return storage.Counts{}, fmt.Errorf("sqlite: database counts: %w", err)
	}
	return c, nil
}
