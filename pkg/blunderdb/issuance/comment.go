package issuance

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// CommentStatement is what a producer signs about one Comment at export: its
// exact text, the author it names, the position it sits on, and who signs
// (ADR-0099). It is the
// Watermark's counterpart for a single note, sealed in the same Envelope.
//
// Field order is the signed byte order, as for Watermark: appending a field is
// safe, reordering or renaming one is not.
type CommentStatement struct {
	Version int    `json:"version"`
	Text    string `json:"text"`
	Author  string `json:"author"`
	// Position is the Zobrist hash of the commented position, in hex: a
	// signed note moved onto another board no longer verifies.
	Position   string `json:"position"`
	IssuerName string `json:"issuerName"`
	SignedAt   string `json:"signedAt"`
}

// SealComment signs text and author, on the position of Zobrist hash
// position, with the producer's identity and returns the encoded Envelope a
// comment row stores in its signature column.
func SealComment(id *Identity, text, author string, position uint64) (string, error) {
	if id == nil {
		return "", fmt.Errorf("no issuer identity")
	}
	payload, err := json.Marshal(CommentStatement{
		Version:    1,
		Text:       text,
		Author:     author,
		Position:   positionKey(position),
		IssuerName: id.Name,
		SignedAt:   time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return "", err
	}
	return EncodeEnvelope(Envelope{
		Alg:       algEd25519,
		Payload:   string(payload),
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(id.priv, payload)),
		PublicKey: base64.StdEncoding.EncodeToString(id.priv.Public().(ed25519.PublicKey)),
	})
}

// CommentSigner is what a stored comment signature says once checked against
// the row it sits on.
type CommentSigner struct {
	// IssuerName is the name the signer gave, as the statement carries it;
	// "" when the signature cannot be read at all.
	IssuerName  string
	Fingerprint string
	// Verified holds when the signature checks out AND covers the row's
	// current text, author and position: a note altered or moved after
	// signing reads as signed by its issuer but not verified, never as
	// unsigned.
	Verified bool
}

// CheckComment reads the signature stored beside a comment and checks it
// against the comment's text, author and position (its Zobrist hash). ok is false when signature is empty:
// the comment is simply not signed. A signature that does not parse is
// reported as signed and unverified, so a damaged file is shown as such
// rather than passed off as an ordinary note.
func CheckComment(signature, text, author string, position uint64) (signer CommentSigner, ok bool) {
	if strings.TrimSpace(signature) == "" {
		return CommentSigner{}, false
	}
	env, err := DecodeEnvelope(signature)
	if err != nil {
		return CommentSigner{}, true
	}
	signer.Fingerprint = env.Fingerprint()
	var st CommentStatement
	if err := json.Unmarshal([]byte(env.Payload), &st); err != nil {
		return signer, true
	}
	signer.IssuerName = st.IssuerName
	signer.Verified = env.Verify() && st.Text == text && st.Author == author && st.Position == positionKey(position)
	return signer, true
}

func positionKey(zobrist uint64) string { return fmt.Sprintf("%016x", zobrist) }
