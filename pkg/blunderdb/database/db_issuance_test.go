package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
)

// isolateIdentity points the Issuer identity at a throwaway config directory. Without it a
// test run would create — or worse, reuse — the identity of whoever is running the suite.
func isolateIdentity(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	xdg.Reload()
	t.Cleanup(xdg.Reload)
}

func issuedCopy(t *testing.T, source *Database, iss IssuanceOptions) IssuedCopy {
	t.Helper()
	copies, err := source.IssueCopies(ExportOptions{
		ExportPath: filepath.Join(t.TempDir(), "copy.db"),
		Metadata:   map[string]string{"user": "Jean Dupont", "description": "Cours"},
	}, iss)
	if err != nil {
		t.Fatalf("IssueCopies: %v", err)
	}
	if len(copies) != 1 {
		t.Fatalf("expected one copy, got %d", len(copies))
	}
	return copies[0]
}

func TestIssueCopyCarriesAVerifiableWatermark(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)

	copy := issuedCopy(t, source, IssuanceOptions{Distribution: "Cours du 12 mars", Recipients: []string{"Kévin Unger"}})

	info, err := InspectIssuance(copy.Path)
	if err != nil {
		t.Fatalf("InspectIssuance: %v", err)
	}
	if !info.IsIssuedCopy || info.Watermark == nil {
		t.Fatal("the produced file must be an issued copy")
	}
	if !info.Watermark.SignatureValid {
		t.Fatal("the watermark must verify")
	}
	if !info.Watermark.IssuedByYou {
		t.Fatal("the issuer must recognise their own copy")
	}
	if info.Watermark.Recipient != "Kévin Unger" || !info.Watermark.Nominative {
		t.Fatalf("unexpected watermark: %+v", info.Watermark)
	}
	if len(info.Holders) != 0 {
		t.Fatal("a freshly issued copy has no holders yet")
	}
}

// The leak the design has to avoid: the issue register lists every recipient of a course
// and the password of every distribution.
func TestIssuedCopyNeverCarriesTheIssueRegister(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)

	first := issuedCopy(t, source, IssuanceOptions{
		Distribution: "Cours du 12 mars",
		Recipients:   []string{"Kévin Unger"},
		Password:     "",
	})
	_ = first

	// A second emission, so the source register is non-empty when the third copy is made.
	second := issuedCopy(t, source, IssuanceOptions{
		Distribution: "Cours du 12 mars",
		Recipients:   []string{"Marie Durand"},
	})

	opened := NewDatabase()
	if err := opened.OpenDatabase(second.Path); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	t.Cleanup(func() { _ = opened.Close() })

	raw, err := opened.readMeta(issuance.KeyIssued)
	if err != nil {
		t.Fatalf("readMeta: %v", err)
	}
	if raw != "" {
		t.Fatalf("the issue register leaked into an issued copy: %s", raw)
	}
	info, err := opened.GetIssuanceInfo()
	if err != nil {
		t.Fatalf("GetIssuanceInfo: %v", err)
	}
	if len(info.Issued) != 0 {
		t.Fatalf("an issued copy must show no issue register, got %d records", len(info.Issued))
	}

	// …while the issuer's own database has both records, with the recipients.
	sourceInfo, err := source.GetIssuanceInfo()
	if err != nil {
		t.Fatalf("GetIssuanceInfo (source): %v", err)
	}
	if len(sourceInfo.Issued) != 2 {
		t.Fatalf("the issuer's register must list both copies, got %d", len(sourceInfo.Issued))
	}
}

func TestUnknownMetadataIsNotCarriedIntoACopy(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)

	copies, err := source.IssueCopies(ExportOptions{
		ExportPath: filepath.Join(t.TempDir(), "copy.db"),
		Metadata: map[string]string{
			"user":        "Jean",
			"description": "Cours",
			"surprise":    "a document added six months from now",
		},
	}, IssuanceOptions{Distribution: "Cours", Recipients: []string{"Kévin"}})
	if err != nil {
		t.Fatalf("IssueCopies: %v", err)
	}

	opened := NewDatabase()
	if err := opened.OpenDatabase(copies[0].Path); err != nil {
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

func TestBatchIssuesOneCopyPerRecipient(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)
	dir := t.TempDir()

	copies, err := source.IssueCopies(ExportOptions{Metadata: map[string]string{}}, IssuanceOptions{
		Distribution: "Cours du 12 mars",
		Recipients:   []string{"Kévin Unger", "Marie Durand", "  ", "Léo Martin"},
		OutputDir:    dir,
	})
	if err != nil {
		t.Fatalf("IssueCopies: %v", err)
	}
	if len(copies) != 3 {
		t.Fatalf("blank recipients must be dropped: %d copies", len(copies))
	}

	seen := map[string]bool{}
	salts := map[string]bool{}
	for i, c := range copies {
		if c.Number != i+1 || c.Total != 3 {
			t.Fatalf("copy %d numbered %d/%d", i, c.Number, c.Total)
		}
		if seen[c.Path] {
			t.Fatalf("two recipients landed on the same file: %s", c.Path)
		}
		seen[c.Path] = true
		if !strings.HasPrefix(filepath.Base(c.Path), "cours-du-12-mars_") {
			t.Fatalf("unexpected file name: %s", filepath.Base(c.Path))
		}
		info, err := InspectIssuance(c.Path)
		if err != nil {
			t.Fatalf("InspectIssuance: %v", err)
		}
		if info.Watermark.Recipient != c.Recipient {
			t.Fatalf("copy for %q names %q", c.Recipient, info.Watermark.Recipient)
		}
		salts[saltOf(t, c.Path)] = true
	}
	if len(salts) != 1 {
		t.Fatalf("copies of one distribution must share a salt, got %d distinct", len(salts))
	}
}

// A second batch for the same course must continue the first, not restart it — otherwise
// two different students both hold "copy 1 of 12".
func TestSecondBatchContinuesTheDistribution(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)

	firstDir, secondDir := t.TempDir(), t.TempDir()
	first, err := source.IssueCopies(ExportOptions{}, IssuanceOptions{
		Distribution: "Cours", Recipients: []string{"A", "B"}, OutputDir: firstDir,
	})
	if err != nil {
		t.Fatalf("IssueCopies: %v", err)
	}
	second, err := source.IssueCopies(ExportOptions{}, IssuanceOptions{
		Distribution: "Cours", Recipients: []string{"C"}, OutputDir: secondDir,
	})
	if err != nil {
		t.Fatalf("IssueCopies (second batch): %v", err)
	}
	if second[0].Number != 3 {
		t.Fatalf("the second batch must continue at 3, got %d", second[0].Number)
	}
	if saltOf(t, first[0].Path) != saltOf(t, second[0].Path) {
		t.Fatal("both batches belong to one distribution and must share its salt")
	}
}

func saltOf(t *testing.T, path string) string {
	t.Helper()
	env, _, err := readIssuanceFrom(path)
	if err != nil {
		t.Fatalf("readIssuanceFrom: %v", err)
	}
	w, err := env.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return w.Salt
}

func TestCollectiveCopyNamesNoRecipient(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)

	copy := issuedCopy(t, source, IssuanceOptions{Distribution: "Promotion 2026"})
	info, err := InspectIssuance(copy.Path)
	if err != nil {
		t.Fatalf("InspectIssuance: %v", err)
	}
	if info.Watermark.Nominative || info.Watermark.Recipient != "" {
		t.Fatalf("a collective copy names no recipient: %+v", info.Watermark)
	}
	if info.Watermark.Distribution != "Promotion 2026" {
		t.Fatalf("unexpected distribution: %q", info.Watermark.Distribution)
	}
}

func TestIssuingRequiresADistributionName(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)
	if _, err := source.IssueCopies(ExportOptions{ExportPath: filepath.Join(t.TempDir(), "x.db")}, IssuanceOptions{}); err == nil {
		t.Fatal("a copy without a distribution cannot be attributed to anything")
	}
}

func TestRecordHolderCountsMachinesAndOnlyForIssuedCopies(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)
	copy := issuedCopy(t, source, IssuanceOptions{Distribution: "Cours", Recipients: []string{"Kévin"}})

	held := NewDatabase()
	if err := held.OpenDatabase(copy.Path); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	t.Cleanup(func() { _ = held.Close() })

	for i := 0; i < 3; i++ {
		if err := held.RecordHolder(); err != nil {
			t.Fatalf("RecordHolder: %v", err)
		}
	}
	info, err := held.GetIssuanceInfo()
	if err != nil {
		t.Fatalf("GetIssuanceInfo: %v", err)
	}
	if len(info.Holders) != 1 {
		t.Fatalf("one machine must produce one entry, got %d", len(info.Holders))
	}
	if info.Holders[0].Openings != 3 {
		t.Fatalf("expected 3 openings, got %d", info.Holders[0].Openings)
	}
	if !info.ChainIntact {
		t.Fatal("a registry blunderDB wrote must read as intact")
	}

	// An ordinary database is left completely alone.
	plain := newTestDB(t)
	if err := plain.RecordHolder(); err != nil {
		t.Fatalf("RecordHolder on an ordinary database: %v", err)
	}
	raw, err := plain.readMeta(issuance.KeyHolders)
	if err != nil {
		t.Fatalf("readMeta: %v", err)
	}
	if raw != "" {
		t.Fatalf("a database that was never issued must carry no holder registry: %s", raw)
	}
}

// The forensic rule: examining a copy that came back must not write the examiner's own
// machine into the evidence.
func TestInspectingACopyDoesNotContaminateIt(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)
	copy := issuedCopy(t, source, IssuanceOptions{Distribution: "Cours", Recipients: []string{"Kévin"}})

	for i := 0; i < 3; i++ {
		if _, err := InspectIssuance(copy.Path); err != nil {
			t.Fatalf("InspectIssuance: %v", err)
		}
	}
	info, err := InspectIssuance(copy.Path)
	if err != nil {
		t.Fatalf("InspectIssuance: %v", err)
	}
	if len(info.Holders) != 0 {
		t.Fatalf("inspection must not add holders, got %d", len(info.Holders))
	}

	// Opening it read-only through the wrapper must not either.
	opened := NewDatabase()
	if err := opened.OpenDatabase(copy.Path); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	t.Cleanup(func() { _ = opened.Close() })
	if _, err := opened.GetIssuanceInfo(); err != nil {
		t.Fatalf("GetIssuanceInfo: %v", err)
	}
	after, err := InspectIssuance(copy.Path)
	if err != nil {
		t.Fatalf("InspectIssuance: %v", err)
	}
	if len(after.Holders) != 0 {
		t.Fatal("opening and reading must not record a holder — only the GUI does that")
	}
}

// The laundering path the whole lineage mechanism exists for.
func TestImportingAnIssuedCopyCarriesItsWatermarkForward(t *testing.T) {
	isolateIdentity(t)
	teacher := newTestDB(t)
	copy := issuedCopy(t, teacher, IssuanceOptions{Distribution: "Cours du 12 mars", Recipients: []string{"Kévin Unger"}})

	student := newTestDB(t)
	if _, err := student.CommitImportDatabase(copy.Path); err != nil {
		t.Fatalf("CommitImportDatabase: %v", err)
	}

	info, err := student.GetIssuanceInfo()
	if err != nil {
		t.Fatalf("GetIssuanceInfo: %v", err)
	}
	if info.IsIssuedCopy {
		t.Fatal("the student's own database is not an issued copy")
	}
	if len(info.Lineage) != 1 {
		t.Fatalf("the import must leave one lineage entry, got %d", len(info.Lineage))
	}
	if !info.Lineage[0].SignatureValid {
		t.Fatal("an inherited watermark must still verify against its original issuer")
	}
	if info.Lineage[0].Recipient != "Kévin Unger" {
		t.Fatalf("unexpected inherited recipient: %q", info.Lineage[0].Recipient)
	}

	// Re-importing the same copy must not grow the list.
	if _, err := student.CommitImportDatabase(copy.Path); err != nil {
		t.Fatalf("CommitImportDatabase (again): %v", err)
	}
	again, err := student.GetIssuanceInfo()
	if err != nil {
		t.Fatalf("GetIssuanceInfo: %v", err)
	}
	if len(again.Lineage) != 1 {
		t.Fatalf("re-importing must be idempotent, got %d entries", len(again.Lineage))
	}

	// …and re-exporting from the student's database carries the trace on.
	relayed, err := student.IssueCopies(ExportOptions{ExportPath: filepath.Join(t.TempDir(), "relayed.db")},
		IssuanceOptions{Distribution: "Mon partage", Recipients: []string{"Un ami"}})
	if err != nil {
		t.Fatalf("IssueCopies: %v", err)
	}
	relayedInfo, err := InspectIssuance(relayed[0].Path)
	if err != nil {
		t.Fatalf("InspectIssuance: %v", err)
	}
	if len(relayedInfo.Lineage) != 1 || relayedInfo.Lineage[0].Recipient != "Kévin Unger" {
		t.Fatalf("the original watermark must survive one more hop: %+v", relayedInfo.Lineage)
	}
}

func TestImportingAnOrdinaryDatabaseLeavesNoLineage(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)
	plainPath := filepath.Join(t.TempDir(), "plain.db")
	if err := source.ExportDatabase(ExportOptions{ExportPath: plainPath, Metadata: map[string]string{}}); err != nil {
		t.Fatalf("ExportDatabase: %v", err)
	}

	target := newTestDB(t)
	if _, err := target.CommitImportDatabase(plainPath); err != nil {
		t.Fatalf("CommitImportDatabase: %v", err)
	}
	raw, err := target.readMeta(issuance.KeyLineage)
	if err != nil {
		t.Fatalf("readMeta: %v", err)
	}
	if raw != "" {
		t.Fatalf("importing an ordinary database must leave no lineage: %s", raw)
	}
}

func TestEncryptedCopyStaysIdentifiableWithoutItsPassword(t *testing.T) {
	isolateIdentity(t)
	source := newTestDB(t)

	dir := t.TempDir()
	copies, err := source.IssueCopies(ExportOptions{Metadata: map[string]string{"user": "Jean"}}, IssuanceOptions{
		Distribution: "Cours du 12 mars",
		Recipients:   []string{"Kévin Unger"},
		OutputDir:    dir,
		Password:     "le-mot-de-passe",
	})
	if err != nil {
		t.Fatalf("IssueCopies: %v", err)
	}
	produced := copies[0]
	if !produced.Encrypted || !strings.HasSuffix(produced.Path, issuance.ContainerExtension) {
		t.Fatalf("expected an encrypted copy, got %s", produced.Path)
	}
	if !issuance.IsContainer(produced.Path) {
		t.Fatal("the produced file must be a container")
	}

	// The forensic property: identifiable without decrypting.
	info, err := InspectIssuance(produced.Path)
	if err != nil {
		t.Fatalf("InspectIssuance: %v", err)
	}
	if !info.IsIssuedCopy || info.Watermark.Recipient != "Kévin Unger" || !info.Watermark.IssuedByYou {
		t.Fatalf("an encrypted copy must stay identifiable: %+v", info.Watermark)
	}

	// The intermediate plain export must not have been left behind: it is the whole
	// database, unprotected, next to the protected copy.
	if entries, _ := filepath.Glob(filepath.Join(dir, "*.plain")); len(entries) > 0 {
		t.Fatalf("the intermediate export was left on disk: %v", entries)
	}

	// And it opens back into an ordinary database.
	opened := filepath.Join(dir, "opened.db")
	if _, err := issuance.UnwrapContainer(produced.Path, opened, "le-mot-de-passe"); err != nil {
		t.Fatalf("UnwrapContainer: %v", err)
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
	if !openedInfo.IsIssuedCopy || openedInfo.Watermark.Recipient != "Kévin Unger" {
		t.Fatalf("the watermark must survive unwrapping: %+v", openedInfo.Watermark)
	}
}

// Producing copies that exist in no register is worse than producing none: the issuer would
// have handed out files they can no longer look up. A read-only source must be refused
// before any file is written.
func TestIssuingFromAReadOnlyDatabaseIsRefusedUpFront(t *testing.T) {
	isolateIdentity(t)
	path := filepath.Join(t.TempDir(), "cours.db")
	source := NewDatabase()
	if err := source.SetupDatabase(path); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	t.Cleanup(func() { _ = source.Close() })

	// A second handle on the same file loses the single-writer lock and opens read-only,
	// exactly as a second blunderDB instance would.
	second := NewDatabase()
	if err := second.OpenDatabase(path); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	t.Cleanup(func() { _ = second.Close() })
	if !second.readOnly {
		t.Skip("the platform did not grant the single-writer lock; nothing to assert here")
	}

	dir := t.TempDir()
	copies, err := second.IssueCopies(ExportOptions{}, IssuanceOptions{
		Distribution: "Cours", Recipients: []string{"Kévin"}, OutputDir: dir,
	})
	if err == nil {
		t.Fatal("issuing from a read-only database must be refused")
	}
	if len(copies) != 0 {
		t.Fatalf("no copy may be reported, got %d", len(copies))
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("no file may be written, got %v", entries)
	}
}

func TestGetIssuanceInfoOnAnOrdinaryDatabase(t *testing.T) {
	isolateIdentity(t)
	db := newTestDB(t)
	info, err := db.GetIssuanceInfo()
	if err != nil {
		t.Fatalf("GetIssuanceInfo: %v", err)
	}
	if info.IsIssuedCopy || info.Watermark != nil || len(info.Holders) != 0 || len(info.Issued) != 0 {
		t.Fatalf("an ordinary database reports nothing: %+v", info)
	}
	// Looking at the panel must not have created an identity as a side effect.
	if info.IssuerFingerprint != "" {
		t.Fatalf("no identity should exist before the first emission, got %q", info.IssuerFingerprint)
	}
	if _, err := issuance.LoadIdentity(issuance.ConfigDir()); err != nil {
		t.Fatalf("LoadIdentity: %v", err)
	}
}
