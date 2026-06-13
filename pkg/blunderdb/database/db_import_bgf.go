package database

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/kevung/bgfparser"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// ============================================================================
// BGBlitz BGF import functions
// ============================================================================

// ImportBGFMatch imports a match from a BGBlitz BGF file, delegating to the
// shared ingest pipeline (ingest.MapBGF -> ingest.WriteMatch) — the same path
// the headless server uses.
func (d *Database) ImportBGFMatch(filePath string) (int64, error) {
	ctx, done := d.beginCancellableImport()
	defer done()

	d.mu.Lock()
	defer d.mu.Unlock()

	graph, err := ingest.MapBGF(filePath)
	if err != nil {
		return 0, err
	}
	matchID, err := d.writeImportedMatch(ctx, graph)
	if err != nil {
		return 0, err
	}
	slog.Info("imported BGF match", "matchID", matchID, "file", filePath)
	return matchID, nil
}

// bgfApplyCheckerMove updates the board state after a BGF checker move.
// BGF from/to use 1-based point numbering from the active player's perspective:
//   - Points 1-24 = board positions (1 = player's 1-point)
//   - 25 = bar (from only)
//   - 0 = bear off (to only)
//
// Board state uses 0-based Green's perspective: indices 0-23 = points 1-24,
// 24 = Green's bar, 25 = Red's bar, 26 = Green off, 27 = Red off.
func bgfApplyCheckerMove(boardState *[28]int, moveData map[string]interface{}, player int) {
	fromArr := bgfGetIntArray(moveData, "from")
	toArr := bgfGetIntArray(moveData, "to")

	for i := 0; i < 4; i++ {
		from := fromArr[i]
		to := toArr[i]
		if from == -1 {
			break // No more submoves
		}

		// from/to come from an untrusted .bgf file. Compute the board indices
		// and skip the whole sub-move if either falls outside the 28-point
		// board, so a malformed file can't panic. Legal BGF values (0..25)
		// always map in range, so valid imports are unaffected.
		if player == -1 {
			// Green moves: Green's point N maps to board index (24-N)
			// BGF board: index 0 = Green's 24-point, index 23 = Green's 1-point
			// Green moves in decreasing direction (24→1→off)
			fromIdx := 24 // Green's bar
			if from != 25 {
				fromIdx = 24 - from // Green's point N → board index (24-N)
			}
			toIdx := 26 // bear-off slot used when to == 0
			if to != 0 {
				toIdx = 24 - to // Green's point N → board index (24-N)
			}
			if fromIdx < 0 || fromIdx >= 28 || toIdx < 0 || toIdx >= 28 {
				continue
			}

			// Remove checker from source
			boardState[fromIdx]--

			if to == 0 {
				// Bear off
				boardState[26]++
			} else {
				// Check for hit
				if boardState[toIdx] < 0 {
					// Hit Red checker - move it to Red's bar
					boardState[25] += boardState[toIdx] // boardState[toIdx] is negative, so this decrements
					boardState[toIdx] = 0
				}
				boardState[toIdx]++
			}
		} else {
			// Red moves: Red's point N maps to board index (N-1)
			// Red's point N = Green's point (25-N) = board index 24-(25-N) = N-1
			// Red moves in increasing direction (from Green's perspective)
			fromIdx := 25 // Red's bar
			if from != 25 {
				fromIdx = from - 1 // Red's point N → board index (N-1)
			}
			toIdx := 27 // bear-off slot used when to == 0
			if to != 0 {
				toIdx = to - 1 // Red's point N → board index (N-1)
			}
			if fromIdx < 0 || fromIdx >= 28 || toIdx < 0 || toIdx >= 28 {
				continue
			}

			// Remove checker from source (Red checkers are negative)
			boardState[fromIdx]++

			if to == 0 {
				// Bear off
				boardState[27]--
			} else {
				// Check for hit
				if boardState[toIdx] > 0 {
					// Hit Green checker - move it to Green's bar
					boardState[24] += boardState[toIdx]
					boardState[toIdx] = 0
				}
				boardState[toIdx]--
			}
		}
	}
}

// ComputeBGFMatchHash generates a unique hash for a BGF match for duplicate detection
func ComputeBGFMatchHash(match *bgfparser.Match) string {
	var hashBuilder strings.Builder

	data := match.Data
	p1 := strings.TrimSpace(strings.ToLower(bgfGetString(data, "nameGreen")))
	p2 := strings.TrimSpace(strings.ToLower(bgfGetString(data, "nameRed")))
	matchLen := bgfGetInt(data, "matchlen")
	hashBuilder.WriteString(fmt.Sprintf("bgf:%s|%s|%d|", p1, p2, matchLen))

	gamesData, _ := data["games"].([]interface{})
	for gameIdx, gameRaw := range gamesData {
		g, ok := gameRaw.(map[string]interface{})
		if !ok {
			continue
		}
		hashBuilder.WriteString(fmt.Sprintf("g%d:%d,%d,%d|",
			gameIdx, bgfGetInt(g, "scoreGreen"), bgfGetInt(g, "scoreRed"), bgfGetInt(g, "wonPoints")))

		movesData, _ := g["moves"].([]interface{})
		for moveIdx, moveRaw := range movesData {
			m, ok := moveRaw.(map[string]interface{})
			if !ok {
				continue
			}
			mtype := bgfGetString(m, "type")
			hashBuilder.WriteString(fmt.Sprintf("m%d:%s,", moveIdx, mtype))
			if mtype == "amove" {
				d1 := bgfGetInt(m, "green")
				d2 := bgfGetInt(m, "red")
				hashBuilder.WriteString(fmt.Sprintf("d%d%d|", d1, d2))
			} else if mtype == "adouble" || mtype == "atake" || mtype == "apass" {
				hashBuilder.WriteString(fmt.Sprintf("c%s|", mtype))
			}
		}
	}

	hash := sha256.Sum256([]byte(hashBuilder.String()))
	return hex.EncodeToString(hash[:])
}

// ComputeCanonicalMatchHashFromBGF computes a format-independent match hash from BGF data.
// Must produce the same hash as ComputeCanonicalMatchHashFromXG for the same match.
// Uses only the first N dice per game for cross-format compatibility.
func ComputeCanonicalMatchHashFromBGF(match *bgfparser.Match) string {
	var hashBuilder strings.Builder

	data := match.Data
	p1 := strings.TrimSpace(strings.ToLower(bgfGetString(data, "nameGreen")))
	p2 := strings.TrimSpace(strings.ToLower(bgfGetString(data, "nameRed")))
	matchLen := bgfGetInt(data, "matchlen")

	if p1 > p2 {
		p1, p2 = p2, p1
	}

	gamesData, _ := data["games"].([]interface{})
	hashBuilder.WriteString(fmt.Sprintf("canonical2:%s|%s|%d|%d|", p1, p2, matchLen, len(gamesData)))

	for gameIdx, gameRaw := range gamesData {
		g, ok := gameRaw.(map[string]interface{})
		if !ok {
			continue
		}
		hashBuilder.WriteString(fmt.Sprintf("g%d|", gameIdx))

		diceCount := 0
		movesData, _ := g["moves"].([]interface{})
		for _, moveRaw := range movesData {
			if diceCount >= maxCanonicalDicePerGame {
				break
			}
			m, ok := moveRaw.(map[string]interface{})
			if !ok {
				continue
			}
			mtype := bgfGetString(m, "type")
			if mtype == "amove" {
				// Skip cube actions encoded as amove (from[0] == -1)
				fromArr := bgfGetIntArray(m, "from")
				if len(fromArr) > 0 && fromArr[0] == -1 {
					continue
				}
				d1 := bgfGetInt(m, "green")
				d2 := bgfGetInt(m, "red")
				if d1 > d2 {
					d1, d2 = d2, d1
				}
				hashBuilder.WriteString(fmt.Sprintf("d%d%d|", d1, d2))
				diceCount++
			}
		}
	}

	hash := sha256.Sum256([]byte(hashBuilder.String()))
	return hex.EncodeToString(hash[:])
}

// ImportBGFPosition imports a single BGBlitz position from a TXT file
func (d *Database) ImportBGFPosition(filePath string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	graphs, err := ingest.MapBGFTextPosition(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to parse BGBlitz position file: %w", err)
	}
	return d.writeImportedPosition(graphs)
}

// ImportBGFPositionFromText imports a BGBlitz position from text content (clipboard/string)
func (d *Database) ImportBGFPositionFromText(content string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	graphs, err := ingest.MapBGFTextPositionText(content)
	if err != nil {
		return 0, fmt.Errorf("failed to parse BGBlitz position text: %w", err)
	}
	return d.writeImportedPosition(graphs)
}

// ImportXGPPosition imports an XG position file (.xgp) as a standalone position with analysis.
// XGP files use the same binary format as .xg match files but contain a single position.
func (d *Database) ImportXGPPosition(filePath string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	graphs, err := ingest.MapXGPPosition(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to parse XGP file: %w", err)
	}
	return d.writeImportedPosition(graphs)
}

func bgfGetString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func bgfGetInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		return bgfToInt(v)
	}
	return 0
}

func bgfGetIntArray(m map[string]interface{}, key string) [4]int {
	var result [4]int
	for i := range result {
		result[i] = -1
	}
	if v, ok := m[key]; ok {
		if arr, ok := v.([]interface{}); ok {
			for i := 0; i < 4 && i < len(arr); i++ {
				result[i] = bgfToInt(arr[i])
			}
		}
	}
	return result
}

func bgfToInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case int64:
		return int(val)
	case string:
		n, _ := strconv.Atoi(val)
		return n
	}
	return 0
}
