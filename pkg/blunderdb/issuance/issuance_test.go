package issuance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustIdentity(t *testing.T, name string) *Identity {
	t.Helper()
	id, err := NewIdentity(name)
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	return id
}

func mustSeal(t *testing.T, id *Identity, w Watermark) Envelope {
	t.Helper()
	env, err := Seal(id, w)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	return env
}

func TestSealRoundTrip(t *testing.T) {
	id := mustIdentity(t, "Jean Dupont")
	env := mustSeal(t, id, Watermark{
		Origin: "Cours de Jean Dupont — 12 mars 2026",
		Note:   "Merci de ne pas rediffuser.",
	})

	if !env.Verify() {
		t.Fatal("a freshly sealed watermark must verify")
	}
	if !env.VerifiedBy(id.Fingerprint()) {
		t.Fatal("the producer must recognise their own mark")
	}
	w, err := env.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if w.Origin != "Cours de Jean Dupont — 12 mars 2026" || w.IssuerName != "Jean Dupont" {
		t.Fatalf("watermark did not survive the round trip: %+v", w)
	}
	if w.Note != "Merci de ne pas rediffuser." {
		t.Fatalf("the note must travel: %q", w.Note)
	}
	if w.IssuedAt == "" {
		t.Fatal("Seal must stamp the date")
	}
}

// A watermark states an origin and nothing else: no recipient, no identifier, nothing
// derived from the machine that produced it. Two marks made from the same identity for the
// same origin must be indistinguishable apart from their timestamp.
func TestWatermarkCarriesNothingButWhatWasWritten(t *testing.T) {
	id := mustIdentity(t, "Jean Dupont")
	env := mustSeal(t, id, Watermark{Origin: "Cours", IssuedAt: "2026-03-12T10:00:00Z"})

	var fields map[string]any
	if err := json.Unmarshal([]byte(env.Payload), &fields); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	want := map[string]bool{"version": true, "origin": true, "issuerName": true, "issuedAt": true}
	for k := range fields {
		if !want[k] {
			t.Fatalf("unexpected field %q in a watermark: %v", k, fields)
		}
	}

	twin := mustSeal(t, id, Watermark{Origin: "Cours", IssuedAt: "2026-03-12T10:00:00Z"})
	if twin.Payload != env.Payload {
		t.Fatalf("two identical marks must be identical:\n%s\n%s", env.Payload, twin.Payload)
	}
}

// An accented origin is the canonical way a re-serialising implementation breaks: the
// signature must be checked against the stored bytes, never a re-marshalled struct.
func TestSealVerifiesAgainstStoredBytes(t *testing.T) {
	id := mustIdentity(t, "Émetteur")
	env := mustSeal(t, id, Watermark{Origin: "Cours élève — Céline Œuvré"})

	stored, err := EncodeEnvelope(env)
	if err != nil {
		t.Fatalf("EncodeEnvelope: %v", err)
	}
	back, err := DecodeEnvelope(stored)
	if err != nil {
		t.Fatalf("DecodeEnvelope: %v", err)
	}
	if !back.Verify() {
		t.Fatal("a stored and reloaded watermark must still verify")
	}
	w, _ := back.Open()
	if w.Origin != "Cours élève — Céline Œuvré" {
		t.Fatalf("accented origin mangled: %q", w.Origin)
	}
}

func TestTamperedWatermarkFailsVerification(t *testing.T) {
	id := mustIdentity(t, "Jean Dupont")
	env := mustSeal(t, id, Watermark{Origin: "Cours de Jean"})

	altered := env
	altered.Payload = strings.Replace(env.Payload, "Cours de Jean", "Cours de Marc", 1)
	if altered.Payload == env.Payload {
		t.Fatal("test setup: payload was not altered")
	}
	if altered.Verify() {
		t.Fatal("rewriting the origin must break the signature")
	}
}

// The property the signature buys: nobody can produce a mark that a given producer's
// fingerprint vouches for.
func TestForgedWatermarkIsNotAttributedToProducer(t *testing.T) {
	producer := mustIdentity(t, "Jean Dupont")
	forger := mustIdentity(t, "Jean Dupont") // same display name, different key

	forged := mustSeal(t, forger, Watermark{Origin: "Cours de Jean Dupont"})
	if !forged.Verify() {
		t.Fatal("the forgery is internally consistent — that is expected")
	}
	if forged.VerifiedBy(producer.Fingerprint()) {
		t.Fatal("a forgery must not verify against the real producer's fingerprint")
	}
}

func TestSealRefusesAnEmptyOrigin(t *testing.T) {
	id := mustIdentity(t, "Jean")
	if _, err := Seal(id, Watermark{Origin: "   "}); err == nil {
		t.Fatal("a watermark with no origin marks nothing")
	}
	if _, err := Seal(nil, Watermark{Origin: "Cours"}); err == nil {
		t.Fatal("sealing without an identity must fail")
	}
}

func TestUnwatermarkedEnvelopeEncodesToNothing(t *testing.T) {
	encoded, err := EncodeEnvelope(Envelope{})
	if err != nil || encoded != "" {
		t.Fatalf("expected no row for an unwatermarked file, got %q / %v", encoded, err)
	}
	env, err := DecodeEnvelope("")
	if err != nil || env.IsWatermarked() {
		t.Fatalf("an absent row must decode to nothing: %+v / %v", env, err)
	}
}

func TestCarriedIsAnAllowList(t *testing.T) {
	got := Carried(map[string]string{
		"user":        "Kévin",
		"description": "Cours",
		"watermark":   "{}",
		"surprise":    "a document added six months from now",
	})
	if _, ok := got[KeyWatermark]; ok {
		t.Fatal("the watermark is written by the export itself, never copied from the source")
	}
	if _, ok := got["surprise"]; ok {
		t.Fatal("an unknown key must not be carried by default")
	}
	if got["user"] != "Kévin" || got["description"] != "Cours" {
		t.Fatalf("ordinary metadata must be carried: %+v", got)
	}
}

func TestIdentityPersistsAndIsCreatedOnce(t *testing.T) {
	dir := t.TempDir()
	first, err := LoadOrCreateIdentity(dir, "Jean Dupont")
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity: %v", err)
	}
	second, err := LoadOrCreateIdentity(dir, "quelqu'un d'autre")
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity (second call): %v", err)
	}
	if first.Fingerprint() != second.Fingerprint() {
		t.Fatal("the identity must be created once and reused")
	}
	if second.Name != "Jean Dupont" {
		t.Fatalf("the stored name must win over the default: %q", second.Name)
	}

	info, err := os.Stat(IdentityPath(dir))
	if err != nil {
		t.Fatalf("stat identity: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("a signing key must not be world-readable: %v", perm)
	}
}

func TestLoadIdentityAbsentIsNotAnError(t *testing.T) {
	id, err := LoadIdentity(t.TempDir())
	if err != nil {
		t.Fatalf("having marked nothing is the normal state: %v", err)
	}
	if id != nil {
		t.Fatal("expected no identity")
	}
}

func TestIdentityTransferKeepsTheSameFingerprint(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	original, err := LoadOrCreateIdentity(from, "Jean Dupont")
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity: %v", err)
	}
	file := filepath.Join(t.TempDir(), "jean"+IdentityFileExtension)
	if err := original.ExportIdentity(file, ""); err != nil {
		t.Fatalf("ExportIdentity: %v", err)
	}
	if needs, err := IdentityFileNeedsPassphrase(file); err != nil || needs {
		t.Fatalf("an unprotected export must not claim a passphrase (%v, %v)", needs, err)
	}

	moved, err := ImportIdentity(to, file, "")
	if err != nil {
		t.Fatalf("ImportIdentity: %v", err)
	}
	if moved.Fingerprint() != original.Fingerprint() {
		t.Fatal("moving an identity between machines must preserve it — it is one person")
	}
	env := mustSeal(t, moved, Watermark{Origin: "Cours"})
	if !env.VerifiedBy(original.Fingerprint()) {
		t.Fatal("marks made from either machine must share one fingerprint")
	}
}

func TestProtectedIdentityFileNeedsItsPassphrase(t *testing.T) {
	dir := t.TempDir()
	id, err := LoadOrCreateIdentity(dir, "Jean")
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity: %v", err)
	}
	file := filepath.Join(t.TempDir(), "jean"+IdentityFileExtension)
	if err := id.ExportIdentity(file, "correct horse"); err != nil {
		t.Fatalf("ExportIdentity: %v", err)
	}
	if needs, err := IdentityFileNeedsPassphrase(file); err != nil || !needs {
		t.Fatalf("a protected export must announce it (%v, %v)", needs, err)
	}
	if _, err := ImportIdentity(t.TempDir(), file, ""); err != ErrPassphraseRequired {
		t.Fatalf("expected ErrPassphraseRequired, got %v", err)
	}
	if _, err := ImportIdentity(t.TempDir(), file, "wrong"); err != ErrWrongPassphrase {
		t.Fatalf("expected ErrWrongPassphrase, got %v", err)
	}
	back, err := ImportIdentity(t.TempDir(), file, "correct horse")
	if err != nil {
		t.Fatalf("ImportIdentity with the right passphrase: %v", err)
	}
	if back.Fingerprint() != id.Fingerprint() {
		t.Fatal("the recovered identity must be the same one")
	}

	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var stored map[string]any
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if stored["kdf"] != kdfArgon2id {
		t.Fatalf("expected the kdf to be recorded, got %v", stored["kdf"])
	}
}

func TestRenameKeepsTheKey(t *testing.T) {
	dir := t.TempDir()
	id, err := LoadOrCreateIdentity(dir, "Jean")
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity: %v", err)
	}
	before := id.Fingerprint()
	if err := id.Rename(dir, "Jean Dupont"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	reloaded, err := LoadIdentity(dir)
	if err != nil {
		t.Fatalf("LoadIdentity: %v", err)
	}
	if reloaded.Name != "Jean Dupont" || reloaded.Fingerprint() != before {
		t.Fatalf("renaming must change the label and nothing else: %q / %s", reloaded.Name, reloaded.Fingerprint())
	}
}

func TestContainerHeaderIsReadableWithoutThePassword(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "cours.db")
	if err := os.WriteFile(dbPath, []byte("SQLite format 3\x00 pretend this is a database"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	id := mustIdentity(t, "Jean Dupont")
	env := mustSeal(t, id, Watermark{Origin: "Cours de Jean Dupont — 12 mars 2026"})

	out := filepath.Join(dir, "cours"+ContainerExtension)
	if err := WrapContainer(dbPath, out, env, "le-mot-de-passe"); err != nil {
		t.Fatalf("WrapContainer: %v", err)
	}
	if !IsContainer(out) {
		t.Fatal("the wrapped file must be recognised as protected")
	}
	if IsContainer(dbPath) {
		t.Fatal("an ordinary database must not look protected")
	}

	header, err := ReadContainerHeader(out)
	if err != nil {
		t.Fatalf("ReadContainerHeader: %v", err)
	}
	if !header.Watermark.VerifiedBy(id.Fingerprint()) {
		t.Fatal("the cleartext header must carry a verifiable watermark")
	}
	w, _ := header.Watermark.Open()
	if w.Origin != "Cours de Jean Dupont — 12 mars 2026" {
		t.Fatalf("unexpected origin in header: %q", w.Origin)
	}

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(raw), "pretend this is a database") {
		t.Fatal("the database must not be readable in the container")
	}
}

func TestContainerWithoutAWatermark(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "cours.db")
	if err := os.WriteFile(dbPath, []byte("SQLite format 3\x00"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	out := filepath.Join(dir, "cours"+ContainerExtension)
	if err := WrapContainer(dbPath, out, Envelope{}, "pw"); err != nil {
		t.Fatalf("a password without a watermark is a valid combination: %v", err)
	}
	header, err := ReadContainerHeader(out)
	if err != nil {
		t.Fatalf("ReadContainerHeader: %v", err)
	}
	if header.Watermark.IsWatermarked() {
		t.Fatal("expected no watermark in the header")
	}
}

func TestContainerRoundTrip(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "cours.db")
	payload := []byte("SQLite format 3\x00 contents that must survive")
	if err := os.WriteFile(dbPath, payload, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	id := mustIdentity(t, "Jean")
	env := mustSeal(t, id, Watermark{Origin: "Cours"})
	out := filepath.Join(dir, "cours"+ContainerExtension)
	if err := WrapContainer(dbPath, out, env, "s3cret"); err != nil {
		t.Fatalf("WrapContainer: %v", err)
	}

	opened := filepath.Join(dir, "opened.db")
	if _, err := UnwrapContainer(out, opened, ""); err != ErrPassphraseRequired {
		t.Fatalf("expected ErrPassphraseRequired, got %v", err)
	}
	if _, err := UnwrapContainer(out, opened, "wrong"); err != ErrWrongPassphrase {
		t.Fatalf("expected ErrWrongPassphrase, got %v", err)
	}
	if _, err := UnwrapContainer(out, opened, "s3cret"); err != nil {
		t.Fatalf("UnwrapContainer: %v", err)
	}
	got, err := os.ReadFile(opened)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatal("the opened database must be byte-identical to the original")
	}
}

func TestVerifyPassword(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "cours.db")
	if err := os.WriteFile(dbPath, []byte("SQLite format 3\x00"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	out := filepath.Join(dir, "cours"+ContainerExtension)
	if err := WrapContainer(dbPath, out, Envelope{}, "s3cret"); err != nil {
		t.Fatalf("WrapContainer: %v", err)
	}

	if err := VerifyPassword(out, "s3cret"); err != nil {
		t.Fatalf("the right password must verify: %v", err)
	}
	if err := VerifyPassword(out, "wrong"); err != ErrWrongPassphrase {
		t.Fatalf("expected ErrWrongPassphrase, got %v", err)
	}
	if err := VerifyPassword(out, ""); err != ErrPassphraseRequired {
		t.Fatalf("expected ErrPassphraseRequired, got %v", err)
	}
	if err := VerifyPassword(dbPath, "s3cret"); err == nil {
		t.Fatal("an ordinary database is not a protected copy")
	}
	// Verifying must not have written anything next to the container.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Fatalf("expected only the database and its container, got %d entries", len(entries))
	}
}

func TestContainerRejectsOrdinaryFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plain.db")
	if err := os.WriteFile(path, []byte("SQLite format 3\x00"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := ReadContainerHeader(path); err == nil {
		t.Fatal("an ordinary database is not a container")
	}
}

func TestDefaultUnwrapPath(t *testing.T) {
	got := DefaultUnwrapPath(filepath.Join("dir", "cours-mars"+ContainerExtension))
	if want := filepath.Join("dir", "cours-mars.db"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// A container does not have to be named .dbx — blunderDB recognises one by its magic bytes.
// Unwrapping one that is called .db must not land on the container itself.
func TestDefaultUnwrapPathNeverOverwritesTheContainer(t *testing.T) {
	for _, name := range []string{"cours.db", "cours", "cours.dbx"} {
		container := filepath.Join("dir", name)
		got := DefaultUnwrapPath(container)
		if got == filepath.Clean(container) {
			t.Fatalf("%s would unwrap onto itself", name)
		}
		if filepath.Ext(got) != ".db" {
			t.Fatalf("%s unwrapped to %q, expected a .db file", name, got)
		}
	}
}

func TestProtectedPath(t *testing.T) {
	cases := map[string]string{
		"cours.db":  "cours" + ContainerExtension,
		"cours":     "cours" + ContainerExtension,
		"cours.dbx": "cours" + ContainerExtension,
		"cours.DBX": "cours.DBX",
	}
	for in, want := range cases {
		if got := ProtectedPath(in); got != want {
			t.Fatalf("ProtectedPath(%q) = %q, want %q", in, got, want)
		}
	}
}
