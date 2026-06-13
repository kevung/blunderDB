package database

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

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
