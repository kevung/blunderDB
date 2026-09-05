package bearoffgen

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func ts(points, checkers int) Domain {
	return Domain{Kind: TwoSidedKind, Points: points, Checkers: checkers}
}

// A paused run and the run that resumes it must land on the table an
// uninterrupted run writes — the file, not just the state in memory.
func TestGenerateWith_PauseThenResumeWritesTheSameFile(t *testing.T) {
	t.Parallel()
	d := ts(6, 5)

	whole := t.TempDir()
	ref, err := GenerateWith(context.Background(), whole, d, RunOptions{Workers: 4})
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(ref)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	_, err = GenerateWith(ctx, dir, d, RunOptions{
		Workers:  4,
		Pausable: true,
		Progress: func(done, total int64) {
			if done*3 >= total {
				cancel()
			}
		},
	})
	cancel()
	if err == nil {
		t.Fatal("the paused run reported success")
	}
	if _, err := os.Stat(filepath.Join(dir, d.FileName())); !os.IsNotExist(err) {
		t.Fatal("a paused run must not leave a table under its final name")
	}

	done, total, err := CheckpointProgress(dir, d)
	if err != nil {
		t.Fatalf("no checkpoint after a pause: %v", err)
	}
	if done <= 0 || done >= total {
		t.Fatalf("checkpoint reports %d/%d pairs, expected a partial run", done, total)
	}

	got, err := GenerateWith(context.Background(), dir, d, RunOptions{Workers: 4})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, raw) {
		t.Fatal("the resumed run wrote a different table")
	}
	if _, _, err := CheckpointProgress(dir, d); !errors.Is(err, ErrNoCheckpoint) {
		t.Error("a finished run must clear its checkpoint")
	}
}

// A checkpoint whose size contradicts its header is debris, not a resume
// point: half a table read as whole would produce a wrong one in silence.
func TestReadCheckpoint_ATruncatedFileIsRefused(t *testing.T) {
	t.Parallel()
	d := ts(6, 3)
	dir := t.TempDir()
	st := NewTwoSidedState(d.Points, d.Checkers)
	if err := ComputeTwoSided(context.Background(), st, 1, nil); err != nil {
		t.Fatal(err)
	}
	st.Diagonal = 3
	if err := WriteCheckpoint(dir, st); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadCheckpoint(dir, d); err != nil {
		t.Fatalf("the checkpoint just written does not read back: %v", err)
	}

	path := CheckpointPath(dir, d)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw[:len(raw)-2], 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadCheckpoint(dir, d); err == nil {
		t.Fatal("a truncated checkpoint was accepted")
	}
	// And a refused checkpoint costs the run nothing but time: the generator
	// starts over rather than failing.
	if _, err := GenerateWith(context.Background(), dir, d, RunOptions{Workers: 2}); err != nil {
		t.Fatalf("generation refused to start over from a bad checkpoint: %v", err)
	}
}

// The `.ckpt` name must be invisible to everything that looks for tables.
func TestCheckpoint_IsNotMistakenForATable(t *testing.T) {
	t.Parallel()
	d := ts(6, 3)
	dir := t.TempDir()
	st := NewTwoSidedState(d.Points, d.Checkers)
	st.Diagonal = 2
	if err := WriteCheckpoint(dir, st); err != nil {
		t.Fatal(err)
	}
	if verdict, _, _ := Verify(CheckpointPath(dir, d)); verdict != Corrupt {
		t.Errorf("Verify accepted a checkpoint as a table (%v)", verdict)
	}
	missing := Missing(dir)
	if len(missing) != len(DefaultDomains()) {
		t.Errorf("a checkpoint made a default domain look present: %v", missing)
	}
}
