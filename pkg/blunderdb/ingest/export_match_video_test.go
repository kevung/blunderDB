package ingest

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// An export carries a match's video source only as an http(s) URL: a local
// path stays with its author, it reveals a directory tree and opens nothing
// on the recipient's machine. The Repères travel whatever the source
// (ADR-0082).
func TestExportCarriesVideoSourceOnlyAsURL(t *testing.T) {
	str := func(s string) *string { return &s }
	roll, done := int64(61_250), int64(64_800)
	cases := []struct {
		name       string
		source     *string
		wantSource *string
	}{
		{"https URL", str("https://example.org/finale.mp4"), str("https://example.org/finale.mp4")},
		{"http URL", str("http://192.168.1.2/m.webm"), str("http://192.168.1.2/m.webm")},
		{"upper-case scheme", str("HTTPS://example.org/m.mp4"), str("HTTPS://example.org/m.mp4")},
		{"URL with surrounding blanks", str("  https://example.org/m.mp4\n"), str("https://example.org/m.mp4")},
		{"blank", str("   "), nil},
		{"local path", str("/home/alice/videos/finale.mkv"), nil},
		{"Windows path", str(`C:\Users\alice\finale.mp4`), nil},
		{"file URL", str("file:///home/alice/finale.mp4"), nil},
		{"no video", nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			src, err := sqlite.Open(ctx, ":memory:", nil)
			if err != nil {
				t.Fatal(err)
			}
			defer src.Close()
			matchID, err := src.Matches().Save(ctx, "", &domain.Match{Player1Name: "A", Player2Name: "B",
				MatchLength: 5, VideoSource: c.source})
			if err != nil {
				t.Fatal(err)
			}
			gameID, err := src.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := src.Matches().CreateMove(ctx, "", &domain.Move{GameID: gameID, MoveNumber: 1,
				MoveType: "checker", Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5",
				RollTickMS: &roll, TickMS: &done}); err != nil {
				t.Fatal(err)
			}
			outPath := filepath.Join(t.TempDir(), "export.sqlite")
			out, err := os.Create(outPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := (SQLiteExporter{S: src}).Export(ctx, "", out, WholeTenant(FormatSQLite)); err != nil {
				t.Fatalf("export: %v", err)
			}
			_ = out.Close()

			dst, err := sqlite.Open(ctx, outPath, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer dst.Close()
			var got *domain.Match
			for mm, err := range dst.Matches().List(ctx, "", storage.MatchListOpts{}) {
				if err != nil {
					t.Fatal(err)
				}
				got = mm
			}
			if got == nil {
				t.Fatal("export holds no match")
			}
			if (got.VideoSource == nil) != (c.wantSource == nil) ||
				(got.VideoSource != nil && *got.VideoSource != *c.wantSource) {
				t.Errorf("exported video source = %v, want %v", deref(got.VideoSource), deref(c.wantSource))
			}
			n := 0
			for mv, err := range dst.Matches().MovesByMatch(ctx, "", got.ID) {
				if err != nil {
					t.Fatal(err)
				}
				n++
				if mv.RollTickMS == nil || *mv.RollTickMS != roll || mv.TickMS == nil || *mv.TickMS != done {
					t.Errorf("exported Repères = %v/%v, want %d/%d", mv.RollTickMS, mv.TickMS, roll, done)
				}
			}
			if n != 1 {
				t.Fatalf("export holds %d moves, want 1", n)
			}
		})
	}
}

func deref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
