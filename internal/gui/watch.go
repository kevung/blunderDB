package gui

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/kevung/blunderdb/pkg/blunderdb/watch"
)

// The watched folder, desktop half (issue #258, fiche I.2).
//
// What this does is deliberately narrow: it LOOKS. When match files appear in
// the folder the user named, it emits their paths to the frontend, and the
// frontend imports them through exactly the path a drag-and-drop takes —
// duplicate detection, import report, automatic analysis, all of it. Nothing
// about importing is written twice, and a watched import is by construction
// the same import as a manual one.
//
// It is off unless the user turns it on and names a folder. blunderDB does
// not guess where somebody's matches live and start reading that directory.

// watchFilesEvent is the event the frontend listens on. Its payload is a list
// of absolute paths.
const watchFilesEvent = "folder-watch:files"

// folderWatch is the single running watch. One, not one per folder: watching
// two folders is a setting nobody asked for, and the second would double
// every question below (which import report? which database?).
type folderWatch struct {
	mu       sync.Mutex
	stop     chan struct{}
	dir      string
	interval time.Duration
}

// WatchStatus is what the settings pane shows: whether a watch is running and
// on what. Reported rather than inferred from the config, so a watch that
// failed to start does not display as running.
type WatchStatus struct {
	Running         bool   `json:"running"`
	Folder          string `json:"folder"`
	IntervalSeconds int    `json:"intervalSeconds"`
}

// StartFolderWatch begins watching dir, replacing any watch already running.
//
// Files already in the folder are recorded and never imported: someone
// pointing a watch at four years of matches wants the next one. Importing
// what is there is the folder import, which is a separate gesture and stays
// one.
//
// intervalSeconds 0 means the package default; anything below the floor is
// raised (watch.ClampInterval).
func (a *App) StartFolderWatch(dir string, intervalSeconds int) (WatchStatus, error) {
	w, err := watch.New(dir)
	if err != nil {
		return WatchStatus{}, err
	}
	interval := watch.ClampInterval(time.Duration(intervalSeconds) * time.Second)

	a.folderWatch.mu.Lock()
	if a.folderWatch.stop != nil {
		close(a.folderWatch.stop)
	}
	stop := make(chan struct{})
	a.folderWatch.stop = stop
	a.folderWatch.dir = w.Dir()
	a.folderWatch.interval = interval
	a.folderWatch.mu.Unlock()

	go a.runFolderWatch(w, interval, stop)

	slog.Info("watching folder for new match files", "folder", w.Dir(), "interval", interval)
	return WatchStatus{Running: true, Folder: w.Dir(), IntervalSeconds: int(interval / time.Second)}, nil
}

// StopFolderWatch stops the watch, if one is running. Idempotent: the
// settings pane calls it whenever the user turns the option off, whatever the
// state was.
func (a *App) StopFolderWatch() {
	a.folderWatch.mu.Lock()
	defer a.folderWatch.mu.Unlock()
	if a.folderWatch.stop != nil {
		close(a.folderWatch.stop)
		a.folderWatch.stop = nil
	}
	a.folderWatch.dir = ""
}

// FolderWatchStatus reports the running watch, if any.
func (a *App) FolderWatchStatus() WatchStatus {
	a.folderWatch.mu.Lock()
	defer a.folderWatch.mu.Unlock()
	if a.folderWatch.stop == nil {
		return WatchStatus{}
	}
	return WatchStatus{
		Running:         true,
		Folder:          a.folderWatch.dir,
		IntervalSeconds: int(a.folderWatch.interval / time.Second),
	}
}

// SuggestWatchFolder returns a folder this machine actually has that is a
// plausible place for match files, or "" when none of the candidates exists.
//
// A SUGGESTION, never a default. The fiche proposed defaulting to XG's own
// folder; a path invented from what XG installs on somebody else's Windows is
// exactly the kind of unverified claim this project refuses to ship. So the
// candidates below are only offered when os.Stat says they are there, and the
// user still has to accept one.
func (a *App) SuggestWatchFolder() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	docs := filepath.Join(home, "Documents")
	for _, c := range []string{
		filepath.Join(docs, "eXtreme Gammon", "Matches"),
		filepath.Join(docs, "eXtreme Gammon", "UserData", "Matches"),
		filepath.Join(docs, "eXtreme Gammon"),
		filepath.Join(home, ".gnubg", "matches"),
		filepath.Join(docs, "Backgammon"),
	} {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	return ""
}

// runFolderWatch is the loop. It emits paths and imports nothing: the
// frontend owns importing, so a watched import and a dropped import are the
// same code.
func (a *App) runFolderWatch(w *watch.Watcher, interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// warned keeps an unreadable folder from filling the log once per tick:
	// an unmounted share can stay unmounted for hours, and the watch is
	// deliberately still running when it comes back.
	warned := false
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
		}

		files, err := w.Poll()
		if err != nil {
			if !warned {
				slog.Warn("watched folder could not be read; still watching", "folder", w.Dir(), "error", err)
				warned = true
			}
			continue
		}
		warned = false
		if len(files) == 0 {
			continue
		}
		slog.Info("new match files in the watched folder", "folder", w.Dir(), "count", len(files))
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, watchFilesEvent, files)
		}
	}
}
