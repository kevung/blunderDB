// Package bearofftest hands tests a real bearoff table.
//
// The tables left the binary with ADR-0027, and with them the guarantee that a
// test could just ask the engine for one.
//
// The two default tables are already in the repository, as the fixtures the
// generator is verified against (bearoffgen/testdata) — they are the reference
// gnubg produced, and the identity tests would be meaningless without them.
// This copies from there, and only generates when a test asks for a domain no
// fixture covers.
//
// Copying rather than generating is what keeps the suite affordable: under
// `-race`, generating OS-06 takes minutes, and the CI job that runs
// internal/gui with the race detector timed out at twenty minutes because of
// it. Reading 8 MB from disk takes milliseconds, race detector or not.
//
// Whatever it hands back is validated with the generator's own fingerprint
// check, so a truncated file — a half-written cache entry from an interrupted
// run — is replaced rather than read.
//
// The cache is shared by every package that asks, and `go test ./...` runs
// those packages as CONCURRENT PROCESSES: the mutex below orders one process,
// nothing orders two. So an entry is always written to a unique temporary file
// and renamed into place, and a rename that loses the race is not an error —
// the winner's file is read instead. Writing in place made internal/gui and
// internal/server read each other's half-written table, and on Windows fail
// outright on the sharing violation.
package bearofftest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen"
)

var mu sync.Mutex

// cacheDir is where test tables live between runs. Deliberately under TempDir:
// it is a cache, and losing it costs a copy.
func cacheDir() string {
	return filepath.Join(os.TempDir(), "blunderdb-bearoff-test")
}

// fixtureDir locates bearoffgen/testdata from this file's own path: a test
// binary runs in its own package's directory, so a relative path would depend
// on which package is asking.
func fixtureDir() string {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Join(filepath.Dir(filepath.Dir(self)), "testdata")
}

// fixtureFor returns the repository fixture for a domain, or "" when there is
// none. The two default domains have one; a wider domain is generated.
func fixtureFor(domain bearoffgen.Domain) string {
	names := map[bearoffgen.Domain]string{
		{Kind: bearoffgen.TwoSidedKind, Points: 6, Checkers: 6}:  "gnubg_ts0.bd",
		{Kind: bearoffgen.OneSidedKind, Points: 6, Checkers: 15}: "gnubg_os6.bd",
	}
	name, ok := names[domain]
	if !ok {
		return ""
	}
	dir := fixtureDir()
	if dir == "" {
		return ""
	}
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}

// valid reports whether `path` already holds the table for `domain`.
func valid(path string, domain bearoffgen.Domain) bool {
	verdict, got, err := bearoffgen.Verify(path)
	return err == nil && got == domain && verdict != bearoffgen.Corrupt
}

// publish writes `raw` at `path` through a unique temporary file. A rename
// that fails while another process holds the destination is not fatal: what
// matters is that the destination is valid afterwards.
func publish(dir, path string, raw []byte, domain bearoffgen.Domain) error {
	tmp, err := os.CreateTemp(dir, "partial-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		if valid(path, domain) {
			return nil
		}
		return err
	}
	return nil
}

// Ensure returns the path of a table for `domain`, making it if the cache does
// not already hold a valid one. It is the error-returning core: a TestMain has
// no *testing.T to fail, and a zero-value one calls runtime.Goexit on the main
// goroutine — which the runtime reports as a deadlock, with the real cause
// nowhere in sight.
func Ensure(domain bearoffgen.Domain) (string, error) {
	mu.Lock()
	defer mu.Unlock()

	dir := cacheDir()
	path := filepath.Join(dir, domain.FileName())
	if valid(path, domain) {
		return path, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	// The repository fixture, when there is one: a copy costs milliseconds
	// where generating costs minutes under -race.
	if fixture := fixtureFor(domain); fixture != "" {
		raw, err := os.ReadFile(fixture)
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", fixture, err)
		}
		if err := publish(dir, path, raw, domain); err != nil {
			return "", fmt.Errorf("writing %s: %w", path, err)
		}
		return path, nil
	}

	// No fixture: generate into a directory of this run's own, then publish.
	scratch, err := os.MkdirTemp(dir, "gen-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	made, err := bearoffgen.Generate(context.Background(), scratch, domain, nil)
	if err != nil {
		return "", fmt.Errorf("generating %s: %w", domain, err)
	}
	raw, err := os.ReadFile(made)
	if err != nil {
		return "", err
	}
	if err := publish(dir, path, raw, domain); err != nil {
		return "", fmt.Errorf("writing %s: %w", path, err)
	}
	return path, nil
}

// EnsureOneSided is Ensure for the OS-06 table the EPC reads — what a TestMain
// needs, without a *testing.T.
func EnsureOneSided() (string, error) {
	return Ensure(bearoffgen.Domain{Kind: bearoffgen.OneSidedKind, Points: 6, Checkers: 15})
}

// EnsureDataDir is Ensure for both default tables, returning the directory
// holding them.
func EnsureDataDir() (string, error) {
	if _, err := Ensure(bearoffgen.Domain{Kind: bearoffgen.TwoSidedKind, Points: 6, Checkers: 6}); err != nil {
		return "", err
	}
	if _, err := EnsureOneSided(); err != nil {
		return "", err
	}
	return cacheDir(), nil
}

// Path returns the path of a table for `domain`, making it if needed. It never
// returns an invalid path: a failure fails the test.
func Path(t *testing.T, domain bearoffgen.Domain) string {
	t.Helper()
	path, err := Ensure(domain)
	if err != nil {
		t.Fatalf("bearofftest: %v", err)
	}
	return path
}

// TwoSidedPath is the TS-06-06 table, the one that used to be embedded.
func TwoSidedPath(t *testing.T) string {
	t.Helper()
	return Path(t, bearoffgen.Domain{Kind: bearoffgen.TwoSidedKind, Points: 6, Checkers: 6})
}

// OneSidedPath is the OS-06 table the EPC reads.
func OneSidedPath(t *testing.T) string {
	t.Helper()
	return Path(t, bearoffgen.Domain{Kind: bearoffgen.OneSidedKind, Points: 6, Checkers: 15})
}

// DataDir makes both default tables and returns the directory holding them —
// what a test needs when it exercises the resolution rather than one table.
func DataDir(t *testing.T) string {
	t.Helper()
	dir, err := EnsureDataDir()
	if err != nil {
		t.Fatalf("bearofftest: %v", err)
	}
	return dir
}
