package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// maxCanonicalDicePerGame bounds the dice included in the canonical hash so it
// is identical across export formats.
const maxCanonicalDicePerGame = 10

// CanonicalMatchHash is the format-independent hash of a match, the one that
// detects the same match arriving in another format (or as a transcription) and
// enriches it instead of storing a second copy. games holds, per game in order,
// the dice of its checker plays — dances included — as the source writes them;
// the order of the two dice does not matter.
//
// Every importer and the transcription hash through it, so the scheme is
// written once.
func CanonicalMatchHash(player1, player2 string, matchLength int, games [][][2]int) string {
	var b strings.Builder
	p1 := strings.TrimSpace(strings.ToLower(player1))
	p2 := strings.TrimSpace(strings.ToLower(player2))
	if p1 > p2 {
		p1, p2 = p2, p1
	}
	fmt.Fprintf(&b, "canonical2:%s|%s|%d|%d|", p1, p2, matchLength, len(games))
	for gi, dice := range games {
		fmt.Fprintf(&b, "g%d|", gi)
		for i, d := range dice {
			if i >= maxCanonicalDicePerGame {
				break
			}
			d1, d2 := d[0], d[1]
			if d1 > d2 {
				d1, d2 = d2, d1
			}
			fmt.Fprintf(&b, "d%d%d|", d1, d2)
		}
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}
