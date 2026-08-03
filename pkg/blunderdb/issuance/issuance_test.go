package issuance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	env := mustSeal(t, id, Watermark{Distribution: "Cours du 12 mars", Recipient: "Kévin Unger", Number: 7, Total: 24})

	if !env.Verify() {
		t.Fatal("a freshly sealed watermark must verify")
	}
	if !env.VerifiedBy(id.Fingerprint()) {
		t.Fatal("the issuer must recognise their own watermark")
	}
	w, err := env.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if w.Recipient != "Kévin Unger" || w.Number != 7 || w.IssuerName != "Jean Dupont" {
		t.Fatalf("watermark did not survive the round trip: %+v", w)
	}
	if !w.Nominative() {
		t.Fatal("a watermark naming a recipient is nominative")
	}
	if w.Salt == "" || w.CopyID == "" || w.IssuedAt == "" {
		t.Fatalf("Seal must fill salt, copy id and date: %+v", w)
	}
}

// An accented recipient name is the canonical way a re-serialising implementation breaks:
// the signature must be checked against the stored bytes, never against a re-marshalled
// struct.
func TestSealVerifiesAgainstStoredBytes(t *testing.T) {
	id := mustIdentity(t, "Émetteur")
	env := mustSeal(t, id, Watermark{Distribution: "Cours élève", Recipient: "Céline Œuvré"})

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
	if w.Recipient != "Céline Œuvré" {
		t.Fatalf("accented recipient mangled: %q", w.Recipient)
	}
}

func TestTamperedWatermarkFailsVerification(t *testing.T) {
	id := mustIdentity(t, "Jean Dupont")
	env := mustSeal(t, id, Watermark{Distribution: "Cours", Recipient: "Kévin"})

	altered := env
	altered.Payload = strings.Replace(env.Payload, "Kévin", "Marc!", 1)
	if len(altered.Payload) == len(env.Payload) && altered.Payload == env.Payload {
		t.Fatal("test setup: payload was not altered")
	}
	if altered.Verify() {
		t.Fatal("a rewritten recipient must break the signature")
	}
}

// The property the signature actually buys: nobody can produce a watermark that a given
// issuer's fingerprint vouches for.
func TestForgedWatermarkIsNotAttributedToIssuer(t *testing.T) {
	issuer := mustIdentity(t, "Jean Dupont")
	forger := mustIdentity(t, "Jean Dupont") // same display name, different key

	forged := mustSeal(t, forger, Watermark{Distribution: "Cours", Recipient: "Marc"})
	if !forged.Verify() {
		t.Fatal("the forgery is internally consistent — that is expected")
	}
	if forged.VerifiedBy(issuer.Fingerprint()) {
		t.Fatal("a forgery must not verify against the real issuer's fingerprint")
	}
}

func TestCollectiveWatermarkIsNotNominative(t *testing.T) {
	id := mustIdentity(t, "Jean")
	env := mustSeal(t, id, Watermark{Distribution: "Promotion 2026"})
	w, _ := env.Open()
	if w.Nominative() {
		t.Fatal("a copy with no recipient is collective")
	}
}

func TestCarriedIsAnAllowList(t *testing.T) {
	got := Carried(map[string]string{
		"user":        "Kévin",
		"description": "Cours",
		"issued":      `{"records":[{"password":"secret"}]}`,
		"watermark":   "{}",
		"holders":     "{}",
		"surprise":    "a document added six months from now",
	})
	if _, ok := got[KeyIssued]; ok {
		t.Fatal("the issue register must never travel inside an issued copy")
	}
	if _, ok := got["surprise"]; ok {
		t.Fatal("an unknown key must not be carried by default")
	}
	if got["user"] != "Kévin" || got["description"] != "Cours" {
		t.Fatalf("ordinary metadata must be carried: %+v", got)
	}
}

func TestLineageInheritDeduplicates(t *testing.T) {
	id := mustIdentity(t, "Jean")
	first := mustSeal(t, id, Watermark{Distribution: "Cours 1", Recipient: "Kévin"})
	second := mustSeal(t, id, Watermark{Distribution: "Cours 2", Recipient: "Kévin"})

	var l Lineage
	l = l.Inherit(first, nil)
	l = l.Inherit(second, Lineage{first}) // second copy already carried the first
	if len(l) != 2 {
		t.Fatalf("re-importing must not grow the lineage: %d entries", len(l))
	}
	l = l.Inherit(first, nil)
	if len(l) != 2 {
		t.Fatalf("importing the same copy twice must be idempotent: %d entries", len(l))
	}
	for _, e := range l {
		if !e.Verify() {
			t.Fatal("inherited watermarks must still verify against their original issuer")
		}
	}
}

func TestLineageIgnoresUnissuedSources(t *testing.T) {
	var l Lineage
	if got := l.Inherit(Envelope{}, nil); len(got) != 0 {
		t.Fatalf("importing an ordinary database must not create a lineage: %d", len(got))
	}
	encoded, err := EncodeLineage(nil)
	if err != nil || encoded != "" {
		t.Fatalf("an empty lineage must encode to nothing, got %q / %v", encoded, err)
	}
}

func TestRegistryRecordsMachinesNotOpenings(t *testing.T) {
	var r Registry
	now := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
	r.Record("genesis", "aaaa", now)
	r.Record("genesis", "aaaa", now.Add(time.Hour))
	r.Record("genesis", "bbbb", now.Add(2*time.Hour))

	if len(r.Holders) != 2 {
		t.Fatalf("one entry per machine, not per opening: %d", len(r.Holders))
	}
	if r.Holders[0].Openings != 2 {
		t.Fatalf("repeat openings must increment the counter: %d", r.Holders[0].Openings)
	}
	if r.Holders[0].FirstSeen == r.Holders[0].LastSeen {
		t.Fatal("last seen must move while first seen stays put")
	}
	if !r.ChainIntact("genesis") {
		t.Fatal("a registry built by Record must be intact")
	}
}

func TestRegistryChainCatchesARemovedEntry(t *testing.T) {
	var r Registry
	now := time.Now()
	for _, fp := range []string{"aaaa", "bbbb", "cccc"} {
		r.Record("genesis", fp, now)
	}
	trimmed := Registry{Holders: append(append([]Holder{}, r.Holders[0]), r.Holders[2])}
	if trimmed.ChainIntact("genesis") {
		t.Fatal("removing a middle entry must break the chain")
	}
	if !r.ChainIntact("genesis") {
		t.Fatal("the untouched registry must stay intact")
	}
	if r.ChainIntact("another-copy") {
		t.Fatal("a registry must not verify against a different copy's genesis")
	}
}

func TestMachineFingerprintIsSaltedPerDistribution(t *testing.T) {
	a := MachineFingerprint("salt-of-course-A")
	b := MachineFingerprint("salt-of-course-B")
	if a == b {
		t.Fatal("the same machine must look different across distributions")
	}
	if a != MachineFingerprint("salt-of-course-A") {
		t.Fatal("the fingerprint must be stable within a distribution")
	}
	if strings.Contains(a, machineTraits()) {
		t.Fatal("the fingerprint must not leak the host traits it derives from")
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
		t.Fatalf("having issued nothing is the normal state: %v", err)
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
	// A watermark sealed on the second machine must be indistinguishable from one sealed
	// on the first.
	env := mustSeal(t, moved, Watermark{Distribution: "Cours", Recipient: "Kévin"})
	if !env.VerifiedBy(original.Fingerprint()) {
		t.Fatal("copies issued from either machine must share one fingerprint")
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

	// The seed must not be sitting in the file in the clear.
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

func TestIssueRegisterLooksUpAFoundCopy(t *testing.T) {
	id := mustIdentity(t, "Jean")
	env := mustSeal(t, id, Watermark{Distribution: "Cours du 12 mars", Recipient: "Kévin", Number: 7, Total: 24})

	var reg IssueRegister
	reg.Add(IssueRecord{
		Distribution: "Cours du 12 mars", Recipient: "Kévin", Number: 7, Total: 24,
		Signature: env.Signature, Password: "le-mot-de-passe",
	}, time.Now())

	found, ok := reg.Find(env.Signature)
	if !ok {
		t.Fatal("a copy that comes back must be found in the register")
	}
	if found.Recipient != "Kévin" || found.Password != "le-mot-de-passe" {
		t.Fatalf("unexpected record: %+v", found)
	}
	if _, ok := reg.Find("somebody else's signature"); ok {
		t.Fatal("an unrelated copy must not match")
	}
	if got := reg.Distributions(); len(got) != 1 || got[0] != "Cours du 12 mars" {
		t.Fatalf("unexpected distributions: %v", got)
	}
}

func TestSaltIsSharedAcrossOneDistribution(t *testing.T) {
	id := mustIdentity(t, "Jean")
	first := mustSeal(t, id, Watermark{Distribution: "Cours", Recipient: "A"})
	w, _ := first.Open()

	var reg IssueRegister
	reg.Add(IssueRecord{Distribution: "Cours", Signature: first.Signature, Salt: w.Salt, Number: 1}, time.Now())

	if got := reg.SaltFor("Cours"); got != w.Salt {
		t.Fatalf("copies of one distribution must share a salt: %q vs %q", got, w.Salt)
	}
	if got := reg.SaltFor("Un autre cours"); got != "" {
		t.Fatalf("a different distribution must get its own salt, got %q", got)
	}
	if got := reg.NextNumber("Cours"); got != 2 {
		t.Fatalf("a second batch must continue the numbering, got %d", got)
	}
	if got := reg.NextNumber("Un autre cours"); got != 1 {
		t.Fatalf("a new distribution starts at one, got %d", got)
	}

	// Sharing the salt is what makes two leaks comparable.
	second := mustSeal(t, id, Watermark{Distribution: "Cours", Recipient: "B", Salt: w.Salt})
	w2, _ := second.Open()
	if MachineFingerprint(w.Salt) != MachineFingerprint(w2.Salt) {
		t.Fatal("one machine must look the same in two copies of the same distribution")
	}
}

func TestContainerHeaderIsReadableWithoutThePassword(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "cours.db")
	if err := os.WriteFile(dbPath, []byte("SQLite format 3\x00 pretend this is a database"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	id := mustIdentity(t, "Jean Dupont")
	env := mustSeal(t, id, Watermark{Distribution: "Cours du 12 mars", Recipient: "Kévin", Number: 7})

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

	// The whole point: a copy found in the wild is identifiable without decrypting it.
	header, err := ReadContainerHeader(out)
	if err != nil {
		t.Fatalf("ReadContainerHeader: %v", err)
	}
	if !header.Watermark.VerifiedBy(id.Fingerprint()) {
		t.Fatal("the cleartext header must carry a verifiable watermark")
	}
	w, _ := header.Watermark.Open()
	if w.Recipient != "Kévin" {
		t.Fatalf("unexpected recipient in header: %q", w.Recipient)
	}

	// …and the payload really is encrypted.
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(raw), "pretend this is a database") {
		t.Fatal("the database must not be readable in the container")
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
	env := mustSeal(t, id, Watermark{Distribution: "Cours"})
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
