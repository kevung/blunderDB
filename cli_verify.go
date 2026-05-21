package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
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
		return fmt.Errorf("failed to get database stats: %v", err)
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

	// If match ID specified, verify that match
	if *matchID != 0 {
		err := cli.verifyMatch(*matchID, *matFile)
		if err != nil {
			return fmt.Errorf("match verification failed: %v", err)
		}
	}

	fmt.Println("Verification complete!")
	return nil
}

// verifyMatch verifies a match against a MAT file
func (cli *CLI) verifyMatch(matchID int64, matFile string) error {
	fmt.Printf("Verifying match %d...\n", matchID)

	// Get match info
	match, err := cli.db.GetMatchByID(matchID)
	if err != nil {
		return fmt.Errorf("failed to get match: %v", err)
	}

	// Get match positions
	positions, err := cli.db.GetMatchMovePositions(matchID)
	if err != nil {
		return fmt.Errorf("failed to get match positions: %v", err)
	}

	fmt.Printf("  Match: %s vs %s\n", match.Player1Name, match.Player2Name)
	fmt.Printf("  Database positions: %d\n", len(positions))

	// If MAT file specified, compare
	if matFile != "" {
		fmt.Printf("  Comparing with MAT file: %s\n", matFile)

		// Read MAT file
		content, err := os.ReadFile(matFile)
		if err != nil {
			return fmt.Errorf("failed to read MAT file: %v", err)
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
