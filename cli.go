package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
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
	case "create":
		return cli.runCreate(commandArgs)
	case "import":
		return cli.runImport(commandArgs)
	case "export":
		return cli.runExport(commandArgs)
	case "list":
		return cli.runList(commandArgs)
	case "delete":
		return cli.runDelete(commandArgs)
	case "match":
		return cli.runMatch(commandArgs)
	case "verify":
		return cli.runVerify(commandArgs)
	case "info":
		return cli.runInfo(commandArgs)
	case "edit":
		return cli.runEdit(commandArgs)
	case "search":
		return cli.runSearch(commandArgs)
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
	fmt.Println("  create    Create a new database with optional metadata")
	fmt.Println("  import    Import data into the database (match, position, batch)")
	fmt.Println("  export    Export data from the database")
	fmt.Println("  list      List database contents")
	fmt.Println("  search    Search positions with filters")
	fmt.Println("  match     Display match positions and analysis")
	fmt.Println("  info      Display database metadata")
	fmt.Println("  edit      Edit database metadata")
	fmt.Println("  verify    Verify database integrity")
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
	importType := importCmd.String("type", "", "Import type: match, position, batch (required)")
	inputFile := importCmd.String("file", "", "Path to the file to import (for match/position)")
	inputDir := importCmd.String("dir", "", "Path to directory for batch import (for batch)")
	recursive := importCmd.Bool("recursive", true, "Recursively scan subdirectories for batch import")

	importCmd.Usage = func() {
		fmt.Println("Usage: blunderdb import [options]")
		fmt.Println()
		fmt.Println("Import data into the database.")
		fmt.Println()
		fmt.Println("Options:")
		importCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Import Types:")
		fmt.Println("  match     Import a single XG match file (.xg)")
		fmt.Println("  position  Import positions from a text file")
		fmt.Println("  batch     Batch import all .xg files from a directory")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Import XG match file")
		fmt.Println("  blunderdb import --db database.db --type match --file match.xg")
		fmt.Println()
		fmt.Println("  # Import position file")
		fmt.Println("  blunderdb import --db database.db --type position --file positions.txt")
		fmt.Println()
		fmt.Println("  # Batch import all .xg files from a directory (recursive)")
		fmt.Println("  blunderdb import --db database.db --type batch --dir ./matches/")
		fmt.Println()
		fmt.Println("  # Batch import (non-recursive)")
		fmt.Println("  blunderdb import --db database.db --type batch --dir ./matches/ --recursive=false")
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

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	// Perform import based on type
	switch strings.ToLower(*importType) {
	case "match":
		if *inputFile == "" {
			fmt.Println("Error: --file flag is required for match import")
			importCmd.Usage()
			return fmt.Errorf("missing required flag: --file")
		}
		// Verify input file exists
		if _, err := os.Stat(*inputFile); os.IsNotExist(err) {
			return fmt.Errorf("input file does not exist: %s", *inputFile)
		}
		return cli.importMatch(*inputFile)
	case "position":
		if *inputFile == "" {
			fmt.Println("Error: --file flag is required for position import")
			importCmd.Usage()
			return fmt.Errorf("missing required flag: --file")
		}
		// Verify input file exists
		if _, err := os.Stat(*inputFile); os.IsNotExist(err) {
			return fmt.Errorf("input file does not exist: %s", *inputFile)
		}
		return cli.importPosition(*inputFile)
	case "batch":
		if *inputDir == "" {
			fmt.Println("Error: --dir flag is required for batch import")
			importCmd.Usage()
			return fmt.Errorf("missing required flag: --dir")
		}
		// Verify directory exists
		if info, err := os.Stat(*inputDir); os.IsNotExist(err) || !info.IsDir() {
			return fmt.Errorf("directory does not exist or is not a directory: %s", *inputDir)
		}
		return cli.importBatch(*inputDir, *recursive)
	default:
		return fmt.Errorf("unknown import type: %s (must be 'match', 'position', or 'batch')", *importType)
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
		// Check if this is a duplicate match error
		if errors.Is(err, ErrDuplicateMatch) {
			return fmt.Errorf("this match has already been imported to the database")
		}
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

	// Export with all data (including played moves)
	err = cli.db.ExportDatabase(outputFile, positions, metadata, true, true, true, true)
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

// runCreate handles the create command
func (cli *CLI) runCreate(args []string) error {
	createCmd := flag.NewFlagSet("create", flag.ExitOnError)

	// Define flags
	dbPath := createCmd.String("db", "", "Path to the database file to create (required)")
	force := createCmd.Bool("force", false, "Overwrite existing database if it exists")
	user := createCmd.String("user", "", "User name (owner of the database)")
	description := createCmd.String("description", "", "Description of the database")

	createCmd.Usage = func() {
		fmt.Println("Usage: blunderdb create [options]")
		fmt.Println()
		fmt.Println("Create a new database with the required schema and optional metadata.")
		fmt.Println()
		fmt.Println("Options:")
		createCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Create a new database")
		fmt.Println("  blunderdb create --db mydb.db")
		fmt.Println()
		fmt.Println("  # Create with metadata")
		fmt.Println("  blunderdb create --db mydb.db --user \"John Doe\" --description \"My backgammon positions\"")
		fmt.Println()
		fmt.Println("  # Force overwrite an existing database")
		fmt.Println("  blunderdb create --db mydb.db --force")
	}

	if err := createCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		fmt.Println("Error: --db flag is required")
		createCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	// Ensure .db extension is present
	if !strings.HasSuffix(strings.ToLower(*dbPath), ".db") {
		*dbPath += ".db"
	}

	// Check if database already exists
	if _, err := os.Stat(*dbPath); err == nil && !*force {
		return fmt.Errorf("database already exists: %s (use --force to overwrite)", *dbPath)
	}

	// Create directory if needed
	dir := filepath.Dir(*dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Create the database
	fmt.Printf("Creating database: %s\n", *dbPath)
	err := cli.db.SetupDatabase(*dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database: %v", err)
	}

	// Save metadata if provided
	metadata := make(map[string]string)
	if *user != "" {
		metadata["user"] = *user
	}
	if *description != "" {
		metadata["description"] = *description
	}
	metadata["dateOfCreation"] = time.Now().Format("2006-01-02 15:04:05")

	if len(metadata) > 0 {
		err = cli.db.SaveMetadata(metadata)
		if err != nil {
			return fmt.Errorf("failed to save metadata: %v", err)
		}
	}

	fmt.Printf("Successfully created database with schema version %s\n", DatabaseVersion)

	// Show database info
	fmt.Println("\nDatabase Information:")
	fmt.Printf("  Version: %s\n", DatabaseVersion)
	if *user != "" {
		fmt.Printf("  User: %s\n", *user)
	}
	if *description != "" {
		fmt.Printf("  Description: %s\n", *description)
	}
	fmt.Printf("  Created: %s\n", metadata["dateOfCreation"])

	return nil
}

// runMatch handles the match command
func (cli *CLI) runMatch(args []string) error {
	matchCmd := flag.NewFlagSet("match", flag.ExitOnError)

	// Define flags
	dbPath := matchCmd.String("db", "", "Path to the database file (required)")
	matchID := matchCmd.Int64("id", 0, "Match ID (required)")
	format := matchCmd.String("format", "json", "Output format: json, text, summary")
	output := matchCmd.String("output", "", "Output file (default: stdout)")

	matchCmd.Usage = func() {
		fmt.Println("Usage: blunderdb match [options]")
		fmt.Println()
		fmt.Println("Display match positions and analysis.")
		fmt.Println()
		fmt.Println("Options:")
		matchCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Display match positions in JSON format")
		fmt.Println("  blunderdb match --db database.db --id 1 --format json")
		fmt.Println()
		fmt.Println("  # Display match summary")
		fmt.Println("  blunderdb match --db database.db --id 1 --format summary")
		fmt.Println()
		fmt.Println("  # Save match positions to file")
		fmt.Println("  blunderdb match --db database.db --id 1 --output match.json")
	}

	if err := matchCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		fmt.Println("Error: --db flag is required")
		matchCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	if *matchID == 0 {
		fmt.Println("Error: --id flag is required")
		matchCmd.Usage()
		return fmt.Errorf("missing required flag: --id")
	}

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	// Get match info
	match, err := cli.db.GetMatchByID(*matchID)
	if err != nil {
		return fmt.Errorf("failed to get match: %v", err)
	}

	// Get match positions
	positions, err := cli.db.GetMatchMovePositions(*matchID)
	if err != nil {
		return fmt.Errorf("failed to get match positions: %v", err)
	}

	// Format output based on requested format
	var outputData string
	switch strings.ToLower(*format) {
	case "json":
		outputData, err = cli.formatMatchJSON(match, positions)
	case "text":
		outputData, err = cli.formatMatchText(match, positions)
	case "summary":
		outputData, err = cli.formatMatchSummary(match, positions)
	default:
		return fmt.Errorf("unknown format: %s (must be 'json', 'text', or 'summary')", *format)
	}

	if err != nil {
		return fmt.Errorf("failed to format output: %v", err)
	}

	// Output results
	if *output != "" {
		err := os.WriteFile(*output, []byte(outputData), 0644)
		if err != nil {
			return fmt.Errorf("failed to write output file: %v", err)
		}
		fmt.Printf("Match data written to: %s\n", *output)
	} else {
		fmt.Println(outputData)
	}

	return nil
}

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
		fmt.Println("Error: --db flag is required")
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

// formatMatchJSON formats match data as JSON
func (cli *CLI) formatMatchJSON(match *Match, positions []MatchMovePosition) (string, error) {
	output := map[string]interface{}{
		"match":          match,
		"positions":      positions,
		"position_count": len(positions),
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

// formatMatchText formats match data as text
func (cli *CLI) formatMatchText(match *Match, positions []MatchMovePosition) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Match ID: %d\n", match.ID))
	sb.WriteString(fmt.Sprintf("Players: %s vs %s\n", match.Player1Name, match.Player2Name))
	if match.Event != "" {
		sb.WriteString(fmt.Sprintf("Event: %s\n", match.Event))
	}
	if match.Location != "" {
		sb.WriteString(fmt.Sprintf("Location: %s\n", match.Location))
	}
	sb.WriteString(fmt.Sprintf("Match Length: %d\n", match.MatchLength))
	sb.WriteString(fmt.Sprintf("Total Positions: %d\n\n", len(positions)))

	for i, movePos := range positions {
		sb.WriteString(fmt.Sprintf("Position %d:\n", i+1))
		sb.WriteString(fmt.Sprintf("  Game: %d, Move: %d\n", movePos.GameNumber, movePos.MoveNumber))

		// Handle XG player encoding: -1 = Player1 (X), 1 = Player2 (O)
		var playerName string
		if movePos.PlayerOnRoll == -1 {
			playerName = match.Player1Name
		} else if movePos.PlayerOnRoll == 1 {
			playerName = match.Player2Name
		} else {
			playerName = "Unknown"
		}
		sb.WriteString(fmt.Sprintf("  Player on roll: %d (%s)\n", movePos.PlayerOnRoll, playerName))
		sb.WriteString(fmt.Sprintf("  Score: %d-%d\n", movePos.Position.Score[0], movePos.Position.Score[1]))
		sb.WriteString(fmt.Sprintf("  Cube: %d (owner: %d)\n", movePos.Position.Cube.Value, movePos.Position.Cube.Owner))
		if movePos.Position.Dice[0] != 0 {
			sb.WriteString(fmt.Sprintf("  Dice: %d-%d\n", movePos.Position.Dice[0], movePos.Position.Dice[1]))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// formatMatchSummary formats match data as a summary
func (cli *CLI) formatMatchSummary(match *Match, positions []MatchMovePosition) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Match: %s vs %s\n", match.Player1Name, match.Player2Name))
	if match.Event != "" {
		sb.WriteString(fmt.Sprintf("Event: %s\n", match.Event))
	}
	sb.WriteString(fmt.Sprintf("Match Length: %d points\n", match.MatchLength))
	sb.WriteString(fmt.Sprintf("Games: %d\n", match.GameCount))
	sb.WriteString(fmt.Sprintf("Total Positions: %d\n\n", len(positions)))

	// Count positions by game
	gamePositions := make(map[int32]int)
	for _, pos := range positions {
		gamePositions[pos.GameNumber]++
	}

	sb.WriteString("Positions per game:\n")
	for gameNum := int32(1); gameNum <= int32(match.GameCount); gameNum++ {
		count := gamePositions[gameNum]
		sb.WriteString(fmt.Sprintf("  Game %d: %d positions\n", gameNum, count))
	}

	return sb.String(), nil
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

// BatchImportResult represents the result of a single file import
type BatchImportResult struct {
	FilePath  string
	Success   bool
	MatchID   int64
	Error     string
	Player1   string
	Player2   string
	Games     int
	Positions int
}

// importBatch imports all .xg files from a directory
func (cli *CLI) importBatch(dirPath string, recursive bool) error {
	fmt.Printf("Batch importing from: %s (recursive: %v)\n\n", dirPath, recursive)

	// Find all .xg files
	var xgFiles []string

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories if not recursive (but always process root)
		if info.IsDir() {
			if !recursive && path != dirPath {
				return filepath.SkipDir
			}
			return nil
		}

		// Check for .xg extension
		if strings.ToLower(filepath.Ext(path)) == ".xg" {
			xgFiles = append(xgFiles, path)
		}

		return nil
	}

	err := filepath.Walk(dirPath, walkFunc)
	if err != nil {
		return fmt.Errorf("failed to scan directory: %v", err)
	}

	if len(xgFiles) == 0 {
		fmt.Println("No .xg files found in directory")
		return nil
	}

	fmt.Printf("Found %d .xg file(s) to import\n\n", len(xgFiles))

	// Import each file and collect results
	var results []BatchImportResult
	successCount := 0
	failCount := 0
	duplicateCount := 0
	totalPositions := 0

	for i, filePath := range xgFiles {
		relPath, _ := filepath.Rel(dirPath, filePath)
		fmt.Printf("[%d/%d] Importing: %s...", i+1, len(xgFiles), relPath)

		result := BatchImportResult{
			FilePath: relPath,
		}

		matchID, err := cli.db.ImportXGMatch(filePath)
		if err != nil {
			if errors.Is(err, ErrDuplicateMatch) {
				fmt.Println(" DUPLICATE")
				result.Error = "duplicate"
				duplicateCount++
			} else {
				fmt.Printf(" ERROR: %v\n", err)
				result.Error = err.Error()
				failCount++
			}
		} else {
			result.Success = true
			result.MatchID = matchID
			successCount++

			// Get match details
			match, err := cli.db.GetMatchByID(matchID)
			if err == nil && match != nil {
				result.Player1 = match.Player1Name
				result.Player2 = match.Player2Name
				result.Games = match.GameCount
			}

			// Get position count
			positions, err := cli.db.GetMatchMovePositions(matchID)
			if err == nil {
				result.Positions = len(positions)
				totalPositions += len(positions)
			}

			fmt.Printf(" OK (ID: %d, %d positions)\n", matchID, result.Positions)
		}

		results = append(results, result)
	}

	// Print summary table
	fmt.Println("\n" + strings.Repeat("=", 100))
	fmt.Println("IMPORT SUMMARY")
	fmt.Println(strings.Repeat("=", 100))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Status\tFile\tID\tPlayer 1\tPlayer 2\tGames\tPositions\tError")
	fmt.Fprintln(w, "------\t----\t--\t--------\t--------\t-----\t---------\t-----")

	for _, r := range results {
		status := "✗"
		if r.Success {
			status = "✓"
		} else if r.Error == "duplicate" {
			status = "⊘"
		}

		idStr := ""
		if r.MatchID > 0 {
			idStr = fmt.Sprintf("%d", r.MatchID)
		}

		errorStr := ""
		if !r.Success && r.Error != "duplicate" {
			errorStr = r.Error
			if len(errorStr) > 30 {
				errorStr = errorStr[:30] + "..."
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
			status, r.FilePath, idStr, r.Player1, r.Player2, r.Games, r.Positions, errorStr)
	}
	w.Flush()

	fmt.Println(strings.Repeat("-", 100))
	fmt.Printf("Total: %d files | Success: %d | Duplicates: %d | Failed: %d | Positions imported: %d\n",
		len(xgFiles), successCount, duplicateCount, failCount, totalPositions)

	return nil
}

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
		fmt.Println("Error: --db flag is required")
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

// runEdit handles the edit command
func (cli *CLI) runEdit(args []string) error {
	editCmd := flag.NewFlagSet("edit", flag.ExitOnError)

	// Define flags
	dbPath := editCmd.String("db", "", "Path to the database file (required)")
	user := editCmd.String("user", "", "Set user name")
	description := editCmd.String("description", "", "Set description")
	clearUser := editCmd.Bool("clear-user", false, "Clear user name")
	clearDescription := editCmd.Bool("clear-description", false, "Clear description")

	editCmd.Usage = func() {
		fmt.Println("Usage: blunderdb edit [options]")
		fmt.Println()
		fmt.Println("Edit database metadata.")
		fmt.Println()
		fmt.Println("Options:")
		editCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Set user name")
		fmt.Println("  blunderdb edit --db database.db --user \"John Doe\"")
		fmt.Println()
		fmt.Println("  # Set description")
		fmt.Println("  blunderdb edit --db database.db --description \"My positions collection\"")
		fmt.Println()
		fmt.Println("  # Clear user name")
		fmt.Println("  blunderdb edit --db database.db --clear-user")
		fmt.Println()
		fmt.Println("  # Set multiple values")
		fmt.Println("  blunderdb edit --db database.db --user \"John\" --description \"Tournament positions\"")
	}

	if err := editCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		fmt.Println("Error: --db flag is required")
		editCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	// Check that at least one edit option is provided
	if *user == "" && *description == "" && !*clearUser && !*clearDescription {
		fmt.Println("Error: at least one edit option is required")
		editCmd.Usage()
		return fmt.Errorf("no edit options provided")
	}

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	// Build metadata updates
	metadata := make(map[string]string)
	changes := []string{}

	if *clearUser {
		metadata["user"] = ""
		changes = append(changes, "Cleared user")
	} else if *user != "" {
		metadata["user"] = *user
		changes = append(changes, fmt.Sprintf("Set user to: %s", *user))
	}

	if *clearDescription {
		metadata["description"] = ""
		changes = append(changes, "Cleared description")
	} else if *description != "" {
		metadata["description"] = *description
		changes = append(changes, fmt.Sprintf("Set description to: %s", *description))
	}

	// Save metadata
	err := cli.db.SaveMetadata(metadata)
	if err != nil {
		return fmt.Errorf("failed to save metadata: %v", err)
	}

	fmt.Println("Database metadata updated:")
	for _, change := range changes {
		fmt.Printf("  - %s\n", change)
	}

	return nil
}

// runSearch handles the search command
func (cli *CLI) runSearch(args []string) error {
	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)

	// Define flags
	dbPath := searchCmd.String("db", "", "Path to the database file (required)")
	outputDB := searchCmd.String("export", "", "Export results to a new database file")
	limit := searchCmd.Int("limit", 0, "Maximum number of results (0 = no limit)")
	format := searchCmd.String("format", "table", "Output format: table, json, xgid")

	// Filter flags
	decisionType := searchCmd.String("decision", "", "Filter by decision type: checker, cube")
	pipMin := searchCmd.Int("pip-min", 0, "Minimum pip count difference")
	pipMax := searchCmd.Int("pip-max", 0, "Maximum pip count difference")
	winRateMin := searchCmd.Float64("winrate-min", 0, "Minimum win rate (%)")
	winRateMax := searchCmd.Float64("winrate-max", 0, "Maximum win rate (%)")
	cubeValue := searchCmd.Int("cube", 0, "Filter by cube value")
	score1 := searchCmd.Int("score1", -1, "Filter by player 1 score")
	score2 := searchCmd.Int("score2", -1, "Filter by player 2 score")
	matchLength := searchCmd.Int("match-length", 0, "Filter by match length")
	errorMin := searchCmd.Float64("error-min", 0, "Minimum equity error (blunders)")
	hasAnalysis := searchCmd.Bool("has-analysis", false, "Only positions with analysis")
	checkerOff1Min := searchCmd.Int("off1-min", 0, "Minimum checkers off for player 1")
	checkerOff2Min := searchCmd.Int("off2-min", 0, "Minimum checkers off for player 2")

	searchCmd.Usage = func() {
		fmt.Println("Usage: blunderdb search [options]")
		fmt.Println()
		fmt.Println("Search for positions in the database using filters.")
		fmt.Println()
		fmt.Println("Options:")
		searchCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # List all positions")
		fmt.Println("  blunderdb search --db database.db")
		fmt.Println()
		fmt.Println("  # Search cube decisions")
		fmt.Println("  blunderdb search --db database.db --decision cube")
		fmt.Println()
		fmt.Println("  # Search positions with errors >= 0.1")
		fmt.Println("  blunderdb search --db database.db --error-min 0.1")
		fmt.Println()
		fmt.Println("  # Search and export to new database")
		fmt.Println("  blunderdb search --db database.db --decision cube --export cubes.db")
		fmt.Println()
		fmt.Println("  # Search bearoff positions")
		fmt.Println("  blunderdb search --db database.db --off1-min 1 --off2-min 1")
		fmt.Println()
		fmt.Println("  # Output as JSON")
		fmt.Println("  blunderdb search --db database.db --format json --limit 10")
	}

	if err := searchCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		fmt.Println("Error: --db flag is required")
		searchCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	// Build filter parameters for LoadPositionsByFilters
	// Create a base filter position with EMPTY board (no checker position filtering)
	// This is different from InitializePosition() which sets up starting position
	filter := Position{
		Board:        Board{Points: [26]Point{}}, // Empty board - matches any position
		Cube:         Cube{None, 0},
		Dice:         [2]int{0, 0},
		Score:        [2]int{-1, -1}, // -1 means no score filter
		PlayerOnRoll: 0,
		DecisionType: CheckerAction,
	}

	// Set decision type filter
	decisionTypeFilter := false
	if *decisionType != "" {
		decisionTypeFilter = true
		switch strings.ToLower(*decisionType) {
		case "checker":
			filter.DecisionType = CheckerAction
		case "cube":
			filter.DecisionType = CubeAction
		default:
			return fmt.Errorf("invalid decision type: %s (must be 'checker' or 'cube')", *decisionType)
		}
	}

	// Build filter strings for the search function
	var pipCountFilter string
	if *pipMin > 0 || *pipMax > 0 {
		if *pipMin > 0 && *pipMax > 0 {
			pipCountFilter = fmt.Sprintf("p%d,%d", *pipMin, *pipMax)
		} else if *pipMin > 0 {
			pipCountFilter = fmt.Sprintf("p>%d", *pipMin)
		} else {
			pipCountFilter = fmt.Sprintf("p<%d", *pipMax)
		}
	}

	var winRateFilter string
	if *winRateMin > 0 || *winRateMax > 0 {
		if *winRateMin > 0 && *winRateMax > 0 {
			winRateFilter = fmt.Sprintf("w%f,%f", *winRateMin, *winRateMax)
		} else if *winRateMin > 0 {
			winRateFilter = fmt.Sprintf("w>%f", *winRateMin)
		} else {
			winRateFilter = fmt.Sprintf("w<%f", *winRateMax)
		}
	}

	var player1CheckerOffFilter string
	if *checkerOff1Min > 0 {
		player1CheckerOffFilter = fmt.Sprintf("o>%d", *checkerOff1Min-1)
	}

	var player2CheckerOffFilter string
	if *checkerOff2Min > 0 {
		player2CheckerOffFilter = fmt.Sprintf("O>%d", *checkerOff2Min-1)
	}

	// Set cube value filter
	includeCube := false
	if *cubeValue > 0 {
		includeCube = true
		filter.Cube.Value = *cubeValue
	}

	// Set score filter
	includeScore := false
	if *score1 >= 0 || *score2 >= 0 || *matchLength > 0 {
		includeScore = true
		if *score1 >= 0 {
			filter.Score[0] = *score1
		}
		if *score2 >= 0 {
			filter.Score[1] = *score2
		}
	}

	// For CLI, we apply decision type filter ourselves since db filter also checks PlayerOnRoll
	// Load positions without decision type filter, we'll apply it manually
	positions, err := cli.db.LoadPositionsByFilters(
		filter,
		includeCube,
		includeScore,
		pipCountFilter,
		winRateFilter,
		"", // gammonRateFilter
		"", // backgammonRateFilter
		"", // player2WinRateFilter
		"", // player2GammonRateFilter
		"", // player2BackgammonRateFilter
		player1CheckerOffFilter,
		player2CheckerOffFilter,
		"",    // player1BackCheckerFilter
		"",    // player2BackCheckerFilter
		"",    // player1CheckerInZoneFilter
		"",    // player2CheckerInZoneFilter
		"",    // searchText
		"",    // player1AbsolutePipCountFilter
		"",    // equityFilter
		false, // decisionTypeFilter - we'll filter ourselves
		false, // diceRollFilter
		"",    // movePatternFilter
		"",    // dateFilter
		"",    // player1OutfieldBlotFilter
		"",    // player2OutfieldBlotFilter
		"",    // player1JanBlotFilter
		"",    // player2JanBlotFilter
		false, // noContactFilter
		false, // mirrorFilter
	)
	if err != nil {
		return fmt.Errorf("failed to search positions: %v", err)
	}

	// Apply additional filters that aren't supported by LoadPositionsByFilters
	var filteredPositions []Position
	for _, pos := range positions {
		// Filter by decision type (CLI only compares DecisionType, not PlayerOnRoll)
		if decisionTypeFilter {
			if pos.DecisionType != filter.DecisionType {
				continue
			}
		}

		// Filter by error minimum if specified
		if *errorMin > 0 || *hasAnalysis {
			analysis, err := cli.db.LoadAnalysis(pos.ID)
			if err != nil || analysis == nil {
				if *hasAnalysis {
					continue
				}
			} else {
				if *errorMin > 0 {
					// Check if this position has an error >= errorMin
					hasError := false
					if analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 1 {
						if analysis.CheckerAnalysis.Moves[1].EquityError != nil {
							if *analysis.CheckerAnalysis.Moves[1].EquityError >= *errorMin {
								hasError = true
							}
						}
					}
					if analysis.DoublingCubeAnalysis != nil {
						if analysis.DoublingCubeAnalysis.CubefulNoDoubleError >= *errorMin ||
							analysis.DoublingCubeAnalysis.CubefulDoubleTakeError >= *errorMin ||
							analysis.DoublingCubeAnalysis.CubefulDoublePassError >= *errorMin {
							hasError = true
						}
					}
					if !hasError {
						continue
					}
				}
			}
		}

		filteredPositions = append(filteredPositions, pos)
	}

	// Apply limit
	if *limit > 0 && len(filteredPositions) > *limit {
		filteredPositions = filteredPositions[:*limit]
	}

	// Output results
	fmt.Printf("Found %d position(s)\n\n", len(filteredPositions))

	if len(filteredPositions) == 0 {
		return nil
	}

	// Format output
	switch strings.ToLower(*format) {
	case "json":
		type PositionResult struct {
			ID           int64   `json:"id"`
			XGID         string  `json:"xgid,omitempty"`
			Score        [2]int  `json:"score"`
			Cube         int     `json:"cube"`
			DecisionType string  `json:"decision_type"`
			Dice         [2]int  `json:"dice"`
			BestMove     string  `json:"best_move,omitempty"`
			Equity       float64 `json:"equity,omitempty"`
		}

		var results []PositionResult
		for _, pos := range filteredPositions {
			result := PositionResult{
				ID:    pos.ID,
				Score: pos.Score,
				Cube:  pos.Cube.Value,
				Dice:  pos.Dice,
			}

			if pos.DecisionType == CheckerAction {
				result.DecisionType = "checker"
			} else {
				result.DecisionType = "cube"
			}

			// Get analysis if available
			analysis, err := cli.db.LoadAnalysis(pos.ID)
			if err == nil && analysis != nil {
				result.XGID = analysis.XGID
				if analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
					result.BestMove = analysis.CheckerAnalysis.Moves[0].Move
					result.Equity = analysis.CheckerAnalysis.Moves[0].Equity
				}
			}

			results = append(results, result)
		}

		jsonData, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format JSON: %v", err)
		}
		fmt.Println(string(jsonData))

	case "xgid":
		for _, pos := range filteredPositions {
			analysis, err := cli.db.LoadAnalysis(pos.ID)
			if err == nil && analysis != nil && analysis.XGID != "" {
				fmt.Println(analysis.XGID)
			}
		}

	default: // table format
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tScore\tCube\tType\tDice\tBest Move\tEquity")
		fmt.Fprintln(w, "--\t-----\t----\t----\t----\t---------\t------")

		for _, pos := range filteredPositions {
			decType := "checker"
			if pos.DecisionType == CubeAction {
				decType = "cube"
			}

			diceStr := ""
			if pos.Dice[0] > 0 {
				diceStr = fmt.Sprintf("%d-%d", pos.Dice[0], pos.Dice[1])
			}

			bestMove := ""
			equityStr := ""

			// Get analysis if available
			analysis, err := cli.db.LoadAnalysis(pos.ID)
			if err == nil && analysis != nil {
				if analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
					bestMove = analysis.CheckerAnalysis.Moves[0].Move
					equityStr = fmt.Sprintf("%.3f", analysis.CheckerAnalysis.Moves[0].Equity)
				} else if analysis.DoublingCubeAnalysis != nil {
					bestMove = analysis.DoublingCubeAnalysis.BestCubeAction
					equityStr = fmt.Sprintf("%.3f", analysis.DoublingCubeAnalysis.CubefulNoDoubleEquity)
				}
			}

			fmt.Fprintf(w, "%d\t%d-%d\t%d\t%s\t%s\t%s\t%s\n",
				pos.ID, pos.Score[0], pos.Score[1], pos.Cube.Value, decType, diceStr, bestMove, equityStr)
		}
		w.Flush()
	}

	// Export to new database if requested
	if *outputDB != "" {
		fmt.Printf("\nExporting %d positions to: %s\n", len(filteredPositions), *outputDB)

		// Get metadata from source database
		metadata, _ := cli.db.LoadMetadata()
		metadata["description"] = fmt.Sprintf("Exported from search: %d positions", len(filteredPositions))
		metadata["dateOfCreation"] = time.Now().Format("2006-01-02 15:04:05")

		err = cli.db.ExportDatabase(*outputDB, filteredPositions, metadata, true, true, false, true)
		if err != nil {
			return fmt.Errorf("failed to export database: %v", err)
		}

		fmt.Println("Export completed successfully")
	}

	return nil
}

// SearchResult represents a position search result for display
type SearchResult struct {
	Position    Position
	Analysis    *PositionAnalysis
	XGID        string
	BestMove    string
	Equity      float64
	EquityError *float64
}

// getSearchResults loads positions with their analysis for display
func (cli *CLI) getSearchResults(positions []Position) []SearchResult {
	var results []SearchResult

	for _, pos := range positions {
		result := SearchResult{
			Position: pos,
		}

		analysis, err := cli.db.LoadAnalysis(pos.ID)
		if err == nil && analysis != nil {
			result.Analysis = analysis
			result.XGID = analysis.XGID

			if analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
				result.BestMove = analysis.CheckerAnalysis.Moves[0].Move
				result.Equity = analysis.CheckerAnalysis.Moves[0].Equity
			}
		}

		results = append(results, result)
	}

	// Sort by ID
	sort.Slice(results, func(i, j int) bool {
		return results[i].Position.ID < results[j].Position.ID
	})

	return results
}
