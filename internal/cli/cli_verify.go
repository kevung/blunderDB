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
	verifyCmd := flag.NewFlagSet("verify", flag.ExitOnError)

	// Define flags
	dbPath := verifyCmd.String("db", "", "Path to the database file (required)")
	matchID := verifyCmd.Int64("match", 0, "Match ID to verify (optional)")
	matFile := verifyCmd.String("mat", "", "MAT file to compare against (optional)")

	verifyCmd.Usage = func() {
		fmt.Println("Usage: blunderdb verify [options]")
		fmt.Println()
		fmt.Println("Verify database integrity and imported data.")
		fmt.Println()
		fmt.Println("Options:")
		verifyCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Verify database integrity")
		fmt.Println("  blunderdb verify --db database.db")
		fmt.Println()
		fmt.Println("  # Verify match against MAT file")
		fmt.Println("  blunderdb verify --db database.db --match 1 --mat test.mat")
	}

	if err := verifyCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		verifyCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	fmt.Println("Verifying database...")
	fmt.Println()

	// Get database stats
	stats, err := cli.db.GetDatabaseStats()
	if err != nil {
		return fmt.Errorf("failed to get database stats: %w", err)
	}

	// Display stats
	fmt.Println("Database Statistics:")
	if posCount, ok := stats["position_count"].(int64); ok {
		fmt.Printf("  Positions: %d\n", posCount)
	}
	if analysisCount, ok := stats["analysis_count"].(int64); ok {
		fmt.Printf("  Analyses: %d\n", analysisCount)
	}
	if matchCount, ok := stats["match_count"].(int64); ok {
		fmt.Printf("  Matches: %d\n", matchCount)
	}
	if gameCount, ok := stats["game_count"].(int64); ok {
		fmt.Printf("  Games: %d\n", gameCount)
	}
	if moveCount, ok := stats["move_count"].(int64); ok {
		fmt.Printf("  Moves: %d\n", moveCount)
	}
	fmt.Println()

	// Referential integrity: child rows whose parent is gone. The schema
	// cascades every one of these deletes, so a healthy database has none;
	// databases written before issue #157 was fixed may carry some.
	orphans, err := cli.db.CountOrphans()
	if err != nil {
		return fmt.Errorf("failed to count orphaned rows: %w", err)
	}
	printOrphans(orphans)

	// Schema drift: what opening the database could not add against the
	// reference DDL (EnsureSchema warns and goes on rather than refuse the
	// file). A query naming one of these elements fails until it is there.
	drift, err := cli.db.CheckSchema()
	if err != nil {
		return fmt.Errorf("failed to check the schema: %w", err)
	}
	printSchemaDrift(drift)

	// If match ID specified, verify that match
	if *matchID != 0 {
		err := cli.verifyMatch(*matchID, *matFile)
		if err != nil {
			return fmt.Errorf("match verification failed: %w", err)
		}
	}

	fmt.Println("Verification complete!")
	return nil
}

// printOrphans reports the referential-integrity check. Orphans are a
// finding, not a failure: the command still exits 0 so a routine verify of a
// database that has carried them for years keeps working, but the WARNING
// line makes them impossible to miss.
func printOrphans(o database.OrphanCounts) {
	if o.Total() == 0 {
		fmt.Println("Orphaned rows: none")
		fmt.Println()
		return
	}
	fmt.Println("Orphaned rows (child rows whose parent row is gone):")
	fmt.Printf("  Games without match: %d\n", o.GamesWithoutMatch)
	fmt.Printf("  Moves without game: %d\n", o.MovesWithoutGame)
	fmt.Printf("  Move analyses without move: %d\n", o.MoveAnalysesWithoutMove)
	fmt.Printf("  Analyses without position: %d\n", o.AnalysesWithoutPosition)
	fmt.Printf("WARNING: %d orphaned row(s) found. They were left behind by deletions made while\n", o.Total())
	fmt.Println("foreign keys were not enforced on every connection (issue #157); the rows are")
	fmt.Println("unreachable from any match and only take up space.")
	fmt.Println()
}

// printSchemaDrift reports the schema check. Like orphans, drift is a
// finding, not a failure — the command exits 0 — but the WARNING line names
// every element so the gap is not mistaken for a healthy database.
func printSchemaDrift(d sqlite.SchemaDrift) {
	if d.Count() == 0 {
		fmt.Println("Schema: matches the reference DDL")
		fmt.Println()
		return
	}
	fmt.Println("Schema drift (what the database lacks against the reference DDL):")
	if len(d.MissingTables) > 0 {
		fmt.Printf("  Missing tables: %s\n", strings.Join(d.MissingTables, ", "))
	}
	if len(d.MissingColumns) > 0 {
		fmt.Printf("  Missing columns: %s\n", strings.Join(d.MissingColumns, ", "))
	}
	if len(d.MissingIndexes) > 0 {
		fmt.Printf("  Missing indexes: %s\n", strings.Join(d.MissingIndexes, ", "))
	}
	fmt.Printf("WARNING: %d schema element(s) missing. Opening the database adds them when it can\n", d.Count())
	fmt.Println("and logs why it could not (a UNIQUE index over duplicate rows, typically); a query")
	fmt.Println("that names one of them fails until the cause is fixed.")
	fmt.Println()
}

// verifyMatch verifies a match against a MAT file
func (cli *CLI) verifyMatch(matchID int64, matFile string) error {
	fmt.Printf("Verifying match %d...\n", matchID)

	// Get match info
	match, err := cli.db.GetMatchByID(matchID)
	if err != nil {
		return fmt.Errorf("failed to get match: %w", err)
	}

	// Get match positions
	positions, err := cli.db.GetMatchMovePositions(matchID)
	if err != nil {
		return fmt.Errorf("failed to get match positions: %w", err)
	}

	fmt.Printf("  Match: %s vs %s\n", match.Player1Name, match.Player2Name)
	fmt.Printf("  Database positions: %d\n", len(positions))

	// If MAT file specified, compare
	if matFile != "" {
		fmt.Printf("  Comparing with MAT file: %s\n", matFile)

		// Read MAT file
		content, err := os.ReadFile(matFile)
		if err != nil {
			return fmt.Errorf("failed to read MAT file: %w", err)
		}

		// Count actual dice rolls in MAT file (each represents a checker move)
		contentStr := string(content)

		// Count dice patterns like "51:", "64:", etc.
		dicePattern := regexp.MustCompile(`[0-9]{2}:`)
		matCheckerMoves := len(dicePattern.FindAllString(contentStr, -1))

		// Count cube actions
		cubePattern := regexp.MustCompile(`(?i)(Doubles|Takes|Drops|Beaver|Passes)`)
		matCubeActions := len(cubePattern.FindAllString(contentStr, -1))

		fmt.Printf("  MAT file checker moves: %d\n", matCheckerMoves)
		fmt.Printf("  MAT file cube actions: %d\n", matCubeActions)
		fmt.Printf("  MAT file total: %d\n", matCheckerMoves+matCubeActions)

		fmt.Printf("  Database total positions: %d\n", len(positions))

		// Verify player1 is always displayed on bottom (stored from POV of player on roll)
		fmt.Println("\n  Verifying position storage (player on roll POV):")
		playerNeg1Count := 0 // XG format: -1 represents Player 1 (X)
		playerPos1Count := 0 // XG format: 1 represents Player 2 (O)
		for _, pos := range positions {
			if pos.PlayerOnRoll == -1 {
				playerNeg1Count++
			} else if pos.PlayerOnRoll == 1 {
				playerPos1Count++
			}
		}
		fmt.Printf("    Positions with Player 1 (X/-1) on roll: %d\n", playerNeg1Count)
		fmt.Printf("    Positions with Player 2 (O/+1) on roll: %d\n", playerPos1Count)
		fmt.Println("    Note: Positions stored from player on roll POV (frontend handles display)")

		fmt.Println("\n  Note: Run database query for accurate move type counts:")
		fmt.Println("    SELECT move_type, COUNT(*) FROM move GROUP BY move_type;")
	}

	fmt.Println()
	return nil
}
