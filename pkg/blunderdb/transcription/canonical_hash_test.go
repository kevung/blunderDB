package transcription

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// TestTranscriptCanonicalHashMatchesImport holds the promise of the canonical
// hash across the two ways a match enters the library: a match imported from a
// file, then reopened as a transcription the way EditMatch does it (stored,
// rendered to .mat, parsed back) and finished, hashes canonically like the
// file. Otherwise a later import of the same match would add a second copy of a
// transcribed one instead of enriching it.
func TestTranscriptCanonicalHashMatchesImport(t *testing.T) {
	root := filepath.Join("..", "..", "..", "testdata")
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "fuzz" {
			return filepath.SkipDir
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".xg", ".sgf", ".mat", ".bgf":
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no match fixture found under testdata")
	}

	for _, path := range files {
		rel, _ := filepath.Rel(root, path)
		t.Run(filepath.ToSlash(rel), func(t *testing.T) {
			var g *ingest.MatchGraph
			var err error
			switch strings.ToLower(filepath.Ext(path)) {
			case ".xg":
				g, err = ingest.MapXG(path)
			case ".bgf":
				g, err = ingest.MapBGF(path)
			default:
				g, err = ingest.MapGnuBG(path)
			}
			if err != nil {
				t.Skipf("not a match fixture: %v", err)
			}
			want := g.Match.CanonicalHash

			ctx := context.Background()
			st := newStore(t)
			tx, err := st.BeginTx(ctx)
			if err != nil {
				t.Fatal(err)
			}
			res, err := ingest.WriteMatch(ctx, tx, "1", g, nil)
			if err != nil {
				_ = tx.Rollback()
				t.Fatal(err)
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}

			m, games, moves, err := ingest.ReadMatchForMAT(ctx, st, "1", res.MatchID)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := transcript.FromMAT(ingest.RenderMAT(m, games, moves))
			if err != nil {
				t.Fatalf("FromMAT: %v", err)
			}
			_, got := MatchHashes(transcript.Build(doc))
			if got != want {
				t.Errorf("canonical hash: transcription %s, import %s", got, want)
			}
		})
	}
}
