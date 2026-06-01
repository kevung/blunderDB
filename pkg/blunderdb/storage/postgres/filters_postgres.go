package postgres

import (
	"context"
	"errors"
	"fmt"
	"iter"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type filterStore struct{ db execer }

var _ storage.FilterStore = (*filterStore)(nil)

// Save stores a new named filter and returns its id. Filter names are unique
// per tenant: a clash reports ErrConflict.
func (s *filterStore) Save(ctx context.Context, scope string, name, command string) (int64, error) {
	tenant := tenantID(scope)
	var existing int64
	err := s.db.QueryRow(ctx,
		`SELECT id FROM filter_library WHERE tenant_id = $1 AND name = $2`, tenant, name).Scan(&existing)
	if err == nil {
		return 0, fmt.Errorf("postgres: save filter %q: %w", name, storage.ErrConflict)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("postgres: save filter %q: %w", name, err)
	}
	var id int64
	if err := s.db.QueryRow(ctx,
		`INSERT INTO filter_library (tenant_id, name, command) VALUES ($1, $2, $3) RETURNING id`,
		tenant, name, command).Scan(&id); err != nil {
		return 0, fmt.Errorf("postgres: save filter %q: %w", name, err)
	}
	return id, nil
}

// Update changes a filter's name and command, or reports ErrNotFound.
func (s *filterStore) Update(ctx context.Context, scope string, id int64, name, command string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE filter_library SET name = $1, command = $2 WHERE id = $3 AND tenant_id = $4`,
		name, command, id, tenantID(scope))
	if err != nil {
		return fmt.Errorf("postgres: update filter %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("postgres: update filter %d: %w", id, storage.ErrNotFound)
	}
	return nil
}

// Delete removes a filter, or reports ErrNotFound.
func (s *filterStore) Delete(ctx context.Context, scope string, id int64) error {
	tag, err := s.db.Exec(ctx,
		`DELETE FROM filter_library WHERE id = $1 AND tenant_id = $2`, id, tenantID(scope))
	if err != nil {
		return fmt.Errorf("postgres: delete filter %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("postgres: delete filter %d: %w", id, storage.ErrNotFound)
	}
	return nil
}

// List streams the saved filters for the tenant, ordered by id.
func (s *filterStore) List(ctx context.Context, scope string) iter.Seq2[*storage.Filter, error] {
	return func(yield func(*storage.Filter, error) bool) {
		rows, err := s.db.Query(ctx,
			`SELECT id, COALESCE(name,''), COALESCE(command,'') FROM filter_library
			 WHERE tenant_id = $1 ORDER BY id ASC`, tenantID(scope))
		if err != nil {
			yield(nil, fmt.Errorf("postgres: list filters: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var f storage.Filter
			if err := rows.Scan(&f.ID, &f.Name, &f.Command); err != nil {
				yield(nil, fmt.Errorf("postgres: list filters: %w", err))
				return
			}
			if !yield(&f, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("postgres: list filters: %w", err))
		}
	}
}

// SaveEditPosition stores the in-progress edit position for a named filter, or
// reports ErrNotFound when no filter carries that name.
func (s *filterStore) SaveEditPosition(ctx context.Context, scope string, filterName, editPosition string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE filter_library SET edit_position = $1 WHERE name = $2 AND tenant_id = $3`,
		editPosition, filterName, tenantID(scope))
	if err != nil {
		return fmt.Errorf("postgres: save edit position for %q: %w", filterName, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("postgres: save edit position for %q: %w", filterName, storage.ErrNotFound)
	}
	return nil
}

// LoadEditPosition returns the stored edit position for a named filter, or ""
// when the filter is unknown or carries no edit position.
func (s *filterStore) LoadEditPosition(ctx context.Context, scope string, filterName string) (string, error) {
	var editPosition *string
	err := s.db.QueryRow(ctx,
		`SELECT edit_position FROM filter_library WHERE name = $1 AND tenant_id = $2`,
		filterName, tenantID(scope)).Scan(&editPosition)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("postgres: load edit position for %q: %w", filterName, err)
	}
	if editPosition == nil {
		return "", nil
	}
	return *editPosition, nil
}
