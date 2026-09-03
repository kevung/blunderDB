package cli

import (
	"flag"
	"fmt"
	"strings"
)

// runDelete handles the delete command
func (cli *CLI) runDelete(args []string) error {
	deleteCmd := flag.NewFlagSet("delete", flag.ContinueOnError)

	// Define flags
	dbPath := deleteCmd.String("db", "", "Path to the database file (required)")
	deleteType := deleteCmd.String("type", "", "Delete type: match (required)")
	id := deleteCmd.Int64("id", 0, "ID of the item to delete (required)")
	confirm := deleteCmd.Bool("confirm", false, "Confirm deletion without prompting")
	format := deleteCmd.String("format", "text", "Output format: text or json")

	deleteCmd.Usage = func() {
		fmt.Println("Usage: blunderdb delete [options]")
		fmt.Println()
		fmt.Println("Delete data from the database.")
		fmt.Println()
		fmt.Println("Options:")
		deleteCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Delete match with ID 1")
		fmt.Println("  blunderdb delete --db database.db --type match --id 1 --confirm")
	}

	if err := deleteCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		deleteCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	if *deleteType == "" {
		deleteCmd.Usage()
		return fmt.Errorf("missing required flag: --type")
	}

	if *id == 0 {
		deleteCmd.Usage()
		return fmt.Errorf("missing required flag: --id")
	}

	formatLower := strings.ToLower(*format)
	if formatLower != "text" && formatLower != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	// Perform deletion based on type
	switch strings.ToLower(*deleteType) {
	case "match":
		return cli.deleteMatch(*id, *confirm, formatLower)
	default:
		return fmt.Errorf("unknown delete type: %s (must be 'match')", *deleteType)
	}
}

// deleteMatch deletes a match from the database
func (cli *CLI) deleteMatch(matchID int64, confirm bool, format string) error {
	text := format != "json"

	// Get match details first
	match, err := cli.db.GetMatchByID(matchID)
	if err != nil {
		return fmt.Errorf("failed to get match: %w", err)
	}
	if match == nil {
		return fmt.Errorf("match with ID %d not found", matchID)
	}

	if text {
		// Show match details
		fmt.Printf("Match ID: %d\n", match.ID)
		fmt.Printf("  Players: %s vs %s\n", match.Player1Name, match.Player2Name)
		if match.Event != "" {
			fmt.Printf("  Event: %s\n", match.Event)
		}
		fmt.Printf("  Games: %d\n", match.GameCount)
		fmt.Println()
	}

	// Confirm deletion
	if !confirm {
		fmt.Print("Are you sure you want to delete this match? (yes/no): ")
		var response string
		_, _ = fmt.Scanln(&response)
		if strings.ToLower(response) != "yes" {
			if text {
				fmt.Println("Deletion cancelled")
				return nil
			}
			return printJSON(struct {
				MatchID   int64 `json:"match_id"`
				Cancelled bool  `json:"cancelled"`
			}{MatchID: matchID, Cancelled: true})
		}
	}

	// Delete the match
	err = cli.db.DeleteMatch(matchID)
	if err != nil {
		return fmt.Errorf("failed to delete match: %w", err)
	}

	if !text {
		return printJSON(struct {
			MatchID int64 `json:"match_id"`
			Deleted bool  `json:"deleted"`
		}{MatchID: matchID, Deleted: true})
	}

	fmt.Printf("Successfully deleted match ID %d\n", matchID)
	return nil
}
