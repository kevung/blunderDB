package gui

import (
	"bytes"
	"os"
	"testing"
)

// TestPrepareDemoDatabase verifies the embedded sample database decompresses to
// a real, non-empty SQLite file that the normal open flow can then load.
func TestPrepareDemoDatabase(t *testing.T) {
	a := NewApp()

	path, err := a.PrepareDemoDatabase()
	if err != nil {
		t.Fatalf("PrepareDemoDatabase: %v", err)
	}
	t.Cleanup(func() { os.Remove(path) })

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading prepared demo db: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("prepared demo db is empty")
	}
	// SQLite database files start with this 16-byte magic header.
	if !bytes.HasPrefix(data, []byte("SQLite format 3\x00")) {
		t.Errorf("prepared demo db is not a SQLite file (header: %q)", data[:min(16, len(data))])
	}

	// Each call must yield a fresh, independent temp file.
	path2, err := a.PrepareDemoDatabase()
	if err != nil {
		t.Fatalf("second PrepareDemoDatabase: %v", err)
	}
	t.Cleanup(func() { os.Remove(path2) })
	if path2 == path {
		t.Errorf("expected a fresh temp file each call, got %q twice", path)
	}
}
