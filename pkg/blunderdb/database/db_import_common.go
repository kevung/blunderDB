package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// openExistingSQLite opens a database file that is only being read from —
// an import source, a file being inspected. The path is checked first (a
// directory is refused): sql.Open would create an empty file at a mistyped one.
func openExistingSQLite(path string) (*sql.DB, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("database file not found: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("database file not found: %s is a directory", path)
	}
	return sql.Open("sqlite", path)
}

// writeImportedMatch persists a mapped MatchGraph through the storage backend,
// shared by the format-specific Import* methods that delegate to the ingest
// pipeline. It preserves the GUI/CLI duplicate contract: an exact same-format
// re-import returns a *DuplicateMatchError, which is ErrDuplicateMatch (the
// ingest layer reports it as a silent skip), while a cross-format canonical duplicate is enriched in place and
// returns the existing match id without error. Callers must hold d.mu.
func (d *Database) writeImportedMatch(ctx context.Context, graph *ingest.MatchGraph) (int64, error) {
	// Stamp the import batch here, the one point every format passes through.
	graph.ImportBatchID = d.importBatchID
	graph.SkipDuplicates = d.skipDuplicates.Load()
	tx, err := d.store.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	res, err := ingest.WriteMatch(ctx, tx, "", graph, nil)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	d.importBatchCounts.AnalysesDropped += res.DroppedAnalyses
	if res.Skipped {
		// A duplicate still carries the study marks added in the source tool
		// since the first import (ADR-0006), and the analyses deeper than the
		// stored ones: they are committed, the rest of the file was never
		// written.
		if res.FlagsApplied > 0 || res.Deepened > 0 {
			if err := tx.Commit(); err != nil {
				return 0, err
			}
		} else {
			_ = tx.Rollback()
		}
		d.importBatchCounts.MatchesSkipped++
		if res.Deepened > 0 {
			d.importBatchCounts.MatchesDeepened++
			d.importBatchCounts.AnalysesDeepened += res.Deepened
		}
		return 0, &DuplicateMatchError{MatchID: res.MatchID, FlagsApplied: res.FlagsApplied, Deepened: res.Deepened}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	if res.ProbableDuplicate != nil {
		d.importBatchCounts.ProbableDuplicates = append(d.importBatchCounts.ProbableDuplicates, *res.ProbableDuplicate)
	}
	// Only the writing path knows written vs enriched; callers see an id either way.
	if res.Enriched {
		d.importBatchCounts.MatchesEnriched++
	} else {
		d.importBatchCounts.MatchesImported++
	}
	d.importBatchCounts.PositionsSaved += res.SavedPositions
	d.positionsSinceStats += res.SavedPositions
	return res.MatchID, nil
}

// writeImportedPosition persists the positions of a single-position import
// (XGP / BGF text) through the storage backend and returns the id of the first
// stored position. Positions dedup by Zobrist hash, so re-importing the same
// position returns its existing id. Callers must hold d.mu and pass the
// context from their own beginCancellableImport, so CancelImport reaches it.
func (d *Database) writeImportedPosition(ctx context.Context, graphs []ingest.PositionGraph) (int64, error) {
	if len(graphs) == 0 {
		return 0, fmt.Errorf("no position to import")
	}
	tx, err := d.store.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	var firstID int64
	for i := range graphs {
		id, err := ingest.WritePosition(ctx, tx, "", &graphs[i])
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}
		if i == 0 {
			firstID = id
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	d.positionsSinceStats += len(graphs)
	return firstID, nil
}

// enginePriority returns a sort priority for analysis engines.
// XG gets priority 0 (first), GNUbg gets 1, unknown/empty gets 2.
func enginePriority(engine string) int {
	switch strings.ToLower(engine) {
	case "xg":
		return 0
	case "gnubg":
		return 1
	default:
		return 2
	}
}

// sortCubeAnalysesByEngine sorts cube analyses so XG comes first, then GNUbg, then others.
func sortCubeAnalysesByEngine(analyses []DoublingCubeAnalysis) {
	sort.SliceStable(analyses, func(i, j int) bool {
		return enginePriority(analyses[i].AnalysisEngine) < enginePriority(analyses[j].AnalysisEngine)
	})
}

// mergeCheckerMoves merges two sets of checker moves, avoiding duplicates
// Moves are considered duplicates if they have the same move string
// Returns moves sorted by equity (highest first), with XG engine preferred as tiebreaker
func mergeCheckerMoves(existing, incoming []CheckerMove) []CheckerMove {
	// Use a map to track unique moves by their move string
	moveMap := make(map[string]CheckerMove)

	// Add existing moves to the map
	for _, m := range existing {
		moveMap[m.Move] = m
	}

	// Add incoming moves, prefer incoming if there's a conflict (newer analysis)
	for _, m := range incoming {
		if existingMove, exists := moveMap[m.Move]; exists {
			// If the incoming move has the same depth or higher quality analysis, use it
			// Otherwise keep the existing one. The labels are free text ("2-ply",
			// "XG Roller++"): compare their rank, never the strings.
			if domain.AnalysisDepthRank(m.AnalysisDepth) >= domain.AnalysisDepthRank(existingMove.AnalysisDepth) {
				moveMap[m.Move] = m
			}
		} else {
			moveMap[m.Move] = m
		}
	}

	// Convert map to slice
	result := make([]CheckerMove, 0, len(moveMap))
	for _, m := range moveMap {
		result = append(result, m)
	}

	// Sort by equity (highest first), with XG engine preferred as tiebreaker
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Equity != result[j].Equity {
			return result[i].Equity > result[j].Equity
		}
		return enginePriority(result[i].AnalysisEngine) < enginePriority(result[j].AnalysisEngine)
	})

	// Recalculate equity errors relative to the best move
	if len(result) > 0 {
		bestEquity := result[0].Equity
		for i := range result {
			result[i].Index = i
			if i == 0 {
				result[i].EquityError = nil
			} else {
				diff := bestEquity - result[i].Equity
				result[i].EquityError = &diff
			}
		}
	}

	return result
}

// mergePlayedMoves merges played moves/cube actions, avoiding duplicates
func mergePlayedMoves(existing, incoming []string) []string {
	// Use a map to track unique moves
	moveSet := make(map[string]bool)

	for _, m := range existing {
		if m != "" {
			moveSet[normalizeMove(m)] = true
		}
	}

	for _, m := range incoming {
		if m != "" {
			moveSet[normalizeMove(m)] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(moveSet))
	for m := range moveSet {
		result = append(result, m)
	}

	sort.Strings(result)
	return result
}

// normalizeMove is re-exported from package engine (see analysiscodec.go).
var normalizeMove = engine.NormalizeMove

// ErrDuplicateMatch is returned when attempting to import a match that already exists
var ErrDuplicateMatch = fmt.Errorf("duplicate match: this match has already been imported")

// DuplicateMatchError is how an import reports a match already stored:
// errors.Is(err, ErrDuplicateMatch) holds. FlagsApplied tells "duplicate, N
// study marks delivered" (committed) from "duplicate, nothing to do" (0);
// Deepened counts the stored analyses the duplicate replaced with deeper ones.
type DuplicateMatchError struct {
	MatchID      int64
	FlagsApplied int
	Deepened     int
}

func (e *DuplicateMatchError) Error() string {
	var extra []string
	if e.FlagsApplied > 0 {
		extra = append(extra, fmt.Sprintf("%d study marks applied", e.FlagsApplied))
	}
	if e.Deepened > 0 {
		extra = append(extra, fmt.Sprintf("%d analyses deepened", e.Deepened))
	}
	if len(extra) == 0 {
		return ErrDuplicateMatch.Error()
	}
	return fmt.Sprintf("%s (%s)", ErrDuplicateMatch.Error(), strings.Join(extra, ", "))
}

// Is makes every DuplicateMatchError match ErrDuplicateMatch.
func (e *DuplicateMatchError) Is(target error) bool { return target == ErrDuplicateMatch }

// computeMatchHashFromStoredData computes a hash for existing matches in the database
// This is used during migration when we don't have access to the original XG file
func computeMatchHashFromStoredData(db *sql.DB, matchID int64, p1Name, p2Name string, matchLength int32) string {
	var hashBuilder strings.Builder

	// Include metadata (normalized)
	p1 := strings.TrimSpace(strings.ToLower(p1Name))
	p2 := strings.TrimSpace(strings.ToLower(p2Name))
	hashBuilder.WriteString(fmt.Sprintf("meta:%s|%s|%d|", p1, p2, matchLength))

	// Query all games for this match
	gameRows, err := db.Query(`
		SELECT id, game_number, initial_score_1, initial_score_2, winner, points_won 
		FROM game WHERE match_id = ? ORDER BY game_number`, matchID)
	if err != nil {
		// Fallback to simple hash
		hash := sha256.Sum256([]byte(hashBuilder.String()))
		return hex.EncodeToString(hash[:])
	}
	defer gameRows.Close()

	for gameRows.Next() {
		var gameID int64
		var gameNum, initScore1, initScore2, winner, pointsWon int32
		if err := gameRows.Scan(&gameID, &gameNum, &initScore1, &initScore2, &winner, &pointsWon); err != nil {
			continue
		}

		hashBuilder.WriteString(fmt.Sprintf("g%d:%d,%d,%d,%d|", gameNum, initScore1, initScore2, winner, pointsWon))

		// Query all moves for this game. A query error skips the game (the
		// hash degrades, as it always has); an iteration error gives up.
		err := forEachRow(db, `
			SELECT move_number, move_type, dice_1, dice_2, checker_move, cube_action 
			FROM move WHERE game_id = ? ORDER BY move_number`, []any{gameID}, func(moveRows *sql.Rows) error {
			var moveNum int32
			var moveType string
			var dice1, dice2 int32
			var checkerMove, cubeAction sql.NullString
			if err := moveRows.Scan(&moveNum, &moveType, &dice1, &dice2, &checkerMove, &cubeAction); err != nil {
				return nil
			}

			hashBuilder.WriteString(fmt.Sprintf("m%d:%s,", moveNum, moveType))
			if moveType == "checker" && checkerMove.Valid {
				hashBuilder.WriteString(fmt.Sprintf("d%d%d,p%s|", dice1, dice2, checkerMove.String))
			}
			if moveType == "cube" && cubeAction.Valid {
				hashBuilder.WriteString(fmt.Sprintf("c%s|", cubeAction.String))
			}
			return nil
		})
		if err != nil {
			return ""
		}
	}
	if err := gameRows.Err(); err != nil {
		return ""
	}

	// Compute SHA256 hash
	hash := sha256.Sum256([]byte(hashBuilder.String()))
	return hex.EncodeToString(hash[:])
}

// CheckMatchExists checks if a match with the given hash already exists in the database
// Returns the existing match ID if found, 0 otherwise
func (d *Database) CheckMatchExists(matchHash string) (int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var existingID int64
	err := d.db.QueryRow(`SELECT id FROM match WHERE match_hash = ?`, matchHash).Scan(&existingID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("error checking for duplicate match: %w", err)
	}
	return existingID, nil
}

// SetSkipDuplicates chooses what the following imports do with an exact
// duplicate: by default (false) its analyses deeper than the stored ones
// replace them and the rest is skipped; true skips it outright, the behaviour
// of `import --skip-duplicates`.
func (d *Database) SetSkipDuplicates(skip bool) { d.skipDuplicates.Store(skip) }

// FindDuplicateMatches lists the pairs of stored matches the dice say are
// probably one (ingest.FindDuplicateSuspects), filling on the way the
// dice_hash of the matches imported before it existed. Nothing is merged.
func (d *Database) FindDuplicateMatches() ([]domain.DuplicateSuspect, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.store == nil {
		return nil, fmt.Errorf("find duplicate matches: %w", storage.ErrInternal)
	}
	out, _, err := ingest.FindDuplicateSuspects(context.Background(), d.store.Matches(), "")
	return out, err
}
