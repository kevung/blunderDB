package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type sessionStore struct{ db execer }

var _ storage.SessionStore = (*sessionStore)(nil)

// Session state is persisted as individual metadata key/value rows under these
// keys.
const (
	sessionKeySearchCommand  = "session_last_search_command"
	sessionKeySearchPosition = "session_last_search_position"
	sessionKeyPositionIndex  = "session_last_position_index"
	sessionKeyPositionIDs    = "session_last_position_ids"
	sessionKeyActiveSearch   = "session_has_active_search"
	sessionKeyViews          = "session_views"
)

var sessionKeys = []string{
	sessionKeySearchCommand, sessionKeySearchPosition, sessionKeyPositionIndex,
	sessionKeyPositionIDs, sessionKeyActiveSearch, sessionKeyViews,
}

// Save persists the UI session state across the six metadata rows.
func (s *sessionStore) Save(ctx context.Context, scope string, state storage.SessionState) error {
	positionIDsJSON, err := json.Marshal(state.LastPositionIDs)
	if err != nil {
		return fmt.Errorf("sqlite: save session: %w", err)
	}
	hasActiveSearch := "false"
	if state.HasActiveSearch {
		hasActiveSearch = "true"
	}
	pairs := [][2]string{
		{sessionKeySearchCommand, state.LastSearchCommand},
		{sessionKeySearchPosition, state.LastSearchPosition},
		{sessionKeyPositionIndex, strconv.Itoa(state.LastPositionIndex)},
		{sessionKeyPositionIDs, string(positionIDsJSON)},
		{sessionKeyActiveSearch, hasActiveSearch},
		{sessionKeyViews, state.ViewsJSON},
	}
	err = withTx(ctx, s.db, func(tx execer) error {
		for _, kv := range pairs {
			if _, err := tx.ExecContext(ctx,
				`INSERT OR REPLACE INTO metadata (key, value) VALUES (?,?)`,
				kv[0], kv[1]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sqlite: save session: %w", err)
	}
	return nil
}

// Load returns the persisted session state. Missing keys yield zero values, so
// a database that never stored a session loads an empty SessionState.
func (s *sessionStore) Load(ctx context.Context, scope string) (*storage.SessionState, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT key, value FROM metadata WHERE key IN (?,?,?,?,?,?)`,
		sessionKeySearchCommand, sessionKeySearchPosition, sessionKeyPositionIndex,
		sessionKeyPositionIDs, sessionKeyActiveSearch, sessionKeyViews)
	if err != nil {
		return nil, fmt.Errorf("sqlite: load session: %w", err)
	}
	defer rows.Close()

	state := &storage.SessionState{}
	for rows.Next() {
		var key string
		var value sql.NullString
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("sqlite: load session: %w", err)
		}
		if !value.Valid {
			continue
		}
		switch key {
		case sessionKeySearchCommand:
			state.LastSearchCommand = value.String
		case sessionKeySearchPosition:
			state.LastSearchPosition = value.String
		case sessionKeyPositionIndex:
			if n, err := strconv.Atoi(value.String); err == nil {
				state.LastPositionIndex = n
			}
		case sessionKeyPositionIDs:
			if value.String != "" {
				var ids []int64
				if err := json.Unmarshal([]byte(value.String), &ids); err == nil {
					state.LastPositionIDs = ids
				}
			}
		case sessionKeyActiveSearch:
			state.HasActiveSearch = value.String == "true"
		case sessionKeyViews:
			state.ViewsJSON = value.String
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: load session: %w", err)
	}
	return state, nil
}

// Clear removes the persisted session state.
func (s *sessionStore) Clear(ctx context.Context, scope string) error {
	err := withTx(ctx, s.db, func(tx execer) error {
		for _, key := range sessionKeys {
			if _, err := tx.ExecContext(ctx, `DELETE FROM metadata WHERE key = ?`, key); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sqlite: clear session: %w", err)
	}
	return nil
}
