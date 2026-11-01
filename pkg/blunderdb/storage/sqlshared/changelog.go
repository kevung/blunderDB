package sqlshared

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ChangeJournalStore implements storage.ChangeJournalStore over change_log.
type ChangeJournalStore struct{ DB Execer }

var _ storage.ChangeJournalStore = (*ChangeJournalStore)(nil)

func (s *ChangeJournalStore) Changes(ctx context.Context, scope string, after int64, limit int) ([]storage.Change, error) {
	if limit <= 0 {
		limit = 1000
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	rows, err := s.DB.Query(ctx, `SELECT seq, CAST(uuid AS TEXT), family, op, CAST(changed_at AS TEXT) FROM change_log
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

func (s *ChangeJournalStore) Mark(ctx context.Context, scope, family, uuid, op string) error {
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
		_, err = s.DB.Exec(ctx, `INSERT OR REPLACE INTO change_log (uuid, family, op) VALUES (?, ?, ?)`, uuid, family, op)
	} else {
		targs := s.DB.ScopeArg(scope)
		_, err = s.DB.Exec(ctx, `INSERT INTO change_log (tenant_id, uuid, family, op) VALUES (?, CAST(? AS uuid), ?, ?)
			ON CONFLICT (tenant_id, uuid) DO UPDATE SET seq = nextval('change_log_seq'), family = EXCLUDED.family,
			op = EXCLUDED.op, changed_at = now()`, targs, uuid, family, op)
	}
	if err != nil {
		return errf(s.DB, "mark change", err)
	}
	return nil
}
