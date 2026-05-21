package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

// runExport handles the export command
func (cli *CLI) runExport(args []string) error {
	exportCmd := flag.NewFlagSet("export", flag.ExitOnError)

	// Define flags
	dbPath := exportCmd.String("db", "", "Path to the database file (required)")
	exportType := exportCmd.String("type", "", "Export type: database, positions, matches (required)")
	outputFile := exportCmd.String("file", "", "Path to the output file (required)")
	includeAnalysis := exportCmd.Bool("analysis", true, "Include analysis in database export (default: true)")
	includeComments := exportCmd.Bool("comments", true, "Include comments in database export (default: true)")
	includeFilterLibrary := exportCmd.Bool("filters", true, "Include filter library in database export (default: true)")
	includePlayedMoves := exportCmd.Bool("played-moves", true, "Include played moves in analysis (default: true)")
	includeMatches := exportCmd.Bool("matches", true, "Include matches in database export (default: true)")
	includeCollections := exportCmd.Bool("collections", false, "Include collections in database export (default: false)")
	collectionIDsStr := exportCmd.String("collection-ids", "", "Comma-separated list of collection IDs to export")
	matchIDsStr := exportCmd.String("match-ids", "", "Comma-separated list of match IDs to export (empty = all)")
	tournamentIDsStr := exportCmd.String("tournament-ids", "", "Comma-separated list of tournament IDs to export")

	exportCmd.Usage = func() {
		fmt.Println("Usage: blunderdb export [options]")
		fmt.Println()
		fmt.Println("Export data from the database.")
		fmt.Println()
		fmt.Println("Options:")
		exportCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Export Types:")
		fmt.Println("  database   Export entire database (positions, analysis, comments, matches)")
		fmt.Println("  positions  Export positions to text file (JSON format)")
		fmt.Println("  matches    Export only matches to a new database")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Export entire database with all matches")
		fmt.Println("  blunderdb export --db database.db --type database --file export.db")
		fmt.Println()
		fmt.Println("  # Export database without matches")
		fmt.Println("  blunderdb export --db database.db --type database --file export.db --matches=false")
		fmt.Println()
		fmt.Println("  # Export without analysis or played moves")
		fmt.Println("  blunderdb export --db database.db --type database --file export.db --analysis=false")
		fmt.Println()
		fmt.Println("  # Export with analysis but without played moves")
		fmt.Println("  blunderdb export --db database.db --type database --file export.db --played-moves=false")
		fmt.Println()
		fmt.Println("  # Export with specific collections")
		fmt.Println("  blunderdb export --db database.db --type database --file export.db --collections --collection-ids=1,2,3")
		fmt.Println()
		fmt.Println("  # Export with specific tournaments")
		fmt.Println("  blunderdb export --db database.db --type database --file export.db --tournament-ids=1,2")
		fmt.Println()
		fmt.Println("  # Export positions to text file")
		fmt.Println("  blunderdb export --db database.db --type positions --file positions.txt")
		fmt.Println()
		fmt.Println("  # Export only matches to a new database")
		fmt.Println("  blunderdb export --db database.db --type matches --file matches.db")
	}

	if err := exportCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		exportCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	if *exportType == "" {
		exportCmd.Usage()
		return fmt.Errorf("missing required flag: --type")
	}

	if *outputFile == "" {
		exportCmd.Usage()
		return fmt.Errorf("missing required flag: --file")
	}

	// Parse comma-separated ID lists
	var collectionIDs, matchIDs, tournamentIDs []int64
	if *collectionIDsStr != "" {
		var parseErr error
		collectionIDs, parseErr = parseIDList(*collectionIDsStr)
		if parseErr != nil {
			return fmt.Errorf("invalid collection-ids: %v", parseErr)
		}
	}
	if *matchIDsStr != "" {
		var parseErr error
		matchIDs, parseErr = parseIDList(*matchIDsStr)
		if parseErr != nil {
			return fmt.Errorf("invalid match-ids: %v", parseErr)
		}
	}
	if *tournamentIDsStr != "" {
		var parseErr error
		tournamentIDs, parseErr = parseIDList(*tournamentIDsStr)
		if parseErr != nil {
			return fmt.Errorf("invalid tournament-ids: %v", parseErr)
		}
	}

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	// Perform export based on type
	switch strings.ToLower(*exportType) {
	case "database":
		return cli.exportDatabaseWithOptions(*outputFile, *includeAnalysis, *includeComments,
			*includeFilterLibrary, *includePlayedMoves, *includeMatches,
			*includeCollections, collectionIDs, matchIDs, tournamentIDs)
	case "positions":
		return cli.exportPositions(*outputFile)
	case "matches":
		return cli.exportMatchesOnly(*outputFile)
	default:
		return fmt.Errorf("unknown export type: %s (must be 'database', 'positions', or 'matches')", *exportType)
	}
}

// exportDatabaseWithOptions exports the database with configurable options
func (cli *CLI) exportDatabaseWithOptions(outputFile string, includeAnalysis bool, includeComments bool,
	includeFilterLibrary bool, includePlayedMoves bool, includeMatches bool,
	includeCollections bool, collectionIDs []int64, matchIDs []int64, tournamentIDs []int64) error {
	fmt.Printf("Exporting database to: %s\n", outputFile)

	// Get all positions
	positions, err := cli.db.LoadAllPositions()
	if err != nil {
		return fmt.Errorf("failed to load positions: %v", err)
	}

	// Get metadata
	metadata := make(map[string]string)
	version, err := cli.db.GetDatabaseVersion()
	if err == nil {
		metadata["database_version"] = version
	}

	// Export with the specified options
	err = cli.db.ExportDatabase(ExportOptions{
		ExportPath:           outputFile,
		Positions:            positions,
		Metadata:             metadata,
		IncludeAnalysis:      includeAnalysis,
		IncludeComments:      includeComments,
		IncludeFilterLibrary: includeFilterLibrary,
		IncludePlayedMoves:   includePlayedMoves,
		IncludeMatches:       includeMatches,
		IncludeCollections:   includeCollections,
		CollectionIDs:        collectionIDs,
		MatchIDs:             matchIDs,
		TournamentIDs:        tournamentIDs,
	})
	if err != nil {
		return fmt.Errorf("failed to export database: %v", err)
	}

	// Get file size
	info, err := os.Stat(outputFile)
	if err == nil {
		fmt.Printf("Successfully exported database (%d bytes)\n", info.Size())
	} else {
		fmt.Println("Successfully exported database")
	}

	return nil
}

// exportMatchesOnly exports only the matches to a new database
func (cli *CLI) exportMatchesOnly(outputFile string) error {
	fmt.Printf("Exporting matches to: %s\n", outputFile)

	// Get matches count first
	matches, err := cli.db.GetAllMatches()
	if err != nil {
		return fmt.Errorf("failed to load matches: %v", err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("no matches found in database")
	}

	fmt.Printf("Found %d match(es) to export\n", len(matches))

	// Export with empty positions but with matches
	metadata := make(map[string]string)
	version, err := cli.db.GetDatabaseVersion()
	if err == nil {
		metadata["database_version"] = version
	}

	// We need to export positions that are linked to matches
	// Get all positions linked to moves in the matches
	positions, err := cli.db.LoadAllPositions()
	if err != nil {
		return fmt.Errorf("failed to load positions: %v", err)
	}

	// Export with positions, analysis, comments disabled, but matches enabled
	err = cli.db.ExportDatabase(ExportOptions{
		ExportPath:         outputFile,
		Positions:          positions,
		Metadata:           metadata,
		IncludeAnalysis:    true,
		IncludeComments:    true,
		IncludePlayedMoves: true,
		IncludeMatches:     true,
	})
	if err != nil {
		return fmt.Errorf("failed to export matches: %v", err)
	}

	// Get file size
	info, err := os.Stat(outputFile)
	if err == nil {
		fmt.Printf("Successfully exported matches (%d bytes)\n", info.Size())
	} else {
		fmt.Println("Successfully exported matches")
	}

	return nil
}

// exportPositions exports positions to a text file
func (cli *CLI) exportPositions(outputFile string) error {
	fmt.Printf("Exporting positions to: %s\n", outputFile)

	// Get all positions
	positions, err := cli.db.LoadAllPositions()
	if err != nil {
		return fmt.Errorf("failed to get positions: %v", err)
	}

	// Create output file
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	// Write positions as JSON, one per line
	for _, pos := range positions {
		posJSON, err := json.Marshal(pos)
		if err != nil {
			continue
		}
		fmt.Fprintf(file, "%s\n", string(posJSON))
	}

	fmt.Printf("Successfully exported %d positions\n", len(positions))
	return nil
}
