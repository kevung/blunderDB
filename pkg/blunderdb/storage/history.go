package storage

import "context"

// CommandHistoryStore persists the in-app command-line history.
type CommandHistoryStore interface {
	// Save appends a command to the history.
	Save(ctx context.Context, scope string, command string) error

	// Load returns the command history, oldest first.
	Load(ctx context.Context, scope string) ([]string, error)

	// Clear empties the command history.
	Clear(ctx context.Context, scope string) error
}
