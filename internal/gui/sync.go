package gui

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/zalando/go-keyring"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/syncremote"
)

// syncKeyringService names the sync tokens in the system keyring, one entry
// per server address. A token never reaches the database, the settings file,
// the sync state or a log.
const syncKeyringService = "blunderDB sync"

// syncStatusEvent carries a SyncStatus to the window.
const syncStatusEvent = "sync:status"

// SyncSettingsSource is what the settings expose to the synchronisation: the
// server address and whether to sync at startup.
type SyncSettingsSource interface {
	SyncServerAddress() (server string, auto bool)
}

// SyncError is why the last run failed. Kind lets the window word it:
// refused, unavailable, rate_limited, too_large, address, no_token,
// no_library or other. Message is the detail, free of the token.
type SyncError struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// SyncStatus is what the window shows: whether a run is under way, the last
// success for this library and server (RFC 3339, empty if none), and how the
// last run ended.
type SyncStatus struct {
	Running     bool               `json:"running"`
	Server      string             `json:"server"`
	HasToken    bool               `json:"hasToken"`
	LastSuccess string             `json:"lastSuccess"`
	Error       *SyncError         `json:"error,omitempty"`
	Result      *syncremote.Result `json:"result,omitempty"`
}

type syncState struct {
	mu      sync.Mutex
	running bool
	lastErr *SyncError
	result  *syncremote.Result
	source  SyncSettingsSource
	// path overrides the state file, for tests.
	path string
}

// syncEntry is what the file keeps per library and server.
type syncEntry struct {
	Cursors     syncremote.Cursors `json:"cursors"`
	LastSuccess string             `json:"lastSuccess,omitempty"`
}

func (s *syncState) file() string {
	if s.path != "" {
		return s.path
	}
	return filepath.Join(xdg.ConfigHome, "blunderDB", "sync-state.json")
}

// load reads the state file; a missing or unreadable one is an empty state,
// which only costs a full pass.
func (s *syncState) load() map[string]syncEntry {
	out := map[string]syncEntry{}
	if raw, err := os.ReadFile(s.file()); err == nil {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}

func (s *syncState) store(m map[string]syncEntry) error {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	path := s.file()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "sync-state-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// syncKey names one library's exchange with one server: the cursors are
// meaningless for any other pair.
func syncKey(db *database.Database, server string) string {
	return database.OpenPath(db) + "\x00" + server
}

func (a *App) syncSettings() (server string, auto bool) {
	if a.sync.source == nil {
		return "", false
	}
	return a.sync.source.SyncServerAddress()
}

// SyncSetToken stores the token for a server in the system keyring; an empty
// token removes it.
func (a *App) SyncSetToken(server, token string) error {
	addr, err := syncremote.NormalizeAddress(server)
	if err != nil {
		return newGUIError(CodeInvalid, "%v", err)
	}
	token = strings.TrimSpace(token)
	if token == "" {
		err = keyring.Delete(syncKeyringService, addr)
		if errors.Is(err, keyring.ErrNotFound) {
			err = nil
		}
	} else {
		err = keyring.Set(syncKeyringService, addr, token)
	}
	if err != nil {
		return newGUIError(CodeInternal, "keyring: %v", err)
	}
	return nil
}

func syncHasToken(server string) bool {
	addr, err := syncremote.NormalizeAddress(server)
	if err != nil {
		return false
	}
	_, err = keyring.Get(syncKeyringService, addr)
	return err == nil
}

// SyncStatus reports the state of the synchronisation of the open library.
func (a *App) SyncStatus() SyncStatus {
	server, _ := a.syncSettings()
	a.sync.mu.Lock()
	defer a.sync.mu.Unlock()
	return a.statusLocked(server)
}

func (a *App) statusLocked(server string) SyncStatus {
	st := SyncStatus{Running: a.sync.running, Server: server, HasToken: syncHasToken(server), Error: a.sync.lastErr, Result: a.sync.result}
	if a.db != nil {
		if addr, err := syncremote.NormalizeAddress(server); err == nil {
			st.LastSuccess = a.sync.load()[syncKey(a.db, addr)].LastSuccess
		}
	}
	return st
}

// SyncNow starts a run in the background and returns at once; the window
// follows it by the sync:status event. Nothing happens if one is under way.
func (a *App) SyncNow() SyncStatus {
	server, _ := a.syncSettings()
	a.sync.mu.Lock()
	if a.sync.running {
		defer a.sync.mu.Unlock()
		return a.statusLocked(server)
	}
	a.sync.running, a.sync.lastErr = true, nil
	st := a.statusLocked(server)
	a.sync.mu.Unlock()
	go func() {
		a.syncRun(context.Background())
		a.emitSync()
	}()
	return st
}

// syncAtStartup runs once when the window opens, for a user who asked for it
// and has both an address and a token. It stays silent otherwise: a library
// that is not open yet, or no account, is not a failure.
func (a *App) syncAtStartup() {
	server, auto := a.syncSettings()
	if !auto || strings.TrimSpace(server) == "" || a.db == nil || !syncHasToken(server) {
		return
	}
	a.SyncNow()
}

func (a *App) emitSync() {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, syncStatusEvent, a.SyncStatus())
}

// syncRun is one run, and always ends by clearing the running mark and
// recording how it ended. Offline is a result, not a block: the library is
// untouched and usable, and the error says what failed.
func (a *App) syncRun(ctx context.Context) {
	db := a.db
	var runErr *SyncError
	var res *syncremote.Result
	defer func() {
		a.sync.mu.Lock()
		a.sync.running, a.sync.lastErr, a.sync.result = false, runErr, res
		a.sync.mu.Unlock()
	}()

	server, _ := a.syncSettings()
	addr, err := syncremote.NormalizeAddress(server)
	if err != nil {
		runErr = &SyncError{Kind: "address", Message: err.Error()}
		return
	}
	if db == nil {
		runErr = &SyncError{Kind: "no_library", Message: "no library is open"}
		return
	}
	token, err := keyring.Get(syncKeyringService, addr)
	if err != nil {
		runErr = &SyncError{Kind: "no_token", Message: "no token is stored for this server"}
		return
	}
	client, err := syncremote.NewClient(addr, token, nil)
	if err != nil {
		runErr = syncError(err)
		return
	}

	key := syncKey(db, addr)
	entries := a.sync.load()
	entry := entries[key]
	save := func(c syncremote.Cursors) error {
		entry.Cursors = c
		entries[key] = entry
		return a.sync.store(entries)
	}
	_, r, err := syncremote.Run(ctx, db, client, entry.Cursors, syncremote.Options{Save: save})
	res = &r
	if err != nil {
		runErr = syncError(err)
		slog.Warn("sync failed", "server", addr, "kind", runErr.Kind, "err", runErr.Message)
		return
	}
	entry.LastSuccess = time.Now().UTC().Format(time.RFC3339)
	entries[key] = entry
	if err := a.sync.store(entries); err != nil {
		runErr = &SyncError{Kind: "other", Message: "state file: " + err.Error()}
	}
}

func syncError(err error) *SyncError {
	kind := "other"
	switch {
	case errors.Is(err, syncremote.ErrRefused):
		kind = "refused"
	case errors.Is(err, syncremote.ErrRateLimited):
		kind = "rate_limited"
	case errors.Is(err, syncremote.ErrTooLarge):
		kind = "too_large"
	case errors.Is(err, syncremote.ErrBadAddress):
		kind = "address"
	case errors.Is(err, syncremote.ErrUnavailable):
		kind = "unavailable"
	case errors.Is(err, storage.ErrCursorStale):
		kind = "other"
	}
	return &SyncError{Kind: kind, Message: err.Error()}
}
