package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	fmt.Println("  create    Create a new database")
	fmt.Println("  import    Import data into the database")
	fmt.Println("  export    Export data from the database")
	fmt.Println("  list      List database contents")
	fmt.Println("  match     Display match positions and analysis")
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

// runCreate handles the create command
func (cli *CLI) runCreate(args []string) error {
	createCmd := flag.NewFlagSet("create", flag.ExitOnError)

	// Define flags
	dbPath := createCmd.String("db", "", "Path to the database file to create (required)")
	force := createCmd.Bool("force", false, "Overwrite existing database if it exists")

	createCmd.Usage = func() {
		fmt.Println("Usage: blunderdb create [options]")
		fmt.Println()
		fmt.Println("Create a new database with the required schema.")
		fmt.Println()
		fmt.Println("Options:")
		createCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Create a new database")
		fmt.Println("  blunderdb create --db mydb.db")
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

	fmt.Printf("Successfully created database with schema version %s\n", DatabaseVersion)

	// Show database info
	stats, err := cli.db.GetDatabaseStats()
	if err == nil {
		fmt.Println("\nDatabase initialized with:")
		if posCount, ok := stats["position_count"].(int64); ok {
			fmt.Printf("  Positions: %d\n", posCount)
		}
		if matchCount, ok := stats["match_count"].(int64); ok {
			fmt.Printf("  Matches: %d\n", matchCount)
		}
	}

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
