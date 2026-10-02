package database

import (
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// ============================================================================
// GnuBG / Jellyfish import functions (SGF, MAT, TXT formats)
// ============================================================================

// ImportGnuBGMatchFromText imports a match from clipboard/string content in
// MAT/TXT format, delegating to the shared ingest pipeline (ingest.MapGnuBGText
// → ingest.WriteMatch).
func (d *Database) ImportGnuBGMatchFromText(content string) (int64, error) {
	ctx, done := d.beginCancellableImport()
	defer done()

	d.mu.Lock()
	defer d.mu.Unlock()

	graph, err := ingest.MapGnuBGText(content)
	if err != nil {
		return 0, fmt.Errorf("failed to parse match text: %w", err)
	}
	return d.writeImportedMatch(ctx, graph)
}

// ImportGnuBGMatch imports a match from a GnuBG file (SGF, MAT, or TXT format),
// delegating to the shared ingest pipeline (ingest.MapGnuBG → ingest.WriteMatch)
// — the same path the headless server uses.
func (d *Database) ImportGnuBGMatch(filePath string) (int64, error) {
	ctx, done := d.beginCancellableImport()
	defer done()

	d.mu.Lock()
	defer d.mu.Unlock()

	graph, err := ingest.MapGnuBG(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to parse file: %w", err)
	}
	matchID, err := d.writeImportedMatch(ctx, graph)
	if err != nil {
		return 0, err
	}
	slog.Info("imported gnubg match", "matchID", matchID, "file", filePath)
	return matchID, nil
}

// convertGnuBGMoveToString converts a gnubgparser Move[8]int to standard notation.
// This function handles moves in ABSOLUTE coordinates (SGF format).
// Move encoding: 0-23 = board points (absolute), 24 = bar, 25 = off, -1 = unused
// For player 0: gnubg point i → standard point (i+1)
// For player 1: gnubg point i → standard point (24-i)
func convertGnuBGMoveToString(move [8]int, player int) string {
	formatPoint := func(pt int, p int) string {
		if pt == 24 {
			return "bar"
		}
		if pt == 25 {
			return "off"
		}
		if p == 0 {
			return fmt.Sprintf("%d", pt+1) // Player 0: 0→1, 23→24
		}
		return fmt.Sprintf("%d", 24-pt) // Player 1: 0→24, 23→1
	}

	return formatGnuBGMoveItems(move, player, formatPoint)
}

// formatGnuBGMoveItems is a helper that formats move items using a point formatter.
func formatGnuBGMoveItems(move [8]int, player int, formatPoint func(int, int) string) string {

	type moveItem struct {
		from string
		to   string
	}

	var items []moveItem
	for i := 0; i < 8; i += 2 {
		from := move[i]
		to := move[i+1]
		if from == -1 {
			break
		}
		items = append(items, moveItem{
			from: formatPoint(from, player),
			to:   formatPoint(to, player),
		})
	}

	if len(items) == 0 {
		return "Cannot Move"
	}

	// Sort by 'from' point descending (standard notation)
	sort.Slice(items, func(i, j int) bool {
		// Parse to int for comparison
		fi, _ := strconv.Atoi(items[i].from)
		fj, _ := strconv.Atoi(items[j].from)
		if items[i].from == "bar" {
			fi = 25
		}
		if items[j].from == "bar" {
			fj = 25
		}
		return fi > fj
	})

	// Group identical moves with multiplier
	var moves []string
	for i := 0; i < len(items); {
		item := items[i]
		count := 1
		for j := i + 1; j < len(items); j++ {
			if items[j].from == item.from && items[j].to == item.to {
				count++
			} else {
				break
			}
		}
		if count > 1 {
			moves = append(moves, fmt.Sprintf("%s/%s(%d)", item.from, item.to, count))
		} else {
			moves = append(moves, fmt.Sprintf("%s/%s", item.from, item.to))
		}
		i += count
	}

	return strings.Join(moves, " ")
}
