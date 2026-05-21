package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
)

// runInfo handles the info command
func (cli *CLI) runInfo(args []string) error {
	infoCmd := flag.NewFlagSet("info", flag.ExitOnError)

	// Define flags
	dbPath := infoCmd.String("db", "", "Path to the database file (required)")
	format := infoCmd.String("format", "text", "Output format: text, json")

	infoCmd.Usage = func() {
		fmt.Println("Usage: blunderdb info [options]")
		fmt.Println()
		fmt.Println("Display database metadata and statistics.")
		fmt.Println()
		fmt.Println("Options:")
		infoCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Display database info")
		fmt.Println("  blunderdb info --db database.db")
		fmt.Println()
		fmt.Println("  # Output as JSON")
		fmt.Println("  blunderdb info --db database.db --format json")
	}

	if err := infoCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		infoCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	// Get metadata
	metadata, err := cli.db.LoadMetadata()
	if err != nil {
		metadata = make(map[string]string)
	}

	// Get stats
	stats, err := cli.db.GetDatabaseStats()
	if err != nil {
		return fmt.Errorf("failed to get database stats: %v", err)
	}

	// Format output
	if strings.ToLower(*format) == "json" {
		output := map[string]interface{}{
			"path":     *dbPath,
			"metadata": metadata,
			"stats":    stats,
		}
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format JSON: %v", err)
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Println("Database Information")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("Path: %s\n\n", *dbPath)

		fmt.Println("Metadata:")
		if v, ok := metadata["database_version"]; ok {
			fmt.Printf("  Version: %s\n", v)
		}
		if v, ok := metadata["user"]; ok && v != "" {
			fmt.Printf("  User: %s\n", v)
		}
		if v, ok := metadata["description"]; ok && v != "" {
			fmt.Printf("  Description: %s\n", v)
		}
		if v, ok := metadata["dateOfCreation"]; ok && v != "" {
			fmt.Printf("  Date of Creation: %s\n", v)
		}

		fmt.Println("\nStatistics:")
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
	}

	return nil
}
