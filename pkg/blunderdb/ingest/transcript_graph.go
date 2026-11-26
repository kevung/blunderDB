package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// What a match the transcript package built is written as — a finished
// transcription, a Duel, an archived match read back (ADR-0119).

// RestoreTimecodes puts each Move's Repères back on the Action that produced it.
// The .mat replay lines them up: the game's index, the Move's number in it. An
// Action that does not find its Move — another kind, another roll — gets none
// rather than a stranger's; a resignation, which produces no Move, has none to
// find.
func RestoreTimecodes(doc *transcript.Document, games []*domain.Game, moves map[int64][]*domain.Move) {
	ann := transcript.Replay(*doc, 0)
	for i, info := range ann.Actions {
		if info.MoveNumber < 0 || info.GameIndex < 0 || info.GameIndex >= len(games) {
			continue
		}
		var mv *domain.Move
		for _, cand := range moves[games[info.GameIndex].ID] {
			if cand.MoveNumber == info.MoveNumber {
				mv = cand
				break
			}
		}
		if mv == nil {
			continue
		}
		a := &doc.Actions[i]
		switch a.Kind {
		case transcript.KindChecker, transcript.KindDance, transcript.KindUnrecorded:
			d := info.Before.Dice
			if mv.MoveType != "checker" || mv.Dice != [2]int32{int32(d[0]), int32(d[1])} {
				continue
			}
			a.RollTickMS = mv.RollTickMS
		default:
			if mv.MoveType != "cube" {
				continue
			}
		}
		a.TickMS = mv.TickMS
	}
}

// TranscriptGraph turns what the transcript package returns into the graph
// WriteMatch writes. It carries no analysis and no comment: the
// analysis is the batch's job afterwards (ADR-0045 §8), and a transcription
// has no notes to attach. The transcript's own ids are not database ids and
// are dropped; so are the hashes (see writeMatch).
func TranscriptGraph(parts transcript.Parts) *MatchGraph {
	// The graph is written in the seats the document holds: the callers
	// restate the header in that order once the match is written, and a
	// Duel keeps the seats it was played in. A finished transcription puts
	// the Main player first after that restatement (placeMainPlayerFirst).
	g := &MatchGraph{Match: *parts.Match, KeepSeats: true}
	g.Match.MatchHash, g.Match.CanonicalHash = "", ""

	for _, game := range parts.Games {
		moves := parts.Moves[game.ID]
		positions := parts.Positions[game.ID]

		gg := GameGraph{Game: *game}
		gg.Game.ID = 0
		gg.Game.MatchID = 0
		for i, mv := range moves {
			mg := MoveGraph{Move: *mv}
			mg.Move.GameID = 0
			if i < len(positions) {
				pos := positions[i]
				mg.Position = &pos
			}
			gg.Moves = append(gg.Moves, mg)
		}
		g.Games = append(g.Games, gg)
	}
	return g
}

// TranscriptHashes are the two content hashes of a match the transcript package built —
// transcribed, or played in a Duel.
//
// The canonical one is the importers' own ([CanonicalMatchHash]), so a
// later XG/GnuBG import of the same match enriches instead of duplicating. The format-specific one
// hashes the plays as typed and changes whenever the document does.
func TranscriptHashes(parts transcript.Parts) (matchHash, canonicalHash string) {
	m := parts.Match

	var b strings.Builder
	p1 := strings.TrimSpace(strings.ToLower(m.Player1Name))
	p2 := strings.TrimSpace(strings.ToLower(m.Player2Name))
	fmt.Fprintf(&b, "transcript1:%s|%s|%d|", p1, p2, m.MatchLength)
	for gi, game := range parts.Games {
		// The winner is hashed as a side (0/1/-1), like the document states it:
		// the hash follows the transcription, not the storage encoding.
		fmt.Fprintf(&b, "g%d:%d,%d,%d,%d|", gi,
			game.InitialScore[0], game.InitialScore[1], domain.WinnerSide(game.Winner), game.PointsWon)
		for mi, mv := range parts.Moves[game.ID] {
			fmt.Fprintf(&b, "m%d:%s,%d,d%d%d,p%s|", mi, mv.MoveType, mv.Player,
				mv.Dice[0], mv.Dice[1], mv.CheckerMove+mv.CubeAction)
		}
	}
	matchHash = transcriptHashHex(b.String())

	games := make([][][2]int, len(parts.Games))
	for gi, game := range parts.Games {
		for _, mv := range parts.Moves[game.ID] {
			if mv.MoveType == "checker" {
				games[gi] = append(games[gi], [2]int{int(mv.Dice[0]), int(mv.Dice[1])})
			}
		}
	}
	canonicalHash = CanonicalMatchHash(m.Player1Name, m.Player2Name, int(m.MatchLength), games)
	return matchHash, canonicalHash
}

func transcriptHashHex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
