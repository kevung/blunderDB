package storagetest

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// signedStudentLibrary returns a library holding a coach's signed note and an
// imported XG note on provenancePos(21), as a student receives them.
func signedStudentLibrary(t *testing.T, coach *issuance.Identity) storage.Storage {
	t.Helper()
	ctx := context.Background()
	open := func(name string) storage.Storage {
		t.Helper()
		st, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), name), nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { st.Close() })
		return st
	}
	src := open("coach.db")
	p := provenancePos(21)
	id, err := src.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := src.Comments().Add(storage.WithCommentAuthor(ctx, "  Martin  "), "", id, "Joue 8/4 6/4."); err != nil {
		t.Fatal(err)
	}
	if _, err := src.Comments().AddFrom(ctx, "", id, "XG: -0.120", domain.CommentOriginXG); err != nil {
		t.Fatal(err)
	}
	opts := ingest.WholeTenant(ingest.FormatSQLite)
	opts.Signer = coach
	file := filepath.Join(t.TempDir(), "coach-export.db")
	if _, err := ingest.ExportSQLite(ctx, src, "", file, opts); err != nil {
		t.Fatalf("ExportSQLite: %v", err)
	}
	mid := open("student.db")
	if _, err := (ingest.DBImporter{S: mid}).Import(ctx, "", ingest.Source{Format: ingest.FormatNativeDB, Path: file}, nil); err != nil {
		t.Fatalf("Import: %v", err)
	}
	return mid
}

func commentsByText(t *testing.T, s storage.Storage) map[string]*domain.CommentEntry {
	t.Helper()
	out := map[string]*domain.CommentEntry{}
	for c, err := range s.Comments().ListAll(context.Background(), "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		out[c.Text] = c
	}
	return out
}

func wantCoachSigned(t *testing.T, c *domain.CommentEntry, coach *issuance.Identity, via string) {
	t.Helper()
	if c == nil || c.Signer == nil || !c.Signer.Verified ||
		c.Signer.Name != "Coach Martin" || c.Signer.Fingerprint != coach.Fingerprint() {
		t.Fatalf("signed note after %s: %+v; want verified, signed by Coach Martin", via, c)
	}
	if c.Origin != domain.CommentOriginUser || c.Author != "Martin" {
		t.Errorf("signed note after %s: origin %q, author %q; want user, Martin", via, c.Origin, c.Author)
	}
}

// testCommentSignatureJSON: a signed note keeps its signature, its author
// and its origin through the JSON interchange; an imported note keeps its
// origin and stays unsigned (ADR-0099).
func testCommentSignatureJSON(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	coach, err := issuance.NewIdentity("Coach Martin")
	if err != nil {
		t.Fatal(err)
	}
	var stream bytes.Buffer
	if err := (ingest.JSONExporter{S: signedStudentLibrary(t, coach)}).Export(ctx, "", &stream, ingest.ExportOptions{}); err != nil {
		t.Fatalf("JSON export: %v", err)
	}
	if _, err := (ingest.JSONImporter{S: s}).Import(ctx, "", ingest.Source{Reader: &stream}, nil); err != nil {
		t.Fatalf("JSON import: %v", err)
	}
	got := commentsByText(t, s)
	wantCoachSigned(t, got["Joue 8/4 6/4."], coach, "JSON import")
	if x := got["XG: -0.120"]; x == nil || x.Signer != nil || x.Origin != domain.CommentOriginXG {
		t.Errorf("an importer's note must stay unsigned with its origin: %+v", x)
	}
}

// testCommentSignatureCarried: a write that copies a signed note from
// elsewhere stores its signature, author and origin as they stand, never a
// recomputed seal; the same words written here stay unsigned (ADR-0099).
func testCommentSignatureCarried(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	coach, err := issuance.NewIdentity("Coach Martin")
	if err != nil {
		t.Fatal(err)
	}
	signed := commentsByText(t, signedStudentLibrary(t, coach))["Joue 8/4 6/4."]
	p := provenancePos(21)
	id, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := s.Comments().AddFrom(storage.WithCarriedComment(ctx, signed.Author, signed.Signature), "", id, signed.Text, signed.Origin); err != nil {
		t.Fatalf("AddFrom carried: %v", err)
	}
	if _, err := s.Comments().Add(storage.WithCommentAuthor(ctx, "Martin"), "", id, "Same words, no seal."); err != nil {
		t.Fatal(err)
	}
	got := commentsByText(t, s)
	wantCoachSigned(t, got[signed.Text], coach, "carried write")
	if got[signed.Text].Signature != signed.Signature {
		t.Error("the signature was not stored as it stands")
	}
	if c := got["Same words, no seal."]; c == nil || c.Signer != nil || c.Signature != "" {
		t.Errorf("a note written here must stay unsigned: %+v", c)
	}
}
