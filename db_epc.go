package main

import "github.com/kevung/blunderdb/pkg/blunderdb/engine"

// ComputeEPCFromPosition computes the EPC for both players from a full board position.
// It extracts checkers on points 1-6 for each side (the bearing-off zone).
// Returns EPC for bottom player and top player.
func (d *Database) ComputeEPCFromPosition(position Position) (map[string]interface{}, error) {
	// In the board model: Points[1..6] = bottom player's bearing-off zone (points 1-6)
	// Points[19..24] = top player's bearing-off zone (from top player's perspective, points 24-19)
	// But we need to check from each player's perspective.

	// Bottom player (Black, index 0): points 1-6 in the board
	// Board.Points[1] to Board.Points[6]
	var bottomBoard [6]int
	bottomTotal := 0
	bottomAllInHome := true
	for i := 0; i < 6; i++ {
		pt := position.Board.Points[i+1]
		if pt.Color == Black {
			bottomBoard[i] = pt.Checkers
			bottomTotal += pt.Checkers
		}
	}
	// Check if all bottom player's checkers are in home board or borne off
	for i := 7; i <= 24; i++ {
		pt := position.Board.Points[i]
		if pt.Color == Black && pt.Checkers > 0 {
			bottomAllInHome = false
			break
		}
	}
	// Also check bar
	if position.Board.Points[BlackBar].Color == Black && position.Board.Points[BlackBar].Checkers > 0 {
		bottomAllInHome = false
	}

	// Top player (White, index 1): points 24-19 in the board (indices 24,23,22,21,20,19)
	// From White's perspective, point 24 is their 1-point, 23 is their 2-point, etc.
	var topBoard [6]int
	topTotal := 0
	topAllInHome := true
	for i := 0; i < 6; i++ {
		pt := position.Board.Points[24-i]
		if pt.Color == White {
			topBoard[i] = pt.Checkers
			topTotal += pt.Checkers
		}
	}
	// Check if all top player's checkers are in home board or borne off
	for i := 1; i <= 18; i++ {
		pt := position.Board.Points[i]
		if pt.Color == White && pt.Checkers > 0 {
			topAllInHome = false
			break
		}
	}
	if position.Board.Points[WhiteBar].Color == White && position.Board.Points[WhiteBar].Checkers > 0 {
		topAllInHome = false
	}

	result := map[string]interface{}{
		"bottomEPC":          nil,
		"topEPC":             nil,
		"bottomAllInHome":    bottomAllInHome,
		"topAllInHome":       topAllInHome,
		"bottomCheckerCount": bottomTotal,
		"topCheckerCount":    topTotal,
	}

	if bottomAllInHome && bottomTotal > 0 {
		epc, err := engine.ComputeEPC(bottomBoard)
		if err == nil {
			result["bottomEPC"] = epc
		}
	}

	if topAllInHome && topTotal > 0 {
		epc, err := engine.ComputeEPC(topBoard)
		if err == nil {
			result["topEPC"] = epc
		}
	}

	return result, nil
}
