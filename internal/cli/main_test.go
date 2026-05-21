package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestMain changes the working directory to the repository root before
// running the package tests. The CLI tests reference fixture files under
// testdata/ with paths relative to the repo root, but `go test` runs with
// the package directory as the working directory.
func TestMain(m *testing.M) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot determine test file location")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	if err := os.Chdir(repoRoot); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
