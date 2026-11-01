package sqlshared

import (
	"context"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ChangeJournalStore implements storage.ChangeJournalStore over change_log.
type ChangeJournalStore struct{ DB Execer }

var _ storage.ChangeJournalStore = (*ChangeJournalStore)(nil)

// changedAtText renders changed_at as UTC RFC 3339 with milliseconds on both
// backends (SQLite stores that text; PostgreSQL a timestamptz).
func (s *ChangeJournalStore) changedAtText() string {
	if s.DB.Name() == "sqlite" {
		return "changed_at"
	}
	return `to_char(changed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')`
}

func (s *ChangeJournalStore) Changes(ctx context.Context, scope string, after int64, limit int) ([]storage.Change, error) {
	if limit <= 0 {
		limit = 1000
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	rows, err := s.DB.Query(ctx, `SELECT seq, CAST(uuid AS TEXT), family, op, `+s.changedAtText()+` FROM change_log
		WHERE `+tenant+` AND seq > ? ORDER BY seq LIMIT ?`, append(targs, after, limit)...)
	if err != nil {
		return nil, errf(s.DB, "changes", err)
	}
	defer rows.Close()
	var out []storage.Change
	for rows.Next() {
		var c storage.Change
		if err := rows.Scan(&c.Seq, &c.UUID, &c.Family, &c.Op, &c.ChangedAt); err != nil {
			return nil, errf(s.DB, "changes", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "changes", err)
	}
	return out, nil
}

func (s *ChangeJournalStore) Entry(ctx context.Context, scope, uuid string) (storage.Change, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	var c storage.Change
	err := s.DB.QueryRow(ctx, `SELECT seq, CAST(uuid AS TEXT), family, op, `+s.changedAtText()+` FROM change_log
		WHERE `+tenant+` AND CAST(uuid AS TEXT) = ?`, append(targs, uuid)...).Scan(&c.Seq, &c.UUID, &c.Family, &c.Op, &c.ChangedAt)
	if errors.Is(err, ErrNoRows) {
		return storage.Change{}, errf(s.DB, "change entry", storage.ErrNotFound)
	}
	if err != nil {
		return storage.Change{}, errf(s.DB, "change entry", err)
	}
	return c, nil
}

func (s *ChangeJournalStore) Mark(ctx context.Context, scope, family, uuid, op, at string) error {
	if uuid == "" {
		return nil
	}
	if _, err := objectUUIDTable(family); err != nil {
		return errf(s.DB, "mark change", err)
	}
	switch op {
	case storage.ChangeCreate, storage.ChangeUpdate, storage.ChangeDelete, storage.ChangeTrash:
	default:
		return errf(s.DB, "mark change", fmt.Errorf("operation %q: %w", op, storage.ErrInvalid))
	}
	var err error
	if s.DB.Name() == "sqlite" {
		if at == "" {
			_, err = s.DB.Exec(ctx, `INSERT OR REPLACE INTO change_log (uuid, family, op) VALUES (?, ?, ?)`, uuid, family, op)
		} else {
			_, err = s.DB.Exec(ctx, `INSERT OR REPLACE INTO change_log (uuid, family, op, changed_at) VALUES (?, ?, ?, ?)`, uuid, family, op, at)
		}
	} else {
		// at is bound as text and cast; "" falls to now().
		_, err = s.DB.Exec(ctx, `INSERT INTO change_log (tenant_id, uuid, family, op, changed_at)
			VALUES (?, CAST(? AS uuid), ?, ?, COALESCE(CAST(NULLIF(?, '') AS timestamptz), now()))
			ON CONFLICT (tenant_id, uuid) DO UPDATE SET seq = nextval('change_log_seq'), family = EXCLUDED.family,
			op = EXCLUDED.op, changed_at = EXCLUDED.changed_at`, s.DB.ScopeArg(scope), uuid, family, op, at)
	}
	if err != nil {
		return errf(s.DB, "mark change", err)
	}
	return nil
}
