package postgres

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type commandHistoryStore struct{ db execer }

var _ storage.CommandHistoryStore = (*commandHistoryStore)(nil)

// commandHistoryLimit caps the per-tenant command history.
const commandHistoryLimit = 1000

// Save appends a command to the tenant's history and trims it to the most
// recent commandHistoryLimit entries.
func (s *commandHistoryStore) Save(ctx context.Context, scope string, command string) error {
	tenant := tenantID(scope)
	if _, err := s.db.Exec(ctx,
		`INSERT INTO command_history (tenant_id, command) VALUES ($1, $2)`, tenant, command); err != nil {
		return fmt.Errorf("postgres: save command: %w", err)
	}
	if _, err := s.db.Exec(ctx,
		`DELETE FROM command_history WHERE tenant_id = $1 AND id NOT IN (
			SELECT id FROM command_history WHERE tenant_id = $1 ORDER BY timestamp DESC, id DESC LIMIT $2
		)`, tenant, commandHistoryLimit); err != nil {
		return fmt.Errorf("postgres: trim command history: %w", err)
	}
	return nil
}

// Load returns the tenant's command history, oldest first.
func (s *commandHistoryStore) Load(ctx context.Context, scope string) ([]string, error) {
	rows, err := s.db.Query(ctx,
		`SELECT command FROM command_history WHERE tenant_id = $1 ORDER BY timestamp ASC, id ASC`,
		tenantID(scope))
	if err != nil {
		return nil, fmt.Errorf("postgres: load command history: %w", err)
	}
	defer rows.Close()
	var history []string
	for rows.Next() {
		var command string
		if err := rows.Scan(&command); err != nil {
			return nil, fmt.Errorf("postgres: load command history: %w", err)
		}
		history = append(history, command)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: load command history: %w", err)
	}
	return history, nil
}

// Clear empties the tenant's command history.
func (s *commandHistoryStore) Clear(ctx context.Context, scope string) error {
	if _, err := s.db.Exec(ctx, `DELETE FROM command_history WHERE tenant_id = $1`, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: clear command history: %w", err)
	}
	return nil
}
