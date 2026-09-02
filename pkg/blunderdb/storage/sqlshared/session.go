package sqlshared

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// SessionStore implements storage.SessionStore over the session_state table:
// one row per key and per scope, confined to the scope by the per-scope
// column both schemas carry (Dialect.ScopeColumn — `scope` on SQLite,
// `tenant_id` on PostgreSQL, where Row-Level Security covers it too).
//
// Until schema 2.15.0 the six keys below were rows of the global metadata
// table, prefixed "<scope>:" for a non-empty scope, and a tenant could read
// every other tenant's session through metadata.load (issue #156). Session
// state is per-tenant data; metadata is database infrastructure.
type SessionStore struct{ DB Execer }

var _ storage.SessionStore = (*SessionStore)(nil)

const (
	sessionKeySearchCommand  = "session_last_search_command"
	sessionKeySearchPosition = "session_last_search_position"
	sessionKeyPositionIndex  = "session_last_position_index"
	sessionKeyPositionIDs    = "session_last_position_ids"
	sessionKeyActiveSearch   = "session_has_active_search"
	sessionKeyViews          = "session_views"
)

// Save persists the UI session state across the six rows of the scope, in
// one transaction. The ON CONFLICT form is common to PostgreSQL and SQLite
// (≥ 3.24).
func (s *SessionStore) Save(ctx context.Context, scope string, state storage.SessionState) error {
	positionIDsJSON, err := json.Marshal(state.LastPositionIDs)
	if err != nil {
		return errf(s.DB, "save session", err)
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
	col, arg := s.DB.ScopeColumn(), s.DB.ScopeArg(scope)
	upsert := `INSERT INTO session_state (` + col + `, key, value) VALUES (?,?,?)
		ON CONFLICT (` + col + `, key) DO UPDATE SET value = EXCLUDED.value`
	err = s.DB.Transact(ctx, func(tx Execer) error {
		for _, kv := range pairs {
			if _, err := tx.Exec(ctx, upsert, arg, kv[0], kv[1]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return errf(s.DB, "save session", err)
	}
	return nil
}

// Load returns the persisted session state for the scope. Missing keys yield
// zero values, so a scope that never stored a session loads an empty
// SessionState.
func (s *SessionStore) Load(ctx context.Context, scope string) (*storage.SessionState, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT key, COALESCE(value,'') FROM session_state WHERE `+s.DB.ScopeColumn()+` = ?`,
		s.DB.ScopeArg(scope))
	if err != nil {
		return nil, errf(s.DB, "load session", err)
	}
	defer rows.Close()

	state := &storage.SessionState{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, errf(s.DB, "load session", err)
		}
		switch key {
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
		return nil, errf(s.DB, "load session", err)
	}
	return state, nil
}

// Clear removes the persisted session state for the scope.
func (s *SessionStore) Clear(ctx context.Context, scope string) error {
	if _, err := s.DB.Exec(ctx,
		`DELETE FROM session_state WHERE `+s.DB.ScopeColumn()+` = ?`, s.DB.ScopeArg(scope)); err != nil {
		return errf(s.DB, "clear session", err)
	}
	return nil
}
