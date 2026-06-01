package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type sessionStore struct{ db execer }

var _ storage.SessionStore = (*sessionStore)(nil)

// Session state is persisted as individual metadata key/value rows. The keys
// are namespaced by scope so several tenants can co-exist in one database; the
// empty scope (single-user) is left unprefixed.
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

func sessionScopedKey(scope, key string) string {
	if scope == "" {
		return key
	}
	return scope + ":" + key
}

// Save persists the UI session state across the six (scoped) metadata rows.
func (s *sessionStore) Save(ctx context.Context, scope string, state storage.SessionState) error {
	positionIDsJSON, err := json.Marshal(state.LastPositionIDs)
	if err != nil {
		return fmt.Errorf("postgres: save session: %w", err)
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
			if _, err := tx.Exec(ctx,
				`INSERT INTO metadata (key, value) VALUES ($1, $2)
				 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
				sessionScopedKey(scope, kv[0]), kv[1]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("postgres: save session: %w", err)
	}
	return nil
}

// Load returns the persisted session state for the scope. Missing keys yield
// zero values, so a scope that never stored a session loads empty.
func (s *sessionStore) Load(ctx context.Context, scope string) (*storage.SessionState, error) {
	baseByScoped := make(map[string]string, len(sessionKeys))
	scoped := make([]string, len(sessionKeys))
	for i, k := range sessionKeys {
		sk := sessionScopedKey(scope, k)
		baseByScoped[sk] = k
		scoped[i] = sk
	}

	rows, err := s.db.Query(ctx,
		`SELECT key, COALESCE(value, '') FROM metadata WHERE key = ANY($1)`, scoped)
	if err != nil {
		return nil, fmt.Errorf("postgres: load session: %w", err)
	}
	defer rows.Close()

	state := &storage.SessionState{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("postgres: load session: %w", err)
		}
		switch baseByScoped[key] {
		case sessionKeySearchCommand:
			state.LastSearchCommand = value
		case sessionKeySearchPosition:
			state.LastSearchPosition = value
		case sessionKeyPositionIndex:
			if n, err := strconv.Atoi(value); err == nil {
				state.LastPositionIndex = n
			}
		case sessionKeyPositionIDs:
			if value != "" {
				var ids []int64
				if err := json.Unmarshal([]byte(value), &ids); err == nil {
					state.LastPositionIDs = ids
				}
			}
		case sessionKeyActiveSearch:
			state.HasActiveSearch = value == "true"
		case sessionKeyViews:
			state.ViewsJSON = value
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: load session: %w", err)
	}
	return state, nil
}

// Clear removes the persisted session state for the scope.
func (s *sessionStore) Clear(ctx context.Context, scope string) error {
	scoped := make([]string, len(sessionKeys))
	for i, k := range sessionKeys {
		scoped[i] = sessionScopedKey(scope, k)
	}
	if _, err := s.db.Exec(ctx, `DELETE FROM metadata WHERE key = ANY($1)`, scoped); err != nil {
		return fmt.Errorf("postgres: clear session: %w", err)
	}
	return nil
}
