package issuance

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ContainerExtension is the extension of a password-protected copy.
const ContainerExtension = ".dbx"

var containerMagic = []byte("BDBX\x01")

// maxContainerPayload caps what this package will read into memory at once. AES-GCM
// authenticates a whole message, so wrapping is a single-shot operation; course databases
// are a few megabytes, and refusing an implausibly large one is better than exhausting
// memory on a machine that has none to spare.
const maxContainerPayload = 2 << 30 // 2 GiB

// ContainerHeader travels **in the clear** at the head of a protected copy. That is
// deliberate: it lets anyone see where the file came from before deciding whether to ask for
// its password, and it keeps the origin readable on a file nobody can open any more.
//
// It follows that the header must never carry anything the password is meant to protect. It
// carries the Watermark, which the producer wanted attached to the file anyway, and the
// parameters needed to derive the key — nothing else.
type ContainerHeader struct {
	Version   int      `json:"version"`
	Watermark Envelope `json:"watermark"`
	KDF       string   `json:"kdf"`
	Salt      string   `json:"salt"`
	Nonce     string   `json:"nonce"`
}

// WrapContainer writes dbPath into outPath as an encrypted copy: cleartext header, then the
// database sealed under a passphrase.
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
	info, err := os.Stat(dbPath)
	if err != nil {
		return fmt.Errorf("cannot read the database to protect: %w", err)
	}
	if info.Size() > maxContainerPayload {
		return fmt.Errorf("database too large to protect in one file (%d bytes)", info.Size())
	}
	plaintext, err := os.ReadFile(dbPath)
	if err != nil {
		return fmt.Errorf("cannot read the database to protect: %w", err)
	}
	sealed, salt, nonce, err := encryptSecret(plaintext, passphrase)
	if err != nil {
		return err
	}
	header, err := json.Marshal(ContainerHeader{
		Version:   1,
		Watermark: watermark,
		KDF:       kdfArgon2id,
		Salt:      base64.StdEncoding.EncodeToString(salt),
		Nonce:     base64.StdEncoding.EncodeToString(nonce),
	})
	if err != nil {
		return err
	}
	out := make([]byte, 0, len(containerMagic)+4+len(header)+len(sealed))
	out = append(out, containerMagic...)
	out = binary.BigEndian.AppendUint32(out, uint32(len(header)))
	out = append(out, header...)
	out = append(out, sealed...)
	return os.WriteFile(outPath, out, 0o644)
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
	if _, err := f.Read(magic); err != nil {
		return false
	}
	return string(magic) == string(containerMagic)
}

// ReadContainerHeader returns the cleartext header — Watermark included — **without the
// password**. This is what keeps a protected copy's origin readable.
func ReadContainerHeader(path string) (ContainerHeader, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ContainerHeader{}, fmt.Errorf("cannot read file: %w", err)
	}
	header, _, err := splitContainer(raw)
	return header, err
}

// VerifyPassword reports whether the password opens this protected copy, without writing
// anything. AES-GCM authenticates the whole payload, so this necessarily decrypts it and
// throws the result away — there is no cheaper honest check, and a wrong password must never
// be mistaken for a right one.
func VerifyPassword(path, password string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}
	header, sealed, err := splitContainer(raw)
	if err != nil {
		return err
	}
	if password == "" {
		return ErrPassphraseRequired
	}
	salt, err := base64.StdEncoding.DecodeString(header.Salt)
	if err != nil {
		return fmt.Errorf("corrupt file: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(header.Nonce)
	if err != nil {
		return fmt.Errorf("corrupt file: %w", err)
	}
	_, err = decryptSecret(sealed, password, salt, nonce)
	return err
}

// UnwrapContainer opens a protected copy into an ordinary database at outPath, the
// one time a passphrase is ever asked for. From then on the recipient works with a normal
// file: no prompt, no re-encryption, no ceremony.
func UnwrapContainer(path, outPath, passphrase string) (ContainerHeader, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ContainerHeader{}, fmt.Errorf("cannot read file: %w", err)
	}
	header, sealed, err := splitContainer(raw)
	if err != nil {
		return ContainerHeader{}, err
	}
	if passphrase == "" {
		return header, ErrPassphraseRequired
	}
	salt, err := base64.StdEncoding.DecodeString(header.Salt)
	if err != nil {
		return header, fmt.Errorf("corrupt file: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(header.Nonce)
	if err != nil {
		return header, fmt.Errorf("corrupt file: %w", err)
	}
	plaintext, err := decryptSecret(sealed, passphrase, salt, nonce)
	if err != nil {
		return header, err
	}
	if err := os.WriteFile(outPath, plaintext, 0o644); err != nil {
		return header, fmt.Errorf("cannot write the opened database: %w", err)
	}
	return header, nil
}

func splitContainer(raw []byte) (ContainerHeader, []byte, error) {
	const prefix = 4
	if len(raw) < len(containerMagic)+prefix {
		return ContainerHeader{}, nil, fmt.Errorf("not a protected blunderDB file")
	}
	if string(raw[:len(containerMagic)]) != string(containerMagic) {
		return ContainerHeader{}, nil, fmt.Errorf("not a protected blunderDB file")
	}
	at := len(containerMagic)
	headerLen := int(binary.BigEndian.Uint32(raw[at : at+prefix]))
	at += prefix
	if headerLen < 0 || at+headerLen > len(raw) {
		return ContainerHeader{}, nil, fmt.Errorf("corrupt protected file: truncated header")
	}
	var header ContainerHeader
	if err := json.Unmarshal(raw[at:at+headerLen], &header); err != nil {
		return ContainerHeader{}, nil, fmt.Errorf("corrupt protected file: %w", err)
	}
	return header, raw[at+headerLen:], nil
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
