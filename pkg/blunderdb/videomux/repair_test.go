package videomux

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const noFFmpeg = "/nonexistent/blunderdb-test/ffmpeg"

// copyFixture puts testdata/name in a fresh directory, as the user's recording.
func copyFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func needFFmpeg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not on the PATH")
	}
}

func assertSound(t *testing.T, path string) {
	t.Helper()
	rep, err := CheckFile(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Partial || len(rep.Faulty) != 0 || rep.Clusters == 0 {
		t.Fatalf("%s: report = %+v, want a sound copy", path, rep)
	}
}

func assertUntouched(t *testing.T, path, fixture string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := os.ReadFile(filepath.Join("testdata", fixture))
	if string(got) != string(want) {
		t.Fatalf("%s was modified", path)
	}
}

// noParts fails when a temporary file of a run is left in dir.
func noParts(t *testing.T, dir string) {
	t.Helper()
	parts, _ := filepath.Glob(filepath.Join(dir, ".*.part*"))
	if len(parts) != 0 {
		t.Fatalf("temporary files left: %v", parts)
	}
}

func TestRepair_WritesSoundCopyBeside(t *testing.T) {
	needFFmpeg(t)
	in := copyFixture(t, "delta-cluster.webm")
	out, err := Repair(context.Background(), in, RepairOptions{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if out != RemuxOutput(in) {
		t.Errorf("copy = %s, want %s", out, RemuxOutput(in))
	}
	assertSound(t, out)
	assertUntouched(t, in, "delta-cluster.webm")
	noParts(t, filepath.Dir(in))
}

func TestRepair_FallsBackToCacheWhenDirectoryIsReadOnly(t *testing.T) {
	needFFmpeg(t)
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a directory the test cannot write to")
	}
	in := copyFixture(t, "delta-cluster.webm")
	dir := filepath.Dir(in)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	cache := filepath.Join(t.TempDir(), "remux")

	out, err := Repair(context.Background(), in, RepairOptions{CacheDir: cache})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(out) != cache || !strings.HasSuffix(out, ".remux.webm") {
		t.Fatalf("copy = %s, want a .remux.webm in %s", out, cache)
	}
	assertSound(t, out)
	noParts(t, cache)

	// Named after the file's state: the same file finds the same copy, without ffmpeg.
	again, err := Repair(context.Background(), in, RepairOptions{CacheDir: cache, FFmpeg: noFFmpeg})
	if err != nil || again != out {
		t.Fatalf("second Repair = %q, %v; want %q", again, err, out)
	}
}

func TestRepair_ReusesSoundCopyWithoutFFmpeg(t *testing.T) {
	in := copyFixture(t, "delta-cluster.webm")
	sound, err := os.ReadFile("testdata/remuxed.webm")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(RemuxOutput(in), sound, 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := Repair(context.Background(), in, RepairOptions{FFmpeg: noFFmpeg, CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if out != RemuxOutput(in) {
		t.Errorf("copy = %s, want the existing %s", out, RemuxOutput(in))
	}
}

func TestRepair_FaultyCopyIsNotReused(t *testing.T) {
	in := copyFixture(t, "delta-cluster.webm")
	faulty, _ := os.ReadFile("testdata/delta-cluster.webm")
	if err := os.WriteFile(RemuxOutput(in), faulty, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Repair(context.Background(), in, RepairOptions{FFmpeg: noFFmpeg, CacheDir: t.TempDir()}); !errors.Is(err, ErrNoFFmpeg) {
		t.Fatalf("err = %v, want ErrNoFFmpeg: a faulty copy is not a repair", err)
	}
}

func TestRepair_WithoutFFmpeg(t *testing.T) {
	in := copyFixture(t, "delta-cluster.webm")
	_, err := Repair(context.Background(), in, RepairOptions{FFmpeg: noFFmpeg, CacheDir: t.TempDir()})
	if !errors.Is(err, ErrNoFFmpeg) {
		t.Fatalf("err = %v, want ErrNoFFmpeg", err)
	}
	if _, err := os.Stat(RemuxOutput(in)); !os.IsNotExist(err) {
		t.Errorf("a copy was written without ffmpeg: %v", err)
	}
}

// A stand-in ffmpeg that never finishes: the run ends only through the context.
func TestRepair_CancelledRunLeavesNothing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in ffmpeg is a shell script")
	}
	fake := filepath.Join(t.TempDir(), "ffmpeg")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexec sleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	in := copyFixture(t, "delta-cluster.webm")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := Repair(ctx, in, RepairOptions{FFmpeg: fake, CacheDir: t.TempDir()})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want the context's", err)
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Errorf("cancellation took %v", d)
	}
	if _, err := os.Stat(RemuxOutput(in)); !os.IsNotExist(err) {
		t.Errorf("a copy was left after cancellation: %v", err)
	}
	noParts(t, filepath.Dir(in))
}

func TestRepair_FailingFFmpegSaysWhy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in ffmpeg is a shell script")
	}
	fake := filepath.Join(t.TempDir(), "ffmpeg")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho 'Invalid data found when processing input' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	in := copyFixture(t, "delta-cluster.webm")
	_, err := Repair(context.Background(), in, RepairOptions{FFmpeg: fake, CacheDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "Invalid data found") {
		t.Fatalf("err = %v, want ffmpeg's last line", err)
	}
	noParts(t, filepath.Dir(in))
}
