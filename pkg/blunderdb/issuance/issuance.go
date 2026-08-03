// Package issuance covers what happens when a database is produced to be handed out:
// marking where it comes from, and protecting it while it travels.
//
// # Two independent, optional mechanisms
//
//   - A **Watermark** states the origin of a database — who produced it and for what. It
//     is signed with the producer's Issuer identity, so it cannot be altered and cannot be
//     fabricated in someone else's name.
//   - A **password** wraps the file in an encrypted container, so a stray copy cannot be
//     opened by whoever happens to find it.
//
// Both are chosen at export, both are optional, and they combine freely.
//
// # What this is not
//
// Nothing here tracks a recipient, a machine, or a share. **The recipient's side writes
// nothing**: opening a watermarked database is exactly like opening any other, and no
// register, log or counter grows anywhere. Earlier iterations of this package carried a
// per-recipient watermark, a holder registry and an import lineage; they were removed
// deliberately. In practice an author does not litigate over a position database, and a
// mechanism that quietly records who opened what costs far more — in trust, in privacy
// obligations, in code — than the disputes it would settle.
//
// A Watermark therefore says *where this came from*, not *who leaked it*. It is
// tamper-evident and unforgeable; it is not unremovable, and it prevents nothing. See
// docs/adr/0007-watermarks-mark-origin-and-nothing-else.md.
//
// # Why a document and not a table
//
// The Watermark is stored as canonical JSON in a `metadata` key/value row, never in a table
// of its own. That keeps a file-level concern out of the schema (no DatabaseVersion bump, no
// PostgreSQL migration, nothing for the serve daemon), and — the decisive reason — it means
// the signature is always checked against the exact bytes that were signed. Rebuilding a
// byte-identical document from normalised columns is the classic source of signature bugs
// that surfaces months later on an accent or an empty field; storing the signed bytes
// verbatim removes the class entirely.
//
// This package is pure: no SQL, and no filesystem beyond the identity file and the
// container. The database glue lives in pkg/blunderdb/database/db_issuance.go.
package issuance

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// KeyWatermark is the metadata key the sealed Watermark lives under. It is deliberately
// absent from every schema file: a database that was never watermarked carries no such row.
const KeyWatermark = "watermark"

// CarriedMetadataKeys is the allow-list of ordinary metadata copied into an exported file.
// The export path copies these and nothing else, then writes the Watermark itself.
//
// An allow-list rather than a deny-list is deliberate: an exported file is handed to someone
// else, and a document added to `metadata` in six months must not travel by default just
// because nobody remembered to exclude it.
var CarriedMetadataKeys = []string{"user", "description", "dateOfCreation"}

// Carried returns the subset of md that may travel inside an exported file.
func Carried(md map[string]string) map[string]string {
	out := make(map[string]string, len(CarriedMetadataKeys))
	for _, k := range CarriedMetadataKeys {
		if v := md[k]; v != "" {
			out[k] = v
		}
	}
	return out
}

// Watermark is the producer's signed statement about where a database comes from.
//
// It names no recipient. Whatever the producer writes in Origin and Note is what travels —
// nothing is derived from the machine, the reader, or the file's later history.
//
// Field order is the signed byte order: appending a field is safe (existing Envelopes keep
// verifying against their own stored payload), reordering or renaming one is not.
type Watermark struct {
	Version int `json:"version"`
	// Origin says what this database is and where it comes from, in the producer's own
	// words: "Cours de Jean Dupont — 12 mars 2026".
	Origin     string `json:"origin"`
	IssuerName string `json:"issuerName"`
	// Note is free text the producer wants attached to the file — terms of use, a contact
	// address, a request not to redistribute. Optional.
	Note     string `json:"note,omitempty"`
	IssuedAt string `json:"issuedAt"`
}

// Envelope is a Watermark together with the signature over its exact bytes. Payload is
// stored verbatim and is the only thing ever verified against.
type Envelope struct {
	Alg       string `json:"alg"`
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
	PublicKey string `json:"publicKey"`
}

const algEd25519 = "ed25519"

// Seal signs w with the producer's identity and returns the Envelope to store.
func Seal(id *Identity, w Watermark) (Envelope, error) {
	if id == nil {
		return Envelope{}, fmt.Errorf("no issuer identity")
	}
	if strings.TrimSpace(w.Origin) == "" {
		return Envelope{}, fmt.Errorf("a watermark needs an origin")
	}
	if w.Version == 0 {
		w.Version = 1
	}
	if w.IssuedAt == "" {
		w.IssuedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if w.IssuerName == "" {
		w.IssuerName = id.Name
	}
	payload, err := json.Marshal(w)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{
		Alg:       algEd25519,
		Payload:   string(payload),
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(id.priv, payload)),
		PublicKey: base64.StdEncoding.EncodeToString(id.priv.Public().(ed25519.PublicKey)),
	}, nil
}

// Open returns the Watermark carried by the Envelope, whether or not the signature checks
// out. Callers that display the content must report Verify alongside it: an unverified
// watermark claims an origin nobody vouched for.
func (e Envelope) Open() (Watermark, error) {
	var w Watermark
	if err := json.Unmarshal([]byte(e.Payload), &w); err != nil {
		return Watermark{}, fmt.Errorf("unreadable watermark: %w", err)
	}
	return w, nil
}

// Verify reports whether the signature matches the payload bytes and the embedded key. It
// proves the Watermark was produced by the holder of that key and has not been altered —
// not that the key belongs to the person named in it. Only comparison against a known
// fingerprint does that, which is what VerifiedBy is for.
func (e Envelope) Verify() bool {
	if e.Alg != algEd25519 {
		return false
	}
	pub, err := base64.StdEncoding.DecodeString(e.PublicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return false
	}
	sig, err := base64.StdEncoding.DecodeString(e.Signature)
	if err != nil {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(pub), []byte(e.Payload), sig)
}

// VerifiedBy reports whether the Envelope verifies *and* was signed by the given
// fingerprint — the check a producer runs against their own identity to confirm a file is
// theirs, and the check a recipient runs against a fingerprint the producer published.
func (e Envelope) VerifiedBy(fingerprint string) bool {
	return e.Verify() && e.Fingerprint() == fingerprint
}

// Fingerprint is the display form of the signing key, e.g. "A3F1-9C24-7B05-E1D8" — what a
// producer can publish so recipients can check where a file came from.
func (e Envelope) Fingerprint() string {
	pub, err := base64.StdEncoding.DecodeString(e.PublicKey)
	if err != nil {
		return ""
	}
	return FormatFingerprint(pub)
}

// IsWatermarked reports whether the Envelope carries anything at all.
func (e Envelope) IsWatermarked() bool { return e.Payload != "" }

// EncodeEnvelope renders an Envelope for storage in a metadata row. An unwatermarked
// Envelope encodes to "" so no row is written for it.
func EncodeEnvelope(e Envelope) (string, error) {
	if !e.IsWatermarked() {
		return "", nil
	}
	b, err := json.Marshal(e)
	return string(b), err
}

// DecodeEnvelope parses a metadata row back into an Envelope. An empty or absent value
// yields the zero Envelope and no error: not being watermarked is the normal case.
func DecodeEnvelope(s string) (Envelope, error) {
	if strings.TrimSpace(s) == "" {
		return Envelope{}, nil
	}
	var e Envelope
	if err := json.Unmarshal([]byte(s), &e); err != nil {
		return Envelope{}, fmt.Errorf("unreadable watermark document: %w", err)
	}
	return e, nil
}

// FormatFingerprint renders key material as four dash-separated groups of four hex digits,
// the form a person can read aloud or write on a whiteboard.
func FormatFingerprint(key []byte) string {
	sum := hashBytes(key)
	h := strings.ToUpper(hex.EncodeToString(sum[:8]))
	var parts []string
	for i := 0; i < len(h); i += 4 {
		parts = append(parts, h[i:i+4])
	}
	return strings.Join(parts, "-")
}
