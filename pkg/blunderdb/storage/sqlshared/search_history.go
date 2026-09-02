package sqlshared

import (
	"context"
	"iter"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// SearchHistoryStore implements storage.SearchHistoryStore over the
// search_history table, which carries a per-scope column in both schemas
// (Dialect.ScopeColumn).
type SearchHistoryStore struct{ DB Execer }

var _ storage.SearchHistoryStore = (*SearchHistoryStore)(nil)

// searchHistoryLimit caps the per-scope search history to its most recent
// entries.
const searchHistoryLimit = 100

// Save appends an executed search — its command, include position and
// optional "Sauf" exclusion structure — to the scope's history and trims it
// to the most recent searchHistoryLimit entries.
func (s *SearchHistoryStore) Save(ctx context.Context, scope string, command, position, excludePosition string) error {
	col, arg := s.DB.ScopeColumn(), s.DB.ScopeArg(scope)
	if _, err := s.DB.Exec(ctx,
		`INSERT INTO search_history (`+col+`, command, position, exclude_position, timestamp) VALUES (?,?,?,?,?)`,
		arg, command, position, excludePosition, time.Now().UnixMilli()); err != nil {
		return errf(s.DB, "save search history", err)
	}
	if _, err := s.DB.Exec(ctx,
		`DELETE FROM search_history WHERE `+col+` = ? AND id NOT IN (
			SELECT id FROM search_history WHERE `+col+` = ? ORDER BY timestamp DESC, id DESC LIMIT ?
		)`, arg, arg, searchHistoryLimit); err != nil {
		return errf(s.DB, "trim search history", err)
	}
	return nil
}

// List streams the scope's search history, most recent first. id DESC breaks
// ties when several entries share a millisecond timestamp, keeping the order
// deterministic.
func (s *SearchHistoryStore) List(ctx context.Context, scope string) iter.Seq2[*storage.SearchHistory, error] {
	return func(yield func(*storage.SearchHistory, error) bool) {
		rows, err := s.DB.Query(ctx,
			`SELECT id, COALESCE(command,''), COALESCE(position,''), COALESCE(exclude_position,''), COALESCE(timestamp,0)
			 FROM search_history WHERE `+s.DB.ScopeColumn()+` = ? ORDER BY timestamp DESC, id DESC LIMIT ?`,
			s.DB.ScopeArg(scope), searchHistoryLimit)
		if err != nil {
			yield(nil, errf(s.DB, "list search history", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var e storage.SearchHistory
			if err := rows.Scan(&e.ID, &e.Command, &e.Position, &e.ExcludePosition, &e.Timestamp); err != nil {
				yield(nil, errf(s.DB, "list search history", err))
				return
			}
			if !yield(&e, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, errf(s.DB, "list search history", err))
		}
	}
}

// DeleteEntry removes the scope's search history entry with the given
// timestamp.
func (s *SearchHistoryStore) DeleteEntry(ctx context.Context, scope string, timestamp int64) error {
	if _, err := s.DB.Exec(ctx,
		`DELETE FROM search_history WHERE timestamp = ? AND `+s.DB.ScopeColumn()+` = ?`,
		timestamp, s.DB.ScopeArg(scope)); err != nil {
		return errf(s.DB, "delete search history entry", err)
	}
	return nil
}
