package sqlshared

import (
	"context"
	"errors"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// MetadataStore implements storage.MetadataStore. The metadata table is
// database-global (not tenant-scoped) in both schemas; only Counts, which reads
// the domain tables, is confined to the scope's tenant.
type MetadataStore struct{ DB Execer }

var _ storage.MetadataStore = (*MetadataStore)(nil)

// metadataVersionKey is the metadata row that records the schema version.
const metadataVersionKey = "database_version"

// upsertMetadataSQL writes one key/value pair, replacing an existing key. The
// ON CONFLICT form is common to PostgreSQL and SQLite (≥ 3.24).
const upsertMetadataSQL = `INSERT INTO metadata (key, value) VALUES (?,?)
	ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`

// Version returns the recorded schema version, or ErrNotFound when the row is
// absent.
func (s *MetadataStore) Version(ctx context.Context, scope string) (string, error) {
	var v string
	err := s.DB.QueryRow(ctx, `SELECT value FROM metadata WHERE key = ?`, metadataVersionKey).Scan(&v)
	if errors.Is(err, ErrNoRows) {
		return "", errf(s.DB, "database version", storage.ErrNotFound)
	}
	if err != nil {
		return "", errf(s.DB, "database version", err)
	}
	return v, nil
}

// SetVersion records the schema version.
func (s *MetadataStore) SetVersion(ctx context.Context, scope string, version string) error {
	if _, err := s.DB.Exec(ctx, upsertMetadataSQL, metadataVersionKey, version); err != nil {
		return errf(s.DB, "set database version", err)
	}
	return nil
}

// Load returns every metadata key/value pair.
func (s *MetadataStore) Load(ctx context.Context, scope string) (map[string]string, error) {
	rows, err := s.DB.Query(ctx, `SELECT key, COALESCE(value,'') FROM metadata`)
	if err != nil {
		return nil, errf(s.DB, "load metadata", err)
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, errf(s.DB, "load metadata", err)
		}
		out[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "load metadata", err)
	}
	return out, nil
}

// Save writes the given metadata key/value pairs, replacing existing keys.
func (s *MetadataStore) Save(ctx context.Context, scope string, metadata map[string]string) error {
	err := s.DB.Transact(ctx, func(tx Execer) error {
		for key, value := range metadata {
			if _, err := tx.Exec(ctx, upsertMetadataSQL, key, value); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return errf(s.DB, "save metadata", err)
	}
	return nil
}

// Counts returns the headline row counts of the scope's tenant.
func (s *MetadataStore) Counts(ctx context.Context, scope string) (storage.Counts, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	tables := []string{
		"position", "analysis", "match", "game", "move",
		"position WHERE " + s.DB.Bool("individually_imported", true) + " AND",
		"anki_card",
	}
	selects := make([]string, len(tables))
	var args []any
	for i, t := range tables {
		if strings.Contains(t, " WHERE ") {
			selects[i] = "(SELECT COUNT(*) FROM " + t + " " + tenant + ")"
		} else {
			selects[i] = "(SELECT COUNT(*) FROM " + t + " WHERE " + tenant + ")"
		}
		args = append(args, targs...)
	}
	var c storage.Counts
	err := s.DB.QueryRow(ctx, "SELECT "+strings.Join(selects, ", "), args...).
		Scan(&c.Positions, &c.Analyses, &c.Matches, &c.Games, &c.Moves,
			&c.IndividualPositions, &c.AnkiCards)
	if err != nil {
		return storage.Counts{}, errf(s.DB, "database counts", err)
	}
	return c, nil
}
