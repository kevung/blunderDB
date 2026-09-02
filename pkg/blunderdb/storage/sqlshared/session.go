package sqlshared

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// SessionStore implements storage.SessionStore. Session state is persisted as
// individual metadata key/value rows under the keys below.
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

var sessionKeys = []string{
	sessionKeySearchCommand, sessionKeySearchPosition, sessionKeyPositionIndex,
	sessionKeyPositionIDs, sessionKeyActiveSearch, sessionKeyViews,
}

// sessionScopedKey namespaces a session metadata key by scope so several
// tenants can co-exist in one database. The empty scope (single-user GUI/CLI)
// is left unprefixed, so existing databases keep working with no migration.
func sessionScopedKey(scope, key string) string {
	if scope == "" {
		return key
	}
	return scope + ":" + key
}

// SessionKeys returns the scope's six session metadata keys. The PostgreSQL
// tenant purge deletes exactly these rows (rather than a LIKE prefix pattern,
// which a scope containing % or _ could over-match) so a decommissioned
// tenant leaves no session crumbs behind; taking them from here keeps the
// purge in lockstep with Save/Load/Clear.
func SessionKeys(scope string) []string {
	keys := make([]string, len(sessionKeys))
	for i, k := range sessionKeys {
		keys[i] = sessionScopedKey(scope, k)
	}
	return keys
}

// scopedSessionKeys returns the scope's six metadata keys as query arguments.
func scopedSessionKeys(scope string) []any {
	args := make([]any, len(sessionKeys))
	for i, k := range SessionKeys(scope) {
		args[i] = k
	}
	return args
}

// Save persists the UI session state across the six (scoped) metadata rows.
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
	err = s.DB.Transact(ctx, func(tx Execer) error {
		for _, kv := range pairs {
			if _, err := tx.Exec(ctx, upsertMetadataSQL, sessionScopedKey(scope, kv[0]), kv[1]); err != nil {
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
	// Query the scope-namespaced keys and map each back to its base key.
	baseByScoped := make(map[string]string, len(sessionKeys))
	for _, k := range sessionKeys {
		baseByScoped[sessionScopedKey(scope, k)] = k
	}
	rows, err := s.DB.Query(ctx,
		`SELECT key, COALESCE(value,'') FROM metadata WHERE key IN (`+Placeholders(len(sessionKeys))+`)`,
		scopedSessionKeys(scope)...)
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
		return nil, errf(s.DB, "load session", err)
	}
	return state, nil
}

// Clear removes the persisted session state for the scope.
func (s *SessionStore) Clear(ctx context.Context, scope string) error {
	if _, err := s.DB.Exec(ctx,
		`DELETE FROM metadata WHERE key IN (`+Placeholders(len(sessionKeys))+`)`,
		scopedSessionKeys(scope)...); err != nil {
		return errf(s.DB, "clear session", err)
	}
	return nil
}
