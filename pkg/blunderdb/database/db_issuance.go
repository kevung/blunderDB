package database

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
)

// This file is the database glue for pkg/blunderdb/issuance: reading and writing the single
// watermark document that lives in `metadata`, and opening a protected copy. All the
// reasoning lives in the issuance package doc and in
// docs/adr/0007-watermarks-mark-origin-and-nothing-else.md; what follows is plumbing.
//
// Note what is absent, and deliberately so: there is no write path on the recipient's side.
// Opening a watermarked database records nothing, anywhere.

// readMeta returns a metadata value, treating an absent row as empty. The watermark row is
// absent from most databases, so absence is the normal case and not an error.
func (d *Database) readMeta(key string) (string, error) {
	var value string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// GetIssuanceInfo reports where this database says it comes from. It never writes.
func (d *Database) GetIssuanceInfo() (IssuanceInfo, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return IssuanceInfo{}, fmt.Errorf("no database is currently open")
	}
	raw, err := d.readMeta(issuance.KeyWatermark)
	if err != nil {
		return IssuanceInfo{}, err
	}
	env, err := issuance.DecodeEnvelope(raw)
	if err != nil {
		return IssuanceInfo{}, err
	}
	return buildIssuanceInfo(env), nil
}

// buildIssuanceInfo turns a stored envelope into what the UI and the CLI display, adding this
// machine's own identity when one exists. The identity is only *read* here: looking at a
// panel must not create one as a side effect — that happens on the first watermarked export
// and nowhere else.
func buildIssuanceInfo(env issuance.Envelope) IssuanceInfo {
	info := IssuanceInfo{Watermarked: env.IsWatermarked()}
	var mine string
	if id, err := issuance.LoadIdentity(issuance.ConfigDir()); err == nil && id != nil {
		mine = id.Fingerprint()
		info.IssuerFingerprint = mine
		info.IssuerName = id.Name
	}
	if info.Watermarked {
		wm := watermarkInfo(env, mine)
		info.Watermark = &wm
	}
	return info
}

func watermarkInfo(e issuance.Envelope, myFingerprint string) WatermarkInfo {
	info := WatermarkInfo{
		IssuerFingerprint: e.Fingerprint(),
		SignatureValid:    e.Verify(),
	}
	w, err := e.Open()
	if err != nil {
		return info
	}
	info.Origin = w.Origin
	info.IssuerName = w.IssuerName
	info.Note = w.Note
	info.IssuedAt = w.IssuedAt
	info.IssuedByYou = myFingerprint != "" && e.VerifiedBy(myFingerprint)
	return info
}

// IssuerIdentity returns this machine's signing identity, creating it silently the first time
// a watermark is applied. Nothing in the UI asks the user to configure it.
func IssuerIdentity() (*issuance.Identity, error) {
	return issuance.LoadOrCreateIdentity(issuance.ConfigDir(), issuance.DefaultIssuerName())
}

// sealWatermark builds the document an export writes, or "" when the export carries no
// watermark. It is the only place an identity is ever created.
func sealWatermark(origin, note string) (string, error) {
	if origin == "" {
		return "", nil
	}
	identity, err := IssuerIdentity()
	if err != nil {
		return "", err
	}
	env, err := issuance.Seal(identity, issuance.Watermark{Origin: origin, Note: note})
	if err != nil {
		return "", err
	}
	return issuance.EncodeEnvelope(env)
}

// IsProtectedCopy reports whether a path is a password-protected copy rather than an ordinary
// database, so a caller can ask for the password instead of failing to open a file that was
// never meant to be opened directly.
func IsProtectedCopy(path string) bool { return issuance.IsContainer(path) }

// OpenProtectedCopy turns a protected copy into an ordinary database and returns its path.
// This is the one time a password is ever asked for: from then on the recipient works with a
// normal file, with no prompt and no re-encryption.
//
// The result lands beside the container with a `.db` extension. An existing file is returned
// as-is rather than overwritten — a second open would otherwise silently discard the work
// already done in the database from the first one.
func OpenProtectedCopy(path, password string) (string, error) {
	target := issuance.DefaultUnwrapPath(path)
	if _, err := os.Stat(target); err == nil {
		// The copy was opened here before, so the database already exists and must not be
		// overwritten — someone may have worked in it since. But it must NOT be handed back
		// without checking the password first: doing so let any password through from the
		// second open onwards, which is no protection at all.
		if err := issuance.VerifyPassword(path, password); err != nil {
			return "", err
		}
		return target, nil
	}
	if _, err := issuance.UnwrapContainer(path, target, password); err != nil {
		return "", err
	}
	return target, nil
}

// IsProtectedCopyPath is the bound form of IsProtectedCopy, for the frontend.
func (d *Database) IsProtectedCopyPath(path string) bool { return IsProtectedCopy(path) }

// OpenProtectedCopyPath is the bound form of OpenProtectedCopy, for the frontend.
func (d *Database) OpenProtectedCopyPath(path, password string) (string, error) {
	return OpenProtectedCopy(path, password)
}

// InspectIssuance reports where a file on disk says it comes from, without opening it as a
// working database — and without writing to it.
//
// It reads a protected copy's cleartext header too, so the origin of a file stays readable
// even when its password is not to hand.
func InspectIssuance(path string) (IssuanceInfo, error) {
	if issuance.IsContainer(path) {
		header, err := issuance.ReadContainerHeader(path)
		if err != nil {
			return IssuanceInfo{}, err
		}
		return buildIssuanceInfo(header.Watermark), nil
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return IssuanceInfo{}, err
	}
	defer db.Close()

	var raw string
	if err := db.QueryRow(`SELECT value FROM metadata WHERE key = ?`, issuance.KeyWatermark).Scan(&raw); err != nil {
		// No row, or no metadata table at all: simply not a watermarked database.
		return buildIssuanceInfo(issuance.Envelope{}), nil
	}
	env, err := issuance.DecodeEnvelope(raw)
	if err != nil {
		return IssuanceInfo{}, err
	}
	return buildIssuanceInfo(env), nil
}
