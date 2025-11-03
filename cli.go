package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CLI represents the command-line interface
type CLI struct {
	db  *Database
	cfg *Config
}

// NewCLI creates a new CLI instance
func NewCLI() *CLI {
	return &CLI{
		db:  NewDatabase(),
		cfg: NewConfig(),
	}
}

// Run executes the CLI
func (cli *CLI) Run(args []string) error {
	if len(args) < 1 {
		cli.printUsage()
		return nil
	}

	// Parse the command
	command := args[0]
	commandArgs := args[1:]

	switch command {
	case "import":
		return cli.runImport(commandArgs)
	case "export":
		return cli.runExport(commandArgs)
	case "list":
		return cli.runList(commandArgs)
	case "delete":
		return cli.runDelete(commandArgs)
	case "help":
		cli.printUsage()
		return nil
	case "version":
		cli.printVersion()
		return nil
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		cli.printUsage()
		return fmt.Errorf("unknown command: %s", command)
	}
}

// printUsage prints the usage information
func (cli *CLI) printUsage() {
	fmt.Println("blunderDB CLI - Backgammon Database Management Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  blunderdb <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  import    Import data into the database")
	fmt.Println("  export    Export data from the database")
	fmt.Println("  list      List database contents")
	fmt.Println("  delete    Delete data from the database")
	fmt.Println("  help      Show this help message")
	fmt.Println("  version   Show version information")
	fmt.Println()
	fmt.Println("Use 'blunderdb <command> --help' for more information about a command.")
}

// printVersion prints version information
func (cli *CLI) printVersion() {
	fmt.Printf("blunderDB version %s\n", DatabaseVersion)
}

// runImport handles the import command
func (cli *CLI) runImport(args []string) error {
	importCmd := flag.NewFlagSet("import", flag.ExitOnError)

	// Define flags
	dbPath := importCmd.String("db", "", "Path to the database file (required)")
	importType := importCmd.String("type", "", "Import type: match, position (required)")
	inputFile := importCmd.String("file", "", "Path to the file to import (required)")

	importCmd.Usage = func() {
		fmt.Println("Usage: blunderdb import [options]")
		fmt.Println()
		fmt.Println("Import data into the database.")
		fmt.Println()
		fmt.Println("Options:")
		importCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Import XG match file")
		fmt.Println("  blunderdb import --db database.db --type match --file match.xg")
		fmt.Println()
		fmt.Println("  # Import position file")
		fmt.Println("  blunderdb import --db database.db --type position --file positions.txt")
	}

	if err := importCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		fmt.Println("Error: --db flag is required")
		importCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	if *importType == "" {
		fmt.Println("Error: --type flag is required")
		importCmd.Usage()
		return fmt.Errorf("missing required flag: --type")
	}

	if *inputFile == "" {
		fmt.Println("Error: --file flag is required")
		importCmd.Usage()
		return fmt.Errorf("missing required flag: --file")
	}

	// Verify input file exists
	if _, err := os.Stat(*inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", *inputFile)
	}

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	// Perform import based on type
	switch strings.ToLower(*importType) {
	case "match":
		return cli.importMatch(*inputFile)
	case "position":
		return cli.importPosition(*inputFile)
	default:
		return fmt.Errorf("unknown import type: %s (must be 'match' or 'position')", *importType)
	}
}

// runExport handles the export command
func (cli *CLI) runExport(args []string) error {
	exportCmd := flag.NewFlagSet("export", flag.ExitOnError)

	// Define flags
	dbPath := exportCmd.String("db", "", "Path to the database file (required)")
	exportType := exportCmd.String("type", "", "Export type: database, positions (required)")
	outputFile := exportCmd.String("file", "", "Path to the output file (required)")

	exportCmd.Usage = func() {
		fmt.Println("Usage: blunderdb export [options]")
		fmt.Println()
		fmt.Println("Export data from the database.")
		fmt.Println()
		fmt.Println("Options:")
		exportCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Export entire database")
		fmt.Println("  blunderdb export --db database.db --type database --file export.db")
		fmt.Println()
		fmt.Println("  # Export positions to text file")
		fmt.Println("  blunderdb export --db database.db --type positions --file positions.txt")
	}

	if err := exportCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		fmt.Println("Error: --db flag is required")
		exportCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	if *exportType == "" {
		fmt.Println("Error: --type flag is required")
		exportCmd.Usage()
		return fmt.Errorf("missing required flag: --type")
	}

	if *outputFile == "" {
		fmt.Println("Error: --file flag is required")
		exportCmd.Usage()
		return fmt.Errorf("missing required flag: --file")
	}

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	// Perform export based on type
	switch strings.ToLower(*exportType) {
	case "database":
		return cli.exportDatabase(*outputFile)
	case "positions":
		return cli.exportPositions(*outputFile)
	default:
		return fmt.Errorf("unknown export type: %s (must be 'database' or 'positions')", *exportType)
	}
}

// runList handles the list command
func (cli *CLI) runList(args []string) error {
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)

	// Define flags
	dbPath := listCmd.String("db", "", "Path to the database file (required)")
	listType := listCmd.String("type", "", "List type: matches, positions, stats (required)")
	limit := listCmd.Int("limit", 10, "Maximum number of items to list")

	listCmd.Usage = func() {
		fmt.Println("Usage: blunderdb list [options]")
		fmt.Println()
		fmt.Println("List database contents.")
		fmt.Println()
		fmt.Println("Options:")
		listCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # List all matches")
		fmt.Println("  blunderdb list --db database.db --type matches")
		fmt.Println()
		fmt.Println("  # List first 20 positions")
		fmt.Println("  blunderdb list --db database.db --type positions --limit 20")
		fmt.Println()
		fmt.Println("  # Show database statistics")
		fmt.Println("  blunderdb list --db database.db --type stats")
	}

	if err := listCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		fmt.Println("Error: --db flag is required")
		listCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	if *listType == "" {
		fmt.Println("Error: --type flag is required")
		listCmd.Usage()
		return fmt.Errorf("missing required flag: --type")
	}

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	// Perform listing based on type
	switch strings.ToLower(*listType) {
	case "matches":
		return cli.listMatches(*limit)
	case "positions":
		return cli.listPositions(*limit)
	case "stats":
		return cli.showStats()
	default:
		return fmt.Errorf("unknown list type: %s (must be 'matches', 'positions', or 'stats')", *listType)
	}
}

// runDelete handles the delete command
func (cli *CLI) runDelete(args []string) error {
	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)

	// Define flags
	dbPath := deleteCmd.String("db", "", "Path to the database file (required)")
	deleteType := deleteCmd.String("type", "", "Delete type: match (required)")
	id := deleteCmd.Int64("id", 0, "ID of the item to delete (required)")
	confirm := deleteCmd.Bool("confirm", false, "Confirm deletion without prompting")

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
		fmt.Println("Error: --db flag is required")
		deleteCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	if *deleteType == "" {
		fmt.Println("Error: --type flag is required")
		deleteCmd.Usage()
		return fmt.Errorf("missing required flag: --type")
	}

	if *id == 0 {
		fmt.Println("Error: --id flag is required")
		deleteCmd.Usage()
		return fmt.Errorf("missing required flag: --id")
	}

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	// Perform deletion based on type
	switch strings.ToLower(*deleteType) {
	case "match":
		return cli.deleteMatch(*id, *confirm)
	default:
		return fmt.Errorf("unknown delete type: %s (must be 'match')", *deleteType)
	}
}

// initDatabase initializes the database connection
func (cli *CLI) initDatabase(dbPath string) error {
	// Check if database file exists
	fileExists := true
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fileExists = false
		fmt.Printf("Database file does not exist, creating new database: %s\n", dbPath)
		// Ensure directory exists
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	}

	// For new databases, use SetupDatabase to create the schema
	// For existing databases, use OpenDatabase
	var err error
	if !fileExists {
		err = cli.db.SetupDatabase(dbPath)
	} else {
		err = cli.db.OpenDatabase(dbPath)
	}

	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	fmt.Printf("Connected to database: %s\n", dbPath)
	return nil
}

// importMatch imports an XG match file
func (cli *CLI) importMatch(filePath string) error {
	fmt.Printf("Importing match from: %s\n", filePath)

	// Verify file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".xg" {
		return fmt.Errorf("invalid file type: %s (expected .xg)", ext)
	}

	// Import the match
	matchID, err := cli.db.ImportXGMatch(filePath)
	if err != nil {
		return fmt.Errorf("failed to import match: %v", err)
	}

	fmt.Printf("Successfully imported match (ID: %d)\n", matchID)

	// Display match details
	match, err := cli.db.GetMatchByID(matchID)
	if err == nil && match != nil {
		fmt.Println("\nMatch Details:")
		fmt.Printf("  Players: %s vs %s\n", match.Player1Name, match.Player2Name)
		if match.Event != "" {
			fmt.Printf("  Event: %s\n", match.Event)
		}
		if match.Location != "" {
			fmt.Printf("  Location: %s\n", match.Location)
		}
		fmt.Printf("  Match Length: %d\n", match.MatchLength)
		fmt.Printf("  Games: %d\n", match.GameCount)
	}

	return nil
}

// importPosition imports a position file
func (cli *CLI) importPosition(filePath string) error {
	fmt.Printf("Importing positions from: %s\n", filePath)

	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// Parse positions (assuming position JSON format, one per line)
	lines := strings.Split(string(content), "\n")
	imported := 0
	errors := 0

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Try to parse as position JSON
		var pos Position
		if err := json.Unmarshal([]byte(line), &pos); err != nil {
			fmt.Printf("Error parsing line %d: %v\n", i+1, err)
			errors++
			continue
		}

		// Save position
		_, err := cli.db.SavePosition(&pos)
		if err != nil {
			fmt.Printf("Error importing line %d: %v\n", i+1, err)
			errors++
			continue
		}
		imported++
	}

	fmt.Printf("Successfully imported %d positions\n", imported)
	if errors > 0 {
		fmt.Printf("Failed to import %d positions\n", errors)
	}

	return nil
}

// exportDatabase exports the entire database
func (cli *CLI) exportDatabase(outputFile string) error {
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

	// Export with all data
	err = cli.db.ExportDatabase(outputFile, positions, metadata, true, true, true)
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

// listMatches lists all matches in the database
func (cli *CLI) listMatches(limit int) error {
	matches, err := cli.db.GetAllMatches()
	if err != nil {
		return fmt.Errorf("failed to get matches: %v", err)
	}

	if len(matches) == 0 {
		fmt.Println("No matches found in database")
		return nil
	}

	fmt.Printf("Found %d match(es):\n\n", len(matches))

	displayCount := len(matches)
	if limit > 0 && limit < len(matches) {
		displayCount = limit
	}

	for i := 0; i < displayCount; i++ {
		match := matches[i]
		fmt.Printf("ID: %d\n", match.ID)
		fmt.Printf("  Players: %s vs %s\n", match.Player1Name, match.Player2Name)
		if match.Event != "" {
			fmt.Printf("  Event: %s\n", match.Event)
		}
		if match.Location != "" {
			fmt.Printf("  Location: %s\n", match.Location)
		}
		fmt.Printf("  Match Length: %d\n", match.MatchLength)
		fmt.Printf("  Games: %d\n", match.GameCount)
		fmt.Printf("  Imported: %s\n", match.ImportDate.Format("2006-01-02 15:04:05"))
		if match.FilePath != "" {
			fmt.Printf("  File: %s\n", match.FilePath)
		}
		fmt.Println()
	}

	if limit > 0 && len(matches) > limit {
		fmt.Printf("(Showing %d of %d matches, use --limit to see more)\n", displayCount, len(matches))
	}

	return nil
}

// listPositions lists positions in the database
func (cli *CLI) listPositions(limit int) error {
	positions, err := cli.db.LoadAllPositions()
	if err != nil {
		return fmt.Errorf("failed to get positions: %v", err)
	}

	if len(positions) == 0 {
		fmt.Println("No positions found in database")
		return nil
	}

	fmt.Printf("Found %d position(s):\n\n", len(positions))

	displayCount := len(positions)
	if limit > 0 && limit < len(positions) {
		displayCount = limit
	}

	for i := 0; i < displayCount; i++ {
		pos := positions[i]

		fmt.Printf("ID: %d\n", pos.ID)
		fmt.Printf("  Score: %d-%d\n", pos.Score[0], pos.Score[1])
		fmt.Printf("  Player on roll: %d\n", pos.PlayerOnRoll)
		if pos.DecisionType == CheckerAction {
			fmt.Printf("  Decision: Checker play\n")
		} else {
			fmt.Printf("  Decision: Cube action\n")
		}
		fmt.Println()
	}

	if limit > 0 && len(positions) > limit {
		fmt.Printf("(Showing %d of %d positions, use --limit to see more)\n", displayCount, len(positions))
	}

	return nil
}

// showStats displays database statistics
func (cli *CLI) showStats() error {
	stats, err := cli.db.GetDatabaseStats()
	if err != nil {
		return fmt.Errorf("failed to get database stats: %v", err)
	}

	fmt.Println("Database Statistics:")
	fmt.Println()

	// Cast stats to appropriate types and display
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

	return nil
}

// deleteMatch deletes a match from the database
func (cli *CLI) deleteMatch(matchID int64, confirm bool) error {
	// Get match details first
	match, err := cli.db.GetMatchByID(matchID)
	if err != nil {
		return fmt.Errorf("failed to get match: %v", err)
	}
	if match == nil {
		return fmt.Errorf("match with ID %d not found", matchID)
	}

	// Show match details
	fmt.Printf("Match ID: %d\n", match.ID)
	fmt.Printf("  Players: %s vs %s\n", match.Player1Name, match.Player2Name)
	if match.Event != "" {
		fmt.Printf("  Event: %s\n", match.Event)
	}
	fmt.Printf("  Games: %d\n", match.GameCount)
	fmt.Println()

	// Confirm deletion
	if !confirm {
		fmt.Print("Are you sure you want to delete this match? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "yes" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	// Delete the match
	err = cli.db.DeleteMatch(matchID)
	if err != nil {
		return fmt.Errorf("failed to delete match: %v", err)
	}

	fmt.Printf("Successfully deleted match ID %d\n", matchID)
	return nil
}
