package ingest

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestMapMatchSourceLooksUnderTheLibraryDirectory(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "..", "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MapMatchSource("test.xg", t.TempDir(), dir); err != nil {
		t.Errorf("relative path under the second directory: %v", err)
	}
	for _, p := range []string{"", "missing.xg", filepath.Join(dir, "missing.xg")} {
		if _, err := MapMatchSource(p, dir); !errors.Is(err, ErrRepairNoSource) {
			t.Errorf("%q: err = %v, want ErrRepairNoSource", p, err)
		}
	}
}
