package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/xgparser/xgparser"
)

// Match import and management functions

// Import XG match file using xgparser library
// ImportXGMatch imports an eXtreme Gammon .xg match file. It delegates to the
// shared ingest pipeline (ingest.MapXG → ingest.WriteMatch) — the same path the
// headless server uses — persisting through the storage backend. The former
// in-Database XG mapping it replaced now lives only inside the ingest package.
func (d *Database) ImportXGMatch(filePath string) (int64, error) {
	ctx, done := d.beginCancellableImport()
	defer done()

	d.mu.Lock()
	defer d.mu.Unlock()

	graph, err := ingest.MapXG(filePath)
	if err != nil {
		return 0, err
	}
	matchID, err := d.writeImportedMatch(ctx, graph)
	if err != nil {
		return 0, err
	}
	slog.Info("imported match", "matchID", matchID, "file", filePath)
	return matchID, nil
}

// RawCubeAction stores raw cube action data from XG file
type RawCubeAction struct {
	Double   int32
	Take     int32
	ActiveP  int32
	CubeB    int32
	Position [26]int8
	Doubled  *xgparser.EngineStructDoubleAction // Full cube analysis data
}

// createPositionFromXG converts xgparser.Position to blunderDB Position
// activePlayer indicates which XG player (-1 or 1) is on roll in this position
func (d *Database) createPositionFromXG(xgPos xgparser.Position, game *xgparser.Game, matchLength int32, moveNum int32, activePlayer int32) (*Position, error) {
	// Convert XG player encoding to blunderDB encoding
	// XG: -1 = Player 1, 1 = Player 2
	// blunderDB: 0 = Player 1, 1 = Player 2
	activePlayerBlunderDB := convertXGPlayerToBlunderDB(activePlayer)
	opponentPlayerBlunderDB := 1 - activePlayerBlunderDB

	// Calculate away scores from current scores
	// In blunderDB, scores are "points away from winning"
	// game.InitialScore contains current scores (e.g., 2-3 in a 7-point match)
	// We need to convert to away scores (e.g., 5-away, 4-away)
	awayScore1 := int(matchLength) - int(game.InitialScore[0])
	awayScore2 := int(matchLength) - int(game.InitialScore[1])

	// Handle unlimited match (matchLength == 0)
	if matchLength == 0 {
		awayScore1 = -1
		awayScore2 = -1
	}

	// Map XG cube position to blunderDB format
	// XG CubePos is RELATIVE to active player:
	//   CubePos = 0: Center (no owner)
	//   CubePos = 1: Active player owns the cube
	//   CubePos = -1: Opponent owns the cube
	// blunderDB uses absolute encoding:
	//   -1 = center (no owner)
	//   0 = Player 1 owns (bottom, black)
	//   1 = Player 2 owns (top, white)
	cubeOwner := -1 // Default: center (no owner)
	if xgPos.CubePos == 1 {
		// Active player owns the cube
		cubeOwner = activePlayerBlunderDB
	} else if xgPos.CubePos == -1 {
		// Opponent owns the cube
		cubeOwner = opponentPlayerBlunderDB
	}
	// CubePos == 0 means center, cubeOwner stays -1

	// Convert cube value from XG (direct value 1,2,4,8...) to blunderDB (exponent 0,1,2,3...)
	// blunderDB displays 2^value, so we need log2(xgCube)
	cubeValue := 0
	if xgPos.Cube > 0 {
		// Calculate log2 of cube value
		for v := int(xgPos.Cube); v > 1; v >>= 1 {
			cubeValue++
		}
	}

	pos := &Position{
		PlayerOnRoll: 0, // Will be set from move context
		DecisionType: 0, // Checker decision by default
		Score:        [2]int{awayScore1, awayScore2},
		Cube: Cube{
			Value: cubeValue,
			Owner: cubeOwner,
		},
		Dice: [2]int{0, 0}, // Will be set from move data
	}

	// Convert checkers from XG format to blunderDB format
	// XG format: index 0-23 are points 1-24, index 24=bar, 25=opponent bar
	// Positive values = active player's checkers, negative = opponent's checkers
	//
	// In blunderDB:
	// - Color 0 = Player 1 (always at bottom, black, moves 24→1) - NEVER CHANGES
	// - Color 1 = Player 2 (always at top, white, moves 1→24) - NEVER CHANGES
	// - Points 1-24 with indices 1-24 in the array
	// - Index 0 = Player 2's bar (white), Index 25 = Player 1's bar (black)
	//
	// XG stores positions from the active player's perspective:
	// - Positive checkers = active player's checkers
	// - Negative checkers = opponent's checkers
	// - Point numbering is from active player's perspective
	//
	// Player mapping:
	// - XG Player 0 = blunderDB Player 1 (Color 0, bottom, black)
	// - XG Player 1 = blunderDB Player 2 (Color 1, top, white)
	//
	// Strategy:
	// 1. Determine which player owns the checkers based on sign AND activePlayer
	// 2. Assign colors based on OWNER, not on sign
	// 3. Mirror positions if activePlayer == 1 (since XG encodes from active player's view)
	for i := 0; i < 26; i++ {
		checkerCount := xgPos.Checkers[i]

		// Determine WHERE to place them (calculate targetIndex for ALL points, even empty)
		// XG uses 1-based indexing: index 1-24 = points 1-24, index 0 = opponent bar, index 25 = active bar
		// blunderDB also uses same: index 1-24 = points 1-24, index 0 = P2 bar, index 25 = P1 bar
		targetIndex := i
		if activePlayerBlunderDB == 1 {
			// Player 2's perspective, need to mirror to Player 1's perspective
			if i >= 1 && i <= 24 {
				// XG index i = Player 2's point i → Player 1's point (25 - i) → same index (25 - i)
				targetIndex = 25 - i
			} else if i == 0 {
				// Opponent's bar from Player 2's view = Player 1's bar
				targetIndex = 25
			} else if i == 25 {
				// Active player's bar (Player 2) → Player 2's bar
				targetIndex = 0
			}
		}
		// Player 1's perspective: direct mapping (XG index = blunderDB index)
		// XG index 1-24 = Player 1's points 1-24 → blunderDB index 1-24 (same)
		// XG index 0 = opponent bar (Player 2) → blunderDB index 0 (Player 2's bar)
		// XG index 25 = active bar (Player 1) → blunderDB index 25 (Player 1's bar)

		if checkerCount == 0 {
			pos.Board.Points[targetIndex] = Point{Checkers: 0, Color: -1}
			continue
		}

		// Determine WHO owns these checkers
		var ownerColor int // 0=Player1(black), 1=Player2(white)
		if checkerCount > 0 {
			// Positive = active player
			ownerColor = activePlayerBlunderDB
		} else {
			// Negative = opponent
			ownerColor = opponentPlayerBlunderDB
		}

		// Place checkers with FIXED color based on owner
		pos.Board.Points[targetIndex] = Point{
			Checkers: int(abs(checkerCount)),
			Color:    ownerColor,
		}
	}

	// Calculate bearoff (checkers borne off)
	player1Total := 0
	player2Total := 0
	for i := 0; i < 26; i++ {
		if pos.Board.Points[i].Color == 0 {
			player1Total += pos.Board.Points[i].Checkers
		} else if pos.Board.Points[i].Color == 1 {
			player2Total += pos.Board.Points[i].Checkers
		}
	}
	pos.Board.Bearoff = [2]int{15 - player1Total, 15 - player2Total}

	return pos, nil
}

// Helper function for absolute value
func abs(x int8) int {
	if x < 0 {
		return int(-x)
	}
	return int(x)
}

// convertXGPlayerToBlunderDB converts XG player encoding to blunderDB encoding
// XG: -1 = Player 1, 1 = Player 2
// blunderDB: 0 = Player 1, 1 = Player 2
func convertXGPlayerToBlunderDB(xgPlayer int32) int {
	// XG uses 1 for first player, -1 for second player
	// blunderDB uses 0 for Player 1 (black, bottom), 1 for Player 2 (white, top)
	if xgPlayer == 1 {
		return 0 // Player 1
	}
	return 1 // Player 2
}

// convertBlunderDBPlayerToXG converts blunderDB player encoding (0/1) to XG encoding (1/-1)
// This is used when storing GnuBG-imported moves in the DB, so that GetMatchMovePositions
// can apply convertXGPlayerToBlunderDB uniformly for both XG and GnuBG imports.
func convertBlunderDBPlayerToXG(blunderDBPlayer int) int32 {
	if blunderDBPlayer == 0 {
		return 1 // Player 1
	}
	return -1 // Player 2
}

// saveCheckerAnalysisToPositionInTx converts XG checker analysis to PositionAnalysis and saves it
// playedMove is optional - if provided, it will be used as the source of truth for the first analysis
// (workaround for xgparser bug where analysis.Move may be incomplete for multi-submove bear-offs)
func (d *Database) saveCheckerAnalysisToPositionInTx(tx *sql.Tx, positionID int64, analyses []xgparser.CheckerAnalysis, initialPosition *xgparser.Position, activePlayer int32, playedMove *[8]int32) error {
	if len(analyses) == 0 {
		return nil
	}

	// Create PositionAnalysis structure
	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "XG",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	// Build checker analysis with all moves
	checkerMoves := make([]CheckerMove, 0, len(analyses))
	for i, analysis := range analyses {
		// Convert move from [8]int8 to [8]int32 for convertXGMoveToString
		var move [8]int32

		// For the first analysis (i=0), use playedMove if available
		// This is a workaround for xgparser bug where analysis.Move may be incomplete
		// for multi-submove bear-offs or other complex moves
		if i == 0 && playedMove != nil {
			// Check if playedMove has more info than analysis.Move
			playedMoveCount := 0
			analysisMoveCount := 0
			for j := 0; j < 8; j += 2 {
				if (*playedMove)[j] != -1 {
					playedMoveCount++
				}
				if analysis.Move[j] != -1 {
					analysisMoveCount++
				}
			}
			// Use playedMove if it has more sub-moves
			if playedMoveCount > analysisMoveCount {
				for j := 0; j < 8; j++ {
					move[j] = (*playedMove)[j]
				}
			} else {
				for j := 0; j < 8; j++ {
					move[j] = int32(analysis.Move[j])
				}
			}
		} else {
			for j := 0; j < 8; j++ {
				move[j] = int32(analysis.Move[j])
			}
		}
		// Infer multipliers from position changes
		// XG stores moves compactly - e.g., 1/off(4) is stored as just 1/off once
		if initialPosition != nil {
			move = inferMoveMultipliers(move, initialPosition, &analysis.Position, activePlayer)
		}

		// Use move string with hit detection if initial position is available
		var moveStr string
		if initialPosition != nil {
			moveStr = d.convertXGMoveToStringWithHits(move, initialPosition, activePlayer)
		} else {
			moveStr = d.convertXGMoveToString(move, activePlayer)
		}

		var equityError *float64
		if i > 0 {
			diff := float64(analyses[0].Equity - analysis.Equity)
			equityError = &diff
		}

		checkerMove := CheckerMove{
			Index:                    i,
			AnalysisDepth:            translateAnalysisDepth(int(analysis.AnalysisDepth)),
			AnalysisEngine:           "XG",
			Move:                     moveStr,
			Equity:                   float64(analysis.Equity),
			EquityError:              equityError,
			PlayerWinChance:          float64(analysis.Player1WinRate) * 100.0,
			PlayerGammonChance:       float64(analysis.Player1GammonRate) * 100.0,
			PlayerBackgammonChance:   float64(analysis.Player1BgRate) * 100.0,
			OpponentWinChance:        float64(1.0-analysis.Player1WinRate) * 100.0,
			OpponentGammonChance:     float64(analysis.Player2GammonRate) * 100.0,
			OpponentBackgammonChance: float64(analysis.Player2BgRate) * 100.0,
		}
		checkerMoves = append(checkerMoves, checkerMove)
	}

	posAnalysis.CheckerAnalysis = &CheckerAnalysis{
		Moves: checkerMoves,
	}

	// Set played move on the analysis if available
	if playedMove != nil {
		playedMoveStr := d.convertXGMoveToStringWithHits(*playedMove, initialPosition, activePlayer)
		if playedMoveStr != "" {
			posAnalysis.PlayedMoves = []string{playedMoveStr}
		}
	}

	// Save to analysis table
	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// saveCubeAnalysisToPositionInTx converts XG cube analysis to PositionAnalysis and saves it.
// playedCubeAction is the cube action the player actually took (e.g. "No Double", "Double/Take",
// "Double/Pass", "Double"). When non-empty it is stored in PlayedCubeActions so that
// populateAnalysisColumns can compute cube_error.
func (d *Database) saveCubeAnalysisToPositionInTx(tx *sql.Tx, positionID int64, analysis *xgparser.CubeAnalysis, playedCubeAction string) error {
	if analysis == nil {
		return nil
	}

	// Create PositionAnalysis structure
	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "DoublingCube",
		AnalysisEngineVersion: "XG",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}
	if playedCubeAction != "" {
		posAnalysis.PlayedCubeActions = []string{playedCubeAction}
	}

	cubeAnalysis := buildDoublingCubeAnalysis(cubeAnalysisParams{
		Depth:                     translateAnalysisDepth(int(analysis.AnalysisDepth)),
		Engine:                    "XG",
		PlayerWinChances:          float64(analysis.Player1WinRate) * 100.0,
		PlayerGammonChances:       float64(analysis.Player1GammonRate) * 100.0,
		PlayerBackgammonChances:   float64(analysis.Player1BgRate) * 100.0,
		OpponentWinChances:        float64(1.0-analysis.Player1WinRate) * 100.0,
		OpponentGammonChances:     float64(analysis.Player2GammonRate) * 100.0,
		OpponentBackgammonChances: float64(analysis.Player2BgRate) * 100.0,
		CubelessNoDoubleEquity:    float64(analysis.CubelessNoDouble),
		CubelessDoubleEquity:      float64(analysis.CubelessDouble),
		CubefulNoDoubleEquity:     float64(analysis.CubefulNoDouble),
		CubefulDoubleTakeEquity:   float64(analysis.CubefulDoubleTake),
		CubefulDoublePassEquity:   float64(analysis.CubefulDoublePass),
		WrongPassPercentage:       float64(analysis.WrongPassTakePercent) * 100.0,
	})

	posAnalysis.DoublingCubeAnalysis = &cubeAnalysis

	// Save to analysis table
	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// inferMoveMultipliers analyzes a partial move array and the position difference
// to infer the correct number of repetitions for each move.
// XG sometimes stores only one instance of a move even when multiple checkers make the same move.
// This function expands the move array to include all repetitions.
// Returns the expanded move array with correct multipliers.
func inferMoveMultipliers(partialMove [8]int32, initialPos, finalPos *xgparser.Position, activePlayer int32) [8]int32 {
	if initialPos == nil || finalPos == nil {
		return partialMove
	}

	// First, count how many moves are explicitly in the input
	// and count occurrences of each unique (from,to) pair
	type moveSpec struct {
		from int32
		to   int32
	}
	moveCount := make(map[moveSpec]int32)
	totalInputMoves := 0

	for i := 0; i < 8; i += 2 {
		from := partialMove[i]
		to := partialMove[i+1]
		if from == -1 {
			break
		}
		// Handle implicit bear-off
		if from >= 1 && from <= 6 && to <= 0 && to != -2 {
			to = -2
		}
		moveCount[moveSpec{from, to}]++
		totalInputMoves++
	}

	if totalInputMoves == 0 {
		return partialMove
	}

	// If we already have multiple moves in input (not a compact representation),
	// just return the input as-is - no inference needed
	// XG uses compact representation only for doublets where same move repeats
	if totalInputMoves > 1 {
		// Check if all moves are the same (compact doublet notation)
		allSame := true
		var firstMove moveSpec
		first := true
		for ms := range moveCount {
			if first {
				firstMove = ms
				first = false
			} else if ms != firstMove {
				allSame = false
				break
			}
		}
		// If moves are different, no inference needed - return as-is
		if !allSame {
			return partialMove
		}
	}

	// At this point, either:
	// 1. We have a single move that might need expansion (e.g., [8,3,-1,-1,-1,-1,-1,-1] for 8/3(4))
	// 2. We have multiple identical moves (already explicit, no expansion needed)

	// Get the unique moves preserving order
	var uniqueMoves []moveSpec
	seen := make(map[moveSpec]bool)
	for i := 0; i < 8; i += 2 {
		from := partialMove[i]
		to := partialMove[i+1]
		if from == -1 {
			break
		}
		if from >= 1 && from <= 6 && to <= 0 && to != -2 {
			to = -2
		}
		ms := moveSpec{from, to}
		if !seen[ms] {
			seen[ms] = true
			uniqueMoves = append(uniqueMoves, ms)
		}
	}

	// If we have multiple identical moves in input, they're already explicit
	if len(uniqueMoves) == 1 && moveCount[uniqueMoves[0]] > 1 {
		return partialMove
	}

	// Calculate how many checkers left each source point
	// netChange[point] = initialCheckers - finalCheckers (positive = net loss of checkers)
	netChange := make(map[int32]int32)

	for _, ms := range uniqueMoves {
		if ms.from == 25 {
			netChange[25] = int32(initialPos.Checkers[25]) - int32(finalPos.Checkers[25])
		} else if ms.from >= 1 && ms.from <= 24 {
			netChange[ms.from] = int32(initialPos.Checkers[ms.from]) - int32(finalPos.Checkers[ms.from])
		}
		// Track destination changes too
		if ms.to >= 1 && ms.to <= 24 {
			netChange[ms.to] = int32(initialPos.Checkers[ms.to]) - int32(finalPos.Checkers[ms.to])
		}
	}

	// Build a flow model: how many checkers move from each source
	// Track arriving checkers for intermediate points
	arriving := make(map[int32]int32) // checkers arriving at each point

	var expandedMove [8]int32
	for i := range expandedMove {
		expandedMove[i] = -1
	}
	moveIndex := 0

	// Process moves in order
	for _, ms := range uniqueMoves {
		var count int32 = 1

		if ms.to == -2 {
			// Bear-off: count = net checkers that left this point + checkers arriving from other points
			count = netChange[ms.from] + arriving[ms.from]
		} else if ms.to >= 1 && ms.to <= 24 {
			// Move to board point
			// Count = how many checkers left the source AND didn't come back
			// For most cases, this is simply the net change of the source
			// But we need to also account for checker flow

			// Net change at source tells us how many checkers total left
			srcLoss := netChange[ms.from]

			// If source also receives checkers (from another move), we need less moves
			srcReceive := arriving[ms.from]

			// The move count is the source loss minus any checkers that arrived
			count = srcLoss - srcReceive
			if count <= 0 {
				count = 1
			}

			// After this move, these checkers arrive at the destination
			arriving[ms.to] += count
		}

		// Cap at 4 (maximum for doubles) or remaining slots
		maxMoves := int32(4)
		remainingSlots := int32((8 - moveIndex) / 2)
		if count > maxMoves {
			count = maxMoves
		}
		if count > remainingSlots {
			count = remainingSlots
		}
		if count < 1 {
			count = 1
		}

		for j := int32(0); j < count && moveIndex < 8; j++ {
			expandedMove[moveIndex] = ms.from
			expandedMove[moveIndex+1] = ms.to
			moveIndex += 2
		}
	}

	return expandedMove
}

// translateAnalysisDepth converts XG analysis depth codes to human-readable strings
// XG depth codes:
//   - 0-9 = (N+1)-ply search depth (XG stores ply-1 internally)
//   - 998-1000 = Book moves (different footnote references)
//   - 1001 = XG Roller (neural network)
//   - 1002 = XG Roller++ (extended neural network analysis)
func translateAnalysisDepth(depth int) string {
	switch {
	case depth >= 0 && depth <= 9:
		return fmt.Sprintf("%d-ply", depth+1)
	case depth >= 998 && depth <= 1000:
		return "Book"
	case depth == 1001:
		return "XG Roller"
	case depth == 1002:
		return "XG Roller++"
	default:
		return fmt.Sprintf("%d", depth)
	}
}

// convertXGMoveToString converts XG move format to readable string
// XG move format: [from, to, from, to, ...] where:
//   - 1-24 are board points (from active player's perspective)
//   - 25 is the bar
//   - -2 is bear off
//   - -1 is unused/end of move
//
// Moves in XG files and analysis are stored from the player on roll's perspective.
// They should be displayed as-is without mirroring, per standard backgammon notation.
func (d *Database) convertXGMoveToString(playedMove [8]int32, activePlayer int32) string {
	// Note: activePlayer is kept for API compatibility but moves don't need transformation
	// Moves are always from the roller's perspective (24 = furthest from home, 1 = closest)
	_ = activePlayer // unused but kept for signature compatibility

	// Parse raw moves into from/to pairs
	var fromPts []int32
	var toPts []int32
	for i := 0; i < 8; i += 2 {
		from := playedMove[i]
		to := playedMove[i+1]
		// Check for end of move marker (-1 when from is also -1)
		if from == -1 {
			break
		}
		// Handle implicit bear-off: XG sometimes encodes bear-off as to=-1 or to<=0
		// when the calculated destination (from - die) would be <= 0
		// This happens when bearing off from the home board (points 1-6)
		if from >= 1 && from <= 6 && to <= 0 && to != -2 {
			to = -2 // Convert to explicit bear-off
		}
		// Skip invalid point values (should be 1-24 for points, 25 for bar, -2 for bear off)
		if (from != 25 && from != -2 && (from < 1 || from > 24)) ||
			(to != 25 && to != -2 && (to < 1 || to > 24)) {
			break
		}
		fromPts = append(fromPts, from)
		toPts = append(toPts, to)
	}

	if len(fromPts) == 0 {
		return "Cannot Move"
	}

	// Try to merge consecutive checker slides (same checker moving through points)
	// E.g., 24/23 23/22 22/21 21/20 -> 24/20 when dice are all same
	fromPts, toPts = d.mergeSlides(fromPts, toPts)

	// Create sortable items
	type moveItem struct {
		from int32
		to   int32
	}
	items := make([]moveItem, len(fromPts))
	for i := range fromPts {
		items[i] = moveItem{from: fromPts[i], to: toPts[i]}
	}

	// Sort moves by 'from' point descending (standard backgammon notation)
	// bar (25) comes first, then higher points before lower points
	sort.Slice(items, func(i, j int) bool {
		return items[i].from > items[j].from
	})

	// Format each move as string
	formatPoint := func(p int32) string {
		if p == 25 {
			return "bar"
		} else if p == -2 {
			return "off"
		} else if p >= 1 && p <= 24 {
			return fmt.Sprintf("%d", p)
		}
		return fmt.Sprintf("?%d", p)
	}

	// Build move string, grouping identical moves with multiplier
	var moves []string
	for i := 0; i < len(items); {
		item := items[i]
		count := 1
		// Count consecutive identical moves
		for j := i + 1; j < len(items); j++ {
			if items[j].from == item.from && items[j].to == item.to {
				count++
			} else {
				break
			}
		}
		if count > 1 {
			moves = append(moves, fmt.Sprintf("%s/%s(%d)", formatPoint(item.from), formatPoint(item.to), count))
		} else {
			moves = append(moves, fmt.Sprintf("%s/%s", formatPoint(item.from), formatPoint(item.to)))
		}
		i += count
	}

	return strings.Join(moves, " ")
}

// convertXGMoveToStringWithHits converts XG move format to readable string with hit indicators (*)
// It uses the initial position to detect when a blot is hit
// Moves are displayed from the player on roll's perspective (standard notation).
// activePlayer: XG encoding: 1 = Player 1 (X), -1 = Player 2 (O)
func (d *Database) convertXGMoveToStringWithHits(playedMove [8]int32, initialPos *xgparser.Position, activePlayer int32) string {
	if initialPos == nil {
		return d.convertXGMoveToString(playedMove, activePlayer)
	}

	// Note: No mirroring needed - moves are always from the roller's perspective

	// Create a mutable copy of the position to track changes as we process moves
	// XG position format: Checkers[1-24] are points 1-24 (1-based indexing)
	// [0]=opponent's bar, [25]=player's bar
	// Positive values = player's checkers, negative = opponent's checkers
	positionCopy := make([]int8, 26)
	copy(positionCopy, initialPos.Checkers[:])

	// Parse raw moves into from/to pairs and track hits
	var items []xgMoveWithHit
	for i := 0; i < 8; i += 2 {
		from := playedMove[i]
		to := playedMove[i+1]
		// Check for end of move marker (-1 when from is also -1)
		if from == -1 {
			break
		}
		// Handle implicit bear-off: XG sometimes encodes bear-off as to=-1 or to<=0
		// when the calculated destination (from - die) would be <= 0
		// This happens when bearing off from the home board (points 1-6)
		if from >= 1 && from <= 6 && to <= 0 && to != -2 {
			to = -2 // Convert to explicit bear-off
		}
		// Skip invalid point values (should be 1-24 for points, 25 for bar, -2 for bear off)
		if (from != 25 && from != -2 && (from < 1 || from > 24)) ||
			(to != 25 && to != -2 && (to < 1 || to > 24)) {
			break
		}

		// Check if this move hits a blot
		// The destination point must have exactly one opponent checker (negative value in XG format)
		isHit := false
		if to >= 1 && to <= 24 {
			// Position.Checkers uses 1-based indexing: Checkers[1] = point 1, Checkers[24] = point 24
			if positionCopy[to] == -1 {
				isHit = true
				// Update position: opponent checker goes to bar
				positionCopy[to] = 0
			}
		}

		// Update position: move our checker
		if from >= 1 && from <= 24 {
			// Position.Checkers uses 1-based indexing
			if positionCopy[from] > 0 {
				positionCopy[from]--
			}
		} else if from == 25 {
			// From bar - player's bar is at index 25
			if positionCopy[25] > 0 {
				positionCopy[25]--
			}
		}

		if to >= 1 && to <= 24 {
			// Position.Checkers uses 1-based indexing
			positionCopy[to]++
		}

		// Store directly - no conversion needed since moves are already in roller's perspective
		items = append(items, xgMoveWithHit{from: from, to: to, isHit: isHit})
	}

	if len(items) == 0 {
		return "Cannot Move"
	}

	// Try to merge consecutive checker slides - but preserve hit info for the final move
	// For slides, only the last move in a chain can be a hit
	items = d.mergeSlidesWithHits(items)

	// Sort moves by 'from' point descending (standard backgammon notation)
	sort.Slice(items, func(i, j int) bool {
		return items[i].from > items[j].from
	})

	// Format each move as string
	formatPoint := func(p int32) string {
		if p == 25 {
			return "bar"
		} else if p == -2 {
			return "off"
		} else if p >= 1 && p <= 24 {
			return fmt.Sprintf("%d", p)
		}
		return fmt.Sprintf("?%d", p)
	}

	// Build move string, grouping identical moves with multiplier
	var moves []string
	for i := 0; i < len(items); {
		item := items[i]
		count := 1
		allHits := item.isHit
		// Count consecutive identical moves
		for j := i + 1; j < len(items); j++ {
			if items[j].from == item.from && items[j].to == item.to {
				count++
				allHits = allHits && items[j].isHit
			} else {
				break
			}
		}

		hitMarker := ""
		if item.isHit || allHits {
			hitMarker = "*"
		}

		if count > 1 {
			moves = append(moves, fmt.Sprintf("%s/%s%s(%d)", formatPoint(item.from), formatPoint(item.to), hitMarker, count))
		} else {
			moves = append(moves, fmt.Sprintf("%s/%s%s", formatPoint(item.from), formatPoint(item.to), hitMarker))
		}
		i += count
	}

	return strings.Join(moves, " ")
}

// mergeSlidesWithHits merges consecutive moves of the same checker, preserving hit info
// For example: 14/12 12/8 becomes 14/8, but only if 12 was just a waypoint (not hit)
// If there was a hit at the intermediate point, we keep both moves to show the hit
func (d *Database) mergeSlidesWithHits(items []xgMoveWithHit) []xgMoveWithHit {
	if len(items) <= 1 {
		return items
	}

	// Try to merge chains: if move[i].to == move[j].from and move[i] is not a hit,
	// they can be merged (the intermediate point was just a waypoint)
	result := make([]xgMoveWithHit, 0, len(items))
	used := make([]bool, len(items))

	for i := 0; i < len(items); i++ {
		if used[i] {
			continue
		}

		// Start a chain from this item
		chainFrom := items[i].from
		chainTo := items[i].to
		chainHit := items[i].isHit
		used[i] = true

		// Only extend if the current segment doesn't end with a hit
		// (we want to show hits, so don't merge past them)
		if !chainHit {
			// Extend chain forward: find items where from == chainTo
			for changed := true; changed; {
				changed = false
				for j := 0; j < len(items); j++ {
					if used[j] {
						continue
					}
					if items[j].from == chainTo {
						chainTo = items[j].to
						chainHit = items[j].isHit
						used[j] = true
						changed = true
						// If this new segment has a hit, stop extending
						if chainHit {
							break
						}
					}
				}
				if chainHit {
					break
				}
			}
		}

		result = append(result, xgMoveWithHit{from: chainFrom, to: chainTo, isHit: chainHit})
	}

	return result
}

// xgMoveWithHit represents a single move in XG format with hit information
type xgMoveWithHit struct {
	from  int32
	to    int32
	isHit bool
}

// mergeSlides merges consecutive moves of the same checker
// For example: 14/12 12/8 becomes 14/8
func (d *Database) mergeSlides(fromPts, toPts []int32) ([]int32, []int32) {
	if len(fromPts) <= 1 {
		return fromPts, toPts
	}

	// Count how many times each destination point is used
	// If multiple moves end at the same point, we should NOT merge through that point
	// because it means different checkers are moving, not the same checker sliding
	toCount := make(map[int32]int)
	for _, t := range toPts {
		toCount[t]++
	}

	// Also count how many times each point is a source
	fromCount := make(map[int32]int)
	for _, f := range fromPts {
		fromCount[f]++
	}

	// Try to merge chains: if move[i].to == move[j].from, they can be merged
	// BUT only if that intermediate point appears exactly once as a destination AND once as a source
	resultFrom := make([]int32, 0, len(fromPts))
	resultTo := make([]int32, 0, len(toPts))
	used := make([]bool, len(fromPts))

	for i := 0; i < len(fromPts); i++ {
		if used[i] {
			continue
		}

		// Start a chain from this item
		chainFrom := fromPts[i]
		chainTo := toPts[i]
		used[i] = true

		// Extend chain forward: find items where from == chainTo
		// Only merge if the intermediate point is not used by multiple checkers
		for changed := true; changed; {
			changed = false
			for j := 0; j < len(fromPts); j++ {
				if used[j] {
					continue
				}
				if fromPts[j] == chainTo {
					// Check if this is a valid merge (same checker moving)
					// Don't merge if chainTo is a destination for multiple moves
					// or if chainTo is a source for multiple moves
					// This indicates different checkers
					if toCount[chainTo] > 1 || fromCount[chainTo] > 1 {
						continue // Don't merge - different checkers
					}
					chainTo = toPts[j]
					used[j] = true
					changed = true
				}
			}
		}

		resultFrom = append(resultFrom, chainFrom)
		resultTo = append(resultTo, chainTo)
	}

	return resultFrom, resultTo
}

// convertCubeAction converts cube action code to string
func (d *Database) convertCubeAction(action int32) string {
	switch action {
	case 0:
		return "No Double"
	case 1:
		return "Double"
	case 2:
		return "Take"
	case 3:
		return "Pass"
	default:
		return fmt.Sprintf("Unknown(%d)", action)
	}
}
