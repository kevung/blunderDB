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

// ContainerExtension is the extension of an encrypted Issued copy.
const ContainerExtension = ".bdbx"

var containerMagic = []byte("BDBX\x01")

// maxContainerPayload caps what this package will read into memory at once. AES-GCM
// authenticates a whole message, so wrapping is a single-shot operation; course databases
// are a few megabytes, and refusing an implausibly large one is better than exhausting
// memory on a machine that has none to spare.
const maxContainerPayload = 2 << 30 // 2 GiB

// ContainerHeader travels **in the clear** at the head of an encrypted Issued copy. That is
// deliberate and it is the whole point of encrypting the transport rather than the database:
// a copy found on a forum stays identifiable without anybody having to decrypt it.
//
// It follows that the header must never carry anything the passphrase is meant to protect.
// It carries the Watermark, which is an assertion the Issuer wanted attached to the file
// anyway, and the parameters needed to derive the key — nothing else.
type ContainerHeader struct {
	Version   int      `json:"version"`
	Watermark Envelope `json:"watermark"`
	KDF       string   `json:"kdf"`
	Salt      string   `json:"salt"`
	Nonce     string   `json:"nonce"`
}

// WrapContainer writes dbPath into outPath as an encrypted Issued copy: cleartext header,
// then the database sealed under a passphrase.
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

// IsContainer reports whether path is an encrypted Issued copy rather than an ordinary
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
// passphrase**. This is what makes a leaked encrypted copy identifiable.
func ReadContainerHeader(path string) (ContainerHeader, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ContainerHeader{}, fmt.Errorf("cannot read file: %w", err)
	}
	header, _, err := splitContainer(raw)
	return header, err
}

// UnwrapContainer opens an encrypted Issued copy into an ordinary database at outPath, the
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

// DefaultUnwrapPath is where an encrypted copy lands when opened: beside it, same name,
// ordinary extension.
func DefaultUnwrapPath(containerPath string) string {
	dir := filepath.Dir(containerPath)
	base := filepath.Base(containerPath)
	base = strings.TrimSuffix(base, ContainerExtension)
	return filepath.Join(dir, base+".db")
}
