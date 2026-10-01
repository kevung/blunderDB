package sqlshared

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The properties of a table (ADR-0058) live in table_setting under one of two
// owners: the Rencontre whose events share the tables, or a Tournament run on
// its own. Both owners go through the same three functions below, named by
// their owner column, so the two cases cannot drift apart.

const (
	ownerRencontre  = "rencontre_id"
	ownerTournament = "tournament_id"
)

// SetTableSettings replaces the Rencontre's table properties.
func (s *RencontreStore) SetTableSettings(ctx context.Context, scope string, rencontreID int64, settings []domain.TableSetting) error {
	return s.setTableSettings(ctx, scope, "rencontre", ownerRencontre, rencontreID, settings)
}

// TournamentTableSettings returns the table properties a Tournament owns.
func (s *RencontreStore) TournamentTableSettings(ctx context.Context, scope string, tournamentID int64) ([]domain.TableSetting, error) {
	if err := s.ownerExists(ctx, s.DB, scope, "tournament", tournamentID); err != nil {
		return nil, err
	}
	return s.tableSettings(ctx, scope, ownerTournament, tournamentID)
}

// SetTournamentTableSettings replaces the table properties a Tournament owns.
func (s *RencontreStore) SetTournamentTableSettings(ctx context.Context, scope string, tournamentID int64, settings []domain.TableSetting) error {
	return s.setTableSettings(ctx, scope, "tournament", ownerTournament, tournamentID, settings)
}

// SetEventRooms writes the rooms a Tournament may play in; an empty list is
// stored as NULL, every table.
func (s *RencontreStore) SetEventRooms(ctx context.Context, scope string, tournamentID int64, rooms []string) error {
	var value any
	if len(rooms) > 0 {
		blob, err := json.Marshal(rooms)
		if err != nil {
			return err
		}
		value = string(blob)
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	n, err := s.DB.Exec(ctx, `UPDATE tournament SET rencontre_rooms = ? WHERE id = ? AND `+tenant,
		append([]any{value, tournamentID}, targs...)...)
	if err != nil {
		return errf(s.DB, "set event rooms", err)
	}
	if n == 0 {
		return fmt.Errorf("%s: set event rooms of tournament %d: %w", s.DB.Name(), tournamentID, storage.ErrNotFound)
	}
	return nil
}

// ValidateTableSettings refuses what the schema would: a number that is not
// positive, or the same number twice. Checked before writing so a bad list is
// an ErrInvalid naming the table, not a constraint violation halfway through.
func ValidateTableSettings(settings []domain.TableSetting) error {
	seen := make(map[int]bool, len(settings))
	for _, t := range settings {
		if t.Number <= 0 {
			return fmt.Errorf("table number %d: %w", t.Number, storage.ErrInvalid)
		}
		if seen[t.Number] {
			return fmt.Errorf("table %d given twice: %w", t.Number, storage.ErrInvalid)
		}
		seen[t.Number] = true
	}
	return nil
}

// ownerExists checks the owner row in the scope: a missing owner is
// ErrNotFound, never an empty list.
func (s *RencontreStore) ownerExists(ctx context.Context, db Execer, scope, table string, id int64) error {
	tenant, targs := db.TenantFilter("", scope)
	var one int
	err := db.QueryRow(ctx, `SELECT 1 FROM `+table+` WHERE id = ? AND `+tenant,
		append([]any{id}, targs...)...).Scan(&one)
	if errors.Is(err, ErrNoRows) {
		return fmt.Errorf("%s: %s %d: %w", db.Name(), table, id, storage.ErrNotFound)
	}
	if err != nil {
		return errf(db, "probe "+table, err)
	}
	return nil
}

func (s *RencontreStore) setTableSettings(ctx context.Context, scope, table, owner string, ownerID int64, settings []domain.TableSetting) error {
	if err := ValidateTableSettings(settings); err != nil {
		return fmt.Errorf("%s: set table settings of %s %d: %w", s.DB.Name(), table, ownerID, err)
	}
	return s.DB.Transact(ctx, func(tx Execer) error {
		if err := s.ownerExists(ctx, tx, scope, table, ownerID); err != nil {
			return err
		}
		tenant, targs := tx.TenantFilter("", scope)
		if _, err := tx.Exec(ctx, `DELETE FROM table_setting WHERE `+owner+` = ? AND `+tenant,
			append([]any{ownerID}, targs...)...); err != nil {
			return errf(tx, "clear table settings", err)
		}
		for _, t := range settings {
			assigned := t.AssignedTo
			if assigned == nil {
				assigned = []string{}
			}
			blob, err := json.Marshal(assigned)
			if err != nil {
				return err
			}
			cols, args := tx.TenantColumns(scope)
			cols = append(cols, owner, "number", "name", "room", "reserved", "assigned_to")
			args = append(args, ownerID, t.Number, t.Name, t.Room, tx.BoolArg(t.Reserved), string(blob))
			if _, err := tx.Exec(ctx,
				`INSERT INTO table_setting (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)`,
				args...); err != nil {
				return errf(tx, fmt.Sprintf("write table %d", t.Number), err)
			}
		}
		return nil
	})
}

// tableSettings reads the rows of one owner, by number. The list is never
// nil, nor is any AssignedTo.
func (s *RencontreStore) tableSettings(ctx context.Context, scope, owner string, ownerID int64) ([]domain.TableSetting, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	rows, err := s.DB.Query(ctx,
		`SELECT number, name, room, `+s.DB.BoolAsInt("reserved")+`, assigned_to FROM table_setting
		 WHERE `+owner+` = ? AND `+tenant+` ORDER BY number`,
		append([]any{ownerID}, targs...)...)
	if err != nil {
		return nil, errf(s.DB, "list table settings", err)
	}
	defer rows.Close()
	out := []domain.TableSetting{}
	for rows.Next() {
		var t domain.TableSetting
		var reserved int
		var assigned string
		if err := rows.Scan(&t.Number, &t.Name, &t.Room, &reserved, &assigned); err != nil {
			return nil, errf(s.DB, "list table settings", err)
		}
		t.Reserved = reserved != 0
		if err := json.Unmarshal([]byte(assigned), &t.AssignedTo); err != nil {
			return nil, errf(s.DB, fmt.Sprintf("table %d assigned_to", t.Number), err)
		}
		if t.AssignedTo == nil {
			t.AssignedTo = []string{}
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// decodeRooms reads tournament.rencontre_rooms, selected through COALESCE so
// NULL arrives as "": that or an empty array is no restriction, nil.
func decodeRooms(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	var rooms []string
	if err := json.Unmarshal([]byte(raw), &rooms); err != nil {
		return nil, err
	}
	if len(rooms) == 0 {
		return nil, nil
	}
	return rooms, nil
}
