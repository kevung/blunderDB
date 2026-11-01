package sqlshared

import (
	"context"
	"errors"
	"fmt"
	"strconv"

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
	return s.ChangesOf(ctx, scope, nil, storage.Cursor{Seq: after}, limit)
}

func (s *ChangeJournalStore) ChangesOf(ctx context.Context, scope string, families []string, after storage.Cursor, limit int) ([]storage.Change, error) {
	if limit <= 0 {
		limit = 1000
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	var args []any
	args = append(args, targs...)
	where, order, xid := tenant+` AND seq > ?`, "seq", "0"
	args = append(args, after.Seq)
	if s.DB.Name() == "postgres" {
		// Only entries whose writer, and every older transaction, has ended,
		// read in the order of those transactions: a change cannot still be
		// committed behind the cursor.
		// A cursor without Xid is a bare seq (Changes, an export from 0): the
		// row comparison would pass every row, since any xid is above 0.
		if after.Xid > 0 {
			where = tenant + ` AND (xid, seq) > (CAST(CAST(? AS TEXT) AS xid8), ?)`
			args = append(args[:len(targs)], strconv.FormatInt(after.Xid, 10), after.Seq)
		}
		where += ` AND xid < pg_snapshot_xmin(pg_current_snapshot())`
		order, xid = "xid, seq", "CAST(xid AS TEXT) AS xid_text"
	}
	if len(families) > 0 {
		where += ` AND family IN (` + Placeholders(len(families)) + `)`
		for _, f := range families {
			args = append(args, f)
		}
	}
	rows, err := s.DB.Query(ctx, `SELECT seq, CAST(uuid AS TEXT), family, op, `+s.changedAtText()+`, `+xid+` FROM change_log
		WHERE `+where+` ORDER BY `+order+` LIMIT ?`, append(args, limit)...)
	if err != nil {
		return nil, errf(s.DB, "changes", err)
	}
	defer rows.Close()
	var out []storage.Change
	for rows.Next() {
		var c storage.Change
		var x string
		if err := rows.Scan(&c.Seq, &c.UUID, &c.Family, &c.Op, &c.ChangedAt, &x); err != nil {
			return nil, errf(s.DB, "changes", err)
		}
		c.Xid, _ = strconv.ParseInt(x, 10, 64)
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "changes", err)
	}
	return out, nil
}

func (s *ChangeJournalStore) Unjournaled(ctx context.Context, scope, family string, afterID int64, limit int) ([]string, int64, error) {
	table, err := objectUUIDTable(family)
	if err != nil {
		return nil, 0, errf(s.DB, "unjournaled", err)
	}
	tenant, targs := s.DB.TenantFilter("t", scope)
	same := ""
	if s.DB.Name() == "postgres" {
		same = ` AND c.tenant_id = t.tenant_id`
	}
	rows, err := s.DB.Query(ctx, `SELECT t.id, CAST(t.uuid AS TEXT) FROM `+table+` t WHERE `+tenant+` AND t.uuid IS NOT NULL AND t.id > ?
		AND NOT EXISTS (SELECT 1 FROM change_log c WHERE c.uuid = t.uuid`+same+`) ORDER BY t.id LIMIT ?`, append(append(targs, afterID), limit)...)
	if err != nil {
		return nil, 0, errf(s.DB, "unjournaled", err)
	}
	defer rows.Close()
	var out []string
	last := afterID
	for rows.Next() {
		var u string
		if err := rows.Scan(&last, &u); err != nil {
			return nil, 0, errf(s.DB, "unjournaled", err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, errf(s.DB, "unjournaled", err)
	}
	return out, last, nil
}

func (s *ChangeJournalStore) Entry(ctx context.Context, scope, uuid string) (storage.Change, error) {
	if !isUUID(uuid) {
		return storage.Change{}, errf(s.DB, "change entry", storage.ErrNotFound)
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	var c storage.Change
	err := s.DB.QueryRow(ctx, `SELECT seq, CAST(uuid AS TEXT), family, op, `+s.changedAtText()+` FROM change_log
		WHERE `+tenant+` AND uuid = ?`, append(targs, uuid)...).Scan(&c.Seq, &c.UUID, &c.Family, &c.Op, &c.ChangedAt)
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
			op = EXCLUDED.op, changed_at = EXCLUDED.changed_at, xid = pg_current_xact_id()`, s.DB.ScopeArg(scope), uuid, family, op, at)
	}
	if err != nil {
		return errf(s.DB, "mark change", err)
	}
	return nil
}

// Adopt — see storage.ChangeJournalStore.
func (s *ChangeJournalStore) Adopt(ctx context.Context, scope, family string, uuids []string, at string) error {
	if _, err := objectUUIDTable(family); err != nil {
		return errf(s.DB, "adopt", err)
	}
	for _, u := range uuids {
		var err error
		if s.DB.Name() == "sqlite" {
			_, err = s.DB.Exec(ctx, `INSERT OR IGNORE INTO change_log (uuid, family, op, changed_at) VALUES (?, ?, 'create', ?)`, u, family, at)
		} else {
			_, err = s.DB.Exec(ctx, `INSERT INTO change_log (tenant_id, uuid, family, op, changed_at)
				VALUES (?, CAST(? AS uuid), ?, 'create', CAST(? AS timestamptz)) ON CONFLICT (tenant_id, uuid) DO NOTHING`,
				s.DB.ScopeArg(scope), u, family, at)
		}
		if err != nil {
			return errf(s.DB, "adopt", err)
		}
	}
	return nil
}
