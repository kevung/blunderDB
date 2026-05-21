package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type filterStore struct{ db execer }

var _ storage.FilterStore = (*filterStore)(nil)

// Save stores a new named filter and returns its id. A filter name must be
// unique: a clash reports ErrConflict.
func (s *filterStore) Save(ctx context.Context, scope string, name, command string) (int64, error) {
	var existing int64
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM filter_library WHERE name = ?`, name).Scan(&existing)
	if err == nil {
		return 0, fmt.Errorf("sqlite: save filter %q: %w", name, storage.ErrConflict)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("sqlite: save filter %q: %w", name, err)
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO filter_library (name, command) VALUES (?,?)`, name, command)
	if err != nil {
		return 0, fmt.Errorf("sqlite: save filter %q: %w", name, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("sqlite: save filter %q id: %w", name, err)
	}
	return id, nil
}

// Update changes a filter's name and command, or reports ErrNotFound.
func (s *filterStore) Update(ctx context.Context, scope string, id int64, name, command string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE filter_library SET name = ?, command = ? WHERE id = ?`, name, command, id)
	if err != nil {
		return fmt.Errorf("sqlite: update filter %d: %w", id, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("sqlite: update filter %d: %w", id, storage.ErrNotFound)
	}
	return nil
}

// Delete removes a filter, or reports ErrNotFound.
func (s *filterStore) Delete(ctx context.Context, scope string, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM filter_library WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("sqlite: delete filter %d: %w", id, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("sqlite: delete filter %d: %w", id, storage.ErrNotFound)
	}
	return nil
}

// List streams the saved filters, ordered by id.
func (s *filterStore) List(ctx context.Context, scope string) iter.Seq2[*storage.Filter, error] {
	return func(yield func(*storage.Filter, error) bool) {
		rows, err := s.db.QueryContext(ctx,
			`SELECT id, COALESCE(name,''), COALESCE(command,'') FROM filter_library ORDER BY id ASC`)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: list filters: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var f storage.Filter
			if err := rows.Scan(&f.ID, &f.Name, &f.Command); err != nil {
				yield(nil, fmt.Errorf("sqlite: list filters: %w", err))
				return
			}
			if !yield(&f, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("sqlite: list filters: %w", err))
		}
	}
}

// SaveEditPosition stores the in-progress edit position for a named filter,
// or reports ErrNotFound when no filter carries that name.
func (s *filterStore) SaveEditPosition(ctx context.Context, scope string, filterName, editPosition string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE filter_library SET edit_position = ? WHERE name = ?`, editPosition, filterName)
	if err != nil {
		return fmt.Errorf("sqlite: save edit position for %q: %w", filterName, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("sqlite: save edit position for %q: %w", filterName, storage.ErrNotFound)
	}
	return nil
}

// LoadEditPosition returns the stored edit position for a named filter, or ""
// when the filter is unknown or carries no edit position.
func (s *filterStore) LoadEditPosition(ctx context.Context, scope string, filterName string) (string, error) {
	var editPosition sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT edit_position FROM filter_library WHERE name = ?`, filterName).Scan(&editPosition)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("sqlite: load edit position for %q: %w", filterName, err)
	}
	return editPosition.String, nil
}
