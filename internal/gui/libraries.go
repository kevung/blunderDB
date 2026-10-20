package gui

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// libraryGroups holds the libraries the desktop keeps open besides the active
// one (ADR-0101). The *Database bound to Wails always carries the active
// library; each other group waits in a handle of its own, its file open, and
// a switch exchanges the two (database.Exchange) without opening or closing
// anything. Groups are presentation: neither the CLI nor the daemon has them.
type libraryGroups struct {
	mu     sync.Mutex
	active int
	next   int
	parked map[int]*database.Database
}

// LibraryGroup is one open library as the tab bar shows it.
type LibraryGroup struct {
	ID       int    `json:"id"`
	Path     string `json:"path"`
	Active   bool   `json:"active"`
	ReadOnly bool   `json:"readOnly"`
}

// ErrLibraryAlreadyOpen refuses a file already open in a group; the message
// carries the group so the desktop can bring it forward instead.
type ErrLibraryAlreadyOpen struct{ Group int }

func (e *ErrLibraryAlreadyOpen) Error() string {
	return fmt.Sprintf("library already open in group %d", e.Group)
}

func (g *libraryGroups) init() {
	if g.parked == nil {
		g.parked = map[int]*database.Database{}
		g.active = 1
		g.next = 2
	}
}

// canonicalPath names a file once whatever the spelling: absolute, symlinks
// resolved. A file that does not exist yet keeps its absolute spelling.
func canonicalPath(path string) string {
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		return real
	}
	return filepath.Clean(abs)
}

// holdingLocked returns the group holding path, 0 when none.
func (a *App) holdingLocked(path string) int {
	want := canonicalPath(path)
	if want == "" {
		return 0
	}
	if a.db != nil && canonicalPath(database.OpenPath(a.db)) == want {
		return a.libs.active
	}
	for id, d := range a.libs.parked {
		if canonicalPath(database.OpenPath(d)) == want {
			return id
		}
	}
	return 0
}

// LibraryGroupHolding returns the group whose library is path, 0 when no
// group holds it. The desktop asks before opening or creating a file.
func (a *App) LibraryGroupHolding(path string) int {
	a.libs.mu.Lock()
	defer a.libs.mu.Unlock()
	a.libs.init()
	return a.holdingLocked(path)
}

// ListLibraryGroups returns the open groups, in the order they were opened.
func (a *App) ListLibraryGroups() []LibraryGroup {
	a.libs.mu.Lock()
	defer a.libs.mu.Unlock()
	a.libs.init()
	var out []LibraryGroup
	if a.db != nil {
		out = append(out, LibraryGroup{ID: a.libs.active, Path: database.OpenPath(a.db), Active: true, ReadOnly: a.db.IsReadOnly()})
	}
	for id, d := range a.libs.parked {
		out = append(out, LibraryGroup{ID: id, Path: database.OpenPath(d), ReadOnly: d.IsReadOnly()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// libraryBusy refuses a switch while a job of the desktop writes the
// active library: its writes go through the bound handle, which a switch
// hands to another file.
func (a *App) libraryBusy() error {
	a.gnBatchMu.Lock()
	batch := a.gnBatchDone != nil
	a.gnBatchMu.Unlock()
	if batch {
		return &database.BusyError{Job: "gammonnet-batch"}
	}
	a.roMu.Lock()
	rollout := a.roDone != nil
	a.roMu.Unlock()
	if rollout {
		return &database.BusyError{Job: database.JobRollout}
	}
	if a.FolderWatchStatus().Running {
		return &database.BusyError{Job: "folder-watch"}
	}
	if a.GetTournamentDisplayStatus().Running {
		return &database.BusyError{Job: "tournament-display"}
	}
	return nil
}

// OpenLibraryGroup opens path in a new group and makes it active; the library
// active until then waits, open, in its group. A file some group already holds
// is refused (*ErrLibraryAlreadyOpen). With no library open, the active group
// takes the file.
func (a *App) OpenLibraryGroup(path string) (LibraryGroup, error) {
	if a.db == nil {
		return LibraryGroup{}, errors.New("no database handle")
	}
	a.libs.mu.Lock()
	defer a.libs.mu.Unlock()
	a.libs.init()
	if id := a.holdingLocked(path); id != 0 {
		return LibraryGroup{}, &ErrLibraryAlreadyOpen{Group: id}
	}
	if err := a.libraryBusy(); err != nil {
		return LibraryGroup{}, err
	}
	fresh := database.NewDatabase()
	if err := fresh.OpenDatabase(path); err != nil {
		return LibraryGroup{}, err
	}
	if database.OpenPath(a.db) == "" {
		// Nothing open: the active group takes the file, no group to add.
		err := database.Exchange(a.db, fresh)
		_ = fresh.Close()
		if err != nil {
			return LibraryGroup{}, err
		}
		return LibraryGroup{ID: a.libs.active, Path: database.OpenPath(a.db), Active: true, ReadOnly: a.db.IsReadOnly()}, nil
	}
	if err := database.Exchange(a.db, fresh); err != nil {
		_ = fresh.Close()
		return LibraryGroup{}, err
	}
	a.libs.parked[a.libs.active] = fresh
	a.libs.active = a.libs.next
	a.libs.next++
	return LibraryGroup{ID: a.libs.active, Path: database.OpenPath(a.db), Active: true, ReadOnly: a.db.IsReadOnly()}, nil
}

// ActivateLibraryGroup makes group id the active one. Nothing is opened,
// closed or read: the two handles exchange their libraries.
func (a *App) ActivateLibraryGroup(id int) error {
	if a.db == nil {
		return errors.New("no database handle")
	}
	a.libs.mu.Lock()
	defer a.libs.mu.Unlock()
	a.libs.init()
	if id == a.libs.active {
		return nil
	}
	parked, ok := a.libs.parked[id]
	if !ok {
		return fmt.Errorf("no library group %d", id)
	}
	if err := a.libraryBusy(); err != nil {
		return err
	}
	if err := database.Exchange(a.db, parked); err != nil {
		return err
	}
	delete(a.libs.parked, id)
	a.libs.parked[a.libs.active] = parked
	a.libs.active = id
	return nil
}

// CloseLibraryGroup closes the library of a waiting group and drops the group.
// The active group is closed by activating another first; when it is the only
// one, its library is closed and the group stays, empty.
func (a *App) CloseLibraryGroup(id int) error {
	a.libs.mu.Lock()
	defer a.libs.mu.Unlock()
	a.libs.init()
	if id == a.libs.active {
		if len(a.libs.parked) > 0 {
			return errors.New("activate another library group before closing the active one")
		}
		if a.db == nil {
			return nil
		}
		return a.db.Close()
	}
	parked, ok := a.libs.parked[id]
	if !ok {
		return nil
	}
	delete(a.libs.parked, id)
	return parked.Close()
}

// closeParkedLibraries closes every waiting library, at shutdown.
func (a *App) closeParkedLibraries() {
	a.libs.mu.Lock()
	defer a.libs.mu.Unlock()
	for id, d := range a.libs.parked {
		if err := d.Close(); err != nil {
			slog.Warn("closing a library group on shutdown", "group", id, "err", err)
		}
		delete(a.libs.parked, id)
	}
}
