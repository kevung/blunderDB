package sqlshared

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"slices"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// FilterStore implements storage.FilterStore over the filter_library table,
// which carries a per-scope column in both schemas (Dialect.ScopeColumn).
//
// The pins (storage.Filter.Pinned) are not a column of filter_library: they
// are one session_state row of the scope, key filterPinsKey, holding the
// pinned ids as a JSON array. A pin is a reading habit of the scope, like the
// open views next to it, and keeping it there spared a schema change on three
// backends for one boolean. Deleting a filter drops its id from the row in the
// same transaction; Session().Clear leaves the row alone.
type FilterStore struct{ DB Execer }

// filterPinsKey is the session_state key of the pinned filter ids.
const filterPinsKey = "filter_pins"

var _ storage.FilterStore = (*FilterStore)(nil)

// Save stores a new named filter and returns its id. A filter name is unique
// within its scope: a clash reports ErrConflict.
func (s *FilterStore) Save(ctx context.Context, scope string, name, command string) (int64, error) {
	col, arg := s.DB.ScopeColumn(), s.DB.ScopeArg(scope)
	what := fmt.Sprintf("save filter %q", name)
	var existing int64
	err := s.DB.QueryRow(ctx,
		`SELECT id FROM filter_library WHERE name = ? AND `+col+` = ?`, name, arg).Scan(&existing)
	if err == nil {
		return 0, errf(s.DB, what, storage.ErrConflict)
	}
	if !errors.Is(err, ErrNoRows) {
		return 0, errf(s.DB, what, err)
	}
	id, err := s.DB.Insert(ctx,
		`INSERT INTO filter_library (`+col+`, name, command) VALUES (?,?,?)`, arg, name, command)
	if err != nil {
		return 0, errf(s.DB, what, err)
	}
	return id, nil
}

// Update changes a filter's name and command, or reports ErrNotFound.
func (s *FilterStore) Update(ctx context.Context, scope string, id int64, name, command string) error {
	n, err := s.DB.Exec(ctx,
		`UPDATE filter_library SET name = ?, command = ? WHERE id = ? AND `+s.DB.ScopeColumn()+` = ?`,
		name, command, id, s.DB.ScopeArg(scope))
	if err != nil {
		return errf(s.DB, fmt.Sprintf("update filter %d", id), err)
	}
	if n == 0 {
		return errf(s.DB, fmt.Sprintf("update filter %d", id), storage.ErrNotFound)
	}
	return nil
}

// Delete removes a filter, or reports ErrNotFound. Its pin goes with it.
func (s *FilterStore) Delete(ctx context.Context, scope string, id int64) error {
	what := fmt.Sprintf("delete filter %d", id)
	err := s.DB.Transact(ctx, func(tx Execer) error {
		n, err := tx.Exec(ctx,
			`DELETE FROM filter_library WHERE id = ? AND `+tx.ScopeColumn()+` = ?`, id, tx.ScopeArg(scope))
		if err != nil {
			return err
		}
		if n == 0 {
			return storage.ErrNotFound
		}
		return setPin(ctx, tx, scope, id, false)
	})
	if err != nil {
		return errf(s.DB, what, err)
	}
	return nil
}

// SetPinned pins or unpins a filter of the scope, or reports ErrNotFound.
func (s *FilterStore) SetPinned(ctx context.Context, scope string, id int64, pinned bool) error {
	what := fmt.Sprintf("pin filter %d", id)
	err := s.DB.Transact(ctx, func(tx Execer) error {
		var existing int64
		err := tx.QueryRow(ctx,
			`SELECT id FROM filter_library WHERE id = ? AND `+tx.ScopeColumn()+` = ?`, id, tx.ScopeArg(scope)).Scan(&existing)
		if errors.Is(err, ErrNoRows) {
			return storage.ErrNotFound
		}
		if err != nil {
			return err
		}
		return setPin(ctx, tx, scope, id, pinned)
	})
	if err != nil {
		return errf(s.DB, what, err)
	}
	return nil
}

// loadPins reads the scope's pinned ids. A missing row, or one that no longer
// parses (edited by hand), reads as no pin at all: a pin is a convenience, and
// losing one must never stop the library from listing.
func loadPins(ctx context.Context, db Execer, scope string) ([]int64, error) {
	var raw *string
	err := db.QueryRow(ctx,
		`SELECT value FROM session_state WHERE `+db.ScopeColumn()+` = ? AND key = ?`,
		db.ScopeArg(scope), filterPinsKey).Scan(&raw)
	if errors.Is(err, ErrNoRows) || (err == nil && raw == nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []int64
	if json.Unmarshal([]byte(*raw), &ids) != nil {
		return nil, nil
	}
	return ids, nil
}

// setPin adds or removes one id from the scope's pins, writing the row only
// when it changes. The ids are kept sorted, the order of the library.
func setPin(ctx context.Context, db Execer, scope string, id int64, pinned bool) error {
	ids, err := loadPins(ctx, db, scope)
	if err != nil {
		return err
	}
	has := slices.Contains(ids, id)
	switch {
	case pinned && !has:
		ids = append(ids, id)
		slices.Sort(ids)
	case !pinned && has:
		ids = slices.DeleteFunc(ids, func(v int64) bool { return v == id })
	default:
		return nil
	}
	col, arg := db.ScopeColumn(), db.ScopeArg(scope)
	if len(ids) == 0 {
		_, err := db.Exec(ctx, `DELETE FROM session_state WHERE `+col+` = ? AND key = ?`, arg, filterPinsKey)
		return err
	}
	value, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `INSERT INTO session_state (`+col+`, key, value) VALUES (?,?,?)
		ON CONFLICT (`+col+`, key) DO UPDATE SET value = EXCLUDED.value`, arg, filterPinsKey, string(value))
	return err
}

// List streams the scope's saved filters, ordered by id, each marked with its
// pin.
func (s *FilterStore) List(ctx context.Context, scope string) iter.Seq2[*storage.Filter, error] {
	return func(yield func(*storage.Filter, error) bool) {
		// Read before the rows are open: an in-memory SQLite database has a
		// single connection, which the open cursor would hold.
		pins, err := loadPins(ctx, s.DB, scope)
		if err != nil {
			yield(nil, errf(s.DB, "list filters", err))
			return
		}
		rows, err := s.DB.Query(ctx,
			`SELECT id, COALESCE(name,''), COALESCE(command,'') FROM filter_library
			 WHERE `+s.DB.ScopeColumn()+` = ? ORDER BY id ASC`, s.DB.ScopeArg(scope))
		if err != nil {
			yield(nil, errf(s.DB, "list filters", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var f storage.Filter
			if err := rows.Scan(&f.ID, &f.Name, &f.Command); err != nil {
				yield(nil, errf(s.DB, "list filters", err))
				return
			}
			f.Pinned = slices.Contains(pins, f.ID)
			if !yield(&f, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, errf(s.DB, "list filters", err))
		}
	}
}

// SaveEditPosition stores the in-progress edit position for a named filter,
// or reports ErrNotFound when no filter carries that name.
func (s *FilterStore) SaveEditPosition(ctx context.Context, scope string, filterName, editPosition string) error {
	n, err := s.DB.Exec(ctx,
		`UPDATE filter_library SET edit_position = ? WHERE name = ? AND `+s.DB.ScopeColumn()+` = ?`,
		editPosition, filterName, s.DB.ScopeArg(scope))
	if err != nil {
		return errf(s.DB, fmt.Sprintf("save edit position for %q", filterName), err)
	}
	if n == 0 {
		return errf(s.DB, fmt.Sprintf("save edit position for %q", filterName), storage.ErrNotFound)
	}
	return nil
}

// LoadEditPosition returns the stored edit position for a named filter, or ""
// when the filter is unknown or carries no edit position.
func (s *FilterStore) LoadEditPosition(ctx context.Context, scope string, filterName string) (string, error) {
	var editPosition *string
	err := s.DB.QueryRow(ctx,
		`SELECT edit_position FROM filter_library WHERE name = ? AND `+s.DB.ScopeColumn()+` = ?`,
		filterName, s.DB.ScopeArg(scope)).Scan(&editPosition)
	if errors.Is(err, ErrNoRows) || (err == nil && editPosition == nil) {
		return "", nil
	}
	if err != nil {
		return "", errf(s.DB, fmt.Sprintf("load edit position for %q", filterName), err)
	}
	return *editPosition, nil
}

// SaveExcludePosition stores the "Sauf" exclusion structure of a named filter,
// or reports ErrNotFound when no filter carries that name.
func (s *FilterStore) SaveExcludePosition(ctx context.Context, scope string, filterName, excludePosition string) error {
	n, err := s.DB.Exec(ctx,
		`UPDATE filter_library SET exclude_position = ? WHERE name = ? AND `+s.DB.ScopeColumn()+` = ?`,
		excludePosition, filterName, s.DB.ScopeArg(scope))
	if err != nil {
		return errf(s.DB, fmt.Sprintf("save exclude position for %q", filterName), err)
	}
	if n == 0 {
		return errf(s.DB, fmt.Sprintf("save exclude position for %q", filterName), storage.ErrNotFound)
	}
	return nil
}

// LoadExcludePosition returns the stored exclusion structure of a named
// filter, or "" when the filter is unknown or carries none.
func (s *FilterStore) LoadExcludePosition(ctx context.Context, scope string, filterName string) (string, error) {
	var excludePosition *string
	err := s.DB.QueryRow(ctx,
		`SELECT exclude_position FROM filter_library WHERE name = ? AND `+s.DB.ScopeColumn()+` = ?`,
		filterName, s.DB.ScopeArg(scope)).Scan(&excludePosition)
	if errors.Is(err, ErrNoRows) || (err == nil && excludePosition == nil) {
		return "", nil
	}
	if err != nil {
		return "", errf(s.DB, fmt.Sprintf("load exclude position for %q", filterName), err)
	}
	return *excludePosition, nil
}
