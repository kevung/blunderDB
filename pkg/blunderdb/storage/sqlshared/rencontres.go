package sqlshared

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// RencontreStore implements storage.RencontreStore over the rencontre table
// and tournament.rencontre_id. Every statement is confined to the scope's
// tenant: a Tournament of another tenant can neither join nor be listed.
type RencontreStore struct{ DB Execer }

var _ storage.RencontreStore = (*RencontreStore)(nil)

func (s *RencontreStore) selectCols() string {
	return `id, name, COALESCE(starts_on,''), COALESCE(ends_on,''), tables, COALESCE(output_dir,''), ` +
		s.DB.TimestampText("created_at") + `, ` + s.DB.TimestampText("updated_at")
}

func scanRencontre(sc interface{ Scan(...any) error }) (*domain.Rencontre, error) {
	var r domain.Rencontre
	if err := sc.Scan(&r.ID, &r.Name, &r.StartsOn, &r.EndsOn, &r.Tables, &r.OutputDir, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	return &r, nil
}

// Create stores a new Rencontre and returns its id. Members are attached
// separately: r.TournamentIDs is ignored here.
func (s *RencontreStore) Create(ctx context.Context, scope string, r domain.Rencontre) (int64, error) {
	cols, args := s.DB.TenantColumns(scope)
	cols = append(cols, "name", "starts_on", "ends_on", "tables", "output_dir")
	args = append(args, r.Name, r.StartsOn, r.EndsOn, r.Tables, r.OutputDir)
	id, err := s.DB.Insert(ctx,
		`INSERT INTO rencontre (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)`, args...)
	if err != nil {
		return 0, errf(s.DB, "create rencontre", err)
	}
	return id, nil
}

// Get returns one Rencontre with its members, or ErrNotFound.
func (s *RencontreStore) Get(ctx context.Context, scope string, id int64) (*domain.Rencontre, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	r, err := scanRencontre(s.DB.QueryRow(ctx,
		`SELECT `+s.selectCols()+` FROM rencontre WHERE id = ? AND `+tenant,
		append([]any{id}, targs...)...))
	if errors.Is(err, ErrNoRows) {
		return nil, fmt.Errorf("%s: get rencontre %d: %w", s.DB.Name(), id, storage.ErrNotFound)
	}
	if err != nil {
		return nil, errf(s.DB, "get rencontre", err)
	}
	if err := s.fill(ctx, scope, r); err != nil {
		return nil, err
	}
	return r, nil
}

// List returns every Rencontre, newest first, each with its members.
func (s *RencontreStore) List(ctx context.Context, scope string) ([]*domain.Rencontre, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	rows, err := s.DB.Query(ctx,
		`SELECT `+s.selectCols()+` FROM rencontre WHERE `+tenant+` ORDER BY id DESC`, targs...)
	if err != nil {
		return nil, errf(s.DB, "list rencontres", err)
	}
	var out []*domain.Rencontre
	for rows.Next() {
		r, err := scanRencontre(rows)
		if err != nil {
			rows.Close()
			return nil, errf(s.DB, "list rencontres", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, errf(s.DB, "list rencontres", err)
	}
	rows.Close()
	// Members read after the cursor is closed: SQLite's in-memory test
	// database runs on a single connection.
	for _, r := range out {
		if err := s.fill(ctx, scope, r); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// fill reads what a Rencontre holds beyond its own row: members, the rooms
// of each, and its table settings.
func (s *RencontreStore) fill(ctx context.Context, scope string, r *domain.Rencontre) error {
	if err := s.members(ctx, scope, r); err != nil {
		return err
	}
	var err error
	r.TableSettings, err = s.tableSettings(ctx, scope, ownerRencontre, r.ID)
	return err
}

func (s *RencontreStore) members(ctx context.Context, scope string, r *domain.Rencontre) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	rows, err := s.DB.Query(ctx,
		`SELECT id, COALESCE(rencontre_rooms, '') FROM tournament WHERE rencontre_id = ? AND `+tenant+` ORDER BY id`,
		append([]any{r.ID}, targs...)...)
	if err != nil {
		return errf(s.DB, "list rencontre members", err)
	}
	defer rows.Close()
	r.TournamentIDs = []int64{}
	r.EventRooms = map[int64][]string{}
	for rows.Next() {
		var t int64
		var raw string
		if err := rows.Scan(&t, &raw); err != nil {
			return errf(s.DB, "list rencontre members", err)
		}
		r.TournamentIDs = append(r.TournamentIDs, t)
		rooms, err := decodeRooms(raw)
		if err != nil {
			return errf(s.DB, fmt.Sprintf("rooms of tournament %d", t), err)
		}
		if rooms != nil {
			r.EventRooms[t] = rooms
		}
	}
	return rows.Err()
}

// Update rewrites the room's own facts.
func (s *RencontreStore) Update(ctx context.Context, scope string, r domain.Rencontre) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	n, err := s.DB.Exec(ctx,
		`UPDATE rencontre SET name = ?, starts_on = ?, ends_on = ?, tables = ?, output_dir = ?,
		 updated_at = CURRENT_TIMESTAMP WHERE id = ? AND `+tenant,
		append([]any{r.Name, r.StartsOn, r.EndsOn, r.Tables, r.OutputDir, r.ID}, targs...)...)
	if err != nil {
		return errf(s.DB, "update rencontre", err)
	}
	if n == 0 {
		return fmt.Errorf("%s: update rencontre %d: %w", s.DB.Name(), r.ID, storage.ErrNotFound)
	}
	return nil
}

// Delete removes the Rencontre and its table settings. The foreign keys
// detach its Tournaments (ON DELETE SET NULL) and drop its settings (CASCADE);
// the explicit statements say so for a connection that runs without foreign
// keys enforced.
func (s *RencontreStore) Delete(ctx context.Context, scope string, id int64) error {
	return s.DB.Transact(ctx, func(tx Execer) error {
		tenant, targs := tx.TenantFilter("", scope)
		if _, err := tx.Exec(ctx, `UPDATE tournament SET rencontre_id = NULL, rencontre_rooms = NULL WHERE rencontre_id = ? AND `+tenant,
			append([]any{id}, targs...)...); err != nil {
			return errf(tx, "detach rencontre members", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM table_setting WHERE rencontre_id = ? AND `+tenant,
			append([]any{id}, targs...)...); err != nil {
			return errf(tx, "delete rencontre table settings", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM rencontre WHERE id = ? AND `+tenant,
			append([]any{id}, targs...)...); err != nil {
			return errf(tx, "delete rencontre", err)
		}
		return nil
	})
}

// Attach puts a Tournament in a Rencontre, or takes it out when rencontreID
// is 0. Both must exist in the scope. Its rooms are named in the Rencontre it
// leaves, so they are cleared whenever the membership changes; attaching to
// the Rencontre it already plays in keeps them.
func (s *RencontreStore) Attach(ctx context.Context, scope string, tournamentID, rencontreID int64) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	var target any
	if rencontreID != 0 {
		if _, err := s.Get(ctx, scope, rencontreID); err != nil {
			return err
		}
		target = rencontreID
	}
	n, err := s.DB.Exec(ctx,
		`UPDATE tournament SET rencontre_rooms = CASE WHEN rencontre_id = ? THEN rencontre_rooms END,
		 rencontre_id = ? WHERE id = ? AND `+tenant,
		append([]any{rencontreID, target, tournamentID}, targs...)...)
	if err != nil {
		return errf(s.DB, "attach tournament", err)
	}
	if n == 0 {
		return fmt.Errorf("%s: attach tournament %d: %w", s.DB.Name(), tournamentID, storage.ErrNotFound)
	}
	return nil
}

// Of returns the Rencontre a Tournament plays in, 0 when none.
func (s *RencontreStore) Of(ctx context.Context, scope string, tournamentID int64) (int64, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	var id int64
	err := s.DB.QueryRow(ctx, `SELECT COALESCE(rencontre_id, 0) FROM tournament WHERE id = ? AND `+tenant,
		append([]any{tournamentID}, targs...)...).Scan(&id)
	if errors.Is(err, ErrNoRows) {
		return 0, fmt.Errorf("%s: rencontre of tournament %d: %w", s.DB.Name(), tournamentID, storage.ErrNotFound)
	}
	if err != nil {
		return 0, errf(s.DB, "rencontre of tournament", err)
	}
	return id, nil
}
