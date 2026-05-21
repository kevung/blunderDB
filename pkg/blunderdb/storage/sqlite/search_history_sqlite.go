package sqlite

import (
	"context"
	"fmt"
	"iter"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type searchHistoryStore struct{ db execer }

var _ storage.SearchHistoryStore = (*searchHistoryStore)(nil)

// searchHistoryLimit caps the search history to its most recent entries.
const searchHistoryLimit = 100

// Save appends an executed search to the history and trims it to the most
// recent searchHistoryLimit entries.
func (s *searchHistoryStore) Save(ctx context.Context, scope string, command, position string) error {
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO search_history (command, position, timestamp) VALUES (?,?,?)`,
		command, position, time.Now().UnixMilli()); err != nil {
		return fmt.Errorf("sqlite: save search history: %w", err)
	}
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM search_history WHERE id NOT IN (
			SELECT id FROM search_history ORDER BY timestamp DESC LIMIT ?
		)`, searchHistoryLimit); err != nil {
		return fmt.Errorf("sqlite: trim search history: %w", err)
	}
	return nil
}

// List streams the search history, most recent first.
func (s *searchHistoryStore) List(ctx context.Context, scope string) iter.Seq2[*storage.SearchHistory, error] {
	return func(yield func(*storage.SearchHistory, error) bool) {
		// id DESC breaks ties when several entries share a millisecond
		// timestamp, keeping the most-recent-first order deterministic.
		rows, err := s.db.QueryContext(ctx,
			`SELECT id, COALESCE(command,''), COALESCE(position,''), COALESCE(timestamp,0)
			 FROM search_history ORDER BY timestamp DESC, id DESC LIMIT ?`, searchHistoryLimit)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: list search history: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var e storage.SearchHistory
			if err := rows.Scan(&e.ID, &e.Command, &e.Position, &e.Timestamp); err != nil {
				yield(nil, fmt.Errorf("sqlite: list search history: %w", err))
				return
			}
			if !yield(&e, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("sqlite: list search history: %w", err))
		}
	}
}

// DeleteEntry removes the search history entry with the given timestamp.
func (s *searchHistoryStore) DeleteEntry(ctx context.Context, scope string, timestamp int64) error {
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM search_history WHERE timestamp = ?`, timestamp); err != nil {
		return fmt.Errorf("sqlite: delete search history entry: %w", err)
	}
	return nil
}
