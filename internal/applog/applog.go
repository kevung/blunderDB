// Package applog is the desktop GUI's on-disk log file: location, size-based
// rotation, and the "open the log folder" affordance. It exists so the GUI
// has somewhere durable to write diagnostics to — logging.go's stderr-only
// setup (package main, repo root) is invisible once the app is launched by a
// double-click or a desktop-file entry, which has no attached terminal
// (#241). It has no dependency on package main or internal/gui so either can
// import it without a cycle: main wires it into slog at startup, gui reads
// Path()/Dir() for the "open logs folder" button and for pointing an
// unexpected-error dialog at the right file.
package applog

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/adrg/xdg"
)

// appDirName is the directory name under $XDG_STATE_HOME (or the platform
// default when unset) blunderDB's log file lives in — the same "blunderDB"
// name config.go's $XDG_CONFIG_HOME directory already uses.
const appDirName = "blunderDB"

// fileName is the current log file's name; rotation keeps at most one
// previous generation alongside it, fileName+".1".
const fileName = "blunderdb.log"

// maxFileBytes rotates the log once the current file would exceed this size:
// a long-running GUI session must not grow blunderdb.log without bound.
const maxFileBytes = 5 << 20 // 5 MiB

// Dir returns the directory blunderDB's GUI log file lives in.
func Dir() string {
	return filepath.Join(xdg.StateHome, appDirName)
}

// Path returns the full path to the current log file.
func Path() string {
	return filepath.Join(Dir(), fileName)
}

// rotatingWriter is an io.WriteCloser over Path() that rotates to
// Path()+".1" once the current file would exceed maxFileBytes. Every write
// this process makes to it goes through one shared *slog.Logger, whose
// Handler already serializes calls to Handle — so the mutex here guards
// against nothing that shouldn't already be single-threaded in practice, but
// costs nothing to keep honest rather than assumed.
type rotatingWriter struct {
	mu   sync.Mutex
	path string
	f    *os.File
	size int64
}

// Open returns an io.WriteCloser over the current log file, creating
// Dir() if needed, ready to back a slog.Handler. Rotation is by size only
// (#241); callers that also want output on stderr (a terminal launch, `wails
// dev`) should combine it themselves, e.g. io.MultiWriter(os.Stderr, w) —
// Open never assumes stderr is wanted, since a `serve`/CLI invocation
// already has its own stderr-only logging and must not also start writing a
// GUI log file it will never rotate or expose a "open folder" button for.
func Open() (io.WriteCloser, error) {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("applog: %w", err)
	}
	path := Path()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("applog: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("applog: %w", err)
	}
	return &rotatingWriter{path: path, f: f, size: info.Size()}, nil
}

func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.size > 0 && w.size+int64(len(p)) > maxFileBytes {
		if err := w.rotateLocked(); err != nil {
			// Losing the rotation must not lose the log line itself: keep
			// writing to the still-open (now oversized) file and just note
			// the failure where it can still be seen.
			fmt.Fprintf(os.Stderr, "applog: rotate %s: %v\n", w.path, err)
		}
	}
	n, err := w.f.Write(p)
	w.size += int64(n)
	return n, err
}

// rotateLocked closes the current file, moves it to path+".1" (replacing any
// previous backup), and reopens path fresh. Called with mu held.
func (w *rotatingWriter) rotateLocked() error {
	if err := w.f.Close(); err != nil {
		return err
	}
	backup := w.path + ".1"
	_ = os.Remove(backup) // best-effort; a missing backup is not an error
	if err := os.Rename(w.path, backup); err != nil {
		// The old file is still there, just not renamed — reopen it in
		// append mode rather than lose it, and try rotating again next time.
		f, ferr := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if ferr != nil {
			return fmt.Errorf("rename: %w (and reopen failed: %w)", err, ferr)
		}
		w.f = f
		return fmt.Errorf("rename: %w", err)
	}
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	w.f = f
	w.size = 0
	return nil
}

func (w *rotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.f.Close()
}

// TailLines returns the last n lines of the current log file, oldest first.
//
// Reading the tail rather than the whole file is what makes this usable from
// the interface: the log rotates at 5 MiB, and a panel that had to hold five
// megabytes to show forty lines would be a panel nobody opens.
//
// A missing file is not an error — a fresh install has logged nothing — and
// neither is a truncated read: the point of the panel is to show what there
// is, not to prove the file is well formed.
func TailLines(n int) ([]string, error) {
	return tailFile(Path(), n)
}

// tailFile is TailLines over an explicit path, so the behaviour can be tested
// without moving the process's XDG directories.
func tailFile(path string, n int) ([]string, error) {
	if n <= 0 {
		n = 200
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("applog: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("applog: %w", err)
	}
	// Read at most the last tailWindowBytes: enough for n lines of slog output
	// by a wide margin, and bounded whatever the file grew to.
	const tailWindowBytes = 512 << 10
	size := info.Size()
	offset := int64(0)
	if size > tailWindowBytes {
		offset = size - tailWindowBytes
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("applog: %w", err)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("applog: %w", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	// The first line of a windowed read is almost certainly cut in half;
	// dropping it is better than showing a fragment as if it were a record.
	if offset > 0 && len(lines) > 1 {
		lines = lines[1:]
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}
	return lines, nil
}
