package issuance

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// identityFile is the name of the Issuer identity inside the config directory. A writable
// config directory is an *essential* host capability (ADR-0004), so failing to create the
// identity there is a real error rather than a degraded mode.
const identityFile = "identity.json"

// IdentityFileExtension is the extension of the single file an Issuer moves between
// machines.
const IdentityFileExtension = ".bdbid"

// Identity is the durable signing identity of an Issuer. It belongs to a person, not to a
// database: every copy the same Issuer produces carries the same fingerprint, which is what
// lets a copy found in the wild be traced back to one emission.
//
// It comes into existence without being asked for — there is no "do not sign" option,
// because nobody rationally chooses to make their own Watermark forgeable, and an unsigned
// regime would only double the states the verification path has to explain.
type Identity struct {
	Name string
	priv ed25519.PrivateKey
}

// storedIdentity is the on-disk shape. The seed is kept rather than the expanded key so the
// file stays small and the transfer format stays obvious.
type storedIdentity struct {
	Version int    `json:"version"`
	Name    string `json:"name"`
	Seed    string `json:"seed"`
	// KDF is empty for the local file and set on an exported file protected by a
	// passphrase. The local file is deliberately unprotected: it is a signing key whose
	// theft lets someone forge emissions in the Issuer's name — real, but requiring
	// access to their machine — and a passphrase there would put a forgotten secret
	// between an ordinary user and a feature they never asked to configure.
	KDF   string `json:"kdf,omitempty"`
	Salt  string `json:"salt,omitempty"`
	Nonce string `json:"nonce,omitempty"`
}

// NewIdentity mints a fresh Issuer identity. Callers normally want LoadOrCreateIdentity.
func NewIdentity(name string) (*Identity, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("cannot generate issuer identity: %w", err)
	}
	return &Identity{Name: strings.TrimSpace(name), priv: priv}, nil
}

// IdentityPath is where the identity lives inside a config directory.
func IdentityPath(configDir string) string { return filepath.Join(configDir, identityFile) }

// LoadIdentity reads the identity from configDir. It returns (nil, nil) when none exists
// yet — not having issued anything is the normal state.
func LoadIdentity(configDir string) (*Identity, error) {
	raw, err := os.ReadFile(IdentityPath(configDir))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read issuer identity: %w", err)
	}
	return decodeIdentity(raw, "")
}

// LoadOrCreateIdentity returns the Issuer identity, creating it silently on first use.
// defaultName seeds the display name when the identity has to be created; it is only a
// label and the Issuer can change it later.
func LoadOrCreateIdentity(configDir, defaultName string) (*Identity, error) {
	if id, err := LoadIdentity(configDir); err != nil || id != nil {
		return id, err
	}
	id, err := NewIdentity(defaultName)
	if err != nil {
		return nil, err
	}
	if err := id.save(configDir); err != nil {
		return nil, err
	}
	return id, nil
}

// Rename changes the display name carried by future Watermarks. Copies already issued keep
// the name they were sealed with, which is the point of sealing them.
func (id *Identity) Rename(configDir, name string) error {
	id.Name = strings.TrimSpace(name)
	return id.save(configDir)
}

func (id *Identity) save(configDir string) error {
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}
	raw, err := json.Marshal(storedIdentity{
		Version: 1,
		Name:    id.Name,
		Seed:    base64.StdEncoding.EncodeToString(id.priv.Seed()),
	})
	if err != nil {
		return err
	}
	// 0600: the file is enough to sign in the Issuer's name.
	return os.WriteFile(IdentityPath(configDir), raw, 0o600)
}

// PublicKey exposes the verifying key embedded in every Envelope this identity seals.
func (id *Identity) PublicKey() ed25519.PublicKey {
	return id.priv.Public().(ed25519.PublicKey)
}

// Fingerprint is the public form an Issuer can write on a whiteboard so recipients can
// check where a file came from.
func (id *Identity) Fingerprint() string { return FormatFingerprint(id.PublicKey()) }

// ExportIdentity writes the single file an Issuer carries to another machine. The
// passphrase is optional and applies only here: this is the copy that travels by mail or
// on a USB stick, which is the exposed one.
func (id *Identity) ExportIdentity(path, passphrase string) error {
	stored := storedIdentity{Version: 1, Name: id.Name}
	seed := id.priv.Seed()
	if passphrase == "" {
		stored.Seed = base64.StdEncoding.EncodeToString(seed)
	} else {
		sealed, salt, nonce, err := encryptSecret(seed, passphrase)
		if err != nil {
			return err
		}
		stored.KDF = kdfArgon2id
		stored.Seed = base64.StdEncoding.EncodeToString(sealed)
		stored.Salt = base64.StdEncoding.EncodeToString(salt)
		stored.Nonce = base64.StdEncoding.EncodeToString(nonce)
	}
	raw, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

// ImportIdentity reads a transferred identity file and installs it as the identity of this
// machine, replacing any existing one. Holding the same identity on two machines is the
// intended outcome: it is one person.
func ImportIdentity(configDir, path, passphrase string) (*Identity, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read identity file: %w", err)
	}
	id, err := decodeIdentity(raw, passphrase)
	if err != nil {
		return nil, err
	}
	if err := id.save(configDir); err != nil {
		return nil, err
	}
	return id, nil
}

// IdentityFileNeedsPassphrase reports whether the file at path is passphrase-protected, so
// a caller can ask for one only when it is actually needed.
func IdentityFileNeedsPassphrase(path string) (bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("cannot read identity file: %w", err)
	}
	var stored storedIdentity
	if err := json.Unmarshal(raw, &stored); err != nil {
		return false, fmt.Errorf("not a blunderDB identity file: %w", err)
	}
	return stored.KDF != "", nil
}

func decodeIdentity(raw []byte, passphrase string) (*Identity, error) {
	var stored storedIdentity
	if err := json.Unmarshal(raw, &stored); err != nil {
		return nil, fmt.Errorf("not a blunderDB identity file: %w", err)
	}
	sealed, err := base64.StdEncoding.DecodeString(stored.Seed)
	if err != nil {
		return nil, fmt.Errorf("corrupt identity file: %w", err)
	}
	seed := sealed
	if stored.KDF != "" {
		if passphrase == "" {
			return nil, ErrPassphraseRequired
		}
		salt, err := base64.StdEncoding.DecodeString(stored.Salt)
		if err != nil {
			return nil, fmt.Errorf("corrupt identity file: %w", err)
		}
		nonce, err := base64.StdEncoding.DecodeString(stored.Nonce)
		if err != nil {
			return nil, fmt.Errorf("corrupt identity file: %w", err)
		}
		if seed, err = decryptSecret(sealed, passphrase, salt, nonce); err != nil {
			return nil, err
		}
	}
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("corrupt identity file: unexpected key size")
	}
	return &Identity{Name: stored.Name, priv: ed25519.NewKeyFromSeed(seed)}, nil
}

func hashBytes(b []byte) []byte {
	sum := sha256.Sum256(b)
	return sum[:]
}
