package database

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// A database shared by its producer keeps each comment's signature through the
// desktop import, on a position the recipient already holds (merged) as on a
// new one (copied whole), and the recipient's own name signs none of them.
func TestCommitImportDatabaseKeepsCommentAuthors(t *testing.T) {
	isolateIdentity(t)
	src := newTestDB(t)
	shared := initialPosition()
	sharedID, err := src.SavePosition(&shared)
	if err != nil {
		t.Fatal(err)
	}
	fresh := initialPosition()
	fresh.Board.Points[1].Checkers, fresh.Board.Points[2].Checkers = fresh.Board.Points[2].Checkers, fresh.Board.Points[1].Checkers
	fresh.Board.Points[1].Color, fresh.Board.Points[2].Color = fresh.Board.Points[2].Color, fresh.Board.Points[1].Color
	freshID, err := src.SavePosition(&fresh)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		author, text string
		pos          int64
	}{{"Élodie", "cube too early", sharedID}, {"", "unsigned note", sharedID}, {"Bob", "second row", freshID}, {"Carol", "third row", freshID}} {
		src.SetCommentAuthor(c.author)
		if err := src.AddComment(c.pos, c.text); err != nil {
			t.Fatal(err)
		}
	}
	path := exportTo(t, src, filepath.Join(t.TempDir(), "partage.db"), ExportOptions{
		AllPositions: true, IncludeAnalysis: true, IncludeComments: true,
	})

	dst := newTestDB(t)
	dst.SetCommentAuthor("Importer")
	mine := initialPosition()
	mineID, err := dst.SavePosition(&mine)
	if err != nil {
		t.Fatal(err)
	}
	if err := dst.AddComment(mineID, "my own note"); err != nil {
		t.Fatal(err)
	}
	if _, err := dst.CommitImportDatabase(path); err != nil {
		t.Fatal(err)
	}

	all, err := dst.GetAllComments()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, c := range all {
		got[c.Text] = c.Author
	}
	want := map[string]string{"my own note": "Importer", "cube too early": "Élodie", "unsigned note": "", "second row": "Bob", "third row": "Carol"}
	for text, author := range want {
		if a, ok := got[text]; !ok || a != author {
			t.Errorf("comment %q signed %q (present %v), want %q", text, a, ok, author)
		}
	}
}

// An author the source file records unnormalised comes out of an import —
// desktop and daemon alike — normalised by the comment contract when the note
// is unsigned (trimmed, cut to MaxCommentAuthorLen runes), and verbatim when it
// is signed: the signature covers the author byte for byte, so normalising it
// would leave a genuine note unverified.
func TestImportCarriesCommentAuthorLikeDaemon(t *testing.T) {
	isolateIdentity(t)
	long := strings.Repeat("é", storage.MaxCommentAuthorLen+10)
	for _, c := range []struct {
		name, author, want string
		signed             bool
	}{
		{"unsigned, padded", "  Coach  ", "Coach", false},
		{"unsigned, too long", " " + long, strings.Repeat("é", storage.MaxCommentAuthorLen), false},
		{"signed, padded", "  Coach  ", "  Coach  ", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			src := newTestDB(t)
			pos := initialPosition()
			id, err := src.SavePosition(&pos)
			if err != nil {
				t.Fatal(err)
			}
			const text = "watch the gap"
			if err := src.AddComment(id, text); err != nil {
				t.Fatal(err)
			}
			path := exportTo(t, src, filepath.Join(t.TempDir(), "partage.db"), ExportOptions{
				AllPositions: true, IncludeAnalysis: true, IncludeComments: true,
			})
			raw, err := openExistingSQLite(path)
			if err != nil {
				t.Fatal(err)
			}
			var signature any
			if c.signed {
				var hash int64
				if err := raw.QueryRow(`SELECT zobrist_hash FROM position`).Scan(&hash); err != nil {
					t.Fatal(err)
				}
				coach, err := issuance.NewIdentity("Coach")
				if err != nil {
					t.Fatal(err)
				}
				if signature, err = issuance.SealComment(coach, text, c.author, uint64(hash)); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := raw.Exec(`UPDATE comment SET author = ?, signature = ?`, c.author, signature); err != nil {
				t.Fatal(err)
			}
			raw.Close()

			check := func(mode string, all []*domain.CommentEntry) {
				t.Helper()
				if len(all) != 1 {
					t.Fatalf("%s import: %d comments, want 1", mode, len(all))
				}
				if all[0].Author != c.want {
					t.Errorf("%s import: author %q, want %q", mode, all[0].Author, c.want)
				}
				if verified := all[0].Signer != nil && all[0].Signer.Verified; verified != c.signed {
					t.Errorf("%s import: verified %v, want %v", mode, verified, c.signed)
				}
			}

			dst := newTestDB(t)
			if _, err := dst.CommitImportDatabase(path); err != nil {
				t.Fatal(err)
			}
			desk, err := dst.GetAllComments()
			if err != nil {
				t.Fatal(err)
			}
			var deskEntries []*domain.CommentEntry
			for i := range desk {
				deskEntries = append(deskEntries, &desk[i])
			}
			check("desktop", deskEntries)

			ctx := context.Background()
			daemon, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "daemon.db"), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer daemon.Close()
			if _, err := (ingest.DBImporter{S: daemon}).Import(ctx, "", ingest.Source{Format: ingest.FormatNativeDB, Path: path}, nil); err != nil {
				t.Fatal(err)
			}
			var served []*domain.CommentEntry
			for e, err := range daemon.Comments().ListAll(ctx, "", storage.ListOpts{}) {
				if err != nil {
					t.Fatal(err)
				}
				served = append(served, e)
			}
			check("daemon", served)
		})
	}
}
