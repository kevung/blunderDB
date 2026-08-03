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

// Argon2id parameters. Sized for an interactive prompt on a laptop: a second or so, 64 MiB.
// They are deliberately fixed rather than configurable — a knob here only lets a user make
// their own file weaker.
const (
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
)

// ErrPassphraseRequired is returned when a protected file is opened without one.
var ErrPassphraseRequired = errors.New("this file is protected by a passphrase")

// ErrWrongPassphrase is returned when the passphrase does not open the file. AES-GCM
// authenticates, so a wrong passphrase is detected rather than yielding garbage.
var ErrWrongPassphrase = errors.New("wrong passphrase")

func deriveKey(passphrase string, salt []byte) []byte {
	return argon2.IDKey([]byte(passphrase), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
}

// encryptSecret seals plaintext under a passphrase, returning the ciphertext with the salt
// and nonce needed to open it again.
func encryptSecret(plaintext []byte, passphrase string) (sealed, salt, nonce []byte, err error) {
	salt = make([]byte, 16)
	if _, err = rand.Read(salt); err != nil {
		return nil, nil, nil, fmt.Errorf("no entropy available: %w", err)
	}
	gcm, err := newGCM(deriveKey(passphrase, salt))
	if err != nil {
		return nil, nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, nil, nil, fmt.Errorf("no entropy available: %w", err)
	}
	return gcm.Seal(nil, nonce, plaintext, nil), salt, nonce, nil
}

// decryptSecret opens what encryptSecret sealed.
func decryptSecret(sealed []byte, passphrase string, salt, nonce []byte) ([]byte, error) {
	gcm, err := newGCM(deriveKey(passphrase, salt))
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("corrupt file: unexpected nonce size")
	}
	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
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
