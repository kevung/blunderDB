package cli

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// runVerify handles the verify command
func (cli *CLI) runVerify(args []string) error {
	verifyCmd := flag.NewFlagSet("verify", flag.ContinueOnError)

	// Define flags
	dbPath := verifyCmd.String("db", "", "Path to the database file (required)")
	matchID := verifyCmd.Int64("match", 0, "Match ID to verify (optional)")
	matFile := verifyCmd.String("mat", "", "MAT file to compare against (optional)")
	format := verifyCmd.String("format", "text", "Output format: text or json")

	verifyCmd.Usage = func() {
		fmt.Fprintln(cli.stdout(), "Usage: blunderdb verify [options]")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Verify database integrity and imported data.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Options:")
		verifyCmd.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		fmt.Fprintln(cli.stdout(), "  # Verify database integrity")
		fmt.Fprintln(cli.stdout(), "  blunderdb verify --db database.db")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "  # Verify match against MAT file")
		fmt.Fprintln(cli.stdout(), "  blunderdb verify --db database.db --match 1 --mat test.mat")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "  # Machine-readable output")
		fmt.Fprintln(cli.stdout(), "  blunderdb verify --db database.db --format json")
	}

	if err := verifyCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		verifyCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	formatLower := strings.ToLower(*format)
	if formatLower != "text" && formatLower != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}
	text := formatLower != "json"

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	if text {
		fmt.Fprintln(cli.stdout(), "Verifying database...")
		fmt.Fprintln(cli.stdout())
	}

	// Get database stats
	stats, err := cli.db.GetDatabaseStats()
	if err != nil {
		return fmt.Errorf("failed to get database stats: %w", err)
	}

	if text {
		// Display stats
		fmt.Fprintln(cli.stdout(), "Database Statistics:")
		if posCount, ok := stats["position_count"].(int64); ok {
			fmt.Fprintf(cli.stdout(), "  Positions: %d\n", posCount)
		}
		if analysisCount, ok := stats["analysis_count"].(int64); ok {
			fmt.Fprintf(cli.stdout(), "  Analyses: %d\n", analysisCount)
		}
		if matchCount, ok := stats["match_count"].(int64); ok {
			fmt.Fprintf(cli.stdout(), "  Matches: %d\n", matchCount)
		}
		if gameCount, ok := stats["game_count"].(int64); ok {
			fmt.Fprintf(cli.stdout(), "  Games: %d\n", gameCount)
		}
		if moveCount, ok := stats["move_count"].(int64); ok {
			fmt.Fprintf(cli.stdout(), "  Moves: %d\n", moveCount)
		}
		fmt.Fprintln(cli.stdout())
	}

	// Referential integrity: child rows whose parent is gone. The schema
	// cascades every one of these deletes, so a healthy database has none.
	orphans, err := cli.db.CountOrphans()
	if err != nil {
		return fmt.Errorf("failed to count orphaned rows: %w", err)
	}
	if text {
		cli.printOrphans(orphans)
	}

	// Schema drift: what opening the database could not add against the
	// reference DDL (EnsureSchema warns and goes on rather than refuse the
	// file). A query naming one of these elements fails until it is there.
	drift, err := cli.db.CheckSchema()
	if err != nil {
		return fmt.Errorf("failed to check the schema: %w", err)
	}
	if text {
		cli.printSchemaDrift(drift)
	}

	// Constraints the current DDL states but SQLite cannot add to a table that
	// already exists (CHECK, NOT NULL): a fresh database enforces them, an
	// upgraded one is judged against them here.
	violations, err := cli.db.CheckConstraints()
	if err != nil {
		return fmt.Errorf("failed to check the constraints: %w", err)
	}
	if text {
		cli.printConstraintViolations(violations)
	}

	// The two denormalised counters, recomputed from the rows.
	counters, err := cli.db.CheckCounters()
	if err != nil {
		return fmt.Errorf("failed to recompute the counters: %w", err)
	}
	if text {
		cli.printCounterDrift(counters)
	}

	// If match ID specified, verify that match
	var matchResult *verifyMatchResult
	if *matchID != 0 {
		var err error
		matchResult, err = cli.verifyMatch(*matchID, *matFile, text)
		if err != nil {
			return fmt.Errorf("match verification failed: %w", err)
		}
	}

	if !text {
		return cli.printJSON(verifyResult{
			Stats:                    stats,
			Orphans:                  orphans,
			OrphanTotal:              orphans.Total(),
			SchemaDrift:              drift,
			SchemaDriftCount:         drift.Count(),
			ConstraintViolations:     violations,
			ConstraintViolationTotal: database.TotalConstraintViolations(violations),
			CounterDrift:             counters,
			CounterDriftTotal:        counters.Total(),
			Match:                    matchResult,
		})
	}

	fmt.Fprintln(cli.stdout(), "Verification complete!")
	return nil
}

// verifyResult is the --format json shape for `verify`.
type verifyResult struct {
	Stats                    map[string]interface{}         `json:"stats"`
	Orphans                  database.OrphanCounts          `json:"orphans"`
	OrphanTotal              int64                          `json:"orphan_total"`
	SchemaDrift              sqlite.SchemaDrift             `json:"schema_drift"`
	SchemaDriftCount         int                            `json:"schema_drift_count"`
	ConstraintViolations     []database.ConstraintViolation `json:"constraint_violations"`
	ConstraintViolationTotal int64                          `json:"constraint_violation_total"`
	CounterDrift             database.CounterDrift          `json:"counter_drift"`
	CounterDriftTotal        int64                          `json:"counter_drift_total"`
	Match                    *verifyMatchResult             `json:"match,omitempty"`
}

// verifyMatchResult is the --format json shape for `verify --match`.
type verifyMatchResult struct {
	MatchID           int64  `json:"match_id"`
	Player1           string `json:"player1"`
	Player2           string `json:"player2"`
	DatabasePositions int    `json:"database_positions"`
	MatFile           string `json:"mat_file,omitempty"`
	MatCheckerMoves   int    `json:"mat_checker_moves,omitempty"`
	MatCubeActions    int    `json:"mat_cube_actions,omitempty"`
}

// printOrphans reports the referential-integrity check. Orphans are a
// finding, not a failure: the command still exits 0 so a routine verify of a
// database that has carried them for years keeps working, but the WARNING
// line makes them impossible to miss.
func (cli *CLI) printOrphans(o database.OrphanCounts) {
	if o.Total() == 0 {
		fmt.Fprintln(cli.stdout(), "Orphaned rows: none")
		fmt.Fprintln(cli.stdout())
		return
	}
	fmt.Fprintln(cli.stdout(), "Orphaned rows (child rows whose parent row is gone):")
	fmt.Fprintf(cli.stdout(), "  Games without match: %d\n", o.GamesWithoutMatch)
	fmt.Fprintf(cli.stdout(), "  Moves without game: %d\n", o.MovesWithoutGame)
	fmt.Fprintf(cli.stdout(), "  Move analyses without move: %d\n", o.MoveAnalysesWithoutMove)
	fmt.Fprintf(cli.stdout(), "  Analyses without position: %d\n", o.AnalysesWithoutPosition)
	fmt.Fprintf(cli.stdout(), "  Reviews without deck: %d\n", o.ReviewsWithoutDeck)
	fmt.Fprintf(cli.stdout(), "  Reviews without position: %d\n", o.ReviewsWithoutPosition)
	fmt.Fprintf(cli.stdout(), "  Training items without session: %d\n", o.TrainingItemsWithoutSession)
	fmt.Fprintf(cli.stdout(), "WARNING: %d orphaned row(s) found. They were left behind by deletions made while\n", o.Total())
	fmt.Fprintln(cli.stdout(), "foreign keys were not enforced on every connection (issue #157), or before the")
	fmt.Fprintln(cli.stdout(), "review journal's own keys existed (issue #185); the rows are unreachable from any")
	fmt.Fprintln(cli.stdout(), "match or deck and only take up space.")
	fmt.Fprintln(cli.stdout())
}

// printSchemaDrift reports the schema check. Like orphans, drift is a
// finding, not a failure — the command exits 0 — but the WARNING line names
// every element so the gap is not mistaken for a healthy database.
func (cli *CLI) printSchemaDrift(d sqlite.SchemaDrift) {
	if d.Count() == 0 {
		fmt.Fprintln(cli.stdout(), "Schema: matches the reference DDL")
		fmt.Fprintln(cli.stdout())
		return
	}
	fmt.Fprintln(cli.stdout(), "Schema drift (what the database lacks against the reference DDL):")
	if len(d.MissingTables) > 0 {
		fmt.Fprintf(cli.stdout(), "  Missing tables: %s\n", strings.Join(d.MissingTables, ", "))
	}
	if len(d.MissingColumns) > 0 {
		fmt.Fprintf(cli.stdout(), "  Missing columns: %s\n", strings.Join(d.MissingColumns, ", "))
	}
	if len(d.MissingIndexes) > 0 {
		fmt.Fprintf(cli.stdout(), "  Missing indexes: %s\n", strings.Join(d.MissingIndexes, ", "))
	}
	fmt.Fprintf(cli.stdout(), "WARNING: %d schema element(s) missing. Opening the database adds them when it can\n", d.Count())
	fmt.Fprintln(cli.stdout(), "and logs why it could not (a UNIQUE index over duplicate rows, typically); a query")
	fmt.Fprintln(cli.stdout(), "that names one of them fails until the cause is fixed.")
	fmt.Fprintln(cli.stdout())
}

// printConstraintViolations reports the rules of the current DDL an upgraded
// database does not satisfy. Like orphans and drift it is a finding, not a
// failure — the command exits 0 — and only the breached rules are listed, so
// the healthy case is one line.
func (cli *CLI) printConstraintViolations(v []database.ConstraintViolation) {
	total := database.TotalConstraintViolations(v)
	if total == 0 {
		fmt.Fprintf(cli.stdout(), "Constraints: every row satisfies the current DDL (%d rules checked)\n", len(v))
		fmt.Fprintln(cli.stdout())
		return
	}
	fmt.Fprintln(cli.stdout(), "Constraint violations (rules a fresh database enforces, this one does not):")
	for _, c := range v {
		if c.Count > 0 {
			fmt.Fprintf(cli.stdout(), "  %s: %d row(s)\n", c.Name, c.Count)
		}
	}
	fmt.Fprintf(cli.stdout(), "WARNING: %d row(s) would not be accepted by the current schema. SQLite cannot add a\n", total)
	fmt.Fprintln(cli.stdout(), "CHECK or a NOT NULL to a table that already exists, so these rules are enforced on")
	fmt.Fprintln(cli.stdout(), "databases created since and reported — not repaired — on older ones.")
	fmt.Fprintln(cli.stdout())
}

// printCounterDrift reports the two denormalised counters against the rows
// they claim to count. Like every other section of verify it is a finding and
// not a failure: game_count and move_count record what the SOURCE FILE held, so
// a small gap can be an importer that skipped what it could not map, and the
// worst gap is what separates that from a figure that means nothing.
func (cli *CLI) printCounterDrift(c database.CounterDrift) {
	if c.Total() == 0 {
		fmt.Fprintln(cli.stdout(), "Counters: game_count and move_count agree with the rows")
		fmt.Fprintln(cli.stdout())
		return
	}
	fmt.Fprintln(cli.stdout(), "Counter drift (the stored figure against the rows it counts):")
	if c.MatchesWithWrongGameCount > 0 {
		fmt.Fprintf(cli.stdout(), "  Matches whose game_count is wrong: %d (worst gap: %d game(s))\n",
			c.MatchesWithWrongGameCount, c.WorstGameCountGap)
	}
	if c.GamesWithWrongMoveCount > 0 {
		fmt.Fprintf(cli.stdout(), "  Games whose move_count is wrong: %d (worst gap: %d move(s))\n",
			c.GamesWithWrongMoveCount, c.WorstMoveCountGap)
	}
	fmt.Fprintln(cli.stdout(), "NOTE: both counters record what the SOURCE FILE held and are never recomputed;")
	fmt.Fprintln(cli.stdout(), "they are what the match list and the game view display. A small gap is usually an")
	fmt.Fprintln(cli.stdout(), "import that skipped what it could not map. Nothing is rewritten here.")
	fmt.Fprintln(cli.stdout())
}

// verifyMatch verifies a match against a MAT file
func (cli *CLI) verifyMatch(matchID int64, matFile string, text bool) (*verifyMatchResult, error) {
	if text {
		fmt.Fprintf(cli.stdout(), "Verifying match %d...\n", matchID)
	}

	// Get match info
	match, err := cli.db.GetMatchByID(matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get match: %w", err)
	}

	// Get match positions
	positions, err := cli.db.GetMatchMovePositions(matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get match positions: %w", err)
	}

	result := &verifyMatchResult{
		MatchID:           matchID,
		Player1:           match.Player1Name,
		Player2:           match.Player2Name,
		DatabasePositions: len(positions),
	}

	if text {
		fmt.Fprintf(cli.stdout(), "  Match: %s vs %s\n", match.Player1Name, match.Player2Name)
		fmt.Fprintf(cli.stdout(), "  Database positions: %d\n", len(positions))
	}

	// If MAT file specified, compare
	if matFile != "" {
		result.MatFile = matFile
		if text {
			fmt.Fprintf(cli.stdout(), "  Comparing with MAT file: %s\n", matFile)
		}

		// Read MAT file
		content, err := os.ReadFile(matFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read MAT file: %w", err)
		}

		// Count actual dice rolls in MAT file (each represents a checker move)
		contentStr := string(content)

		// Count dice patterns like "51:", "64:", etc.
		dicePattern := regexp.MustCompile(`[0-9]{2}:`)
		matCheckerMoves := len(dicePattern.FindAllString(contentStr, -1))

		// Count cube actions
		cubePattern := regexp.MustCompile(`(?i)(Doubles|Takes|Drops|Beaver|Passes)`)
		matCubeActions := len(cubePattern.FindAllString(contentStr, -1))
		result.MatCheckerMoves = matCheckerMoves
		result.MatCubeActions = matCubeActions

		if text {
			fmt.Fprintf(cli.stdout(), "  MAT file checker moves: %d\n", matCheckerMoves)
			fmt.Fprintf(cli.stdout(), "  MAT file cube actions: %d\n", matCubeActions)
			fmt.Fprintf(cli.stdout(), "  MAT file total: %d\n", matCheckerMoves+matCubeActions)

			fmt.Fprintf(cli.stdout(), "  Database total positions: %d\n", len(positions))

			// Verify player1 is always displayed on bottom (stored from POV of player on roll)
			fmt.Fprintln(cli.stdout(), "\n  Verifying position storage (player on roll POV):")
			playerNeg1Count := 0 // XG format: -1 represents Player 1 (X)
			playerPos1Count := 0 // XG format: 1 represents Player 2 (O)
			for _, pos := range positions {
				if pos.PlayerOnRoll == -1 {
					playerNeg1Count++
				} else if pos.PlayerOnRoll == 1 {
					playerPos1Count++
				}
			}
			fmt.Fprintf(cli.stdout(), "    Positions with Player 1 (X/-1) on roll: %d\n", playerNeg1Count)
			fmt.Fprintf(cli.stdout(), "    Positions with Player 2 (O/+1) on roll: %d\n", playerPos1Count)
			fmt.Fprintln(cli.stdout(), "    Note: Positions stored from player on roll POV (frontend handles display)")

			fmt.Fprintln(cli.stdout(), "\n  Note: Run database query for accurate move type counts:")
			fmt.Fprintln(cli.stdout(), "    SELECT move_type, COUNT(*) FROM move GROUP BY move_type;")
		}
	}

	if text {
		fmt.Fprintln(cli.stdout())
	}
	return result, nil
}
