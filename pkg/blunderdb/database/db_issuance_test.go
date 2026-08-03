package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
)

// isolateIdentity points the signing identity at a throwaway config directory. Without it a
// test run would create — or worse, reuse — the identity of whoever is running the suite.
func isolateIdentity(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	xdg.Reload()
	t.Cleanup(xdg.Reload)
}

func exportTo(t *testing.T, source *Database, path string, opts ExportOptions) string {
	t.Helper()
	opts.ExportPath = path
	if opts.Metadata == nil {
		opts.Metadata = map[string]string{"user": "Jean Dupont", "description": "Cours"}
	}
	if err := source.ExportDatabase(opts); err != nil {
		t.Fatalf("ExportDatabase: %v", err)
	}
	return path
}

func TestWatermarkedExportCarriesAVerifiableOrigin(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)

	path := exportTo(t, source, filepath.Join(t.TempDir(), "cours.db"), ExportOptions{
		Watermark:     "Cours de Jean Dupont — 12 mars 2026",
		WatermarkNote: "Merci de ne pas rediffuser.",
	})

	info, err := InspectIssuance(path)
	if err != nil {
		t.Fatalf("InspectIssuance: %v", err)
	}
	if !info.Watermarked || info.Watermark == nil {
		t.Fatal("the exported file must carry a watermark")
	}
	if !info.Watermark.SignatureValid || !info.Watermark.IssuedByYou {
		t.Fatalf("the producer must recognise their own mark: %+v", info.Watermark)
	}
	if info.Watermark.Origin != "Cours de Jean Dupont — 12 mars 2026" {
		t.Fatalf("unexpected origin: %q", info.Watermark.Origin)
	}
	if info.Watermark.Note != "Merci de ne pas rediffuser." {
		t.Fatalf("the note must travel: %q", info.Watermark.Note)
	}
}

// An export without a watermark must be exactly what it always was.
func TestPlainExportCarriesNothing(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)
	path := exportTo(t, source, filepath.Join(t.TempDir(), "cours.db"), ExportOptions{})

	info, err := InspectIssuance(path)
	if err != nil {
		t.Fatalf("InspectIssuance: %v", err)
	}
	if info.Watermarked || info.Watermark != nil {
		t.Fatalf("an ordinary export must carry no watermark: %+v", info)
	}
	// …and exporting without a watermark must not have created a signing identity.
	id, err := issuance.LoadIdentity(issuance.ConfigDir())
	if err != nil {
		t.Fatalf("LoadIdentity: %v", err)
	}
	if id != nil {
		t.Fatal("no identity should exist until the first watermark")
	}
}

func TestUnknownMetadataIsNotCarriedIntoAnExport(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)

	path := exportTo(t, source, filepath.Join(t.TempDir(), "cours.db"), ExportOptions{
		Metadata: map[string]string{
			"user":        "Jean",
			"description": "Cours",
			"surprise":    "a document added six months from now",
		},
	})

	opened := NewDatabase()
	if err := opened.OpenDatabase(path); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	t.Cleanup(func() { _ = opened.Close() })

	if got, _ := opened.readMeta("surprise"); got != "" {
		t.Fatalf("metadata must be copied by allow-list, got %q", got)
	}
	if got, _ := opened.readMeta("user"); got != "Jean" {
		t.Fatalf("ordinary metadata must be carried, got %q", got)
	}
}

// The design's central promise: the recipient's side records nothing. Opening a watermarked
// database must leave every issuance row exactly as the producer wrote it, and must not
// create any of the rows earlier iterations used to keep (holders, lineage, register).
//
// Note that opening a database is not a read-only act in general — blunderDB applies WAL
// pragmas and may run ANALYZE to build query-planner statistics — so this asserts on the
// metadata rows rather than on the file's bytes.
func TestOpeningAWatermarkedDatabaseRecordsNothing(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)
	path := exportTo(t, source, filepath.Join(t.TempDir(), "cours.db"), ExportOptions{
		Watermark: "Cours de Jean Dupont",
	})

	// Retired keys: they must never reappear, whatever happens on the recipient's side.
	retired := []string{"holders", "lineage", "issued"}

	var sealed string
	for i := 0; i < 3; i++ {
		opened := NewDatabase()
		if err := opened.OpenDatabase(path); err != nil {
			t.Fatalf("OpenDatabase: %v", err)
		}
		if _, err := opened.GetIssuanceInfo(); err != nil {
			t.Fatalf("GetIssuanceInfo: %v", err)
		}
		if _, err := InspectIssuance(path); err != nil {
			t.Fatalf("InspectIssuance: %v", err)
		}
		got, err := opened.readMeta(issuance.KeyWatermark)
		if err != nil {
			t.Fatalf("readMeta: %v", err)
		}
		if i == 0 {
			sealed = got
		} else if got != sealed {
			t.Fatal("the watermark must not change when the file is opened")
		}
		for _, key := range retired {
			if value, _ := opened.readMeta(key); value != "" {
				t.Fatalf("opening wrote %q = %q", key, value)
			}
		}
		if err := opened.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}
	if sealed == "" {
		t.Fatal("test setup: the export carried no watermark")
	}
}

// Importing a watermarked database into one's own must leave no trace of it either: the
// recipient's database is theirs, and blunderDB records nothing about where its contents
// came from.
func TestImportingAWatermarkedDatabaseLeavesNoTrace(t *testing.T) {
	isolateIdentity(t)
	teacher := newTestDB(t)
	path := exportTo(t, teacher, filepath.Join(t.TempDir(), "cours.db"), ExportOptions{
		Watermark: "Cours de Jean Dupont",
	})

	student := newTestDB(t)
	if _, err := student.CommitImportDatabase(path); err != nil {
		t.Fatalf("CommitImportDatabase: %v", err)
	}
	info, err := student.GetIssuanceInfo()
	if err != nil {
		t.Fatalf("GetIssuanceInfo: %v", err)
	}
	if info.Watermarked || info.Watermark != nil {
		t.Fatalf("importing must not stamp the receiving database: %+v", info)
	}
	for _, key := range []string{"watermark", "holders", "lineage", "issued"} {
		if got, _ := student.readMeta(key); got != "" {
			t.Fatalf("importing wrote %q = %q", key, got)
		}
	}
}

// The GUI's save dialog forces a .db name before the user has chosen a password, so a
// protected export must rename itself rather than ship encrypted bytes under a .db name.
func TestProtectedExportIsRenamedToBdbx(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)
	dir := t.TempDir()
	asked := filepath.Join(dir, "cours.db")

	if err := source.ExportDatabase(ExportOptions{
		ExportPath: asked,
		Metadata:   map[string]string{},
		Password:   "pw",
	}); err != nil {
		t.Fatalf("ExportDatabase: %v", err)
	}

	if _, err := os.Stat(asked); err == nil {
		t.Fatalf("%s must not exist: an encrypted file must not be named .db", asked)
	}
	produced := filepath.Join(dir, "cours.bdbx")
	if !issuance.IsContainer(produced) {
		t.Fatalf("expected a protected container at %s", produced)
	}
	if entries, _ := filepath.Glob(filepath.Join(dir, "*.plain")); len(entries) > 0 {
		t.Fatalf("the intermediate export was left on disk: %v", entries)
	}

	// …and it opens without landing on itself.
	opened, err := OpenProtectedCopy(produced, "pw")
	if err != nil {
		t.Fatalf("OpenProtectedCopy: %v", err)
	}
	if opened == produced {
		t.Fatal("unwrapping must not overwrite the container")
	}
	db := NewDatabase()
	if err := db.OpenDatabase(opened); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	_ = db.Close()
}

func TestPasswordProtectedExport(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)
	dir := t.TempDir()
	container := filepath.Join(dir, "cours.bdbx")

	exportTo(t, source, container, ExportOptions{
		Watermark: "Cours de Jean Dupont — 12 mars 2026",
		Password:  "le-mot-de-passe",
	})

	if !issuance.IsContainer(container) {
		t.Fatal("the produced file must be a protected container")
	}
	// The intermediate plain export must not have been left behind: it is the whole
	// database, unprotected, next to the protected copy.
	if entries, _ := filepath.Glob(filepath.Join(dir, "*.plain")); len(entries) > 0 {
		t.Fatalf("the intermediate export was left on disk: %v", entries)
	}

	// The origin stays readable without the password.
	info, err := InspectIssuance(container)
	if err != nil {
		t.Fatalf("InspectIssuance: %v", err)
	}
	if info.Watermark == nil || !strings.HasPrefix(info.Watermark.Origin, "Cours de Jean Dupont") {
		t.Fatalf("a protected copy must keep its origin readable: %+v", info.Watermark)
	}

	if _, err := OpenProtectedCopy(container, "wrong"); err == nil {
		t.Fatal("a wrong password must be rejected")
	}
	opened, err := OpenProtectedCopy(container, "le-mot-de-passe")
	if err != nil {
		t.Fatalf("OpenProtectedCopy: %v", err)
	}
	db := NewDatabase()
	if err := db.OpenDatabase(opened); err != nil {
		t.Fatalf("the opened container must be an ordinary database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	openedInfo, err := db.GetIssuanceInfo()
	if err != nil {
		t.Fatalf("GetIssuanceInfo: %v", err)
	}
	if openedInfo.Watermark == nil || !openedInfo.Watermark.SignatureValid {
		t.Fatalf("the watermark must survive unwrapping: %+v", openedInfo.Watermark)
	}
}

// A password with no watermark, and a watermark with no password, are both valid: the two
// mechanisms are independent.
func TestPasswordWithoutAWatermark(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)
	container := filepath.Join(t.TempDir(), "cours.bdbx")
	exportTo(t, source, container, ExportOptions{Password: "pw"})

	if !issuance.IsContainer(container) {
		t.Fatal("the produced file must be a protected container")
	}
	info, err := InspectIssuance(container)
	if err != nil {
		t.Fatalf("InspectIssuance: %v", err)
	}
	if info.Watermarked {
		t.Fatal("no watermark was asked for")
	}
	opened, err := OpenProtectedCopy(container, "pw")
	if err != nil {
		t.Fatalf("OpenProtectedCopy: %v", err)
	}
	db := NewDatabase()
	if err := db.OpenDatabase(opened); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	_ = db.Close()
}

// Opening a protected copy twice must not discard the work done in the first result.
func TestOpeningAProtectedCopyTwiceKeepsTheFirstResult(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)
	container := filepath.Join(t.TempDir(), "cours.bdbx")
	exportTo(t, source, container, ExportOptions{Password: "pw"})

	first, err := OpenProtectedCopy(container, "pw")
	if err != nil {
		t.Fatalf("OpenProtectedCopy: %v", err)
	}
	marker := []byte("work done by the recipient")
	if err := os.WriteFile(first, marker, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	second, err := OpenProtectedCopy(container, "pw")
	if err != nil {
		t.Fatalf("OpenProtectedCopy (again): %v", err)
	}
	if second != first {
		t.Fatalf("expected the same path, got %q then %q", first, second)
	}
	got, err := os.ReadFile(first)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != string(marker) {
		t.Fatal("a second open must not overwrite the database already in use")
	}
}

func TestGetIssuanceInfoOnAnOrdinaryDatabase(t *testing.T) {
	isolateIdentity(t)
	db := newTestDB(t)
	info, err := db.GetIssuanceInfo()
	if err != nil {
		t.Fatalf("GetIssuanceInfo: %v", err)
	}
	if info.Watermarked || info.Watermark != nil {
		t.Fatalf("an ordinary database reports nothing: %+v", info)
	}
	if info.IssuerFingerprint != "" {
		t.Fatalf("no identity should exist before the first watermark, got %q", info.IssuerFingerprint)
	}
}
