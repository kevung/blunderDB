package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// write creates or rewrites a file with n bytes and a modification time far
// enough in the past that a later write is distinguishable on a file system
// with a coarse timestamp.
func write(t *testing.T, dir, name string, n int, age time.Duration) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, make([]byte, n), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	when := time.Now().Add(-age)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatalf("chtimes %s: %v", name, err)
	}
	return path
}

// A watch reports what APPEARS, never what was already there. Someone
// starting a watch on a folder holding four years of matches wants the next
// one; importing the rest is a separate, explicit gesture.
func TestWatcherIgnoresWhatWasAlreadyThere(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "old.xg", 100, time.Hour)

	w, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for i := 0; i < 3; i++ {
		got, err := w.Poll()
		if err != nil {
			t.Fatalf("Poll: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("poll %d reported %v; a pre-existing file is not an appearance", i, got)
		}
	}
}

// The stability rule: a file is offered only once two polls have seen it
// unchanged. Importing a match another program is still writing would give
// the user a parse error they cannot act on.
func TestWatcherWaitsForAFileToSettle(t *testing.T) {
	dir := t.TempDir()
	w, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	write(t, dir, "new.xg", 100, time.Hour)
	if got, _ := w.Poll(); len(got) != 0 {
		t.Fatalf("first sighting reported %v, want nothing", got)
	}

	// Still growing: a second, different sighting must not release it.
	write(t, dir, "new.xg", 200, time.Minute)
	if got, _ := w.Poll(); len(got) != 0 {
		t.Fatalf("a file that changed between polls was reported: %v", got)
	}

	// Unchanged now.
	got, err := w.Poll()
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(got) != 1 || filepath.Base(got[0]) != "new.xg" {
		t.Fatalf("stable file: got %v, want [new.xg]", got)
	}

	// And it is reported exactly once.
	if again, _ := w.Poll(); len(again) != 0 {
		t.Fatalf("the same file was reported twice: %v", again)
	}
}

func TestWatcherIgnoresWhatItCannotImport(t *testing.T) {
	dir := t.TempDir()
	w, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	write(t, dir, "notes.pdf", 10, time.Hour)
	write(t, dir, "match.mat", 10, time.Hour)
	if err := os.Mkdir(filepath.Join(dir, "sub.xg"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	_, _ = w.Poll()
	got, err := w.Poll()
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(got) != 1 || filepath.Base(got[0]) != "match.mat" {
		t.Fatalf("got %v, want only match.mat — a directory named like a match is not one", got)
	}
}

// A temporary file another program renames away must not stay in the
// watcher's memory for the life of the watch.
func TestWatcherForgetsAFileThatVanishes(t *testing.T) {
	dir := t.TempDir()
	w, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	path := write(t, dir, "half.xg", 10, time.Hour)
	if got, _ := w.Poll(); len(got) != 0 {
		t.Fatalf("first sighting reported %v", got)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if got, _ := w.Poll(); len(got) != 0 {
		t.Fatalf("a vanished file was reported: %v", got)
	}
	if len(w.pending) != 0 {
		t.Fatalf("the vanished file is still pending: %v", w.pending)
	}
}

// A share that goes away and comes back must not look like a folder full of
// new matches: the watch's memory survives the error.
func TestWatcherKeepsItsMemoryAcrossAnUnreadableFolder(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "old.xg", 10, time.Hour)
	w, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	gone := filepath.Join(t.TempDir(), "not-here")
	w.dir = gone
	if _, err := w.Poll(); err == nil {
		t.Fatal("polling an unreadable folder should be an error")
	}

	w.dir = dir
	if got, _ := w.Poll(); len(got) != 0 {
		t.Fatalf("after the folder came back: %v, want nothing — those files were already known", got)
	}
}

func TestNewRefusesWhatIsNotAFolder(t *testing.T) {
	dir := t.TempDir()
	file := write(t, dir, "match.xg", 10, time.Hour)
	if _, err := New(file); err == nil {
		t.Fatal("a file is not a folder to watch")
	}
	if _, err := New(filepath.Join(dir, "nope")); err == nil {
		t.Fatal("a missing folder should be refused while the user is still looking at the setting")
	}
	if _, err := New(""); err == nil {
		t.Fatal("an empty path should be refused")
	}
}

func TestClampInterval(t *testing.T) {
	if got := ClampInterval(0); got != DefaultInterval {
		t.Errorf("ClampInterval(0) = %v, want %v", got, DefaultInterval)
	}
	if got := ClampInterval(-time.Second); got != DefaultInterval {
		t.Errorf("ClampInterval(-1s) = %v, want %v", got, DefaultInterval)
	}
	if got := ClampInterval(time.Millisecond); got != MinInterval {
		t.Errorf("ClampInterval(1ms) = %v, want %v", got, MinInterval)
	}
	if got := ClampInterval(time.Minute); got != time.Minute {
		t.Errorf("ClampInterval(1m) = %v, want 1m", got)
	}
}
