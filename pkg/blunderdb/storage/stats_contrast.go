package storage

import "sort"

// PlayerContrast lists the positions two players both decided and answered
// differently: one played them well, the other did not. It is the side-by-side
// view of two players turned into positions to study (StatsStore.PlayerContrast).
type PlayerContrast struct {
	PlayerA string `json:"player_a"`
	PlayerB string `json:"player_b"`
	// ThresholdMP is the library's Error threshold: a play costing at least
	// this is not "well played".
	ThresholdMP int `json:"threshold_mp"`
	// CommonPositions counts the positions both players decided, whatever
	// the verdict: the base the contrast is read against.
	CommonPositions int `json:"common_positions"`
	// Positions are the contrasting ones, the widest gap between the two
	// players first.
	Positions []ContrastPosition `json:"positions"`
	// UnscoredMoves counts the two players' analysed plays whose error is not
	// written yet: the contrast leaves them out until the explicit ScoreMoves
	// pass (repair --move-errors) runs, since no import writes it. A play the
	// analysis cannot price stays counted after that pass.
	UnscoredMoves int `json:"unscored_moves"`
}

// ContrastPosition is one position of a PlayerContrast. A player's error is
// their worst play of the position (millipoints of normalised equity, the
// stats' own scale); a player who played it twice and erred once did not play
// it well.
type ContrastPosition struct {
	PositionID int64 `json:"position_id"`
	ErrorMPA   int64 `json:"error_mp_a"`
	ErrorMPB   int64 `json:"error_mp_b"`
	TimesA     int   `json:"times_a"`
	TimesB     int   `json:"times_b"`
	// WellPlayed is "a" or "b": the one who played the position well.
	WellPlayed string `json:"well_played"`
}

// ContrastRow is one player's record on one position, as a backend hands it
// to ContrastPlayers.
type ContrastRow struct {
	PositionID int64
	WorstMP    int64
	Times      int
}

// ContrastPlayers crosses the two players' rows. It is pure and shared by both
// backends so that what counts as a contrast cannot differ between them.
func ContrastPlayers(a, b []ContrastRow, thresholdMP int) (common int, positions []ContrastPosition) {
	byID := make(map[int64]ContrastRow, len(a))
	for _, r := range a {
		byID[r.PositionID] = r
	}
	positions = []ContrastPosition{}
	for _, rb := range b {
		ra, ok := byID[rb.PositionID]
		if !ok {
			continue
		}
		common++
		goodA, goodB := ra.WorstMP < int64(thresholdMP), rb.WorstMP < int64(thresholdMP)
		if goodA == goodB {
			continue
		}
		p := ContrastPosition{PositionID: ra.PositionID, ErrorMPA: ra.WorstMP, ErrorMPB: rb.WorstMP,
			TimesA: ra.Times, TimesB: rb.Times, WellPlayed: "a"}
		if goodB {
			p.WellPlayed = "b"
		}
		positions = append(positions, p)
	}
	gap := func(p ContrastPosition) int64 {
		if p.ErrorMPA > p.ErrorMPB {
			return p.ErrorMPA - p.ErrorMPB
		}
		return p.ErrorMPB - p.ErrorMPA
	}
	sort.Slice(positions, func(i, j int) bool {
		if gi, gj := gap(positions[i]), gap(positions[j]); gi != gj {
			return gi > gj
		}
		return positions[i].PositionID < positions[j].PositionID
	})
	return common, positions
}
