package openapigen

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestMain chdirs to the repository root: openapi-gen's paths are
// repo-root-relative, but `go test` runs in the package directory.
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

// TestGeneratedFilesAreUpToDate regenerates openapi.yaml,
// doc/source/api_reference.rst and the Python client in memory and compares
// them byte-for-byte with the committed files; on mismatch, run
// `go run ./cmd/openapi-gen`.
func TestGeneratedFilesAreUpToDate(t *testing.T) {
	model, err := Parse("internal/server")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	checkUpToDate(t, "openapi.yaml", GenerateOpenAPI(model))
	checkUpToDate(t, "doc/source/api_reference.rst", GenerateAPIReferenceRST(model))
	// The Python client is generated from the same model; a stale one cannot
	// call the new route.
	checkUpToDate(t, "clients/python/blunderdb/_generated.py", GeneratePythonClient(model))
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
