package applog

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// newTestWriter opens a rotatingWriter rooted at a temp dir.
func newTestWriter(t *testing.T) (*rotatingWriter, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, fileName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	w := &rotatingWriter{path: path, f: f}
	return w, path
}

// TestRotatingWriter_WritesAppend guards the ordinary case: no rotation,
// every Write lands in the file in order.
func TestRotatingWriter_WritesAppend(t *testing.T) {
	w, path := newTestWriter(t)
	defer w.Close()

	if _, err := w.Write([]byte("line one\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := w.Write([]byte("line two\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	w.Close()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "line one\nline two\n" {
		t.Errorf("content = %q, want both lines in order", got)
	}
}

// TestRotatingWriter_RotatesPastMaxBytes guards #241's size-based rotation:
// once the current file would exceed maxFileBytes, the next write rotates
// first — the old content ends up in path+".1" and the new write starts a
// fresh file rather than growing the old one without bound. Rather than
// writing 5 MiB of filler to reach the real threshold, this seeds w.size
// directly to just under it — Write only ever consults that field (never
// re-stats the file), so this exercises the exact "> maxFileBytes"
// comparison production code runs, against the real package constant.
func TestRotatingWriter_RotatesPastMaxBytes(t *testing.T) {
	w, path := newTestWriter(t)
	defer w.Close()

	first := bytes.Repeat([]byte("a"), 20)
	if _, err := w.Write(first); err != nil {
		t.Fatalf("Write: %v", err)
	}
	w.mu.Lock()
	w.size = maxFileBytes - 1
	w.mu.Unlock()

	second := []byte("bbbb")
	if _, err := w.Write(second); err != nil {
		t.Fatalf("Write: %v", err)
	}
	w.Close()

	backup, err := os.ReadFile(path + ".1")
	if err != nil {
		t.Fatalf("ReadFile(backup): %v", err)
	}
	if !bytes.Equal(backup, first) {
		t.Errorf("backup content = %q, want %q", backup, first)
	}

	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(current): %v", err)
	}
	if !bytes.Equal(current, second) {
		t.Errorf("current content = %q, want only the post-rotation write %q", current, second)
	}
}

// TestOpen_CreatesDirAndFile guards Open()'s directory-creation contract: a
// fresh $XDG_STATE_HOME/blunderDB that does not exist yet must be created,
// not treated as an error.
func TestOpen_CreatesDirAndFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	w, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer w.Close()

	if _, err := w.Write([]byte("hello\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if _, err := os.Stat(Path()); err != nil {
		t.Errorf("log file not created at %s: %v", Path(), err)
	}
}
