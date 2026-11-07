package gui

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/videomux"
)

func faultyVideo(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../pkg/blunderdb/videomux/testdata/delta-cluster.webm")
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "match.webm")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func withRepairOptions(t *testing.T, opt videomux.RepairOptions) {
	t.Helper()
	saved := repairOptions
	repairOptions = opt
	t.Cleanup(func() { repairOptions = saved })
}

// slowFFmpeg stands in for an ffmpeg that runs until killed, counting its starts in a file.
func slowFFmpeg(t *testing.T) (path, starts string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in ffmpeg is a shell script")
	}
	dir := t.TempDir()
	path, starts = filepath.Join(dir, "ffmpeg"), filepath.Join(dir, "starts")
	script := "#!/bin/sh\necho x >> '" + starts + "'\nexec sleep 30\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path, starts
}

func waitStarted(t *testing.T, starts string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(starts); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the stand-in ffmpeg never started")
}

func TestRepairVideoMux_UnservedPathIsNotTouched(t *testing.T) {
	a := &App{}
	t.Cleanup(a.stopMedia)
	in := faultyVideo(t)
	if got := a.RepairVideoMux(in); got.Path != "" || got.Reason == "" {
		t.Fatalf("unserved: %+v, want a refusal", got)
	}
	if _, err := os.Stat(videomux.RemuxOutput(in)); !os.IsNotExist(err) {
		t.Fatal("a copy was written for a file no pane serves")
	}
}

func TestRepairVideoMux_RealFFmpeg(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not on the PATH")
	}
	withRepairOptions(t, videomux.RepairOptions{CacheDir: t.TempDir()})
	a := &App{}
	t.Cleanup(a.stopMedia)
	in := faultyVideo(t)
	if _, err := a.MediaURL(in); err != nil {
		t.Fatal(err)
	}
	got := a.RepairVideoMux(in)
	if got.Path != videomux.RemuxOutput(in) {
		t.Fatalf("repair = %+v, want %s", got, videomux.RemuxOutput(in))
	}
	if _, err := a.MediaURL(got.Path); err != nil {
		t.Fatal(err)
	}
	if rep := a.CheckVideoMux(got.Path); !rep.Checked || len(rep.Faulty) != 0 {
		t.Errorf("copy report = %+v, want sound", rep)
	}
}

func TestRepairVideoMux_NoFFmpeg(t *testing.T) {
	withRepairOptions(t, videomux.RepairOptions{FFmpeg: "/nonexistent/ffmpeg", CacheDir: t.TempDir()})
	a := &App{}
	t.Cleanup(a.stopMedia)
	in := faultyVideo(t)
	if _, err := a.MediaURL(in); err != nil {
		t.Fatal(err)
	}
	if got := a.RepairVideoMux(in); !got.NoFFmpeg || got.Path != "" {
		t.Fatalf("repair = %+v, want NoFFmpeg", got)
	}
}

// Two panes on one file share one run; it stops only when the last of them gives it up.
func TestRepairVideoMux_OneRunPerFileCancelledByLastPane(t *testing.T) {
	ff, starts := slowFFmpeg(t)
	withRepairOptions(t, videomux.RepairOptions{FFmpeg: ff, CacheDir: t.TempDir()})
	a := &App{}
	t.Cleanup(a.stopMedia)
	in := faultyVideo(t)
	if _, err := a.MediaURL(in); err != nil {
		t.Fatal(err)
	}
	results := make([]VideoRepair, 2)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func() { defer wg.Done(); results[i] = a.RepairVideoMux(in) }()
		if i == 0 {
			waitStarted(t, starts)
		}
	}
	// Both callers hold the run before the first gives it up.
	deadline := time.Now().Add(5 * time.Second)
	for {
		a.media.mu.Lock()
		job := a.media.repairs[in]
		n := 0
		if job != nil {
			n = job.holders
		}
		a.media.mu.Unlock()
		if n == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("holders = %d, want 2", n)
		}
		time.Sleep(10 * time.Millisecond)
	}
	a.CancelVideoRepair(in)
	a.media.mu.Lock()
	stopping := a.media.repairs[in] != nil && a.media.repairs[in].stopping
	a.media.mu.Unlock()
	if stopping {
		t.Fatal("the run stopped while a pane still waits on it")
	}
	a.CancelVideoRepair(in)
	wg.Wait()
	for _, r := range results {
		if !r.Cancelled || r.Path != "" {
			t.Errorf("result = %+v, want cancelled", r)
		}
	}
	if b, _ := os.ReadFile(starts); len(b) != 2 { // "x\n" once
		t.Errorf("ffmpeg started %d times, want once", len(b)/2)
	}
	if _, err := os.Stat(videomux.RemuxOutput(in)); !os.IsNotExist(err) {
		t.Error("a cancelled run left a copy")
	}
}

// Closing the app (or the last pane on the file) stops the run without a CancelVideoRepair.
func TestRepairVideoMux_StopCancels(t *testing.T) {
	ff, starts := slowFFmpeg(t)
	withRepairOptions(t, videomux.RepairOptions{FFmpeg: ff, CacheDir: t.TempDir()})
	a := &App{}
	in := faultyVideo(t)
	u, err := a.MediaURL(in)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan VideoRepair, 1)
	go func() { done <- a.RepairVideoMux(in) }()
	waitStarted(t, starts)
	a.ReleaseMedia(u)
	select {
	case r := <-done:
		if !r.Cancelled {
			t.Errorf("result = %+v, want cancelled", r)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("releasing the file did not stop the repair")
	}

	if _, err := a.MediaURL(in); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(starts)
	go func() { done <- a.RepairVideoMux(in) }()
	waitStarted(t, starts)
	a.stopMedia()
	select {
	case r := <-done:
		if !r.Cancelled {
			t.Errorf("result = %+v, want cancelled", r)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("stopping the media host did not stop the repair")
	}
}
