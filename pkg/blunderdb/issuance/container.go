package issuance

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// ContainerExtension is the extension of a password-protected copy.
const ContainerExtension = ".dbx"

// containerMagic opens every protected copy, whatever its version: it identifies the file
// type, and the version lives in the header that follows. Changing it would make every
// existing container unrecognisable to IsContainer.
var containerMagic = []byte("BDBX\x01")

// Container versions. The layout is the same for both — magic, big-endian header length,
// JSON header, AES-GCM payload — and they differ in one thing: what the AEAD authenticates.
//
//   - Version 1 sealed the payload alone. The cleartext header (watermark, salt, nonce) could
//     be rewritten without the password and without detection: a file could be relabelled as
//     coming from someone else while its contents still opened.
//   - Version 2 passes every byte that precedes the payload — magic, length prefix and header
//     — as the AEAD's additional data. Altering any of them makes Open fail exactly as a
//     wrong passphrase does.
//
// Version 1 is still read, with a logged warning, because files already handed out cannot be
// recalled; nothing writes it any more.
const (
	containerVersionUnauthenticatedHeader = 1
	containerVersionCurrent               = 2
)

// maxContainerPayload caps what this package will hold in memory at once. AES-GCM
// authenticates a whole message before it releases a byte of plaintext, so wrapping and
// unwrapping are single-shot operations; course databases are a few megabytes, and refusing
// an implausibly large one is better than exhausting memory on a machine that has none to
// spare. Read paths check it against the file's size *before* allocating anything.
const maxContainerPayload = 2 << 30 // 2 GiB

// maxContainerHeader bounds the cleartext header. A real one is a few hundred bytes of JSON
// (a watermark is a signed sentence, not a document); the bound exists so a hostile length
// prefix cannot ask for a gigabyte before the password is even wanted.
const maxContainerHeader = 1 << 20

// ContainerHeader travels **in the clear** at the head of a protected copy. That is
// deliberate: it lets anyone see where the file came from before deciding whether to ask for
// its password, and it keeps the origin readable on a file nobody can open any more.
//
// It follows that the header must never carry anything the password is meant to protect. It
// carries the Watermark, which the producer wanted attached to the file anyway, and the
// parameters needed to derive the key — nothing else.
//
// Clear does not mean unprotected: from version 2 the header's bytes are authenticated by
// the payload's AEAD, so it cannot be altered without the password either.
type ContainerHeader struct {
	Version   int      `json:"version"`
	Watermark Envelope `json:"watermark"`
	KDF       string   `json:"kdf"`
	// Argon2 records the derivation's cost parameters. Absent from version-1 files, which
	// all used the defaults; refused when it names anything else (see resolveArgon2).
	Argon2 *Argon2Params `json:"argon2,omitempty"`
	Salt   string        `json:"salt"`
	Nonce  string        `json:"nonce"`
}

// WrapContainer writes dbPath into outPath as an encrypted copy: cleartext header, then the
// database sealed under a passphrase, with the header bound to the payload (version 2).
//
// What this protects is the *transport* — the stray file in a downloads folder, the
// attachment forwarded by mistake, the link shared without the password. It does not
// protect against the legitimate recipient, who holds the passphrase and is the actual
// source of leaks; that is what the Watermark is for. Every label shown to a user must say
// so.
func WrapContainer(dbPath, outPath string, watermark Envelope, passphrase string) error {
	if passphrase == "" {
		return fmt.Errorf("an encrypted copy needs a password")
	}
	f, err := os.Open(dbPath)
	if err != nil {
		return fmt.Errorf("cannot read the database to protect: %w", err)
	}
	defer f.Close()
	// Sized for in-place sealing: one buffer holds the plaintext, then the ciphertext.
	plaintext, err := readExactly(f, 0, maxContainerPayload, gcmOverhead, "database too large to protect in one file")
	if err != nil {
		return fmt.Errorf("cannot read the database to protect: %w", err)
	}
	salt, nonce, err := newSaltAndNonce()
	if err != nil {
		return err
	}
	prefix, err := encodeContainerPrefix(ContainerHeader{
		Version:   containerVersionCurrent,
		Watermark: watermark,
		KDF:       kdfArgon2id,
		Argon2:    &argon2Default,
		Salt:      base64.StdEncoding.EncodeToString(salt),
		Nonce:     base64.StdEncoding.EncodeToString(nonce),
	})
	if err != nil {
		return err
	}
	sealed, err := sealSecret(plaintext, prefix, passphrase, salt, nonce)
	if err != nil {
		return err
	}
	out, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("cannot write the protected copy: %w", err)
	}
	if _, err := out.Write(prefix); err != nil {
		out.Close()
		return fmt.Errorf("cannot write the protected copy: %w", err)
	}
	if _, err := out.Write(sealed); err != nil {
		out.Close()
		return fmt.Errorf("cannot write the protected copy: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("cannot write the protected copy: %w", err)
	}
	return nil
}

// encodeContainerPrefix renders everything that precedes the payload: magic, header length,
// header JSON. In version 2 these exact bytes are the AEAD's additional data.
func encodeContainerPrefix(header ContainerHeader) ([]byte, error) {
	raw, err := json.Marshal(header)
	if err != nil {
		return nil, err
	}
	if len(raw) > maxContainerHeader {
		return nil, fmt.Errorf("protected file header too large (%d bytes)", len(raw))
	}
	prefix := make([]byte, 0, len(containerMagic)+4+len(raw))
	prefix = append(prefix, containerMagic...)
	prefix = binary.BigEndian.AppendUint32(prefix, uint32(len(raw))) //nolint:gosec // G115: bounded by the maxContainerHeader check just above, never negative or truncating
	return append(prefix, raw...), nil
}

// IsContainer reports whether path is a password-protected copy rather than an ordinary
// database. Cheap enough to call on every open.
func IsContainer(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	magic := make([]byte, len(containerMagic))
	if _, err := io.ReadFull(f, magic); err != nil {
		return false
	}
	return bytes.Equal(magic, containerMagic)
}

// ReadContainerHeader returns the cleartext header — Watermark included — **without the
// password**. This is what keeps a protected copy's origin readable. Only the header is
// read: the payload, whatever its size, is never touched.
func ReadContainerHeader(path string) (ContainerHeader, error) {
	f, err := os.Open(path)
	if err != nil {
		return ContainerHeader{}, fmt.Errorf("cannot read file: %w", err)
	}
	defer f.Close()
	c, err := readContainerPrefix(f)
	return c.header, err
}

// VerifyPassword reports whether the password opens this protected copy, without writing
// anything. AES-GCM authenticates the whole payload, so this necessarily decrypts it and
// throws the result away — there is no cheaper honest check, and a wrong password must never
// be mistaken for a right one.
func VerifyPassword(path, password string) error {
	_, _, err := openContainer(path, password)
	return err
}

// UnwrapContainer opens a protected copy into an ordinary database at outPath, the
// one time a passphrase is ever asked for. From then on the recipient works with a normal
// file: no prompt, no re-encryption, no ceremony.
//
// Memory: the payload is held once. AES-GCM must authenticate the whole message before it
// may release a single byte of plaintext, so a container cannot be streamed through a
// bounded buffer without changing the format to chunked AEAD — not worth a third version
// while the payload is capped at maxContainerPayload. What this does instead is decrypt in
// place (the ciphertext buffer becomes the plaintext) and write that buffer straight to
// outPath, so the peak is one copy of the file, checked against the cap before allocation.
func UnwrapContainer(path, outPath, passphrase string) (ContainerHeader, error) {
	header, plaintext, err := openContainer(path, passphrase)
	if err != nil {
		return header, err
	}
	if err := os.WriteFile(outPath, plaintext, 0o600); err != nil {
		return header, fmt.Errorf("cannot write the opened database: %w", err)
	}
	return header, nil
}

// container is a parsed cleartext prefix.
type container struct {
	header ContainerHeader
	// prefix is every byte that precedes the payload — magic, length, header JSON — exactly
	// as read from the file. In version 2 it is the AEAD's additional data, so it is kept
	// verbatim rather than re-encoded: a re-encoding that differed by one byte would refuse
	// every legitimate file.
	prefix []byte
}

// additionalData is what the payload's AEAD binds besides the ciphertext.
func (c container) additionalData() []byte {
	if c.header.Version == containerVersionUnauthenticatedHeader {
		return nil
	}
	return c.prefix
}

// readContainerPrefix parses the cleartext prefix from r, leaving r at the first payload
// byte. It accepts any version whose layout it can parse, so the origin of a file written by
// a newer blunderDB stays readable; refusing to *open* one is openContainer's job.
func readContainerPrefix(r io.Reader) (container, error) {
	const lengthPrefix = 4
	fixed := make([]byte, len(containerMagic)+lengthPrefix)
	if _, err := io.ReadFull(r, fixed); err != nil {
		return container{}, fmt.Errorf("not a protected blunderDB file")
	}
	if !bytes.Equal(fixed[:len(containerMagic)], containerMagic) {
		return container{}, fmt.Errorf("not a protected blunderDB file")
	}
	headerLen := binary.BigEndian.Uint32(fixed[len(containerMagic):])
	if headerLen > maxContainerHeader {
		return container{}, fmt.Errorf("corrupt protected file: header claims %d bytes", headerLen)
	}
	prefix := make([]byte, len(fixed)+int(headerLen))
	copy(prefix, fixed)
	if _, err := io.ReadFull(r, prefix[len(fixed):]); err != nil {
		return container{}, fmt.Errorf("corrupt protected file: truncated header")
	}
	var header ContainerHeader
	if err := json.Unmarshal(prefix[len(fixed):], &header); err != nil {
		return container{}, fmt.Errorf("corrupt protected file: %w", err)
	}
	return container{header: header, prefix: prefix}, nil
}

// openContainer reads and decrypts a protected copy, returning its header and plaintext.
// The plaintext is the file's payload buffer decrypted in place.
func openContainer(path, passphrase string) (ContainerHeader, []byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return ContainerHeader{}, nil, fmt.Errorf("cannot read file: %w", err)
	}
	defer f.Close()
	c, err := readContainerPrefix(f)
	if err != nil {
		return ContainerHeader{}, nil, err
	}
	header := c.header
	switch header.Version {
	case containerVersionUnauthenticatedHeader:
		slog.Warn("opening a version-1 protected file: its cleartext header is not authenticated; export it again to upgrade it",
			"path", path)
	case containerVersionCurrent:
	default:
		return header, nil, fmt.Errorf("protected file version %d: this file needs a newer blunderDB", header.Version)
	}
	if passphrase == "" {
		return header, nil, ErrPassphraseRequired
	}
	params, err := resolveArgon2(header.KDF, header.Argon2)
	if err != nil {
		return header, nil, err
	}
	salt, err := base64.StdEncoding.DecodeString(header.Salt)
	if err != nil {
		return header, nil, fmt.Errorf("corrupt file: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(header.Nonce)
	if err != nil {
		return header, nil, fmt.Errorf("corrupt file: %w", err)
	}
	sealed, err := readExactly(f, int64(len(c.prefix)), maxContainerPayload+gcmOverhead, 0, "protected file too large to open")
	if err != nil {
		return header, nil, fmt.Errorf("cannot read file: %w", err)
	}
	plaintext, err := decryptSecret(sealed, c.additionalData(), passphrase, salt, nonce, params)
	if err != nil {
		return header, nil, err
	}
	return header, plaintext, nil
}

// readExactly reads the rest of f — from offset consumed to its end — into one buffer with
// spare capacity, refusing before allocating when that would exceed limit bytes. The size
// comes from Stat, so a file that grows underneath is read only up to its size at that
// instant; AES-GCM then rejects the truncated payload, which is the right answer.
func readExactly(f *os.File, consumed, limit, spare int64, tooLarge string) ([]byte, error) {
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	remaining := info.Size() - consumed
	if remaining < 0 {
		return nil, fmt.Errorf("truncated file")
	}
	if remaining > limit || remaining+spare > math.MaxInt {
		return nil, fmt.Errorf("%s (%d bytes)", tooLarge, info.Size())
	}
	buf := make([]byte, remaining, remaining+spare)
	if _, err := io.ReadFull(f, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// DefaultUnwrapPath is where a protected copy lands when opened: beside it, same name,
// ordinary extension.
//
// It must never return the container's own path — unwrapping would then overwrite the
// protected file with its own contents. That is not hypothetical: a protected file does not
// have to be named `.dbx` (blunderDB recognises one by its magic bytes, not its extension),
// so a container called `cours.db` is perfectly openable and would otherwise unwrap onto
// itself.
func DefaultUnwrapPath(containerPath string) string {
	dir := filepath.Dir(containerPath)
	base := strings.TrimSuffix(filepath.Base(containerPath), ContainerExtension)
	base = strings.TrimSuffix(base, ".db")
	target := filepath.Join(dir, base+".db")
	if target == filepath.Clean(containerPath) {
		target = filepath.Join(dir, base+"-ouverte.db")
	}
	return target
}

// ProtectedPath is the name a protected export should carry. blunderDB opens a container by
// its magic bytes, so the extension is a convention rather than a requirement — but a file
// whose name says `.db` and whose contents are encrypted misleads every other tool the user
// owns, so an export that asks for a password gets the `.dbx` name to match.
func ProtectedPath(path string) string {
	if strings.HasSuffix(strings.ToLower(path), ContainerExtension) {
		return path
	}
	return strings.TrimSuffix(path, ".db") + ContainerExtension
}
