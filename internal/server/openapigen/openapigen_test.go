package openapigen

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestMain changes the working directory to the repository root before
// running the package tests: cmd/openapi-gen's default output paths
// (openapi.yaml, doc/source/api_reference.rst) and Parse's own default
// input directory (internal/server) are all repo-root-relative, matching
// how a developer actually runs `go run ./cmd/openapi-gen` from the repo
// root — but `go test` runs with the package directory as the working
// directory (see internal/cli/main_test.go for the same pattern).
func TestMain(m *testing.M) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot determine test file location")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	if err := os.Chdir(repoRoot); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// TestGeneratedFilesAreUpToDate is the non-drift guard cmd/openapi-gen's
// -check flag also runs in CI: it regenerates openapi.yaml and
// doc/source/api_reference.rst in memory from the current handlers_*.go
// source and compares the result byte-for-byte against the committed
// files. A mismatch means someone added, removed or reshaped a /v1 route
// (or changed a Req/Resp struct's fields) without running
// `go run ./cmd/openapi-gen` and committing the result — this test's
// failure message says exactly that.
func TestGeneratedFilesAreUpToDate(t *testing.T) {
	model, err := Parse("internal/server")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	checkUpToDate(t, "openapi.yaml", GenerateOpenAPI(model))
	checkUpToDate(t, "doc/source/api_reference.rst", GenerateAPIReferenceRST(model))
}

func checkUpToDate(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading committed %s: %v (run `go run ./cmd/openapi-gen` and commit it)", path, err)
	}
	if string(got) != want {
		t.Errorf("%s is stale: run `go run ./cmd/openapi-gen` and commit the result", path)
	}
}
