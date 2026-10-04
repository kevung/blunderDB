package ingest

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/xgparser/xgparser"
)

// A file whose segment inflates past xgparser's cap fails on its own, with a
// message naming it, and the rest of the batch is imported. The cap is
// lowered between the fixtures' largest segments (test.xg ~1.7 MB,
// match_with_comment.xg ~0.98 MB) so a real file plays the oversized one.
func TestImportFiles_DecompressionLimitFailsOnlyThatFile(t *testing.T) {
	saved := xgparser.MaxDecompressedSize
	xgparser.MaxDecompressedSize = 1_200_000
	t.Cleanup(func() { xgparser.MaxDecompressedSize = saved })

	big := filepath.Join("..", "..", "..", "testdata", "test.xg")
	small := filepath.Join("..", "..", "..", "testdata", "match_with_comment.xg")

	if _, err := MapXG(big); !errors.Is(err, xgparser.ErrDecompressionLimit) {
		t.Fatalf("MapXG(oversized) = %v, want ErrDecompressionLimit", err)
	}

	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	defer s.Close()

	out, err := ImportFiles(ctx, s, []string{big, small}, PipelineOptions{Workers: 1})
	if err != nil {
		t.Fatalf("ImportFiles: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("outcomes = %d, want 2", len(out))
	}
	if out[0].Status != FileFailed {
		t.Fatalf("oversized file status = %q, want %q", out[0].Status, FileFailed)
	}
	for _, want := range []string{"test.xg", "fichier trop gros ou corrompu"} {
		if !strings.Contains(out[0].Error, want) {
			t.Errorf("error %q does not mention %q", out[0].Error, want)
		}
	}
	if out[1].Status != FileImported {
		t.Fatalf("next file status = %q (%s), want %q", out[1].Status, out[1].Error, FileImported)
	}
}
