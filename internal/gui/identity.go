package gui

import (
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// The issuer identity is the key every watermark is signed with. It belongs to a person
// rather than to a database, which is why it lives in the config directory and is managed
// from the settings screen rather than from a database panel.
//
// It is created without being asked for — on the first watermarked export — so these
// methods are only ever about *seeing* it and *moving it to another machine*.

// GetIssuerIdentity reports this machine's identity. It deliberately does not create one:
// opening the settings must not mint a key for someone who never watermarks anything.
func (a *App) GetIssuerIdentity() (domain.IssuerIdentityInfo, error) {
	dir := issuance.ConfigDir()
	id, err := issuance.LoadIdentity(dir)
	if err != nil {
		return domain.IssuerIdentityInfo{}, err
	}
	if id == nil {
		return domain.IssuerIdentityInfo{Path: issuance.IdentityPath(dir)}, nil
	}
	return domain.IssuerIdentityInfo{
		Present:     true,
		Name:        id.Name,
		Fingerprint: id.Fingerprint(),
		Path:        issuance.IdentityPath(dir),
	}, nil
}

// SetIssuerName changes the display name future watermarks carry. Watermarks already
// applied keep the name they were sealed with — that is the point of sealing them — and the
// key itself is untouched, so everything already marked keeps verifying.
func (a *App) SetIssuerName(name string) (domain.IssuerIdentityInfo, error) {
	if strings.TrimSpace(name) == "" {
		return domain.IssuerIdentityInfo{}, fmt.Errorf("the issuer name cannot be empty")
	}
	dir := issuance.ConfigDir()
	id, err := issuance.LoadIdentity(dir)
	if err != nil {
		return domain.IssuerIdentityInfo{}, err
	}
	if id == nil {
		// Naming yourself is an explicit gesture; minting the key here is expected.
		if id, err = issuance.LoadOrCreateIdentity(dir, name); err != nil {
			return domain.IssuerIdentityInfo{}, err
		}
	}
	if err := id.Rename(dir, name); err != nil {
		return domain.IssuerIdentityInfo{}, err
	}
	return a.GetIssuerIdentity()
}

// ExportIssuerIdentity writes the identity to a file the user picks, so it can be carried to
// another machine. The passphrase is optional and applies only to that file: it is the copy
// that travels by mail or on a USB stick, which is the exposed one.
//
// Returns the chosen path, or "" when the user cancels the dialog.
func (a *App) ExportIssuerIdentity(passphrase string) (string, error) {
	dir := issuance.ConfigDir()
	// Asking to save the identity is explicit enough to create it if it does not exist yet.
	id, err := issuance.LoadOrCreateIdentity(dir, issuance.DefaultIssuerName())
	if err != nil {
		return "", err
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:                "Export blunderDB identity",
		DefaultFilename:      issuance.FileSlug(id.Name) + issuance.IdentityFileExtension,
		Filters:              []runtime.FileFilter{{DisplayName: "blunderDB identity (*" + issuance.IdentityFileExtension + ")", Pattern: "*" + issuance.IdentityFileExtension}},
		CanCreateDirectories: true,
	})
	if err != nil || path == "" {
		return "", err
	}
	if !strings.HasSuffix(strings.ToLower(path), issuance.IdentityFileExtension) {
		path += issuance.IdentityFileExtension
	}
	if err := id.ExportIdentity(path, passphrase); err != nil {
		return "", err
	}
	return path, nil
}

// PickIdentityFile asks for the file to import and reports whether it is protected, so the
// caller prompts for a passphrase only when one is actually needed.
func (a *App) PickIdentityFile() (domain.IdentityFilePick, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Import blunderDB identity",
		Filters: []runtime.FileFilter{{DisplayName: "blunderDB identity (*" + issuance.IdentityFileExtension + ")", Pattern: "*" + issuance.IdentityFileExtension}},
	})
	if err != nil {
		return domain.IdentityFilePick{}, err
	}
	if path == "" {
		return domain.IdentityFilePick{Cancelled: true}, nil
	}
	needs, err := issuance.IdentityFileNeedsPassphrase(path)
	if err != nil {
		return domain.IdentityFilePick{}, err
	}
	return domain.IdentityFilePick{Path: path, NeedsPassphrase: needs}, nil
}

// RegenerateIssuerIdentity replaces the signing key with a fresh one.
//
// It is worth being exact about what this does not do. A watermark carries the public key it
// was signed with, so it verifies for ever, on its own. Regenerating therefore **revokes
// nothing**: if the old identity file leaked, whoever holds it can keep signing under the
// old fingerprint and those marks stay valid. What protects a producer after a leak is
// social — publishing the new fingerprint and disowning the old one — not this function.
//
// It is offered for the cases that remain: starting over, or moving to a fresh identity on
// purpose. The caller is expected to have said all of the above and to have offered to save
// the old identity first, because this overwrites it.
func (a *App) RegenerateIssuerIdentity(name string) (domain.IssuerIdentityInfo, error) {
	if strings.TrimSpace(name) == "" {
		name = issuance.DefaultIssuerName()
	}
	id, err := issuance.NewIdentity(name)
	if err != nil {
		return domain.IssuerIdentityInfo{}, err
	}
	if err := id.Rename(issuance.ConfigDir(), name); err != nil {
		return domain.IssuerIdentityInfo{}, err
	}
	return a.GetIssuerIdentity()
}

// ImportIssuerIdentity installs a transferred identity on this machine, replacing any
// existing one. Holding the same identity on two machines is the intended outcome: it is one
// person, and everything they mark from either machine carries one fingerprint.
func (a *App) ImportIssuerIdentity(path, passphrase string) (domain.IssuerIdentityInfo, error) {
	if _, err := issuance.ImportIdentity(issuance.ConfigDir(), path, passphrase); err != nil {
		return domain.IssuerIdentityInfo{}, err
	}
	return a.GetIssuerIdentity()
}
