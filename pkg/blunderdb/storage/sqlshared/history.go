package sqlshared

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// CommandHistoryStore implements storage.CommandHistoryStore over the
// command_history table, which carries a per-scope column in both schemas
// (Dialect.ScopeColumn).
type CommandHistoryStore struct{ DB Execer }

var _ storage.CommandHistoryStore = (*CommandHistoryStore)(nil)

// commandHistoryLimit caps the per-scope command history to bound unbounded
// growth.
const commandHistoryLimit = 1000

// Save appends a command to the scope's history and trims it to the most
// recent commandHistoryLimit entries.
func (s *CommandHistoryStore) Save(ctx context.Context, scope string, command string) error {
	col, arg := s.DB.ScopeColumn(), s.DB.ScopeArg(scope)
	if _, err := s.DB.Exec(ctx,
		`INSERT INTO command_history (`+col+`, command) VALUES (?,?)`, arg, command); err != nil {
		return errf(s.DB, "save command", err)
	}
	if _, err := s.DB.Exec(ctx,
		`DELETE FROM command_history WHERE `+col+` = ? AND id NOT IN (
			SELECT id FROM command_history WHERE `+col+` = ? ORDER BY timestamp DESC, id DESC LIMIT ?
		)`, arg, arg, commandHistoryLimit); err != nil {
		return errf(s.DB, "trim command history", err)
	}
	return nil
}

// Load returns the scope's command history, oldest first. id ASC breaks ties
// when several commands share a timestamp, keeping the order deterministic.
func (s *CommandHistoryStore) Load(ctx context.Context, scope string) ([]string, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT command FROM command_history WHERE `+s.DB.ScopeColumn()+` = ? ORDER BY timestamp ASC, id ASC`,
		s.DB.ScopeArg(scope))
	if err != nil {
		return nil, errf(s.DB, "load command history", err)
	}
	defer rows.Close()
	var history []string
	for rows.Next() {
		var command string
		if err := rows.Scan(&command); err != nil {
			return nil, errf(s.DB, "load command history", err)
		}
		history = append(history, command)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "load command history", err)
	}
	return history, nil
}

// Clear empties the scope's command history.
func (s *CommandHistoryStore) Clear(ctx context.Context, scope string) error {
	if _, err := s.DB.Exec(ctx,
		`DELETE FROM command_history WHERE `+s.DB.ScopeColumn()+` = ?`, s.DB.ScopeArg(scope)); err != nil {
		return errf(s.DB, "clear command history", err)
	}
	return nil
}
