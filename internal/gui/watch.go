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

// The watched folder, desktop half. It only looks: new files' paths go to the
// frontend, which imports them exactly as a drag-and-drop, so importing is
// written once. Off unless the user names a folder.

// watchFilesEvent is the event the frontend listens on. Its payload is a list
// of absolute paths.
const watchFilesEvent = "folder-watch:files"

// folderWatch is the single running watch.
type folderWatch struct {
	mu       sync.Mutex
	stop     chan struct{}
	dir      string
	interval time.Duration
}

// WatchStatus is the running watch as reported, not inferred from the config,
// so a failed start never shows as running.
type WatchStatus struct {
	Running         bool   `json:"running"`
	Folder          string `json:"folder"`
	IntervalSeconds int    `json:"intervalSeconds"`
}

// StartFolderWatch begins watching dir, replacing any running watch. Files
// already present are never imported (that is the folder import).
// intervalSeconds is clamped by watch.ClampInterval, 0 = default.
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

// StopFolderWatch stops the watch, if one is running. Idempotent.
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

// SuggestWatchFolder returns an existing plausible match folder, or "". A
// suggestion the user must accept, never a default, and only if it exists.
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

// runFolderWatch is the loop; it emits paths and imports nothing.
func (a *App) runFolderWatch(w *watch.Watcher, interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// warned logs an unreadable folder once, not per tick; the watch keeps
	// running for when it comes back.
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
