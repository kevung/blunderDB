// Package issuance implements the controlled distribution of a blunderDB database:
// the Watermark an Issuer seals into an Issued copy, the Issuer identity it is signed
// with, the Holder registry a copy accumulates as it travels, and the Lineage an
// ordinary database inherits when it imports Issued copies.
//
// # The promise
//
// A Watermark is tamper-evident and unforgeable, never unremovable. blunderDB is
// MIT-licensed and its distributed artefact is a plain SQLite file, so a Holder who
// sets out to delete a Watermark will succeed; nothing here pretends otherwise. What
// signing buys is that a Watermark cannot be altered and cannot be fabricated in
// someone else's name — so a leaked copy can be attributed, and nobody can be accused
// wrongly. See docs/adr/0007-issued-copies-are-attributable-never-unshareable.md.
//
// # Why documents and not tables
//
// Everything here is stored as canonical JSON in `metadata` key/value rows, never in
// tables of its own. That keeps a file-level concern out of the schema (no
// DatabaseVersion bump, no PostgreSQL migration, nothing for the serve daemon), and —
// the decisive reason — it means a signature is always checked against the exact bytes
// that were signed. Rebuilding a byte-identical document from normalised columns is the
// classic source of signature bugs that surfaces months later on an accent or an empty
// field; storing the signed bytes verbatim removes the class entirely.
//
// This package is pure: no SQL, no filesystem beyond the identity file and the
// container. The database glue lives in pkg/blunderdb/database/db_issuance.go.
package issuance

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Metadata keys under which the four documents live. They are deliberately absent from
// every schema file: a database that was never issued and never imported an Issued copy
// carries none of them.
const (
	// KeyWatermark holds the sealed Envelope identifying this file as an Issued copy.
	KeyWatermark = "watermark"
	// KeyHolders holds the Holder registry — the machines that opened this copy.
	KeyHolders = "holders"
	// KeyLineage holds the Watermarks inherited by importing Issued copies.
	KeyLineage = "lineage"
	// KeyIssued holds the Issuer's own Issue register. It never leaves the database it
	// was written in; exporting it would hand a recipient the list of every other
	// recipient and the passwords of every Distribution.
	KeyIssued = "issued"
)

// CarriedMetadataKeys is the allow-list of ordinary metadata copied into an Issued copy.
// The export path copies these and nothing else, then writes the issuance documents
// itself. An allow-list rather than a deny-list is deliberate: a document added in six
// months must not leak by default, and what is at stake here is recipient names and
// Distribution passwords.
var CarriedMetadataKeys = []string{"user", "description", "dateOfCreation"}

// Carried returns the subset of md that may travel inside an Issued copy.
func Carried(md map[string]string) map[string]string {
	out := make(map[string]string, len(CarriedMetadataKeys))
	for _, k := range CarriedMetadataKeys {
		if v := md[k]; v != "" {
			out[k] = v
		}
	}
	return out
}

// Watermark is the Issuer's assertion about one Issued copy. Recipient is empty in the
// collective regime, where the copy is issued to a Distribution rather than to a person.
//
// Field order is the signed byte order: appending a field is safe (old Envelopes keep
// verifying against their own stored payload), reordering or renaming one is not.
type Watermark struct {
	Version      int    `json:"version"`
	Distribution string `json:"distribution"`
	IssuerName   string `json:"issuerName"`
	Recipient    string `json:"recipient,omitempty"`
	Number       int    `json:"number,omitempty"`
	Total        int    `json:"total,omitempty"`
	IssuedAt     string `json:"issuedAt"`
	// Salt is per Distribution, not per copy: Holder fingerprints are comparable
	// between the copies of one course — "these two leaks came from the same machine" —
	// and meaningless outside it, so no global machine identifier is ever constituted.
	Salt string `json:"salt"`
	// CopyID distinguishes two copies issued to the same recipient in one Distribution.
	CopyID string `json:"copyId"`
}

// Nominative reports whether the copy names a recipient. A collective copy is attributable
// only through its Holder registry.
func (w Watermark) Nominative() bool { return strings.TrimSpace(w.Recipient) != "" }

// Envelope is a Watermark together with the signature over its exact bytes. Payload is
// stored verbatim and is the only thing ever verified against.
type Envelope struct {
	Alg       string `json:"alg"`
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
	PublicKey string `json:"publicKey"`
}

const algEd25519 = "ed25519"

// Seal signs w with the Issuer identity and returns the Envelope to store.
func Seal(id *Identity, w Watermark) (Envelope, error) {
	if id == nil {
		return Envelope{}, fmt.Errorf("no issuer identity")
	}
	if w.Version == 0 {
		w.Version = 1
	}
	if w.IssuedAt == "" {
		w.IssuedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if w.Salt == "" {
		salt, err := randomHex(16)
		if err != nil {
			return Envelope{}, err
		}
		w.Salt = salt
	}
	if w.CopyID == "" {
		cid, err := randomHex(8)
		if err != nil {
			return Envelope{}, err
		}
		w.CopyID = cid
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
// out. Callers that act on the content must report Verify alongside it: a Watermark whose
// signature does not verify names a recipient nobody vouched for.
func (e Envelope) Open() (Watermark, error) {
	var w Watermark
	if err := json.Unmarshal([]byte(e.Payload), &w); err != nil {
		return Watermark{}, fmt.Errorf("unreadable watermark: %w", err)
	}
	return w, nil
}

// Verify reports whether the signature matches the payload bytes and the embedded key.
// It proves the Watermark was produced by the holder of that key and has not been
// altered — not that the key belongs to the person named in it. Only comparison against
// a known fingerprint does that, which is what VerifiedBy is for.
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
// fingerprint. In practice the person examining a leaked copy is the person who signed
// it, so passing their own identity's fingerprint closes the loop locally, without
// anyone having had to publish anything.
func (e Envelope) VerifiedBy(fingerprint string) bool {
	return e.Verify() && e.Fingerprint() == fingerprint
}

// Fingerprint is the display form of the signing key, e.g. "A3F1-9C24-7B05-E1D8". It is
// what an Issuer can communicate out of band so a recipient can check where a file came
// from.
func (e Envelope) Fingerprint() string {
	pub, err := base64.StdEncoding.DecodeString(e.PublicKey)
	if err != nil {
		return ""
	}
	return FormatFingerprint(pub)
}

// EncodeEnvelope renders an Envelope for storage in a metadata row.
func EncodeEnvelope(e Envelope) (string, error) {
	b, err := json.Marshal(e)
	return string(b), err
}

// DecodeEnvelope parses a metadata row back into an Envelope. An empty or absent value
// yields the zero Envelope and no error: not being an Issued copy is the normal case.
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

// IsIssued reports whether the Envelope carries anything at all.
func (e Envelope) IsIssued() bool { return e.Payload != "" }

// Lineage is the ordered list of Watermarks a database inherited by importing Issued
// copies. Importing a course into one's own database is a normal, supported gesture and
// also the easiest way to strip a Watermark; carrying the Envelopes verbatim keeps the
// trace attached to the result, so the default path through the product does not launder
// it. Because the Envelopes travel unchanged, they still verify against their original
// Issuer.
type Lineage []Envelope

// EncodeLineage renders a Lineage for storage. An empty Lineage encodes to "" so that no
// row is written for the overwhelmingly common case.
func EncodeLineage(l Lineage) (string, error) {
	if len(l) == 0 {
		return "", nil
	}
	b, err := json.Marshal(l)
	return string(b), err
}

// DecodeLineage parses a stored Lineage.
func DecodeLineage(s string) (Lineage, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	var l Lineage
	if err := json.Unmarshal([]byte(s), &l); err != nil {
		return nil, fmt.Errorf("unreadable lineage document: %w", err)
	}
	return l, nil
}

// Append adds an Envelope to the Lineage unless an identical copy is already recorded.
// Re-importing the same Issued copy must not grow the list without bound.
func (l Lineage) Append(e Envelope) Lineage {
	if !e.IsIssued() {
		return l
	}
	for _, existing := range l {
		if existing.Signature == e.Signature {
			return l
		}
	}
	return append(l, e)
}

// Inherit merges what an imported file carried — its own Watermark and its own Lineage —
// into the receiving database's Lineage.
func (l Lineage) Inherit(sourceWatermark Envelope, sourceLineage Lineage) Lineage {
	out := l
	for _, e := range sourceLineage {
		out = out.Append(e)
	}
	return out.Append(sourceWatermark)
}

// FormatFingerprint renders key material as four dash-separated groups of four hex
// digits, the form a person can read aloud or copy off a whiteboard.
func FormatFingerprint(key []byte) string {
	sum := hashBytes(key)
	h := strings.ToUpper(hex.EncodeToString(sum[:8]))
	var parts []string
	for i := 0; i < len(h); i += 4 {
		parts = append(parts, h[i:i+4])
	}
	return strings.Join(parts, "-")
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("no entropy available: %w", err)
	}
	return hex.EncodeToString(b), nil
}
