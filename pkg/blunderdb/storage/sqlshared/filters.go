package sqlshared

import (
	"context"
	"errors"
	"fmt"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// FilterStore implements storage.FilterStore over the filter_library table,
// which carries a per-scope column in both schemas (Dialect.ScopeColumn).
type FilterStore struct{ DB Execer }

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

// Delete removes a filter, or reports ErrNotFound.
func (s *FilterStore) Delete(ctx context.Context, scope string, id int64) error {
	n, err := s.DB.Exec(ctx,
		`DELETE FROM filter_library WHERE id = ? AND `+s.DB.ScopeColumn()+` = ?`, id, s.DB.ScopeArg(scope))
	if err != nil {
		return errf(s.DB, fmt.Sprintf("delete filter %d", id), err)
	}
	if n == 0 {
		return errf(s.DB, fmt.Sprintf("delete filter %d", id), storage.ErrNotFound)
	}
	return nil
}

// List streams the scope's saved filters, ordered by id.
func (s *FilterStore) List(ctx context.Context, scope string) iter.Seq2[*storage.Filter, error] {
	return func(yield func(*storage.Filter, error) bool) {
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
