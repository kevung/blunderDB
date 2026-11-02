package sqlshared

import (
	"context"
	"encoding/json"
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
// against the same clock that wrote deleted_at. "Now minus N days" is spelled
// per dialect here, the one place it shows, rather than as a Dialect method.
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
	var n int64
	err := s.DB.Transact(ctx, func(tx Execer) error {
		held, err := trashedObjects(ctx, tx, where, targs)
		if err != nil {
			return err
		}
		if n, err = tx.Exec(ctx, `DELETE FROM trash WHERE `+where, targs...); err != nil {
			return err
		}
		// What the journal still calls trash is gone for good now.
		j := &ChangeJournalStore{DB: tx}
		for _, o := range held {
			if o.uuid == "" {
				continue
			}
			switch e, err := j.Entry(ctx, scope, o.uuid); {
			case errors.Is(err, storage.ErrNotFound):
			case err != nil:
				return err
			case e.Op == storage.ChangeTrash:
				if err := j.Mark(ctx, scope, o.family, o.uuid, storage.ChangeDelete, ""); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return 0, errf(s.DB, "purge the trash", err)
	}
	return int(n), nil
}

type trashedObject struct{ family, uuid string }

// trashedObjects names, by family and uuid, what the entries matching where
// held: the object itself and, for a position or a match, its notes and
// positions.
func trashedObjects(ctx context.Context, db Execer, where string, args []any) ([]trashedObject, error) {
	rows, err := db.Query(ctx, `SELECT kind, payload FROM trash WHERE `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type position struct {
		UUID         string           `json:"uuid"`
		CommentUUIDs map[int64]string `json:"commentUuids"`
	}
	var out []trashedObject
	addPosition := func(p position) {
		out = append(out, trashedObject{"position", p.UUID})
		for _, u := range p.CommentUUIDs {
			out = append(out, trashedObject{"comment", u})
		}
	}
	for rows.Next() {
		var kind, payload string
		if err := rows.Scan(&kind, &payload); err != nil {
			return nil, err
		}
		var p struct {
			position
			Positions []position `json:"positions"`
		}
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			continue
		}
		switch domain.TrashKind(kind) {
		case domain.TrashPosition:
			addPosition(p.position)
		case domain.TrashCollection:
			out = append(out, trashedObject{"collection", p.UUID})
		case domain.TrashComment:
			out = append(out, trashedObject{"comment", p.UUID})
		case domain.TrashMatch:
			out = append(out, trashedObject{"match", p.UUID})
			for _, pp := range p.Positions {
				addPosition(pp)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
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
