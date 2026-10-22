package issuancetest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGuardFailsWhenRealDirChanges(t *testing.T) {
	real := filepath.Join(t.TempDir(), "blunderDB")
	if code := runGuarded(real, func() int { return 0 }); code != 0 {
		t.Fatalf("untouched dir: code %d", code)
	}
	code := runGuarded(real, func() int {
		if err := os.MkdirAll(real, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(real, "identity.json"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		return 0
	})
	if code == 0 {
		t.Fatal("guard did not fail after the real dir changed")
	}
}

func TestGuardKeepsTestFailureCode(t *testing.T) {
	if code := runGuarded(filepath.Join(t.TempDir(), "x"), func() int { return 3 }); code != 3 {
		t.Fatalf("code = %d, want 3", code)
	}
}
