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
	dice := fmt.Sprintf("%d%d", p.Dice[0], p.Dice[1])
	reverseDice := fmt.Sprintf("%d%d", p.Dice[1], p.Dice[0])
	filterDice := fmt.Sprintf("%d%d", filter.Dice[0], filter.Dice[1])
	return (dice == filterDice || reverseDice == filterDice) && p.PlayerOnRoll == filter.PlayerOnRoll && p.DecisionType == filter.DecisionType
}

func (p *Position) MatchesPlayer1OutfieldBlot(filter string) bool {
	outfieldBlots := 0
	for i := 7; i <= 18; i++ {
		if p.Board.Points[i].Color == 0 && p.Board.Points[i].Checkers == 1 {
			outfieldBlots++
		}
	}

	if strings.HasPrefix(filter, "bo>") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[3:])
			return false
		}
		return outfieldBlots >= value
	} else if strings.HasPrefix(filter, "bo<") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[3:])
			return false
		}
		return outfieldBlots <= value
	} else if strings.HasPrefix(filter, "bo") {
		values := strings.Split(filter[2:], ",")
		if len(values) != 2 {
			slog.Warn("parsing filter values", "value", filter[2:])
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
		return outfieldBlots >= minValue && outfieldBlots <= maxValue
	}
	return false
}

func (p *Position) MatchesPlayer2OutfieldBlot(filter string) bool {
	opponentOutfieldBlots := 0
	for i := 7; i <= 18; i++ {
		if p.Board.Points[i].Color == 1 && p.Board.Points[i].Checkers == 1 {
			opponentOutfieldBlots++
		}
	}

	if strings.HasPrefix(filter, "BO>") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[3:])
			return false
		}
		return opponentOutfieldBlots >= value
	} else if strings.HasPrefix(filter, "BO<") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[3:])
			return false
		}
		return opponentOutfieldBlots <= value
	} else if strings.HasPrefix(filter, "BO") {
		values := strings.Split(filter[2:], ",")
		if len(values) != 2 {
			slog.Warn("parsing filter values", "value", filter[2:])
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
		return opponentOutfieldBlots >= minValue && opponentOutfieldBlots <= maxValue
	}
	return false
}

func (p *Position) MatchesPlayer1JanBlot(filter string) bool {
	janBlots := 0
	for i := 1; i <= 6; i++ {
		if p.Board.Points[i].Color == 0 && p.Board.Points[i].Checkers == 1 {
			janBlots++
		}
	}

	if strings.HasPrefix(filter, "bj>") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[3:])
			return false
		}
		return janBlots >= value
	} else if strings.HasPrefix(filter, "bj<") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[3:])
			return false
		}
		return janBlots <= value
	} else if strings.HasPrefix(filter, "bj") {
		values := strings.Split(filter[2:], ",")
		if len(values) != 2 {
			slog.Warn("parsing filter values", "value", filter[2:])
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
		return janBlots >= minValue && janBlots <= maxValue
	}
	return false
}

func (p *Position) MatchesPlayer2JanBlot(filter string) bool {
	opponentJanBlots := 0
	for i := 19; i <= 24; i++ {
		if p.Board.Points[i].Color == 1 && p.Board.Points[i].Checkers == 1 {
			opponentJanBlots++
		}
	}

	if strings.HasPrefix(filter, "BJ>") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[3:])
			return false
		}
		return opponentJanBlots >= value
	} else if strings.HasPrefix(filter, "BJ<") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[3:])
			return false
		}
		return opponentJanBlots <= value
	} else if strings.HasPrefix(filter, "BJ") {
		values := strings.Split(filter[2:], ",")
		if len(values) != 2 {
			slog.Warn("parsing filter values", "value", filter[2:])
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
		return opponentJanBlots >= minValue && opponentJanBlots <= maxValue
	}
	return false
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
