// Package applog is the desktop GUI's on-disk log file: location and
// size-based rotation. A GUI launched without a terminal has no visible
// stderr. It imports neither main nor gui, so both can use it.
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

// appDirName is the log directory under $XDG_STATE_HOME.
const appDirName = "blunderDB"

// fileName is the current log file; rotation keeps one fileName+".1".
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
// Path()+".1" past maxFileBytes. The mutex does not rely on slog serialising.
type rotatingWriter struct {
	mu   sync.Mutex
	path string
	f    *os.File
	size int64
}

// Open returns an io.WriteCloser over the current log file, creating Dir()
// if needed. Callers wanting stderr too combine it (io.MultiWriter).
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
			// A failed rotation must not lose the line: keep writing.
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
		// Rename failed: reopen the old file in append mode, retry next time.
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

// TailLines returns the last n lines of the current log file, oldest first,
// reading only the tail. A missing file or truncated read is not an error.
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
	// The window's first line is likely cut in half: drop it.
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
