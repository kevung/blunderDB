package sqlite

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type commandHistoryStore struct{ db execer }

var _ storage.CommandHistoryStore = (*commandHistoryStore)(nil)

// commandHistoryLimit caps the command history to bound unbounded growth.
const commandHistoryLimit = 1000

// Save appends a command to the history and trims it to the most recent
// commandHistoryLimit entries.
func (s *commandHistoryStore) Save(ctx context.Context, scope string, command string) error {
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO command_history (command) VALUES (?)`, command); err != nil {
		return fmt.Errorf("sqlite: save command: %w", err)
	}
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM command_history WHERE id NOT IN (
			SELECT id FROM command_history ORDER BY timestamp DESC LIMIT ?
		)`, commandHistoryLimit); err != nil {
		return fmt.Errorf("sqlite: trim command history: %w", err)
	}
	return nil
}

// Load returns the command history, oldest first.
func (s *commandHistoryStore) Load(ctx context.Context, scope string) ([]string, error) {
	// id ASC breaks ties when several commands share a timestamp, keeping the
	// oldest-first order deterministic.
	rows, err := s.db.QueryContext(ctx,
		`SELECT command FROM command_history ORDER BY timestamp ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: load command history: %w", err)
	}
	defer rows.Close()
	var history []string
	for rows.Next() {
		var command string
		if err := rows.Scan(&command); err != nil {
			return nil, fmt.Errorf("sqlite: load command history: %w", err)
		}
		history = append(history, command)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: load command history: %w", err)
	}
	return history, nil
}

// Clear empties the command history.
func (s *commandHistoryStore) Clear(ctx context.Context, scope string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM command_history`); err != nil {
		return fmt.Errorf("sqlite: clear command history: %w", err)
	}
	return nil
}
