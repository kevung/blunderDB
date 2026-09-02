package issuance

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// v1Fixture is a container written by the version-1 code (before the header was bound to
// the payload), frozen in testdata so the read path for files already handed out is checked
// against real bytes rather than a re-implementation of the old writer.
const (
	v1Fixture    = "testdata/container-v1.dbx"
	v1Passphrase = "fixture-v1"
	v1Contents   = "SQLite format 3\x00 v1 fixture: pretend this is a database"
)

func writeContainer(t *testing.T, contents, passphrase string) (dir, out string) {
	t.Helper()
	dir = t.TempDir()
	dbPath := filepath.Join(dir, "cours.db")
	if err := os.WriteFile(dbPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	id := mustIdentity(t, "Jean Dupont")
	env := mustSeal(t, id, Watermark{Origin: "Cours de Jean Dupont"})
	out = filepath.Join(dir, "cours"+ContainerExtension)
	if err := WrapContainer(dbPath, out, env, passphrase); err != nil {
		t.Fatalf("WrapContainer: %v", err)
	}
	return dir, out
}

// flipHeaderByte rewrites one byte of the cleartext header so that the JSON stays valid:
// the last character of the watermark payload's origin, which is inside a string.
func flipHeaderByte(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	headerLen := int(binary.BigEndian.Uint32(raw[len(containerMagic):]))
	start := len(containerMagic) + 4
	header := raw[start : start+headerLen]
	at := bytes.Index(header, []byte("Dupont"))
	if at < 0 {
		t.Fatal("fixture header does not carry the expected origin")
	}
	header[at] = 'X'
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// captureWarnings routes slog's default logger into a buffer for the duration of the test.
func captureWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

func TestContainerWritesVersion2WithArgon2Parameters(t *testing.T) {
	_, out := writeContainer(t, "SQLite format 3\x00", "pw")
	header, err := ReadContainerHeader(out)
	if err != nil {
		t.Fatalf("ReadContainerHeader: %v", err)
	}
	if header.Version != containerVersionCurrent {
		t.Fatalf("new containers must be written as version %d, got %d", containerVersionCurrent, header.Version)
	}
	if header.KDF != kdfArgon2id || header.Argon2 == nil || *header.Argon2 != argon2Default {
		t.Fatalf("the header must record the derivation and its parameters, got %s %+v", header.KDF, header.Argon2)
	}
}

func TestVersion1ContainerStillOpensWithAWarning(t *testing.T) {
	logs := captureWarnings(t)
	header, err := ReadContainerHeader(v1Fixture)
	if err != nil {
		t.Fatalf("ReadContainerHeader: %v", err)
	}
	if header.Version != containerVersionUnauthenticatedHeader {
		t.Fatalf("fixture is not a version-1 container: %d", header.Version)
	}
	if !header.Watermark.Verify() {
		t.Fatal("the fixture's watermark must still verify")
	}
	if err := VerifyPassword(v1Fixture, "wrong"); !errors.Is(err, ErrWrongPassphrase) {
		t.Fatalf("expected ErrWrongPassphrase, got %v", err)
	}
	opened := filepath.Join(t.TempDir(), "cours.db")
	if _, err := UnwrapContainer(v1Fixture, opened, v1Passphrase); err != nil {
		t.Fatalf("a version-1 file handed out before the change must still open: %v", err)
	}
	got, err := os.ReadFile(opened)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != v1Contents {
		t.Fatalf("unexpected contents: %q", got)
	}
	if !strings.Contains(logs.String(), "version-1") {
		t.Fatalf("opening a version-1 file must be logged as a warning, got: %q", logs.String())
	}
}

// The flaw version 2 closes, demonstrated on version 1: the header is not bound to the
// payload, so relabelling the file goes unnoticed. Kept as a test so the reason for the
// second version stays executable rather than folklore.
func TestVersion1ContainerDoesNotNoticeARelabelledHeader(t *testing.T) {
	captureWarnings(t)
	raw, err := os.ReadFile(v1Fixture)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	dir := t.TempDir()
	clone := filepath.Join(dir, "cours"+ContainerExtension)
	if err := os.WriteFile(clone, raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	flipHeaderByte(t, clone)
	if err := VerifyPassword(clone, v1Passphrase); err != nil {
		t.Fatalf("version 1 cannot detect a relabelled header; if it now does, drop this test: %v", err)
	}
}

func TestVersion2ContainerRefusesARelabelledHeader(t *testing.T) {
	dir, out := writeContainer(t, "SQLite format 3\x00", "s3cret")
	flipHeaderByte(t, out)
	header, err := ReadContainerHeader(out)
	if err != nil {
		t.Fatalf("the altered header still parses (that is the point): %v", err)
	}
	if header.Watermark.Verify() {
		t.Fatal("test setup: the flipped byte must land inside the signed payload")
	}
	if err := VerifyPassword(out, "s3cret"); !errors.Is(err, ErrWrongPassphrase) {
		t.Fatalf("a version-2 file whose header changed by one byte must not open, got %v", err)
	}
	opened := filepath.Join(dir, "opened.db")
	if _, err := UnwrapContainer(out, opened, "s3cret"); !errors.Is(err, ErrWrongPassphrase) {
		t.Fatalf("expected ErrWrongPassphrase, got %v", err)
	}
	if _, err := os.Stat(opened); !os.IsNotExist(err) {
		t.Fatal("nothing must be written when the file does not authenticate")
	}
}

func TestVersion2ContainerRefusesAnAlteredPayload(t *testing.T) {
	_, out := writeContainer(t, "SQLite format 3\x00 some more bytes", "s3cret")
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	raw[len(raw)-20] ^= 0x01
	if err := os.WriteFile(out, raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := VerifyPassword(out, "s3cret"); !errors.Is(err, ErrWrongPassphrase) {
		t.Fatalf("expected ErrWrongPassphrase, got %v", err)
	}
}

// rewriteHeader re-encodes the header of a container in place. Used to forge headers the
// writer would never produce; the payload no longer authenticates afterwards, which is fine
// for tests that expect a refusal before decryption.
func rewriteHeader(t *testing.T, path string, edit func(*ContainerHeader)) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	headerLen := int(binary.BigEndian.Uint32(raw[len(containerMagic):]))
	start := len(containerMagic) + 4
	var header ContainerHeader
	if err := json.Unmarshal(raw[start:start+headerLen], &header); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	edit(&header)
	prefix, err := encodeContainerPrefix(header)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if err := os.WriteFile(path, append(prefix, raw[start+headerLen:]...), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestContainerFromANewerVersionIsRefusedButItsOriginStaysReadable(t *testing.T) {
	_, out := writeContainer(t, "SQLite format 3\x00", "pw")
	rewriteHeader(t, out, func(h *ContainerHeader) { h.Version = 3 })
	header, err := ReadContainerHeader(out)
	if err != nil {
		t.Fatalf("the origin of a newer file must stay readable: %v", err)
	}
	if !header.Watermark.Verify() {
		t.Fatal("the watermark must still verify")
	}
	err = VerifyPassword(out, "pw")
	if err == nil || !strings.Contains(err.Error(), "newer blunderDB") {
		t.Fatalf("a newer version must be refused explicitly, got %v", err)
	}
}

func TestContainerRefusesUnknownDerivationParameters(t *testing.T) {
	_, out := writeContainer(t, "SQLite format 3\x00", "pw")
	weak := filepath.Join(t.TempDir(), "weak"+ContainerExtension)
	raw, _ := os.ReadFile(out)
	if err := os.WriteFile(weak, raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	rewriteHeader(t, weak, func(h *ContainerHeader) { h.Argon2 = &Argon2Params{Time: 1, Memory: 8, Threads: 1} })
	if err := VerifyPassword(weak, "pw"); err == nil || errors.Is(err, ErrWrongPassphrase) || !strings.Contains(err.Error(), "parameters") {
		t.Fatalf("parameters this build never used must be refused by name, got %v", err)
	}

	other := filepath.Join(t.TempDir(), "other"+ContainerExtension)
	if err := os.WriteFile(other, raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	rewriteHeader(t, other, func(h *ContainerHeader) { h.KDF = "scrypt" })
	if err := VerifyPassword(other, "pw"); err == nil || errors.Is(err, ErrWrongPassphrase) || !strings.Contains(err.Error(), "scrypt") {
		t.Fatalf("an unknown derivation must be refused by name, got %v", err)
	}
}

func TestContainerHeaderLengthIsBoundedBeforeAllocation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hostile"+ContainerExtension)
	raw := append([]byte{}, containerMagic...)
	raw = binary.BigEndian.AppendUint32(raw, 0xFFFFFFFF)
	raw = append(raw, []byte("{}")...)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := ReadContainerHeader(path); err == nil || !strings.Contains(err.Error(), "header claims") {
		t.Fatalf("a hostile header length must be refused before anything is allocated, got %v", err)
	}
}

// An oversized payload is refused from its size, before any of it is read. The file is
// sparse, so the test costs nothing on a filesystem that supports holes; Windows would
// materialise the zeros.
func TestContainerPayloadIsBoundedBeforeAllocation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs a sparse file")
	}
	_, out := writeContainer(t, "SQLite format 3\x00", "pw")
	f, err := os.OpenFile(out, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.Truncate(maxContainerPayload + gcmOverhead + 4096); err != nil {
		f.Close()
		t.Skipf("cannot create a sparse file here: %v", err)
	}
	f.Close()

	if _, err := ReadContainerHeader(out); err != nil {
		t.Fatalf("the header must stay readable whatever the payload size: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- VerifyPassword(out, "pw") }()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "too large") {
			t.Fatalf("an oversized payload must be refused by size, got %v", err)
		}
	case <-ctx.Done():
		t.Fatal("refusing an oversized payload must not read it")
	}
}

func TestUnwrapRoundTripKeepsLargeContents(t *testing.T) {
	// Several AES blocks and a size that is not a multiple of anything, so in-place
	// sealing and opening are exercised on a buffer the cipher has to grow.
	contents := strings.Repeat("SQLite format 3\x00 page ", 4099)
	dir, out := writeContainer(t, contents, "pw")
	opened := filepath.Join(dir, "opened.db")
	if _, err := UnwrapContainer(out, opened, "pw"); err != nil {
		t.Fatalf("UnwrapContainer: %v", err)
	}
	got, err := os.ReadFile(opened)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != contents {
		t.Fatalf("contents differ after the round trip (%d vs %d bytes)", len(got), len(contents))
	}
}
