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
// register, log or counter grows anywhere. Per-recipient watermarks, a holder registry and an
// import lineage were deliberately removed: recording who opened what costs more in trust and
// privacy than the disputes it would settle.
//
// A Watermark therefore says *where this came from*, not *who leaked it*. It is
// tamper-evident and unforgeable; it is not unremovable, and it prevents nothing. See
// docs/adr/0007-watermarks-mark-origin-and-nothing-else.md.
//
// # Why a document and not a table
//
// The Watermark is stored as canonical JSON in a `metadata` key/value row, never in a table
// of its own. That keeps a file-level concern out of the schema (no DatabaseVersion bump, no
// PostgreSQL migration), and — decisively — the signature is always checked against the
// exact bytes that were signed. Rebuilding them from normalised columns is a classic source of
// signature bugs (an accent, an empty field); storing them verbatim removes the class.
//
// # The container, and what its cleartext header is worth
//
// A protected copy (`.dbx`) is a cleartext prefix — magic bytes, a length, a JSON header
// carrying the Watermark, the KDF name with its Argon2id parameters, salt and nonce — followed
// by the database sealed with AES-256-GCM under a key derived from the passphrase. The header
// is clear so that the origin of a file stays readable without the password, and readable on
// a file nobody can open any more.
//
// Clear is not the same as unprotected. Container version 2 passes the whole prefix, byte for
// byte, as the AEAD's additional data, so relabelling a file — swapping its Watermark for
// someone else's — fails to open exactly as a wrong passphrase does. Version 1 did not bind
// the header; it is still read, with a logged warning, and never written.
//
// Every read is bounded before it allocates: the header by maxContainerHeader, the payload by
// maxContainerPayload (2 GiB) checked against the file's size. AES-GCM authenticates a whole
// message before releasing plaintext, so a container is opened in one piece, decrypted in
// place (peak: one copy of the file). Streaming would need a chunked AEAD and a format v3.
//
// Argon2id's cost parameters are written into every protected file and identity export. A
// file recording parameters this build never used is refused by name rather than tried: a
// wrong cost would only report a wrong passphrase.
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
	"net/url"
	"strings"
	"time"
)

// KeyWatermark is the metadata key the sealed Watermark lives under. It is deliberately
// absent from every schema file: a database that was never watermarked carries no such row.
const KeyWatermark = "watermark"

// CarriedMetadataKeys is the allow-list of ordinary metadata copied into an exported file.
// The export path copies these and nothing else, then writes the Watermark itself.
//
// An allow-list, not a deny-list: a document added to `metadata` later must not travel by
// default because nobody remembered to exclude it.
var CarriedMetadataKeys = []string{"user", "description", "dateOfCreation"}

// CarriedCommentColumns is the allow-list of comment columns an exported file
// carries, for the same reason: a column added to `comment` later stays home
// until it is named here. The author is the producer's statement about who
// wrote the note, so it travels (ADR-0007). The origin travels too: a remark
// an XG file brought the producer stays an importer's remark for the
// recipient, and only the producer's own notes hold their position there
// (ADR-0099). The signature is the producer's sealed statement over text,
// author and position (SealComment).
var CarriedCommentColumns = []string{"text", "created_at", "modified_at", "author", "origin", "signature"}

// CarriedMatchCommentColumns is the same allow-list for the note a match
// carries itself: its text and who signs it — the producer's statement, or
// the author the source file named for its header or footer comment.
var CarriedMatchCommentColumns = []string{"comment", "comment_author"}

// CarriedVideoSource is the video source an exported match carries: an
// http(s) URL without the credentials it may embed, a local file by its name
// alone, nil when there is none (ADR-0082). The name lets the recipient
// relocate the match onto their own copy of the video; the directories above
// it are the producer's and stay home. The Repères keep their meaning either
// way.
//
// Limit: a token written in the URL's path (/s/<token>/video.mp4, the share
// links of Dropbox or Drive) is indistinguishable from a path and is not
// removed.
func CarriedVideoSource(src *string) *string {
	if src == nil {
		return nil
	}
	v := strings.TrimSpace(*src)
	lower := strings.ToLower(v)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		// Anything that is not exactly an http(s) URL (//host/x, host/x,
		// https:/h/x) is read as a path, but may still be a URL typed
		// loosely: the query and the fragment are cut before the name is
		// taken, so a signed token cannot ride along. A local file whose
		// name holds ? or # loses the tail of its name; that is the price of
		// never carrying a token, and the name is kept rather than dropped.
		if i := strings.IndexAny(v, "?#"); i >= 0 {
			v = v[:i]
		}
		v = CarriedFileName(v)
	} else if u, err := url.Parse(v); err != nil {
		// A URL that does not parse cannot be told free of credentials.
		return nil
	} else {
		q, f := keepVideoParams(u.RawQuery), keepVideoParams(u.Fragment)
		if u.User != nil || q != u.RawQuery || f != u.Fragment {
			u.User, u.RawQuery, u.Fragment, u.RawFragment = nil, q, f, ""
			v = u.String()
		}
	}
	if v == "" {
		return nil
	}
	return &v
}

// carriedVideoParams are the query parameters that locate a moment of a
// public video: the video (v), the playlist and its position (list, index),
// the start time (t, start, end) and the share id YouTube appends (si). Any
// other parameter of a URL is the producer's: a signed link (S3 X-Amz-*,
// Drive) carries its access token there, so what is not named stays home.
var carriedVideoParams = map[string]bool{
	"v": true, "t": true, "list": true, "si": true,
	"start": true, "end": true, "index": true,
}

// keepVideoParams filters a query (or a fragment written as one) to the
// allow-list, keeping the order and the first occurrence of each key. A value
// that could hide another parameter (a ; separator, or a % escape that
// decodes to &, = or ;) is dropped with its key; a fragment that is not a
// query is dropped.
func keepVideoParams(raw string) string {
	var kept []string
	seen := map[string]bool{}
	for _, kv := range strings.Split(raw, "&") {
		k, val, _ := strings.Cut(kv, "=")
		if !carriedVideoParams[k] || seen[k] {
			continue
		}
		seen[k] = true
		if strings.Contains(val, ";") {
			continue
		}
		if dec, err := url.QueryUnescape(val); err != nil || strings.ContainsAny(dec, "&=;") {
			continue
		}
		kept = append(kept, kv)
	}
	return strings.Join(kept, "&")
}

// CarriedTranscriptVideoSource is CarriedVideoSource for a transcription's
// header, where no video is the empty string.
func CarriedTranscriptVideoSource(src string) string {
	if v := CarriedVideoSource(&src); v != nil {
		return *v
	}
	return ""
}

// CarriedFileName is a producer's file path as an export carries it: the
// file's name alone, whichever separator the producer's system used. The name
// is enough to find the file again in the recipient's library; the path
// above it describes the producer's disk.
func CarriedFileName(path string) string {
	v := strings.TrimSpace(path)
	return v[strings.LastIndexAny(v, `/\`)+1:]
}

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
