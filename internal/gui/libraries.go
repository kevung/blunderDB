package gui

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"

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
	// locked holds the groups their user locked against transfers into them
	// (SetLibraryGroupLocked): the desktop's memory, never a library's.
	locked map[int]bool
	// exchanging is set while a switch exchanges the two handles: the bound
	// handle's before-switch hook then stops nothing, since the jobs run on
	// borrowed handles that keep their library (borrowLibrary).
	exchanging atomic.Bool
	// shown mirrors active for the before-switch hook, which may run while
	// libs.mu is held (closing the last group closes the bound handle).
	shown atomic.Int64
}

func (g *libraryGroups) setActive(id int) {
	g.active = id
	g.shown.Store(int64(id))
}

// LibraryGroup is one open library as the tab bar shows it.
type LibraryGroup struct {
	ID       int    `json:"id"`
	Path     string `json:"path"`
	Active   bool   `json:"active"`
	ReadOnly bool   `json:"readOnly"`
	// Locked refuses transfers into the group (SetLibraryGroupLocked).
	Locked bool `json:"locked"`
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
		g.locked = map[int]bool{}
		g.setActive(1)
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

// refuseHeldLibrary is the bound handle's open guard: whatever path opens a
// file in the active group (a dialog, a recent file, a drop, the CLI bridge),
// a file another group holds is refused, never opened twice. Reopening the
// active group's own file stays allowed.
func (a *App) refuseHeldLibrary(path string) error {
	a.libs.mu.Lock()
	defer a.libs.mu.Unlock()
	a.libs.init()
	if id := a.holdingLocked(path); id != 0 && id != a.libs.active {
		return &ErrLibraryAlreadyOpen{Group: id}
	}
	return nil
}

// ListLibraryGroups returns the open groups, in the order they were opened.
func (a *App) ListLibraryGroups() []LibraryGroup {
	a.libs.mu.Lock()
	defer a.libs.mu.Unlock()
	a.libs.init()
	var out []LibraryGroup
	if a.db != nil {
		out = append(out, LibraryGroup{ID: a.libs.active, Path: database.OpenPath(a.db), Active: true, ReadOnly: a.db.IsReadOnly(), Locked: a.libs.locked[a.libs.active]})
	}
	for id, d := range a.libs.parked {
		out = append(out, LibraryGroup{ID: id, Path: database.OpenPath(d), ReadOnly: d.IsReadOnly(), Locked: a.libs.locked[id]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// LibrariesWrittenElsewhere returns the groups whose library another process
// (the CLI, a serve daemon) wrote since the previous call; the desktop's own
// writes are left out unless they cannot be told apart (one read too many).
// Reads only, and waits for no library: one held for writing, or whose file
// cannot be read now, is left out and asked again next time, its change kept.
func (a *App) LibrariesWrittenElsewhere() []int {
	a.libs.mu.Lock()
	defer a.libs.mu.Unlock()
	a.libs.init()
	out := []int{}
	ask := func(id int, d *database.Database) {
		written, err := database.WrittenElsewhere(d)
		if err != nil {
			slog.Debug("watching a library for outside writes", "group", id, "err", err)
		}
		if written {
			out = append(out, id)
		}
	}
	if a.db != nil {
		ask(a.libs.active, a.db)
	}
	for id, d := range a.libs.parked {
		ask(id, d)
	}
	sort.Ints(out)
	return out
}

// libraryBusy refuses a switch while a job that follows the screen runs. The
// background jobs (an import, the watched folder, a gammonNet batch) run on a
// handle borrowed from their library and keep it through a switch; a rollout
// shows its result live on the position at the board, and the tournament
// display serves the active library on the network: those two still hold the
// group on screen.
func (a *App) libraryBusy() error {
	a.roMu.Lock()
	rollout := a.roDone != nil
	a.roMu.Unlock()
	if rollout {
		return &database.BusyError{Job: database.JobRollout}
	}
	if a.GetTournamentDisplayStatus().Running {
		return &database.BusyError{Job: "tournament-display"}
	}
	return nil
}

// exchangeLocked swaps the bound handle's library with other's; the caller
// holds libs.mu. The jobs of both libraries run on borrowed handles and are
// left running.
func (a *App) exchangeLocked(other *database.Database) error {
	a.libs.exchanging.Store(true)
	defer a.libs.exchanging.Store(false)
	return database.Exchange(a.db, other)
}

// borrowLibrary lends the library of group (0: the active one) to a
// background job: the returned handle stays on that library whichever group
// comes on screen. The caller lets it go with Close, which closes nothing.
func (a *App) borrowLibrary(group int) (int, *database.Database, error) {
	a.libs.mu.Lock()
	defer a.libs.mu.Unlock()
	return a.borrowLibraryLocked(group)
}

// borrowLibraryLocked is borrowLibrary for a caller holding libs.mu, which
// registers the job before letting go: a group closed meanwhile (under the
// same lock) then finds the job to stop.
func (a *App) borrowLibraryLocked(group int) (int, *database.Database, error) {
	a.libs.init()
	if group == 0 {
		group = a.libs.active
	}
	carrier := a.db
	if group != a.libs.active {
		carrier = a.libs.parked[group]
	}
	if carrier == nil {
		return group, nil, fmt.Errorf("no library group %d", group)
	}
	lent, err := database.Borrow(carrier)
	return group, lent, err
}

// activeGroup is the group on screen.
func (a *App) activeGroup() int {
	a.libs.mu.Lock()
	defer a.libs.mu.Unlock()
	a.libs.init()
	return a.libs.active
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
		err := a.exchangeLocked(fresh)
		_ = fresh.Close()
		if err != nil {
			return LibraryGroup{}, err
		}
		return LibraryGroup{ID: a.libs.active, Path: database.OpenPath(a.db), Active: true, ReadOnly: a.db.IsReadOnly()}, nil
	}
	if err := a.exchangeLocked(fresh); err != nil {
		_ = fresh.Close()
		return LibraryGroup{}, err
	}
	a.libs.parked[a.libs.active] = fresh
	a.libs.setActive(a.libs.next)
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
	if err := a.exchangeLocked(parked); err != nil {
		return err
	}
	delete(a.libs.parked, id)
	a.libs.parked[a.libs.active] = parked
	a.libs.setActive(id)
	return nil
}

// CloseLibraryGroup closes the library of a waiting group and drops the group.
// The active group is closed by activating another first; when it is the only
// one, its library is closed and the group stays, empty. The group's jobs are
// stopped first: they write through handles borrowed from that library.
func (a *App) CloseLibraryGroup(id int) error {
	a.libs.mu.Lock()
	defer a.libs.mu.Unlock()
	a.libs.init()
	if id != a.libs.active {
		// Under libs.mu: no job can borrow this library between the stop
		// and the close.
		a.stopGroupJobs(id, shutdownJobGrace)
		a.stopFolderWatchOf(id)
	}
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
	delete(a.libs.locked, id)
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
