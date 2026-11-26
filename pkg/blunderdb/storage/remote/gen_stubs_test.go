package remote

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The stubs are generated from the storage interfaces; a method added there
// without regenerating leaves the remote backend short of the contract.
func TestUnsupportedGenIsCurrent(t *testing.T) {
	out := filepath.Join(t.TempDir(), "unsupported_gen.go")
	cmd := exec.Command("go", "run", "gen_stubs.go")
	cmd.Env = append(os.Environ(), "GEN_STUBS_OUT="+out)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go run gen_stubs.go: %v\n%s", err, b)
	}
	want, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile("unsupported_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatal("unsupported_gen.go is stale: run go generate ./pkg/blunderdb/storage/remote/")
	}
}
