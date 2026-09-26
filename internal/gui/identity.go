package gui

import (
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// The issuer identity signs every watermark. It belongs to a person, not a database, so it
// lives in the config directory; it is created on the first watermarked export.

// GetIssuerIdentity reports this machine's identity without creating one.
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

// SetIssuerName changes the name future watermarks carry; the key is untouched, so existing
// marks keep their name and keep verifying.
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

// ExportIssuerIdentity writes the identity to a file the user picks, optionally protected by
// a passphrase (the travelling copy is the exposed one). Returns "" on cancel.
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

// RegenerateIssuerIdentity replaces the signing key, overwriting the old one. It revokes
// nothing: a watermark carries its public key and verifies for ever, so a leaked key keeps
// signing valid marks. The caller must say so and offer to save the old identity first.
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

// ImportIssuerIdentity installs a transferred identity, replacing any existing one: one
// person, one fingerprint across machines.
func (a *App) ImportIssuerIdentity(path, passphrase string) (domain.IssuerIdentityInfo, error) {
	if _, err := issuance.ImportIdentity(issuance.ConfigDir(), path, passphrase); err != nil {
		return domain.IssuerIdentityInfo{}, err
	}
	return a.GetIssuerIdentity()
}
