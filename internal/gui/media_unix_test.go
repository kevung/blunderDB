//go:build !windows

package gui

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// A FIFO blocks os.Open until a writer comes: it is refused before any open, whether it is
// offered for serving or swapped in under a served path.
func TestCheckVideoMuxRefusesFIFO(t *testing.T) {
	a := &App{}
	t.Cleanup(a.stopMedia)
	dir := t.TempDir()
	fifo := filepath.Join(dir, "pipe.webm")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skip("mkfifo:", err)
	}
	if _, err := a.MediaURL(fifo); err == nil {
		t.Error("MediaURL served a FIFO")
	}
	served := newMediaFile(t, "swapped.webm", 100)
	if _, err := a.MediaURL(served); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(served); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(served, 0o600); err != nil {
		t.Fatal(err)
	}
	done := make(chan VideoMuxReport, 1)
	go func() { done <- a.CheckVideoMux(served) }()
	select {
	case got := <-done:
		if got.Checked {
			t.Errorf("report = %+v, want the FIFO left unread", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("CheckVideoMux blocked on a FIFO")
	}
}
