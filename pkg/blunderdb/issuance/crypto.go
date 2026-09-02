package issuance

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// kdfArgon2id names the only key derivation this package performs. It is written into every
// protected file so a future change can be recognised rather than silently mis-decrypted.
const kdfArgon2id = "argon2id"

// Argon2Params are the cost parameters of the key derivation, written into every protected
// file next to the KDF name. Recording them is what lets a file written today still open
// after the defaults move, and — the reverse — lets a file claiming parameters this build
// never used be refused instead of being fed to a derivation that will "succeed" with the
// wrong key and report a wrong passphrase.
type Argon2Params struct {
	Time    uint32 `json:"t"`
	Memory  uint32 `json:"m"` // KiB
	Threads uint8  `json:"p"`
}

// argon2Default is the one parameter set this package derives keys with. Sized for an
// interactive prompt on a laptop: a second or so, 64 MiB. Deliberately fixed rather than
// configurable — a knob here only lets a user make their own file weaker.
var argon2Default = Argon2Params{Time: 3, Memory: 64 * 1024, Threads: 4}

const argonKeyLen = 32

// resolveArgon2 returns the parameters a stored file asks for. A file written before the
// parameters were recorded (nil) used argon2Default, the only set that ever existed; a file
// recording any other set was not written by a blunderDB this package knows and is refused
// rather than tried.
func resolveArgon2(kdf string, stored *Argon2Params) (Argon2Params, error) {
	if kdf != kdfArgon2id {
		return Argon2Params{}, fmt.Errorf("unsupported key derivation %q: this file needs a newer blunderDB", kdf)
	}
	if stored == nil {
		return argon2Default, nil
	}
	if *stored != argon2Default {
		return Argon2Params{}, fmt.Errorf("unsupported key derivation parameters (t=%d m=%d p=%d): this file needs a newer blunderDB",
			stored.Time, stored.Memory, stored.Threads)
	}
	return *stored, nil
}

// ErrPassphraseRequired is returned when a protected file is opened without one.
var ErrPassphraseRequired = errors.New("this file is protected by a passphrase")

// ErrWrongPassphrase is returned when the passphrase does not open the file. AES-GCM
// authenticates, so a wrong passphrase is detected rather than yielding garbage. The same
// failure is reported when the file itself — payload or, from container version 2, header —
// was altered: an AEAD cannot tell the two apart, and must not try.
var ErrWrongPassphrase = errors.New("wrong passphrase, or the file was altered")

func deriveKey(passphrase string, salt []byte, p Argon2Params) []byte {
	return argon2.IDKey([]byte(passphrase), salt, p.Time, p.Memory, p.Threads, argonKeyLen)
}

// newSaltAndNonce draws the per-file salt and nonce. Both travel in the clear beside the
// ciphertext; neither is secret, both must be unique per seal.
func newSaltAndNonce() (salt, nonce []byte, err error) {
	salt = make([]byte, 16)
	if _, err = rand.Read(salt); err != nil {
		return nil, nil, fmt.Errorf("no entropy available: %w", err)
	}
	nonce = make([]byte, gcmNonceSize)
	if _, err = rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("no entropy available: %w", err)
	}
	return salt, nonce, nil
}

// gcmNonceSize and gcmOverhead are AES-GCM's standard nonce and tag sizes, stated so the
// container can size its buffers before the cipher exists.
const (
	gcmNonceSize = 12
	gcmOverhead  = 16
)

// sealSecret seals plaintext under a passphrase with the given salt and nonce. additionalData
// is authenticated but not encrypted: the caller passes whatever travels in the clear next to
// the ciphertext, so that altering it is detected exactly like altering the ciphertext. It is
// nil only for the identity file, whose clear fields are the parameters of the seal itself.
//
// The ciphertext is written over plaintext when its capacity allows (len+gcmOverhead): a
// caller holding a large payload then keeps one copy in memory, not two. plaintext must not
// be used afterwards.
func sealSecret(plaintext, additionalData []byte, passphrase string, salt, nonce []byte) ([]byte, error) {
	gcm, err := newGCM(deriveKey(passphrase, salt, argon2Default))
	if err != nil {
		return nil, err
	}
	return gcm.Seal(plaintext[:0], nonce, plaintext, additionalData), nil
}

// encryptSecret is sealSecret with fresh salt and nonce, for callers that store both
// themselves.
func encryptSecret(plaintext, additionalData []byte, passphrase string) (sealed, salt, nonce []byte, err error) {
	salt, nonce, err = newSaltAndNonce()
	if err != nil {
		return nil, nil, nil, err
	}
	sealed, err = sealSecret(plaintext, additionalData, passphrase, salt, nonce)
	if err != nil {
		return nil, nil, nil, err
	}
	return sealed, salt, nonce, nil
}

// decryptSecret opens what encryptSecret sealed, under the same additionalData.
//
// The plaintext is written over sealed: AES-GCM decrypts in place, so a caller holding a
// large payload keeps one copy in memory, not two. sealed must not be used afterwards.
func decryptSecret(sealed, additionalData []byte, passphrase string, salt, nonce []byte, p Argon2Params) ([]byte, error) {
	gcm, err := newGCM(deriveKey(passphrase, salt, p))
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("corrupt file: unexpected nonce size")
	}
	plaintext, err := gcm.Open(sealed[:0], nonce, sealed, additionalData)
	if err != nil {
		return nil, ErrWrongPassphrase
	}
	return plaintext, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("cannot set up cipher: %w", err)
	}
	return cipher.NewGCM(block)
}
