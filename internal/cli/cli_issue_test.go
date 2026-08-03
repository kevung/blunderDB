package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"
	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
)

// isolateIdentity points the issuer identity at a throwaway config directory, so a test run
// never creates — or reuses — the identity of whoever runs the suite.
func isolateIdentity(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	xdg.Reload()
	t.Cleanup(xdg.Reload)
}

func newIssuedDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cours.db")
	cli := NewCLI()
	if err := cli.Run([]string{"create", "--db", path}); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Release the single-writer lock: the CLI holds it for the life of its Database, and a
	// later command in the same process would otherwise open the file read-only.
	if err := cli.db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return path
}

func TestIssueCommandWritesOneFilePerRecipient(t *testing.T) {
	isolateIdentity(t)
	dbPath := newIssuedDB(t)
	outDir := filepath.Join(t.TempDir(), "exemplaires")

	list := filepath.Join(t.TempDir(), "eleves.txt")
	if err := os.WriteFile(list, []byte("Kévin Unger\n# un commentaire\n\nMarie Durand\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cli := NewCLI()
	if err := cli.Run([]string{"issue", "--db", dbPath, "--distribution", "Cours du 12 mars", "--to-file", list, "--dir", outDir}); err != nil {
		t.Fatalf("issue: %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected two copies (comments and blanks dropped), got %d", len(entries))
	}

	for _, e := range entries {
		info, err := database.InspectIssuance(filepath.Join(outDir, e.Name()))
		if err != nil {
			t.Fatalf("InspectIssuance: %v", err)
		}
		if !info.IsIssuedCopy || !info.Watermark.SignatureValid || !info.Watermark.IssuedByYou {
			t.Fatalf("%s is not a verifiable issued copy: %+v", e.Name(), info.Watermark)
		}
		if info.Watermark.Distribution != "Cours du 12 mars" {
			t.Fatalf("unexpected distribution in %s: %q", e.Name(), info.Watermark.Distribution)
		}
		if len(info.Issued) != 0 {
			t.Fatalf("%s carries an issue register — it must never travel", e.Name())
		}
	}

	// The register lives in the source, with both recipients.
	source, err := database.InspectIssuance(dbPath)
	if err != nil {
		t.Fatalf("InspectIssuance (source): %v", err)
	}
	if len(source.Issued) != 2 {
		t.Fatalf("the issuer's register must list both copies, got %d", len(source.Issued))
	}
}

func TestIssueCommandRequiresADistribution(t *testing.T) {
	isolateIdentity(t)
	dbPath := newIssuedDB(t)
	cli := NewCLI()
	err := cli.Run([]string{"issue", "--db", dbPath, "--file", filepath.Join(t.TempDir(), "x.db")})
	if err == nil || !strings.Contains(err.Error(), "distribution") {
		t.Fatalf("expected a missing-distribution error, got %v", err)
	}
}

func TestIssueCommandRefusesABatchWithoutADirectory(t *testing.T) {
	isolateIdentity(t)
	dbPath := newIssuedDB(t)
	cli := NewCLI()
	err := cli.Run([]string{"issue", "--db", dbPath, "--distribution", "Cours", "--to", "A,B"})
	if err == nil || !strings.Contains(err.Error(), "--dir") {
		t.Fatalf("expected a missing --dir error, got %v", err)
	}
}

// The CLI must never record a holder: examining a copy that came back has to leave it
// untouched. This is the one deliberate departure from GUI/CLI parity (ADR-0007).
func TestCLINeverRecordsAHolder(t *testing.T) {
	isolateIdentity(t)
	dbPath := newIssuedDB(t)
	copyPath := filepath.Join(t.TempDir(), "copie.db")

	cli := NewCLI()
	if err := cli.Run([]string{"issue", "--db", dbPath, "--distribution", "Cours", "--to", "Kévin", "--file", copyPath}); err != nil {
		t.Fatalf("issue: %v", err)
	}

	if err := cli.db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	for i := 0; i < 3; i++ {
		inspector := NewCLI()
		if err := inspector.Run([]string{"info", "--db", copyPath}); err != nil {
			t.Fatalf("info: %v", err)
		}
		if err := inspector.db.Close(); err != nil {
			t.Fatalf("close: %v", err)
		}
	}

	info, err := database.InspectIssuance(copyPath)
	if err != nil {
		t.Fatalf("InspectIssuance: %v", err)
	}
	if len(info.Holders) != 0 {
		t.Fatalf("the CLI must not write holders, got %d", len(info.Holders))
	}
}

func TestProtectedCopyRoundTripThroughTheCLI(t *testing.T) {
	isolateIdentity(t)
	dbPath := newIssuedDB(t)
	outDir := filepath.Join(t.TempDir(), "protege")

	cli := NewCLI()
	if err := cli.Run([]string{"issue", "--db", dbPath, "--distribution", "Cours", "--to", "Kévin", "--dir", outDir, "--password", "s3cret"}); err != nil {
		t.Fatalf("issue: %v", err)
	}
	entries, err := os.ReadDir(outDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected one protected copy (%v, %v)", entries, err)
	}
	container := filepath.Join(outDir, entries[0].Name())
	if !strings.HasSuffix(container, issuance.ContainerExtension) {
		t.Fatalf("expected a %s file, got %s", issuance.ContainerExtension, container)
	}

	// Identifiable without the password — the reason the header is cleartext.
	info, err := database.InspectIssuance(container)
	if err != nil {
		t.Fatalf("InspectIssuance: %v", err)
	}
	if info.Watermark == nil || info.Watermark.Recipient != "Kévin" {
		t.Fatalf("a protected copy must stay identifiable: %+v", info.Watermark)
	}

	if err := NewCLI().Run([]string{"open", "--db", container, "--password", "wrong"}); err == nil {
		t.Fatal("a wrong password must be rejected")
	}
	if err := NewCLI().Run([]string{"open", "--db", container, "--password", "s3cret"}); err != nil {
		t.Fatalf("open: %v", err)
	}
	opened := issuance.DefaultUnwrapPath(container)
	if _, err := os.Stat(opened); err != nil {
		t.Fatalf("the opened database is missing: %v", err)
	}
	// A second open must not silently discard the work already done in the opened file.
	if err := NewCLI().Run([]string{"open", "--db", container, "--password", "s3cret"}); err == nil {
		t.Fatal("opening over an existing file must be refused")
	}
}

func TestIdentityCommandRoundTrip(t *testing.T) {
	isolateIdentity(t)
	if err := NewCLI().Run([]string{"identity", "--name", "Jean Dupont"}); err != nil {
		t.Fatalf("identity: %v", err)
	}
	before, err := issuance.LoadIdentity(issuance.ConfigDir())
	if err != nil || before == nil {
		t.Fatalf("LoadIdentity: %v / %v", before, err)
	}
	if before.Name != "Jean Dupont" {
		t.Fatalf("unexpected name: %q", before.Name)
	}

	file := filepath.Join(t.TempDir(), "jean"+issuance.IdentityFileExtension)
	if err := NewCLI().Run([]string{"identity", "--export", file, "--passphrase", "pw"}); err != nil {
		t.Fatalf("identity --export: %v", err)
	}
	if err := NewCLI().Run([]string{"identity", "--import", file}); err == nil {
		t.Fatal("a protected identity file must not import without its passphrase")
	}

	// Import it onto a second "machine" and check the fingerprint survives.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	xdg.Reload()
	if err := NewCLI().Run([]string{"identity", "--import", file, "--passphrase", "pw"}); err != nil {
		t.Fatalf("identity --import: %v", err)
	}
	after, err := issuance.LoadIdentity(issuance.ConfigDir())
	if err != nil || after == nil {
		t.Fatalf("LoadIdentity: %v / %v", after, err)
	}
	if after.Fingerprint() != before.Fingerprint() {
		t.Fatal("moving an identity between machines must preserve it")
	}
}

func TestGatherRecipients(t *testing.T) {
	file := filepath.Join(t.TempDir(), "list.txt")
	if err := os.WriteFile(file, []byte("  Marie Durand \n#skip\n\nLéo Martin\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := gatherRecipients("Kévin Unger, , A", file)
	if err != nil {
		t.Fatalf("gatherRecipients: %v", err)
	}
	want := []string{"Kévin Unger", "A", "Marie Durand", "Léo Martin"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
