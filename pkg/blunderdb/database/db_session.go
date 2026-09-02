package database

import (
	"context"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// This file is the GUI-facing adapter for four small families of per-database
// UI state — the command-line history, the search history, the saved-filter
// library and the last-session state. Every method takes the wrapper's lock,
// then delegates to the SQLite Storage backend under the single implicit
// tenant ("" scope); the SQL lives there (storage/sqlite/{history,search_history,
// filters,session}_sqlite.go) and is held to the shared contract suite.
//
// The public signatures are what Wails binds and the CLI calls; they stay as
// they were. Errors come back as the backend reports them, wrapped around
// storage.ErrConflict (a duplicate filter name) or storage.ErrNotFound (an
// unknown filter id or name), so callers can test with errors.Is.

// errNotOpened is returned by every method here when no database is open. The
// wrapper's other families say "no database is currently open"; this wording is
// the one these methods have always used.
var errNotOpened = errors.New("database is not opened")

// SaveCommand appends a command to the command-line history. The backend keeps
// the most recent 1000 entries.
func (d *Database) SaveCommand(command string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return errNotOpened
	}
	return d.store.History().Save(context.Background(), "", command)
}

// LoadCommandHistory returns the command-line history, oldest first.
func (d *Database) LoadCommandHistory() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return nil, errNotOpened
	}
	return d.store.History().Load(context.Background(), "")
}

// ClearCommandHistory empties the command-line history.
func (d *Database) ClearCommandHistory() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return errNotOpened
	}
	return d.store.History().Clear(context.Background(), "")
}

// SearchHistory represents a search history entry. It mirrors
// storage.SearchHistory field for field and is kept as a distinct type only
// because Wails generates the frontend model from this package.
type SearchHistory struct {
	ID              int    `json:"id"`
	Command         string `json:"command"`
	Position        string `json:"position"`
	ExcludePosition string `json:"excludePosition"`
	Timestamp       int64  `json:"timestamp"`
}

// SaveSearchHistory records a search command, its include position and its
// optional exclude ("Sauf") position. The backend keeps the most recent 100
// entries.
func (d *Database) SaveSearchHistory(command string, position string, excludePosition string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return errNotOpened
	}
	return d.store.SearchHistory().Save(context.Background(), "", command, position, excludePosition)
}

// LoadSearchHistory returns the search history, most recent first.
func (d *Database) LoadSearchHistory() ([]SearchHistory, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return nil, errNotOpened
	}
	var history []SearchHistory
	for e, err := range d.store.SearchHistory().List(context.Background(), "") {
		if err != nil {
			return nil, err
		}
		history = append(history, SearchHistory(*e))
	}
	return history, nil
}

// DeleteSearchHistoryEntry deletes the search history entries carrying a
// timestamp.
func (d *Database) DeleteSearchHistoryEntry(timestamp int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return errNotOpened
	}
	return d.store.SearchHistory().DeleteEntry(context.Background(), "", timestamp)
}

// SessionState represents the last session state for restoring when reopening
// a database. Same remark as SearchHistory: a Wails-facing twin of
// storage.SessionState.
type SessionState struct {
	LastSearchCommand  string  `json:"lastSearchCommand"`  // The last search command executed
	LastSearchPosition string  `json:"lastSearchPosition"` // The position used for the last search (JSON)
	LastPositionIndex  int     `json:"lastPositionIndex"`  // The index of the last viewed position in results
	LastPositionIDs    []int64 `json:"lastPositionIds"`    // The list of position IDs from the last search
	HasActiveSearch    bool    `json:"hasActiveSearch"`    // Whether there was an active search session
	ViewsJSON          string  `json:"viewsJSON"`          // Serialized view tabs state
}

// SaveSessionState persists the session state (six metadata rows, written in
// one transaction).
func (d *Database) SaveSessionState(state SessionState) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return errNotOpened
	}
	return d.store.Session().Save(context.Background(), "", storage.SessionState(state))
}

// LoadSessionState returns the persisted session state; a database that never
// stored one yields an empty (non-nil) state.
func (d *Database) LoadSessionState() (*SessionState, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return nil, errNotOpened
	}
	s, err := d.store.Session().Load(context.Background(), "")
	if err != nil {
		return nil, err
	}
	state := SessionState(*s)
	return &state, nil
}

// ClearSessionState removes the persisted session state.
func (d *Database) ClearSessionState() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return errNotOpened
	}
	return d.store.Session().Clear(context.Background(), "")
}

// SaveFilter adds a named filter to the library. A name already in use
// reports storage.ErrConflict.
func (d *Database) SaveFilter(name, command string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return errNotOpened
	}
	_, err := d.store.Filters().Save(context.Background(), "", name, command)
	return err
}

// UpdateFilter renames and rewrites a filter, or reports storage.ErrNotFound.
func (d *Database) UpdateFilter(id int64, name, command string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return errNotOpened
	}
	return d.store.Filters().Update(context.Background(), "", id, name, command)
}

// DeleteFilter removes a filter, or reports storage.ErrNotFound.
func (d *Database) DeleteFilter(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return errNotOpened
	}
	return d.store.Filters().Delete(context.Background(), "", id)
}

// LoadFilters returns the filter library, oldest first, as the id/name/command
// maps the frontend's filter picker consumes.
func (d *Database) LoadFilters() ([]map[string]interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return nil, errNotOpened
	}
	var filters []map[string]interface{}
	for f, err := range d.store.Filters().List(context.Background(), "") {
		if err != nil {
			return nil, err
		}
		filters = append(filters, map[string]interface{}{
			"id":      f.ID,
			"name":    f.Name,
			"command": f.Command,
		})
	}
	return filters, nil
}

// SaveEditPosition stores the checker structure a saved filter searches for,
// or reports storage.ErrNotFound for an unknown filter name.
func (d *Database) SaveEditPosition(filterName, editPosition string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return errNotOpened
	}
	return d.store.Filters().SaveEditPosition(context.Background(), "", filterName, editPosition)
}

// LoadEditPosition returns the checker structure of a saved filter, or "" when
// the filter is unknown or carries none.
func (d *Database) LoadEditPosition(filterName string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return "", errNotOpened
	}
	return d.store.Filters().LoadEditPosition(context.Background(), "", filterName)
}

// SaveExcludePosition stores the "Sauf" (exclusion) structure of a saved
// filter, or reports storage.ErrNotFound for an unknown filter name.
func (d *Database) SaveExcludePosition(filterName, excludePosition string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return errNotOpened
	}
	return d.store.Filters().SaveExcludePosition(context.Background(), "", filterName, excludePosition)
}

// LoadExcludePosition returns the "Sauf" (exclusion) structure of a saved
// filter, or "" when the filter is unknown or carries none.
func (d *Database) LoadExcludePosition(filterName string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return "", errNotOpened
	}
	return d.store.Filters().LoadExcludePosition(context.Background(), "", filterName)
}
