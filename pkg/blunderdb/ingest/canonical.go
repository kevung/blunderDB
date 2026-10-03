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

// DiceMatchHash is the hash of a match without its names: its length, the
// score at the start of its first game and the dice of the checker moves of
// every game, in order. The same match stored under other player names (or
// with the seats swapped) shares it, which MatchHash and CanonicalHash —
// both over the names — cannot see. The order of the two dice and of the two
// scores does not matter. initial holds each game's starting score; only the
// first is read. It is empty when no game holds a real roll.
func DiceMatchHash(matchLength int, initial [][2]int, games [][][2]int) string {
	// A match without one real roll has no dice to recognise it by: every
	// such match would share a hash and be suspected of being the others.
	rolls := 0
	for _, g := range games {
		for _, d := range g {
			if d[0] >= 1 && d[0] <= 6 && d[1] >= 1 && d[1] <= 6 {
				rolls++
			}
		}
	}
	if rolls == 0 {
		return ""
	}
	var b strings.Builder
	s1, s2 := 0, 0
	if len(initial) > 0 {
		s1, s2 = initial[0][0], initial[0][1]
		if s1 > s2 {
			s1, s2 = s2, s1
		}
	}
	fmt.Fprintf(&b, "dice1:%d|%d-%d|%d|", matchLength, s1, s2, len(games))
	for _, g := range games {
		b.WriteString(diceGameKey(g))
		b.WriteByte('|')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// diceGameKey writes one game's dice as one byte per roll, the smaller die
// first, so two games compare — and one prefixes another — as strings.
func diceGameKey(dice [][2]int) string {
	k := make([]byte, len(dice))
	for i, d := range dice {
		d1, d2 := d[0], d[1]
		if d1 > d2 {
			d1, d2 = d2, d1
		}
		k[i] = byte('0' + d1*8 + d2)
	}
	return string(k)
}

// graphDice is what DiceMatchHash reads from a graph: the dice of its
// checker moves per game and each game's starting score — the same rows
// MatchStore.DiceSequences reads back once the graph is stored.
func graphDice(g *MatchGraph) (initial [][2]int, games [][][2]int) {
	for gi := range g.Games {
		gg := &g.Games[gi]
		initial = append(initial, [2]int{int(gg.Game.InitialScore[0]), int(gg.Game.InitialScore[1])})
		var dice [][2]int
		for mi := range gg.Moves {
			mv := &gg.Moves[mi].Move
			if mv.MoveType == "checker" && mv.Dice[0] > 0 {
				dice = append(dice, [2]int{int(mv.Dice[0]), int(mv.Dice[1])})
			}
		}
		games = append(games, dice)
	}
	return initial, games
}
