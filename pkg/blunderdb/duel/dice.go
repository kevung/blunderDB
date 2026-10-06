package duel

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
)

// seedBytes is the length of a dice seed: 256 bits, drawn from the system.
const seedBytes = 32

// newSeed draws a dice seed from r and returns it hex-encoded, as the draft and
// the Match's origin store it.
func newSeed(r io.Reader) (string, error) {
	b := make([]byte, seedBytes)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", fmt.Errorf("draw the dice seed: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Fingerprint is the SHA-256 of the seed's bytes, hex-encoded. It is published
// when the Duel is created; the seed itself is revealed with the Match, and
// anyone can then check the one against the other (ADR-0072 rule 8).
func Fingerprint(seed string) (string, error) {
	b, err := hex.DecodeString(seed)
	if err != nil {
		return "", fmt.Errorf("dice seed: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// MaxContribution bounds a Side's contribution to a combined seed, in bytes.
const MaxContribution = 64

// CombinedSeed is the seed a Duel created with a combined seed rolls from:
// the sealed seed, whose fingerprint was published before any contribution,
// combined with what the external Sides contributed, in player order — a
// delegated Side contributes the empty string. The Arbiter sealed before
// seeing the contributions and the Sides contributed without knowing the
// seed, so none of the three chose the dice (ADR-0072 rule 8).
//
// It is stated here so that it can be recomputed outside blunderDB: the
// HMAC-SHA256 with key the sealed seed's bytes over, for player 1 then player
// 2, the contribution's length in bytes as 4 bytes big-endian followed by its
// UTF-8 bytes; hex-encoded, it is the seed Roll takes.
func CombinedSeed(seed string, contributions [2]string) (string, error) {
	key, err := hex.DecodeString(seed)
	if err != nil {
		return "", fmt.Errorf("dice seed: %w", err)
	}
	mac := hmac.New(sha256.New, key)
	for _, c := range contributions {
		var n [4]byte
		binary.BigEndian.PutUint32(n[:], uint32(len(c)))
		mac.Write(n[:])
		mac.Write([]byte(c))
	}
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// Roll is the roll of rank n of a Duel, a function of its seed and of n alone:
// resuming a Duel or replaying it never changes a roll.
//
// It is stated here so that it can be recomputed outside blunderDB. The bytes
// of HMAC-SHA256(key = seed bytes, message = n as 8 bytes big-endian, then a
// block counter as 4 bytes big-endian, starting at 0) are read in order; a byte
// below 252 gives the die 1 + byte mod 6, a byte from 252 up is skipped so that
// every face is equally likely; the first two dice kept are the roll. A block
// runs out only when it holds fewer than two bytes below 252, and the next
// block takes over.
//
// The roll is in drawing order. At the opening of a game, the first die is
// player 1's and the second player 2's: a double is drawn again at the next
// rank, and the higher die plays first.
func Roll(seed string, n int) ([2]int, error) {
	key, err := hex.DecodeString(seed)
	if err != nil {
		return [2]int{}, fmt.Errorf("dice seed: %w", err)
	}
	var out [2]int
	got := 0
	for block := uint32(0); got < 2; block++ {
		mac := hmac.New(sha256.New, key)
		var msg [12]byte
		binary.BigEndian.PutUint64(msg[:8], uint64(n))
		binary.BigEndian.PutUint32(msg[8:], block)
		mac.Write(msg[:])
		for _, b := range mac.Sum(nil) {
			if b >= 252 {
				continue
			}
			out[got] = 1 + int(b%6)
			if got++; got == 2 {
				break
			}
		}
	}
	return out, nil
}
