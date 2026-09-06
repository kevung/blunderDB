package sqlshared

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TrashStore implements storage.TrashStore. `trash` is a domain table: every
// statement is confined to the scope's tenant through Dialect.TenantFilter /
// TenantColumns.
//
// The SQL is the same on both backends — a snapshot table has no dialect —
// which is why it is written once here.
type TrashStore struct{ DB Execer }

var _ storage.TrashStore = (*TrashStore)(nil)

func (s *TrashStore) selectCols() string {
	return `id, kind, COALESCE(label,''), ` + s.DB.TimestampText("deleted_at") + `, payload`
}

func scanTrashEntry(sc interface{ Scan(...any) error }) (*domain.TrashEntry, error) {
	var e domain.TrashEntry
	var kind, payload string
	if err := sc.Scan(&e.ID, &kind, &e.Label, &e.DeletedAt, &payload); err != nil {
		return nil, err
	}
	e.Kind = domain.TrashKind(kind)
	e.Payload = []byte(payload)
	return &e, nil
}

// Put writes a snapshot and returns its id.
func (s *TrashStore) Put(ctx context.Context, scope string, kind domain.TrashKind, label string, payload []byte) (int64, error) {
	cols, args := s.DB.TenantColumns(scope)
	cols = append(cols, "kind", "label", "payload")
	args = append(args, string(kind), label, string(payload))
	id, err := s.DB.Insert(ctx,
		`INSERT INTO trash (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)`, args...)
	if err != nil {
		return 0, errf(s.DB, "put in the trash", err)
	}
	return id, nil
}

// List returns the entries, most recently deleted first.
func (s *TrashStore) List(ctx context.Context, scope string, kind domain.TrashKind, opts storage.ListOpts) ([]*domain.TrashEntry, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	where := tenant
	args := append([]any{}, targs...)
	if kind != "" {
		where += " AND kind = ?"
		args = append(args, string(kind))
	}
	limit, largs := s.DB.LimitOffset(opts.Limit, opts.Offset)
	rows, err := s.DB.Query(ctx,
		`SELECT `+s.selectCols()+` FROM trash WHERE `+where+` ORDER BY id DESC`+limit,
		append(args, largs...)...)
	if err != nil {
		return nil, errf(s.DB, "list the trash", err)
	}
	defer rows.Close()
	var out []*domain.TrashEntry
	for rows.Next() {
		e, err := scanTrashEntry(rows)
		if err != nil {
			return nil, errf(s.DB, "list the trash", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "list the trash", err)
	}
	return out, nil
}

// Load returns one entry, or ErrNotFound.
func (s *TrashStore) Load(ctx context.Context, scope string, id int64) (*domain.TrashEntry, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	row := s.DB.QueryRow(ctx,
		`SELECT `+s.selectCols()+` FROM trash WHERE id = ? AND `+tenant,
		append([]any{id}, targs...)...)
	e, err := scanTrashEntry(row)
	if errors.Is(err, ErrNoRows) {
		return nil, fmt.Errorf("%s: load trash entry %d: %w", s.DB.Name(), id, storage.ErrNotFound)
	}
	if err != nil {
		return nil, errf(s.DB, fmt.Sprintf("load trash entry %d", id), err)
	}
	return e, nil
}

// Discard removes one entry without restoring anything.
func (s *TrashStore) Discard(ctx context.Context, scope string, id int64) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	n, err := s.DB.Exec(ctx, `DELETE FROM trash WHERE id = ? AND `+tenant,
		append([]any{id}, targs...)...)
	if err != nil {
		return errf(s.DB, fmt.Sprintf("discard trash entry %d", id), err)
	}
	if n == 0 {
		return fmt.Errorf("%s: discard trash entry %d: %w", s.DB.Name(), id, storage.ErrNotFound)
	}
	return nil
}

// Purge drops the entries older than olderThanDays and returns how many.
//
// The cut-off is computed in SQL rather than in Go so the comparison is
// against the same clock that wrote deleted_at. The two dialects spell "now
// minus N days" differently and this is the ONE place the difference shows,
// which is why it is a small switch rather than a Dialect method nobody else
// would call.
func (s *TrashStore) Purge(ctx context.Context, scope string, olderThanDays int) (int, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	where := tenant
	if olderThanDays > 0 {
		days := strconv.Itoa(olderThanDays)
		if s.DB.Name() == "postgres" {
			where += ` AND deleted_at < now() - INTERVAL '` + days + ` days'`
		} else {
			where += ` AND deleted_at < datetime('now', '-` + days + ` days')`
		}
	}
	n, err := s.DB.Exec(ctx, `DELETE FROM trash WHERE `+where, targs...)
	if err != nil {
		return 0, errf(s.DB, "purge the trash", err)
	}
	return int(n), nil
}

// Count is how many entries the trash holds.
func (s *TrashStore) Count(ctx context.Context, scope string) (int, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	var n int
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM trash WHERE `+tenant, targs...).Scan(&n); err != nil {
		return 0, errf(s.DB, "count the trash", err)
	}
	return n, nil
}
