package domain

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

// MatchesMirrorPosition reports whether the position (or its mirror) matches
// the checker layout of filter.
func (p *Position) MatchesMirrorPosition(filter Position) bool {
	mirroredPosition := p.Mirror()
	return p.MatchesCheckerPosition(filter) || mirroredPosition.MatchesCheckerPosition(filter)
}

// Mirror creates a mirrored version of the current Position.
// It reverses the board points, swaps the bearoff positions,
// changes the player on roll, swaps the scores, and changes the cube owner.
// Returns the mirrored Position.
func (p *Position) Mirror() Position {
	mirrored := *p
	for i, point := range p.Board.Points {
		mirrored.Board.Points[25-i] = Point{
			Color:    point.Color,
			Checkers: point.Checkers,
		}
		if point.Color != -1 {
			mirrored.Board.Points[25-i].Color = 1 - point.Color
		}
	}
	mirrored.Board.Bearoff[0], mirrored.Board.Bearoff[1] = p.Board.Bearoff[1], p.Board.Bearoff[0]
	mirrored.PlayerOnRoll = 1 - p.PlayerOnRoll
	mirrored.Score[0], mirrored.Score[1] = p.Score[1], p.Score[0]
	if p.Cube.Owner != -1 {
		mirrored.Cube.Owner = 1 - p.Cube.Owner
	}
	return mirrored
}

// NormalizeForStorage returns a normalized version of the position for storage.
// Positions are always stored from the player on roll's perspective (player_on_roll = 0).
// If player_on_roll is 1, the position is mirrored so that player_on_roll becomes 0.
// This prevents storing duplicate positions that are just mirror images of each other.
func (p *Position) NormalizeForStorage() Position {
	if p.PlayerOnRoll == 1 {
		return p.Mirror()
	}
	return *p
}

func (p *Position) MatchesDecisionType(filter Position) bool {
	return p.DecisionType == filter.DecisionType && p.PlayerOnRoll == filter.PlayerOnRoll
}

// MatchesPlayer1CheckerOff matches by player 1's borne-off checker count.
func (p *Position) MatchesPlayer1CheckerOff(filter string) bool {
	checkersOff := p.Board.Bearoff[0]

	if strings.HasPrefix(filter, "o>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return checkersOff >= value
	} else if strings.HasPrefix(filter, "o<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return checkersOff <= value
	} else if strings.HasPrefix(filter, "o") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			values = append(values, values[0]) // Handle case where 'ox' means 'ox,x'
		}
		if len(values) != 2 {
			slog.Warn("parsing filter values", "value", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			slog.Warn("parsing filter values", "v1", values[0], "v2", values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return checkersOff >= minValue && checkersOff <= maxValue
	}
	return false
}

// MatchesPlayer2CheckerOff matches by player 2's borne-off checker count.
func (p *Position) MatchesPlayer2CheckerOff(filter string) bool {
	checkersOff := p.Board.Bearoff[1]

	if strings.HasPrefix(filter, "O>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return checkersOff >= value
	} else if strings.HasPrefix(filter, "O<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return checkersOff <= value
	} else if strings.HasPrefix(filter, "O") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			values = append(values, values[0]) // Handle case where 'Ox' means 'Ox,x'
		}
		if len(values) != 2 {
			slog.Warn("parsing filter values", "value", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			slog.Warn("parsing filter values", "v1", values[0], "v2", values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return checkersOff >= minValue && checkersOff <= maxValue
	}
	return false
}

func (p *Position) MatchesScorePosition(filter Position) bool {
	return p.Score[0] == filter.Score[0] && p.Score[1] == filter.Score[1]
}

func (p *Position) MatchesCubePosition(filter Position) bool {
	return p.Cube.Value == filter.Cube.Value && p.Cube.Owner == filter.Cube.Owner
}

func (p *Position) MatchesPipCountFilter(filter string) bool {
	pipCountDiff := p.PipCountDifference()
	if strings.HasPrefix(filter, "p>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return pipCountDiff >= value
	} else if strings.HasPrefix(filter, "p<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return pipCountDiff <= value
	} else if strings.HasPrefix(filter, "p") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			slog.Warn("parsing filter values", "value", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			slog.Warn("parsing filter values", "v1", values[0], "v2", values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return pipCountDiff >= minValue && pipCountDiff <= maxValue
	}
	return false
}

// PipCountDifference returns player 1's pip count minus player 2's.
func (p *Position) PipCountDifference() int {
	player1PipCount, player2PipCount := p.ComputePipCounts()
	return player1PipCount - player2PipCount
}

// ComputePipCounts returns the pip counts of player 1 and player 2.
func (p *Position) ComputePipCounts() (int, int) {
	player1PipCount := 0
	player2PipCount := 0

	for i, point := range p.Board.Points {
		if point.Color == 0 {
			player1PipCount += point.Checkers * i
		} else if point.Color == 1 {
			player2PipCount += point.Checkers * (25 - i)
		}
	}

	return player1PipCount, player2PipCount
}

func (p *Position) MatchesPlayer1BackChecker(filter string) bool {

	backCheckers := 0
	for i := 19; i <= 24; i++ {
		if p.Board.Points[i].Color == 0 {
			backCheckers += p.Board.Points[i].Checkers
		}
	}

	if strings.HasPrefix(filter, "k>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return backCheckers >= value
	} else if strings.HasPrefix(filter, "k<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return backCheckers <= value
	} else if strings.HasPrefix(filter, "k") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			values = append(values, values[0]) // Handle case where 'kx' means 'kx,x'
		}
		if len(values) != 2 {
			slog.Warn("parsing filter values", "value", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			slog.Warn("parsing filter values", "v1", values[0], "v2", values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return backCheckers >= minValue && backCheckers <= maxValue
	}
	return false
}

func (p *Position) MatchesPlayer2BackChecker(filter string) bool {

	backCheckers := 0
	for i := 1; i <= 6; i++ {
		if p.Board.Points[i].Color == 1 {
			backCheckers += p.Board.Points[i].Checkers
		}
	}

	if strings.HasPrefix(filter, "K>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return backCheckers >= value
	} else if strings.HasPrefix(filter, "K<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return backCheckers <= value
	} else if strings.HasPrefix(filter, "K") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			values = append(values, values[0]) // Handle case where 'Kx' means 'Kx,x'
		}
		if len(values) != 2 {
			slog.Warn("parsing filter values", "value", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			slog.Warn("parsing filter values", "v1", values[0], "v2", values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return backCheckers >= minValue && backCheckers <= maxValue
	}
	return false
}

func (p *Position) MatchesPlayer1CheckerInZone(filter string) bool {

	checkersInZone := 0
	for i := 0; i <= 12; i++ {
		if p.Board.Points[i].Color == 0 {
			checkersInZone += p.Board.Points[i].Checkers
		}
	}

	if strings.HasPrefix(filter, "z>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return checkersInZone >= value
	} else if strings.HasPrefix(filter, "z<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return checkersInZone <= value
	} else if strings.HasPrefix(filter, "z") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			values = append(values, values[0]) // Handle case where 'zx' means 'zx,x'
		}
		if len(values) != 2 {
			slog.Warn("parsing filter values", "value", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			slog.Warn("parsing filter values", "v1", values[0], "v2", values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return checkersInZone >= minValue && checkersInZone <= maxValue
	}
	return false
}

func (p *Position) MatchesPlayer2CheckerInZone(filter string) bool {

	checkersInZone := 0
	for i := 13; i <= 25; i++ {
		if p.Board.Points[i].Color == 1 {
			checkersInZone += p.Board.Points[i].Checkers
		}
	}

	if strings.HasPrefix(filter, "Z>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return checkersInZone >= value
	} else if strings.HasPrefix(filter, "Z<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return checkersInZone <= value
	} else if strings.HasPrefix(filter, "Z") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			values = append(values, values[0]) // Handle case where 'Zx' means 'Zx,x'
		}
		if len(values) != 2 {
			slog.Warn("parsing filter values", "value", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			slog.Warn("parsing filter values", "v1", values[0], "v2", values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return checkersInZone >= minValue && checkersInZone <= maxValue
	}
	return false
}

func (p *Position) MatchesPlayer1AbsolutePipCount(filter string) bool {
	player1PipCount, _ := p.ComputePipCounts()

	if strings.HasPrefix(filter, "P>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return player1PipCount >= value
	} else if strings.HasPrefix(filter, "P<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return player1PipCount <= value
	} else if strings.HasPrefix(filter, "P") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			value, err := strconv.Atoi(values[0])
			if err != nil {
				slog.Warn("parsing filter value", "value", values[0])
				return false
			}
			return player1PipCount == value
		} else if len(values) == 2 {
			value1, err1 := strconv.Atoi(values[0])
			value2, err2 := strconv.Atoi(values[1])
			if err1 != nil || err2 != nil {
				slog.Warn("parsing filter values", "v1", values[0], "v2", values[1])
				return false
			}
			minValue := value1
			maxValue := value2
			if value1 > value2 {
				minValue = value2
				maxValue = value1
			}
			return player1PipCount >= minValue && player1PipCount <= maxValue
		}
	}
	return false
}

func (p *Position) MatchesDiceRoll(filter Position) bool {
	return p.MatchesDiceRollMode(filter, "both")
}

func (p *Position) MatchesDiceRollMode(filter Position, mode string) bool {
	if p.PlayerOnRoll != filter.PlayerOnRoll || p.DecisionType != filter.DecisionType {
		return false
	}
	if mode == "first" {
		return p.Dice[0] == filter.Dice[0] || p.Dice[1] == filter.Dice[0]
	}
	dice := fmt.Sprintf("%d%d", p.Dice[0], p.Dice[1])
	reverseDice := fmt.Sprintf("%d%d", p.Dice[1], p.Dice[0])
	filterDice := fmt.Sprintf("%d%d", filter.Dice[0], filter.Dice[1])
	return dice == filterDice || reverseDice == filterDice
}

// ParseExceptDice parses an "except dice" filter string (e.g. "65;54") into
// dice pairs. Each entry is two digits 1-6 (order-insensitive at match time);
// malformed entries are skipped. Backs the xD<d1><d2> search discriminator,
// which excludes positions rolled with any of the listed dice.
func ParseExceptDice(s string) [][2]int {
	var pairs [][2]int
	for _, tok := range strings.Split(s, ";") {
		tok = strings.TrimSpace(tok)
		if len(tok) != 2 {
			continue
		}
		d1 := int(tok[0] - '0')
		d2 := int(tok[1] - '0')
		if d1 < 1 || d1 > 6 || d2 < 1 || d2 > 6 {
			continue
		}
		pairs = append(pairs, [2]int{d1, d2})
	}
	return pairs
}

// MatchesExceptDice reports whether the position's dice avoid every excluded
// roll (each compared in both orders). A position rolled with none of the
// excluded rolls — including cube decisions, which carry no roll — is kept.
func (p *Position) MatchesExceptDice(excluded [][2]int) bool {
	for _, pair := range excluded {
		if (p.Dice[0] == pair[0] && p.Dice[1] == pair[1]) ||
			(p.Dice[0] == pair[1] && p.Dice[1] == pair[0]) {
			return false
		}
	}
	return true
}

// BlotZone is a run of points: the blots a blot filter counts are the points
// of the zone holding exactly one checker of Color.
type BlotZone struct {
	Color  int
	Lo, Hi int
}

// The zones of the four blot filters, by filter prefix. The outfield is
// points 7 to 18 for both sides; each side's jan is its own home board.
var blotZones = map[string]BlotZone{
	"bo": {Color: Black, Lo: 7, Hi: 18},
	"BO": {Color: White, Lo: 7, Hi: 18},
	"bj": {Color: Black, Lo: 1, Hi: 6},
	"BJ": {Color: White, Lo: 19, Hi: 24},
}

// BlotZoneOf is the zone the blot filter of that prefix counts in.
func BlotZoneOf(prefix string) BlotZone { return blotZones[prefix] }

// ParseBlotFilter reads a blot filter: prefix>N is N blots or more, prefix<N
// is N or fewer, prefixA,B is between the two, in either order. ok is false
// for any other spelling, a bare prefixN included.
func ParseBlotFilter(filter, prefix string) (lo, hi int, hasLo, hasHi, ok bool) {
	rest, found := strings.CutPrefix(filter, prefix)
	if !found {
		return 0, 0, false, false, false
	}
	if v, isMin := strings.CutPrefix(rest, ">"); isMin {
		n, err := strconv.Atoi(v)
		return n, 0, true, false, err == nil
	}
	if v, isMax := strings.CutPrefix(rest, "<"); isMax {
		n, err := strconv.Atoi(v)
		return 0, n, false, true, err == nil
	}
	values := strings.Split(rest, ",")
	if len(values) != 2 {
		return 0, 0, false, false, false
	}
	a, err1 := strconv.Atoi(values[0])
	b, err2 := strconv.Atoi(values[1])
	if err1 != nil || err2 != nil {
		return 0, 0, false, false, false
	}
	return min(a, b), max(a, b), true, true, true
}

// matchesBlots counts the blots of the prefix's zone and holds the count to
// the filter. An unreadable filter matches nothing; the search refuses it
// before it reads a position.
func (p *Position) matchesBlots(filter, prefix string) bool {
	lo, hi, hasLo, hasHi, ok := ParseBlotFilter(filter, prefix)
	if !ok {
		return false
	}
	z := blotZones[prefix]
	blots := 0
	for i := z.Lo; i <= z.Hi; i++ {
		if p.Board.Points[i].Color == z.Color && p.Board.Points[i].Checkers == 1 {
			blots++
		}
	}
	return (!hasLo || blots >= lo) && (!hasHi || blots <= hi)
}

func (p *Position) MatchesPlayer1OutfieldBlot(filter string) bool {
	return p.matchesBlots(filter, "bo")
}

func (p *Position) MatchesPlayer2OutfieldBlot(filter string) bool {
	return p.matchesBlots(filter, "BO")
}

func (p *Position) MatchesPlayer1JanBlot(filter string) bool {
	return p.matchesBlots(filter, "bj")
}

func (p *Position) MatchesPlayer2JanBlot(filter string) bool {
	return p.matchesBlots(filter, "BJ")
}

func (p *Position) MatchesNoContact() bool {
	var furthestPlayerChecker, furthestOpponentChecker int

	// Initialize to invalid indices
	furthestPlayerChecker = -1
	furthestOpponentChecker = 26

	for i := 0; i < len(p.Board.Points); i++ {
		if p.Board.Points[i].Color == 0 && p.Board.Points[i].Checkers > 0 {
			furthestPlayerChecker = i
		}
		if p.Board.Points[25-i].Color == 1 && p.Board.Points[25-i].Checkers > 0 {
			furthestOpponentChecker = 25 - i
		}
	}

	// Compare indices to determine if there is no contact
	return furthestPlayerChecker < furthestOpponentChecker
}

// SplitFilterList splits a ";"-separated filter value into its non-empty,
// trimmed, lower-cased items, in the order they were written. It is what
// CommentOriginFilter and GamePhaseFilter hold: a short closed list, kept as
// one string because that is the shape the frontend has always sent and the
// shape ExceptDiceFilter established.
func SplitFilterList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ";") {
		if p := strings.ToLower(strings.TrimSpace(part)); p != "" {
			out = append(out, p)
		}
	}
	return out
}
