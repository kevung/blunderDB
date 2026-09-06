// Package watch reports the importable files that APPEAR in a folder
// (issue #258, fiche I.2).
//
// The intent is the one thing every competition player asks for: play a
// session in eXtreme Gammon, and find the matches already in blunderDB
// without importing them by hand.
//
// # Why this polls rather than subscribing to the file system
//
// The fiche proposed fsnotify. This polls instead, and the reason is
// measured against what the feature needs rather than against what is
// fashionable:
//
//   - The fallback would have to exist anyway. An XG folder very often lives
//     on a network share or a synchronised directory, where inotify and its
//     equivalents report nothing (ADR-0004's whole posture: detect, and fall
//     back). So fsnotify would buy a second code path, not a simpler one.
//   - The latency requirement is loose. A match file appears when a session
//     ends; nobody is watching the second hand. A ten-second poll of one
//     directory costs a readdir.
//   - It adds no dependency and no host capability to probe, on a tool that
//     counts both.
//
// # What "new" and "stable" mean
//
// New means "not present when the watch started, and not reported since". A
// watch does NOT import the folder it finds: someone starting a watch on a
// folder holding four years of matches wants the next one, not all of them.
// Importing what is already there is a separate, explicit gesture, and it
// already exists.
//
// Stable means "seen twice with the same size and modification time". A file
// being written by another program grows between two polls, and importing a
// half-written match would produce a parse error the user cannot act on.
package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// DefaultInterval is how often a watch looks. Ten seconds: fast enough that
// the match is there before the user has switched windows, slow enough that
// the cost of watching a folder is not worth measuring.
const DefaultInterval = 10 * time.Second

// MinInterval is the floor a caller's interval is clamped to. Two seconds,
// because the stability rule needs two polls to conclude anything and a
// sub-second loop on a network share is a way to be unwelcome on it.
const MinInterval = 2 * time.Second

// fileState is what one poll saw of one file.
type fileState struct {
	size    int64
	modTime time.Time
}

// Watcher reports the importable files that appear in one directory. It is
// not safe for concurrent use: one goroutine polls it.
type Watcher struct {
	dir string
	// reported holds every path already handed to a caller, plus everything
	// that was present when the watch started. A path leaves this map only
	// with the Watcher.
	reported map[string]bool
	// pending holds files seen once but not yet stable.
	pending map[string]fileState
}

// New starts a watch on dir. Everything already in the directory is recorded
// as seen and will never be reported: a watch reports what appears, and the
// caller who wants what is already there has a folder import for that.
//
// An unreadable directory is an error here rather than on the first poll, so
// a mistyped path is refused while the user is still looking at the setting.
func New(dir string) (*Watcher, error) {
	if dir == "" {
		return nil, fmt.Errorf("watch: no folder given")
	}
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("watch: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("watch: %s is not a folder", dir)
	}

	w := &Watcher{
		dir:      dir,
		reported: map[string]bool{},
		pending:  map[string]fileState{},
	}
	present, err := w.scan()
	if err != nil {
		return nil, err
	}
	for path := range present {
		w.reported[path] = true
	}
	return w, nil
}

// Dir is the folder being watched.
func (w *Watcher) Dir() string { return w.dir }

// Poll returns the importable files that have appeared since the watch
// started and have stopped changing, sorted by path so two runs over the same
// folder import in the same order.
//
// A file that has appeared but is still growing is held back and offered on a
// later poll. A file that vanishes before it settles is simply forgotten.
//
// A directory that has become unreadable — unmounted share, revoked
// permission — is an error, and the watch's memory is left intact: the caller
// decides whether to stop or to keep trying, and a share that comes back does
// not re-import everything it holds.
func (w *Watcher) Poll() ([]string, error) {
	present, err := w.scan()
	if err != nil {
		return nil, err
	}

	// Forget anything that was pending and is gone: a temporary file another
	// program renamed away must not stay in memory for the life of the watch.
	for path := range w.pending {
		if _, ok := present[path]; !ok {
			delete(w.pending, path)
		}
	}

	var ready []string
	for path, now := range present {
		if w.reported[path] {
			continue
		}
		before, seen := w.pending[path]
		switch {
		case !seen:
			w.pending[path] = now
		case before == now:
			ready = append(ready, path)
			delete(w.pending, path)
			w.reported[path] = true
		default:
			w.pending[path] = now // still being written
		}
	}
	sort.Strings(ready)
	return ready, nil
}

// scan lists the importable files directly inside the directory, with what
// this moment says about each.
//
// It is deliberately NOT recursive. A watched folder is the place a tool
// drops its matches, not a tree to crawl: recursing would make a watch on a
// home directory a plausible mistake, and the cost of every poll would depend
// on what the user happens to keep below it.
func (w *Watcher) scan() (map[string]fileState, error) {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return nil, fmt.Errorf("watch: reading %s: %w", w.dir, err)
	}
	out := make(map[string]fileState, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(w.dir, e.Name())
		if !ingest.IsImportable(path) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue // vanished between the listing and the stat
		}
		out[path] = fileState{size: info.Size(), modTime: info.ModTime()}
	}
	return out, nil
}

// ClampInterval coerces a caller's interval into what a watch will actually
// use: zero means the default, anything shorter than MinInterval is raised.
func ClampInterval(d time.Duration) time.Duration {
	switch {
	case d <= 0:
		return DefaultInterval
	case d < MinInterval:
		return MinInterval
	default:
		return d
	}
}
