package bearoffgen

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// RunOptions tunes one generation run.
type RunOptions struct {
	// Workers is how many cores the sweep may use; 0 means all of them.
	Workers int
	// Progress, when non-nil, is called with the pairs done and the total.
	Progress func(done, total int64)
	// Pausable makes a cancellation write a checkpoint rather than throw the
	// work away, so a later run resumes where this one stopped. Only the
	// two-sided sweep is resumable — the one-sided table takes seconds, and a
	// checkpoint for it would cost more than recomputing it.
	Pausable bool
}

// Generate writes the table for `domain` into dir, and returns its path. It
// uses every core and does not checkpoint; GenerateWith is the form that
// takes a core count and can be paused.
func Generate(ctx context.Context, dir string, domain Domain, progress func(done, total int64)) (string, error) {
	return GenerateWith(ctx, dir, domain, RunOptions{Progress: progress})
}

// GenerateWith writes the table for `domain` into dir, and returns its path.
//
// The finished table is written to a `.part` file which becomes the real name
// only once it has been checked against its recorded fingerprint. A run that
// dies with the machine therefore leaves a `.part` behind and never a
// truncated table that would later be read as authoritative — Resolve skips
// `.part` for exactly that reason.
//
// With opts.Pausable, a cancelled two-sided run leaves a `.ckpt` instead: the
// sweep's state, which the next call picks up. A run that finishes clears it.
//
// A domain with no recorded fingerprint is written all the same: Verify
// answers Unverified, which is a statement about what is known, not a defect.
func GenerateWith(ctx context.Context, dir string, domain Domain, opts RunOptions) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("bearoffgen: create %s: %w", dir, err)
	}
	final := filepath.Join(dir, domain.FileName())
	part := final + ".part"

	// The two-sided sweep computes into memory, so the resumable path can hand
	// its state to a checkpoint. The one-sided one streams and stays as it was.
	if domain.Kind == TwoSidedKind {
		st, err := ReadCheckpoint(dir, domain)
		if err != nil {
			// No checkpoint, or one that contradicts itself: start over. A
			// discarded checkpoint is not an error the caller can act on.
			st = NewTwoSidedState(domain.Points, domain.Checkers)
		}
		if err := ComputeTwoSided(ctx, st, opts.Workers, opts.Progress); err != nil {
			if opts.Pausable {
				if werr := WriteCheckpoint(dir, st); werr != nil {
					return "", fmt.Errorf("bearoffgen: writing the checkpoint of %s: %w", domain, werr)
				}
			}
			return "", err
		}
		f, err := os.Create(part)
		if err != nil {
			return "", fmt.Errorf("bearoffgen: create %s: %w", part, err)
		}
		w := bufio.NewWriterSize(f, 1<<20)
		if err := WriteTwoSided(w, st); err != nil {
			f.Close()
			return "", err
		}
		if err := finish(f, w, part, final); err != nil {
			return "", err
		}
		if err := RemoveCheckpoint(dir, domain); err != nil {
			return "", err
		}
		return final, nil
	}

	f, err := os.Create(part)
	if err != nil {
		return "", fmt.Errorf("bearoffgen: create %s: %w", part, err)
	}
	// Any early return leaves the .part in place: it is the trace of an
	// interrupted run, and the caller may want to see it.
	w := bufio.NewWriterSize(f, 1<<20)
	if err := OneSided(ctx, w, domain.Points, opts.Progress); err != nil {
		f.Close()
		return "", err
	}
	if err := finish(f, w, part, final); err != nil {
		return "", err
	}
	return final, nil
}

// finish flushes, syncs, verifies and renames a completed `.part`.
func finish(f *os.File, w *bufio.Writer, part, final string) error {
	if err := w.Flush(); err != nil {
		f.Close()
		return fmt.Errorf("bearoffgen: write %s: %w", part, err)
	}
	// Sync before the rename: a table that appears under its final name after
	// a power cut, with its tail still in the page cache, would verify as
	// corrupt on the next launch and cost the user the whole run again.
	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("bearoffgen: sync %s: %w", part, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("bearoffgen: close %s: %w", part, err)
	}
	verdict, _, err := Verify(part)
	if verdict == Corrupt {
		return fmt.Errorf("bearoffgen: the table just generated does not verify: %w", err)
	}
	if err := os.Rename(part, final); err != nil {
		return fmt.Errorf("bearoffgen: rename %s: %w", part, err)
	}
	return nil
}

// ErrAlreadyPresent is returned by EnsureDefaults for a table that is already
// there — not a failure, and not silent either.
var ErrAlreadyPresent = errors.New("bearoffgen: already present")

// DefaultDomains are the two tables blunderDB used to embed: the exact
// two-sided table over six points and six chequers, and the one-sided table
// the EPC reads. Together they are the 8.2 MB that ADR-0027 takes out of the
// binary, and they cost about six seconds to make.
func DefaultDomains() []Domain {
	return []Domain{
		{Kind: TwoSidedKind, Points: 6, Checkers: 6},
		{Kind: OneSidedKind, Points: 6, Checkers: osCheckers},
	}
}

// Missing returns the default domains that dir does not already hold. A file
// that is present but corrupt counts as missing: it will be overwritten by a
// fresh run, which is better than reading it.
func Missing(dir string) []Domain {
	var out []Domain
	for _, d := range DefaultDomains() {
		path := filepath.Join(dir, d.FileName())
		if verdict, got, err := Verify(path); err == nil && got == d && verdict != Corrupt {
			continue
		}
		out = append(out, d)
	}
	return out
}
