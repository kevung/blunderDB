package gui

import (
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
)

// isolateIdentityConfig points XDG_CONFIG_HOME at a throwaway directory, so a
// test run never creates or reuses the identity of whoever runs the suite
// (same idiom as database.isolateIdentity / config_test.go's
// isolateXDGConfig).
func isolateIdentityConfig(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	xdg.Reload()
	t.Cleanup(xdg.Reload)
}

func TestGetIssuerIdentityBeforeItExists(t *testing.T) {
	isolateIdentityConfig(t)
	a := NewApp()

	info, err := a.GetIssuerIdentity()
	if err != nil {
		t.Fatalf("GetIssuerIdentity: %v", err)
	}
	if info.Present {
		t.Errorf("Present = true, want false: opening settings must not mint a key: %+v", info)
	}
	if info.Name != "" || info.Fingerprint != "" {
		t.Errorf("expected no name/fingerprint before creation: %+v", info)
	}
	if info.Path == "" {
		t.Error("Path should always be reported, even before the identity exists")
	}
}

func TestSetIssuerNameCreatesAndNames(t *testing.T) {
	isolateIdentityConfig(t)
	a := NewApp()

	info, err := a.SetIssuerName("Jean Dupont")
	if err != nil {
		t.Fatalf("SetIssuerName: %v", err)
	}
	if !info.Present {
		t.Errorf("Present = false, want true: SetIssuerName is an explicit gesture, should mint a key: %+v", info)
	}
	if info.Name != "Jean Dupont" {
		t.Errorf("Name = %q, want %q", info.Name, "Jean Dupont")
	}
	if info.Fingerprint == "" {
		t.Error("expected a non-empty fingerprint once the identity exists")
	}

	// GetIssuerIdentity should now see it too.
	got, err := a.GetIssuerIdentity()
	if err != nil {
		t.Fatalf("GetIssuerIdentity after SetIssuerName: %v", err)
	}
	if got.Fingerprint != info.Fingerprint {
		t.Errorf("GetIssuerIdentity fingerprint = %q, want %q (same identity)", got.Fingerprint, info.Fingerprint)
	}
}

func TestSetIssuerNameRenameKeepsFingerprint(t *testing.T) {
	isolateIdentityConfig(t)
	a := NewApp()

	first, err := a.SetIssuerName("Jean Dupont")
	if err != nil {
		t.Fatalf("first SetIssuerName: %v", err)
	}
	second, err := a.SetIssuerName("J. Dupont")
	if err != nil {
		t.Fatalf("second SetIssuerName: %v", err)
	}
	if second.Name != "J. Dupont" {
		t.Errorf("Name = %q, want %q", second.Name, "J. Dupont")
	}
	if second.Fingerprint != first.Fingerprint {
		t.Errorf("renaming must keep the same signing key: fingerprint changed from %q to %q", first.Fingerprint, second.Fingerprint)
	}
}

func TestSetIssuerNameRejectsBlank(t *testing.T) {
	isolateIdentityConfig(t)
	a := NewApp()

	if _, err := a.SetIssuerName("   "); err == nil {
		t.Error("expected an error naming the issuer with a blank/whitespace-only name")
	}
}

func TestRegenerateIssuerIdentity(t *testing.T) {
	isolateIdentityConfig(t)
	a := NewApp()

	original, err := a.SetIssuerName("Jean Dupont")
	if err != nil {
		t.Fatalf("SetIssuerName: %v", err)
	}

	regenerated, err := a.RegenerateIssuerIdentity("Jean Dupont (new key)")
	if err != nil {
		t.Fatalf("RegenerateIssuerIdentity: %v", err)
	}
	if regenerated.Name != "Jean Dupont (new key)" {
		t.Errorf("Name = %q, want %q", regenerated.Name, "Jean Dupont (new key)")
	}
	if regenerated.Fingerprint == original.Fingerprint {
		t.Error("RegenerateIssuerIdentity must mint a fresh key: fingerprint should differ from the original")
	}

	// The regenerated identity is what's now on disk.
	got, err := a.GetIssuerIdentity()
	if err != nil {
		t.Fatalf("GetIssuerIdentity after regenerate: %v", err)
	}
	if got.Fingerprint != regenerated.Fingerprint {
		t.Errorf("GetIssuerIdentity fingerprint = %q, want the regenerated one %q", got.Fingerprint, regenerated.Fingerprint)
	}
}

func TestRegenerateIssuerIdentityDefaultsBlankName(t *testing.T) {
	isolateIdentityConfig(t)
	a := NewApp()

	info, err := a.RegenerateIssuerIdentity("")
	if err != nil {
		t.Fatalf("RegenerateIssuerIdentity: %v", err)
	}
	if info.Name == "" {
		t.Error("expected a non-empty default name when regenerating with a blank name")
	}
}

// TestImportIssuerIdentityRoundTrip exercises export/import at the issuance
// layer directly (issuance.Identity.ExportIdentity), so it stays clear of the
// Wails dialog in App.ExportIssuerIdentity (which needs a live a.ctx), while
// still exercising the App-level, dialog-free ImportIssuerIdentity.
func TestImportIssuerIdentityRoundTrip(t *testing.T) {
	isolateIdentityConfig(t)

	// Produce an identity file "on another machine": a throwaway config dir,
	// exported without a passphrase.
	sourceDir := t.TempDir()
	source, err := issuance.LoadOrCreateIdentity(sourceDir, "Kévin Unger")
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity (source): %v", err)
	}
	exportPath := filepath.Join(t.TempDir(), "identity.bdbid")
	if err := source.ExportIdentity(exportPath, ""); err != nil {
		t.Fatalf("ExportIdentity: %v", err)
	}

	a := NewApp()
	info, err := a.ImportIssuerIdentity(exportPath, "")
	if err != nil {
		t.Fatalf("ImportIssuerIdentity: %v", err)
	}
	if info.Name != "Kévin Unger" {
		t.Errorf("Name = %q, want %q", info.Name, "Kévin Unger")
	}
	if info.Fingerprint != source.Fingerprint() {
		t.Errorf("Fingerprint = %q, want %q (importing must carry the same key)", info.Fingerprint, source.Fingerprint())
	}
}

func TestImportIssuerIdentityRoundTripWithPassphrase(t *testing.T) {
	isolateIdentityConfig(t)

	sourceDir := t.TempDir()
	source, err := issuance.LoadOrCreateIdentity(sourceDir, "Kévin Unger")
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity (source): %v", err)
	}
	exportPath := filepath.Join(t.TempDir(), "identity.bdbid")
	if err := source.ExportIdentity(exportPath, "correct horse battery staple"); err != nil {
		t.Fatalf("ExportIdentity: %v", err)
	}

	a := NewApp()

	if _, err := a.ImportIssuerIdentity(exportPath, "wrong passphrase"); err == nil {
		t.Error("expected ImportIssuerIdentity to reject the wrong passphrase")
	}
	if _, err := a.ImportIssuerIdentity(exportPath, ""); err == nil {
		t.Error("expected ImportIssuerIdentity to require a passphrase for a protected file")
	}

	info, err := a.ImportIssuerIdentity(exportPath, "correct horse battery staple")
	if err != nil {
		t.Fatalf("ImportIssuerIdentity with correct passphrase: %v", err)
	}
	if info.Fingerprint != source.Fingerprint() {
		t.Errorf("Fingerprint = %q, want %q", info.Fingerprint, source.Fingerprint())
	}
}

func TestImportIssuerIdentityReplacesExisting(t *testing.T) {
	isolateIdentityConfig(t)
	a := NewApp()

	before, err := a.SetIssuerName("Local Identity")
	if err != nil {
		t.Fatalf("SetIssuerName: %v", err)
	}

	sourceDir := t.TempDir()
	source, err := issuance.LoadOrCreateIdentity(sourceDir, "Imported Identity")
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity (source): %v", err)
	}
	exportPath := filepath.Join(t.TempDir(), "identity.bdbid")
	if err := source.ExportIdentity(exportPath, ""); err != nil {
		t.Fatalf("ExportIdentity: %v", err)
	}

	info, err := a.ImportIssuerIdentity(exportPath, "")
	if err != nil {
		t.Fatalf("ImportIssuerIdentity: %v", err)
	}
	if info.Fingerprint == before.Fingerprint {
		t.Error("importing should replace the local identity, not keep it")
	}
	if info.Name != "Imported Identity" {
		t.Errorf("Name = %q, want %q", info.Name, "Imported Identity")
	}
}
