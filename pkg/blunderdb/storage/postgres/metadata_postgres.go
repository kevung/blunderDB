package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type metadataStore struct{ db execer }

var _ storage.MetadataStore = (*metadataStore)(nil)

// metadataVersionKey is the metadata row that records the schema version. The
// metadata table is database-global (not tenant-scoped).
const metadataVersionKey = "database_version"

// Version returns the recorded schema version, or ErrNotFound when absent.
func (s *metadataStore) Version(ctx context.Context, scope string) (string, error) {
	var v string
	err := s.db.QueryRow(ctx, `SELECT value FROM metadata WHERE key = $1`, metadataVersionKey).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("postgres: database version: %w", storage.ErrNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("postgres: database version: %w", err)
	}
	return v, nil
}

// SetVersion records the schema version.
func (s *metadataStore) SetVersion(ctx context.Context, scope string, version string) error {
	if _, err := s.db.Exec(ctx,
		`INSERT INTO metadata (key, value) VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
		metadataVersionKey, version); err != nil {
		return fmt.Errorf("postgres: set database version: %w", err)
	}
	return nil
}

// Load returns every metadata key/value pair.
func (s *metadataStore) Load(ctx context.Context, scope string) (map[string]string, error) {
	rows, err := s.db.Query(ctx, `SELECT key, COALESCE(value, '') FROM metadata`)
	if err != nil {
		return nil, fmt.Errorf("postgres: load metadata: %w", err)
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("postgres: load metadata: %w", err)
		}
		out[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: load metadata: %w", err)
	}
	return out, nil
}

// Save writes the given metadata key/value pairs, replacing existing keys.
func (s *metadataStore) Save(ctx context.Context, scope string, metadata map[string]string) error {
	err := withTx(ctx, s.db, func(tx execer) error {
		for key, value := range metadata {
			if _, err := tx.Exec(ctx,
				`INSERT INTO metadata (key, value) VALUES ($1, $2)
				 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
				key, value); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("postgres: save metadata: %w", err)
	}
	return nil
}

// Counts returns the headline row counts for the tenant scope.
func (s *metadataStore) Counts(ctx context.Context, scope string) (storage.Counts, error) {
	tenant := tenantID(scope)
	var c storage.Counts
	err := s.db.QueryRow(ctx, `SELECT
		(SELECT COUNT(*) FROM position WHERE tenant_id = $1),
		(SELECT COUNT(*) FROM analysis WHERE tenant_id = $1),
		(SELECT COUNT(*) FROM match    WHERE tenant_id = $1),
		(SELECT COUNT(*) FROM game     WHERE tenant_id = $1),
		(SELECT COUNT(*) FROM move     WHERE tenant_id = $1)`, tenant).
		Scan(&c.Positions, &c.Analyses, &c.Matches, &c.Games, &c.Moves)
	if err != nil {
		return storage.Counts{}, fmt.Errorf("postgres: database counts: %w", err)
	}
	return c, nil
}
