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
