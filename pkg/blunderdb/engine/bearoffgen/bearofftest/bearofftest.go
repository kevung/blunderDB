// Package bearofftest hands tests a real bearoff table.
//
// The tables left the binary with ADR-0027, and with them the guarantee that a
// test could just ask the engine for one. Rather than commit a fixture — the
// 8.2 MB that were taken out of the binary would have come back into the
// repository — this generates what a test needs, and caches it under the
// system temporary directory so `go test ./...` pays for it once rather than
// once per package.
//
// The cache is keyed by the domain and validated with the generator's own
// fingerprint check, so a truncated or half-written file from an interrupted
// run is regenerated rather than read.
package bearofftest

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen"
)

var mu sync.Mutex

// cacheDir is where generated test tables live between runs. Deliberately
// under TempDir: it is a cache, and losing it costs six seconds.
func cacheDir() string {
	return filepath.Join(os.TempDir(), "blunderdb-bearoff-test")
}

// Path returns the path of a generated table for `domain`, making it if the
// cache does not already hold a valid one. It never returns an invalid path:
// a failure fails the test.
func Path(t *testing.T, domain bearoffgen.Domain) string {
	t.Helper()
	mu.Lock()
	defer mu.Unlock()

	dir := cacheDir()
	path := filepath.Join(dir, domain.FileName())
	if verdict, got, err := bearoffgen.Verify(path); err == nil && got == domain && verdict != bearoffgen.Corrupt {
		return path
	}
	made, err := bearoffgen.Generate(context.Background(), dir, domain, nil)
	if err != nil {
		t.Fatalf("bearofftest: generating %s: %v", domain, err)
	}
	return made
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

// DataDir generates both default tables into one directory and returns it —
// what a test needs when it exercises the resolution rather than one table.
func DataDir(t *testing.T) string {
	t.Helper()
	TwoSidedPath(t)
	OneSidedPath(t)
	return cacheDir()
}
