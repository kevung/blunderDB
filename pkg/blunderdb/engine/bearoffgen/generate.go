package bearoffgen

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Generate writes the table for `domain` into dir, and returns its path.
//
// The work happens in a `.part` file which becomes the real name only once the
// table is complete and has been checked against its recorded fingerprint. A
// run that is cancelled, or that dies with the machine, therefore leaves a
// `.part` behind and never a truncated table that would later be read as
// authoritative — Resolve skips `.part` for exactly that reason.
//
// A domain with no recorded fingerprint is written all the same: Verify
// answers Unverified, which is a statement about what is known, not a defect.
func Generate(ctx context.Context, dir string, domain Domain, progress func(done, total int64)) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("bearoffgen: create %s: %w", dir, err)
	}
	final := filepath.Join(dir, domain.FileName())
	part := final + ".part"

	f, err := os.Create(part)
	if err != nil {
		return "", fmt.Errorf("bearoffgen: create %s: %w", part, err)
	}
	// Any early return leaves the .part in place: it is the trace of an
	// interrupted run, and the caller may want to see it.
	w := bufio.NewWriterSize(f, 1<<20)

	switch domain.Kind {
	case TwoSidedKind:
		err = TwoSided(ctx, w, domain.Points, domain.Checkers, progress)
	case OneSidedKind:
		err = OneSided(ctx, w, domain.Points, progress)
	default:
		err = fmt.Errorf("bearoffgen: unknown table kind %v", domain.Kind)
	}
	if err != nil {
		f.Close()
		return "", err
	}
	if err := w.Flush(); err != nil {
		f.Close()
		return "", fmt.Errorf("bearoffgen: write %s: %w", part, err)
	}
	// Sync before the rename: a table that appears under its final name after
	// a power cut, with its tail still in the page cache, would verify as
	// corrupt on the next launch and cost the user the whole run again.
	if err := f.Sync(); err != nil {
		f.Close()
		return "", fmt.Errorf("bearoffgen: sync %s: %w", part, err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("bearoffgen: close %s: %w", part, err)
	}

	verdict, _, err := Verify(part)
	if verdict == Corrupt {
		return "", fmt.Errorf("bearoffgen: the table just generated does not verify: %w", err)
	}
	if err := os.Rename(part, final); err != nil {
		return "", fmt.Errorf("bearoffgen: rename %s: %w", part, err)
	}
	return final, nil
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
