package ingest

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func TestMapXG_TruncatedIsInvalid(t *testing.T) {
	for _, name := range []string{"test.xg", "charlot1-charlot2_7p_2025-11-08-2305.xg"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		for _, cut := range []int{1, 2, 40, 100, 1000, 4096, len(data) / 4, len(data) / 2, len(data) - 24056%len(data)} {
			if cut <= 0 || cut >= len(data) {
				continue
			}
			p := filepath.Join(t.TempDir(), "t.xg")
			if err := os.WriteFile(p, data[:len(data)-cut], 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := MapXG(p); err != nil && !errors.Is(err, storage.ErrInvalid) {
				t.Errorf("%s cut %d: not invalid: %v", name, cut, err)
			}
		}
		for _, size := range []int{0, 1, 10, 24056, 24057} {
			if size >= len(data) {
				continue
			}
			p := filepath.Join(t.TempDir(), "t.xg")
			_ = os.WriteFile(p, data[:size], 0o600)
			if _, err := MapXG(p); err == nil || !errors.Is(err, storage.ErrInvalid) {
				t.Errorf("%s size %d: want invalid, got %v", name, size, err)
			}
		}
	}
}
