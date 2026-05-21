package engine

import (
	"encoding/json"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// PositionColumns holds the derived scalar columns stored alongside the
// position JSON in the v2 schema. PopulatePositionColumns computes them from a
// (already-normalized) Position without hitting the DB.
type PositionColumns struct {
	ZobristHash   uint64
	DecisionType  int
	Dice1, Dice2  int
	Pip1, Pip2    int
	PipDiff       int
	Off1, Off2    int
	BackCheckers1 int
	BackCheckers2 int
	NoContact     bool
	Occupancy1    uint32
	Occupancy2    uint32
	PointMask1    uint32
	PointMask2    uint32
	// mirrors of Position fields for indexed columns
	CubeValue int
	CubeOwner int
	Score1    int
	Score2    int
	HasJacoby int
	HasBeaver int
}

// PopulatePositionColumns computes every derived column value for a Position.
// The input should already be normalized (PlayerOnRoll == 0); if it isn't the
// function normalizes it internally.
func PopulatePositionColumns(p *domain.Position) PositionColumns {
	norm := p.NormalizeForStorage()
	var c PositionColumns

	c.ZobristHash = ZobristHash(&norm)
	c.DecisionType = norm.DecisionType
	c.Dice1 = norm.Dice[0]
	c.Dice2 = norm.Dice[1]

	c.Pip1, c.Pip2 = PipCounts(norm.Board)
	c.PipDiff = c.Pip1 - c.Pip2

	c.Off1 = norm.Board.Bearoff[0]
	c.Off2 = norm.Board.Bearoff[1]

	// Back checkers: Black's (player-on-roll's) are in the opponent's home board (points 19-24);
	// White's are in Black's home board (points 1-6).
	for i := 19; i <= 24; i++ {
		if norm.Board.Points[i].Color == domain.Black && norm.Board.Points[i].Checkers > 0 {
			c.BackCheckers1 += norm.Board.Points[i].Checkers
		}
	}
	for i := 1; i <= 6; i++ {
		if norm.Board.Points[i].Color == domain.White && norm.Board.Points[i].Checkers > 0 {
			c.BackCheckers2 += norm.Board.Points[i].Checkers
		}
	}

	c.NoContact = norm.MatchesNoContact()

	c.Occupancy1, c.Occupancy2, c.PointMask1, c.PointMask2 = OccupancyMasks(&norm.Board)

	c.CubeValue = norm.Cube.Value
	c.CubeOwner = norm.Cube.Owner
	c.Score1 = norm.Score[0]
	c.Score2 = norm.Score[1]
	c.HasJacoby = norm.HasJacoby
	c.HasBeaver = norm.HasBeaver

	return c
}

// ---------------------------------------------------------------------------
// Compact board encoding (v2.2.0)
//
// The state column stores ONLY the board layout as a JSON array of 28 signed
// integers. Indices 0-25 correspond to Board.Points: positive values mean
// White checkers, negative mean Black, zero means empty. Indices 26-27 are
// Board.Bearoff[0] and Bearoff[1]. All other Position fields (cube, dice,
// score, decision_type, flags) are reconstructed from the denormalised
// columns on the same row.
//
// Format detection: state[0]=='[' → compact, state[0]=='{' → legacy JSON.
// ---------------------------------------------------------------------------

// EncodeBoardCompact encodes a Board as a compact JSON array of 28 signed ints.
func EncodeBoardCompact(b domain.Board) string {
	var vals [28]int
	for i := 0; i < domain.NumPoints+2; i++ {
		p := b.Points[i]
		if p.Checkers > 0 {
			if p.Color == domain.White {
				vals[i] = p.Checkers
			} else {
				vals[i] = -p.Checkers
			}
		}
	}
	vals[26] = b.Bearoff[0]
	vals[27] = b.Bearoff[1]
	data, _ := json.Marshal(vals)
	return string(data)
}

// DecodeBoardCompact decodes a compact JSON array back into a Board.
func DecodeBoardCompact(s string) domain.Board {
	var vals [28]int
	_ = json.Unmarshal([]byte(s), &vals)
	var b domain.Board
	for i := 0; i < domain.NumPoints+2; i++ {
		v := vals[i]
		if v > 0 {
			b.Points[i] = domain.Point{Checkers: v, Color: domain.White}
		} else if v < 0 {
			b.Points[i] = domain.Point{Checkers: -v, Color: domain.Black}
		}
	}
	b.Bearoff[0] = vals[26]
	b.Bearoff[1] = vals[27]
	return b
}

// IsCompactState returns true if the state string uses the compact array format.
func IsCompactState(s string) bool {
	return len(s) > 0 && s[0] == '['
}

// ReconstructPosition rebuilds a full Position from the compact state string
// and denormalised column values. For legacy JSON state it unmarshals the full
// Position but still overwrites with the column values (they are authoritative
// in the v2 schema).
func ReconstructPosition(id int64, state string, decisionType, playerOnRoll, dice1, dice2, cubeValue, cubeOwner, score1, score2, hasJacoby, hasBeaver int) domain.Position {
	var pos domain.Position
	if IsCompactState(state) {
		pos.Board = DecodeBoardCompact(state)
	} else {
		_ = json.Unmarshal([]byte(state), &pos)
	}
	pos.ID = id
	pos.DecisionType = decisionType
	pos.PlayerOnRoll = playerOnRoll
	pos.Dice = [2]int{dice1, dice2}
	pos.Cube = domain.Cube{Owner: cubeOwner, Value: cubeValue}
	pos.Score = [2]int{score1, score2}
	pos.HasJacoby = hasJacoby
	pos.HasBeaver = hasBeaver
	return pos
}
