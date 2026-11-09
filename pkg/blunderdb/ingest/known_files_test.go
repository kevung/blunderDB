package ingest

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func importBatch(t *testing.T, ctx context.Context, s *sqlite.Storage, paths []string, known func(string) (int64, bool)) []FileOutcome {
	t.Helper()
	id, err := s.ImportBatches().Begin(ctx, "", "corpus", "xg")
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	out, err := ImportFiles(ctx, s, paths, PipelineOptions{Workers: 1, ImportBatchID: id, Known: known})
	if err != nil {
		t.Fatalf("ImportFiles: %v", err)
	}
	return out
}

// A file whose bytes an earlier batch imported is recognised from the journal
// alone: it is a duplicate of the same match, never decoded (its outcome names
// no player, which a decoded duplicate does). One byte more, and it is decoded
// and deduplicated by its content as it always was.
func TestKnownFiles_SameBytesAreNotDecodedAgain(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	dir := t.TempDir()
	src, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "match_with_comment.xg"))
	if err != nil {
		t.Fatal(err)
	}
	orig := filepath.Join(dir, "a.xg")
	same := filepath.Join(dir, "same.xg")
	changed := filepath.Join(dir, "changed.xg")
	for p, b := range map[string][]byte{orig: src, same: src, changed: append(append([]byte{}, src...), 0)} {
		if err := os.WriteFile(p, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	first := importBatch(t, ctx, s, []string{orig}, nil)
	if first[0].Status != FileImported || first[0].MatchID == 0 {
		t.Fatalf("first import: %+v", first[0])
	}

	var asked, hits atomic.Int64
	known := func(sha string) (int64, bool) {
		asked.Add(1)
		id, ok := KnownFiles(ctx, s.ImportBatches(), "", nil)(sha)
		if ok {
			hits.Add(1)
		}
		return id, ok
	}
	out := importBatch(t, ctx, s, []string{same, changed}, known)

	if out[0].Status != FileDuplicate || out[0].MatchID != first[0].MatchID {
		t.Errorf("same bytes: got %+v, want a duplicate of match %d", out[0], first[0].MatchID)
	}
	if out[0].Player1 != "" || out[0].Games != 0 {
		t.Errorf("same bytes were decoded: %+v", out[0])
	}
	if asked.Load() != 2 || hits.Load() != 1 {
		t.Errorf("asked %d digests and knew %d, want 2 and 1", asked.Load(), hits.Load())
	}
	if out[1].SHA256 == out[0].SHA256 {
		t.Fatal("the modified copy has the original digest")
	}
	if out[1].Status == FileImported || out[1].Positions != 0 {
		t.Errorf("the modified copy wrote a second match: %+v", out[1])
	}
	if out[1].Status == FileFailed {
		return
	}
	if out[1].MatchID != first[0].MatchID {
		t.Errorf("the modified copy decoded to match %d, want %d", out[1].MatchID, first[0].MatchID)
	}
}

// A match deleted since unlinks its journal lines: its file imports again.
func TestKnownFiles_DeletedMatchImportsAgain(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	p := filepath.Join("..", "..", "..", "testdata", "match_with_comment.xg")

	first := importBatch(t, ctx, s, []string{p}, nil)
	if _, ok, _ := s.ImportBatches().KnownFile(ctx, "", first[0].SHA256); !ok {
		t.Fatal("the imported file is not known")
	}
	if err := s.Matches().DeleteCascade(ctx, "", first[0].MatchID); err != nil {
		t.Fatalf("delete match: %v", err)
	}
	if id, ok, _ := s.ImportBatches().KnownFile(ctx, "", first[0].SHA256); ok {
		t.Fatalf("a deleted match is still known as %d", id)
	}
	again := importBatch(t, ctx, s, []string{p}, KnownFiles(ctx, s.ImportBatches(), "", nil))
	if again[0].Status != FileImported {
		t.Errorf("after deletion: got %+v, want a new import", again[0])
	}
}

// What a re-imported file costs before and after: decoding it to find its
// fingerprint, against its digest (the whole cost of a file the journal knows).
func BenchmarkReimportDecode(b *testing.B) {
	p := filepath.Join("..", "..", "..", "testdata", "HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg")
	for i := 0; i < b.N; i++ {
		if _, _, err := mapFile(p); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReimportDigest(b *testing.B) {
	p := filepath.Join("..", "..", "..", "testdata", "HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg")
	for i := 0; i < b.N; i++ {
		if _, _, err := digestFile(p); err != nil {
			b.Fatal(err)
		}
	}
}
