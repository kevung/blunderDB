package ingest

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/xgparser/xgparser"
)

// Each file a batch refuses says why in a code a client can act on, its text
// kept beside it.
func TestImportFailureCarriesItsCode(t *testing.T) {
	dir := t.TempDir()
	xg, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "match_with_comment.xg"))
	if err != nil {
		t.Fatal(err)
	}
	write := func(name string, data []byte) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	cases := []struct {
		name, path, want string
	}{
		{"unknown format", write("a.pdf", []byte("not a match")), domain.ImportErrUnknownFormat},
		{"truncated", write("b.xg", xg[:len(xg)/2]), domain.ImportErrTruncated},
		{"corrupt", write("c.xg", []byte("this is no XG container at all, not even close")), domain.ImportErrCorrupt},
		{"empty", write("d.mat", nil), domain.ImportErrEmpty},
	}
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := ImportFiles(ctx, s, []string{c.path}, PipelineOptions{Workers: 1})
			if err != nil {
				t.Fatal(err)
			}
			if out[0].Status != FileFailed || out[0].Code != c.want {
				t.Errorf("status %q code %q (%s), want failed %q", out[0].Status, out[0].Code, out[0].Error, c.want)
			}
			if out[0].Error == "" {
				t.Error("the text must stay beside the code")
			}
		})
	}
}

func TestImportFailureCodeTooLarge(t *testing.T) {
	saved := xgparser.MaxDecompressedSize
	xgparser.MaxDecompressedSize = 1_200_000
	t.Cleanup(func() { xgparser.MaxDecompressedSize = saved })
	_, err := MapXG(filepath.Join("..", "..", "..", "testdata", "test.xg"))
	if got := ImportErrorCode(err); got != domain.ImportErrTooLarge {
		t.Errorf("code %q (%v), want %q", got, err, domain.ImportErrTooLarge)
	}
}

func TestImportFailureCodeInternal(t *testing.T) {
	if got := ImportErrorCode(errors.New("connection reset")); got != domain.ImportErrInternal {
		t.Errorf("code %q, want %q", got, domain.ImportErrInternal)
	}
}

// A second copy of a refused file carries the code of the first, not a generic one.
func TestImportFailureCopyKeepsItsCode(t *testing.T) {
	dir := t.TempDir()
	body := []byte("this is no XG container at all, not even close")
	a, b := filepath.Join(dir, "a.xg"), filepath.Join(dir, "b.xg")
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	out, err := ImportFiles(ctx, s, []string{a, b}, PipelineOptions{Workers: 1})
	if err != nil {
		t.Fatal(err)
	}
	if out[1].Code != domain.ImportErrCorrupt {
		t.Errorf("copy code %q, want %q", out[1].Code, domain.ImportErrCorrupt)
	}
}
