package storagetest

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// testCommentSignatureRoundTrip: an export signed by its producer seals the
// user's notes, not an importer's; the recipient's import keeps each
// signature, verified, and each origin; a note altered or moved to another
// board in the file arrives flagged unverified, not refused; the recipient's
// edit takes the signature away; and a file from before the signature column
// imports with its comments unsigned (ADR-0099).
func testCommentSignatureRoundTrip(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	coach, err := issuance.NewIdentity("Coach Martin")
	if err != nil {
		t.Fatal(err)
	}
	p := provenancePos(11)
	id, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := s.Comments().Add(storage.WithCommentAuthor(ctx, "Martin"), "", id, "Joue 8/4 6/4."); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Comments().Add(ctx, "", id, "Le pip count décide."); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Comments().AddFrom(ctx, "", id, "XG: -0.120", domain.CommentOriginXG); err != nil {
		t.Fatal(err)
	}
	other := provenancePos(12)
	otherID, err := s.Positions().Save(ctx, "", &other)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := s.Comments().Add(ctx, "", otherID, "Ailleurs."); err != nil {
		t.Fatal(err)
	}

	opts := ingest.WholeTenant(ingest.FormatSQLite)
	opts.Signer = coach
	// Unwatermarked, the user's three notes leave unsigned, and the report says so.
	plain := ingest.WholeTenant(ingest.FormatSQLite)
	rep, err := ingest.ExportSQLite(ctx, s, "", filepath.Join(t.TempDir(), "plain.db"), plain)
	if err != nil {
		t.Fatalf("ExportSQLite (no watermark): %v", err)
	}
	if rep.SignedComments != 0 || rep.UnsignedOwnComments != 3 {
		t.Errorf("export without a watermark: signed %d, unsigned own %d; want 0 and 3", rep.SignedComments, rep.UnsignedOwnComments)
	}

	path := filepath.Join(t.TempDir(), "eleve.db")
	rep, err = ingest.ExportSQLite(ctx, s, "", path, opts)
	if err != nil {
		t.Fatalf("ExportSQLite: %v", err)
	}
	if rep.SignedComments != 3 || rep.UnsignedOwnComments != 0 {
		t.Errorf("watermarked export: signed %d, unsigned own %d; want 3 and 0", rep.SignedComments, rep.UnsignedOwnComments)
	}

	importInto := func(file string) map[string]*domain.CommentEntry {
		t.Helper()
		target, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "recipient.db"), nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { target.Close() })
		if _, err := (ingest.DBImporter{S: target}).Import(ctx, "", ingest.Source{Format: ingest.FormatNativeDB, Path: file}, nil); err != nil {
			t.Fatalf("Import: %v", err)
		}
		out := map[string]*domain.CommentEntry{}
		for c, err := range target.Comments().ListAll(ctx, "", storage.ListOpts{}) {
			if err != nil {
				t.Fatal(err)
			}
			out[c.Text] = c
		}
		if c := out["Le pip count décide."]; c != nil {
			// The recipient's edit makes the note theirs: the coach's seal goes.
			if err := target.Comments().Update(ctx, "", c.ID, "Le pip count décide, je crois."); err != nil {
				t.Fatal(err)
			}
			for e, err := range target.Comments().ByPosition(ctx, "", c.PositionID) {
				if err != nil {
					t.Fatal(err)
				}
				if e.ID == c.ID && (e.Signer != nil || e.Signature != "") {
					t.Errorf("edited comment still signed: %+v", e.Signer)
				}
			}
		}
		return out
	}

	got := importInto(path)
	for _, text := range []string{"Joue 8/4 6/4.", "Le pip count décide.", "Ailleurs."} {
		c := got[text]
		if c == nil || c.Signer == nil || !c.Signer.Verified || c.Signer.Name != "Coach Martin" || c.Signer.Fingerprint != coach.Fingerprint() {
			t.Errorf("%q after import: %+v; want verified, signed by Coach Martin", text, c)
		}
		if c != nil && c.Origin != domain.CommentOriginUser {
			t.Errorf("%q after import: origin %q; want user", text, c.Origin)
		}
	}
	if c := got["XG: -0.120"]; c == nil || c.Signer != nil || c.Origin != domain.CommentOriginXG {
		t.Errorf("an importer's note must travel unsigned and keep its origin: %+v", c)
	}

	// The file is altered on its way: the note arrives, flagged.
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`UPDATE comment SET text = 'Joue 13/7 13/9.' WHERE text = 'Joue 8/4 6/4.'`,
		`UPDATE comment SET position_id = (SELECT position_id FROM comment WHERE text = 'Le pip count décide.') WHERE text = 'Ailleurs.'`,
	} {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	raw.Close()
	got = importInto(path)
	for _, text := range []string{"Joue 13/7 13/9.", "Ailleurs."} {
		if c := got[text]; c == nil || c.Signer == nil || c.Signer.Verified || c.Signer.Name != "Coach Martin" {
			t.Errorf("%q altered in the file: %+v; want kept, signed by Coach Martin, unverified", text, c)
		}
	}

	// A file from 2.45.0, the last schema without the column, imports as it
	// stands: never migrated, its comments arrive unsigned.
	raw, err = sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`ALTER TABLE comment DROP COLUMN signature`,
		`UPDATE metadata SET value = '2.45.0' WHERE key = 'database_version'`,
	} {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	raw.Close()
	got = importInto(path)
	if c := got["Ailleurs."]; c == nil || c.Signer != nil || c.Origin != domain.CommentOriginUser {
		t.Errorf("comment of a 2.45.0 file: %+v; want imported, unsigned, of origin user", c)
	}

	// A file older than 2.19.0 has neither origin nor author: its comments
	// arrive of origin unknown, with no author, as the desktop reads them.
	raw, err = sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`DROP INDEX IF EXISTS idx_comment_position`,
		`ALTER TABLE comment DROP COLUMN origin`,
		`ALTER TABLE comment DROP COLUMN author`,
	} {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	raw.Close()
	got = importInto(path)
	if c := got["Ailleurs."]; c == nil || c.Signer != nil || c.Origin != domain.CommentOriginUnknown || c.Author != "" {
		t.Errorf("comment of a file without origin nor author: %+v; want imported, unsigned, origin unknown, no author", c)
	}
}
