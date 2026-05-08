package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

// parseIDList parses a comma-separated string of int64 IDs.
func parseIDList(s string) ([]int64, error) {
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid ID %q: %v", p, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

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
		fmt.Println("  match     Import a single match file (.xg, .sgf, .mat, .txt, .bgf) or XGP position (.xgp)")
		fmt.Println("  position  Import positions from a text file")
		fmt.Println("  batch     Batch import all match/position files from a directory")
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
		importCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	if *importType == "" {
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

// runList handles the list command
func (cli *CLI) runList(args []string) error {
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)

	// Define flags
	dbPath := listCmd.String("db", "", "Path to the database file (required)")
	listType := listCmd.String("type", "", "List type: matches, positions, stats (required)")
	limit := listCmd.Int("limit", 10, "Maximum number of items to list")

	// Stats-specific flags (only used when --type stats)
	statsMetric := listCmd.String("metric", "pr", "Metric to display: pr or mwc (stats only)")
	statsPlayer := listCmd.String("player", "", "Filter by player name (stats only)")
	statsTournament := listCmd.String("tournament", "", "Filter by tournament IDs, comma-separated (stats only)")
	statsFrom := listCmd.String("from", "", "Start date filter YYYY-MM-DD (stats only)")
	statsTo := listCmd.String("to", "", "End date filter YYYY-MM-DD (stats only)")
	statsDecisionType := listCmd.String("decision-type", "all", "Decision type: all, checker, or cube (stats only)")
	statsTopBlunders := listCmd.Int("top-blunders", 10, "Number of top blunders to show (stats only)")
	statsFormat := listCmd.String("format", "text", "Output format: text or json (stats only)")

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
		fmt.Println()
		fmt.Println("  # Show stats as JSON")
		fmt.Println("  blunderdb list --db database.db --type stats --format json")
		fmt.Println()
		fmt.Println("  # Show stats in MWC with player filter")
		fmt.Println("  blunderdb list --db database.db --type stats --metric mwc --player \"Alice\"")
	}

	if err := listCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		listCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	if *listType == "" {
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
		// Build StatsFilter from flags
		filter := StatsFilter{
			PlayerName:   *statsPlayer,
			DateFrom:     *statsFrom,
			DateTo:       *statsTo,
			DecisionType: -1, // default: all
		}
		switch strings.ToLower(*statsDecisionType) {
		case "checker":
			filter.DecisionType = 0
		case "cube":
			filter.DecisionType = 1
		}
		if *statsTournament != "" {
			ids, err := parseIDList(*statsTournament)
			if err != nil {
				return fmt.Errorf("invalid --tournament: %v", err)
			}
			filter.TournamentIDs = ids
		}
		return cli.showStats(filter, *statsMetric, *statsFormat, *statsTopBlunders)
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

// importMatch imports a match file (XG, SGF, MAT, TXT, BGF) or XGP position file
func (cli *CLI) importMatch(filePath string) error {
	fmt.Printf("Importing match from: %s\n", filePath)

	// Verify file extension and route to appropriate importer
	ext := strings.ToLower(filepath.Ext(filePath))
	var matchID int64
	var err error

	switch ext {
	case ".xgp":
		// XGP files are single-position files, not match files
		posID, posErr := cli.db.ImportXGPPosition(filePath)
		if posErr != nil {
			return fmt.Errorf("failed to import XGP position: %v", posErr)
		}
		fmt.Printf("Successfully imported XGP position (ID: %d)\n", posID)
		return nil
	case ".xg":
		matchID, err = cli.db.ImportXGMatch(filePath)
	case ".sgf", ".mat", ".txt":
		matchID, err = cli.db.ImportGnuBGMatch(filePath)
	case ".bgf":
		matchID, err = cli.db.ImportBGFMatch(filePath)
	default:
		return fmt.Errorf("invalid file type: %s (expected .xg, .xgp, .sgf, .mat, .txt, or .bgf)", ext)
	}

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

	// Check if this is a BGBlitz position text file
	// BGBlitz text files contain "Position-ID:" which is unique to BGBlitz format
	contentStr := string(content)
	if strings.Contains(contentStr, "Position-ID:") {
		// BGBlitz position text file
		posID, err := cli.db.ImportBGFPosition(filePath)
		if err != nil {
			return fmt.Errorf("failed to import BGBlitz position: %v", err)
		}
		fmt.Printf("Successfully imported BGBlitz position (ID: %d)\n", posID)
		return nil
	}

	// Parse positions (assuming position JSON format, one per line)
	lines := strings.Split(contentStr, "\n")
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
			slog.Warn("parsing line", "line", i+1, "err", err)
			errors++
			continue
		}

		// Save position
		_, err := cli.db.SavePosition(&pos)
		if err != nil {
			slog.Warn("importing line", "line", i+1, "err", err)
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

// showStats displays database statistics using ComputeStats.
//
// metric is "pr" or "mwc", format is "text" or "json", topN is the number of
// top blunders to display (only relevant for text format, JSON always includes
// the full TopBlunders slice).
func (cli *CLI) showStats(filter StatsFilter, metric, format string, topN int) error {
	result, err := cli.db.ComputeStats(filter)
	if err != nil {
		return fmt.Errorf("failed to compute stats: %v", err)
	}

	if strings.ToLower(format) == "json" {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal stats: %v", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// ── Text format ──────────────────────────────────────────────────────────
	useMWC := strings.ToLower(metric) == "mwc"
	metricLabel := "PR"
	if useMWC {
		metricLabel = "MWC"
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// 1. Header
	fmt.Fprintln(w, "=== blunderDB Statistics ===")
	if filter.PlayerName != "" {
		fmt.Fprintf(w, "Player:\t%s\n", filter.PlayerName)
	}
	if filter.DateFrom != "" || filter.DateTo != "" {
		from, to := filter.DateFrom, filter.DateTo
		if from == "" {
			from = "—"
		}
		if to == "" {
			to = "—"
		}
		fmt.Fprintf(w, "Date range:\t%s → %s\n", from, to)
	}
	switch filter.DecisionType {
	case 0:
		fmt.Fprintln(w, "Decision type:\tchecker only")
	case 1:
		fmt.Fprintln(w, "Decision type:\tcube only")
	}
	fmt.Fprintf(w, "Metric:\t%s\n", metricLabel)
	w.Flush()
	fmt.Println()

	// 2. Totals
	fmt.Println("── Totals ──")
	fmt.Fprintf(w, "  Positions:\t%d\n", result.Totals.NumPositions)
	fmt.Fprintf(w, "  Matches:\t%d\n", result.Totals.NumMatches)
	fmt.Fprintf(w, "  Tournaments:\t%d\n", result.Totals.NumTournaments)
	fmt.Fprintf(w, "  Decisions:\t%d\n", result.Totals.NumDecisions)
	w.Flush()
	fmt.Println()

	// 3. PR / MWC global
	fmt.Printf("── %s ──\n", metricLabel)
	if useMWC {
		mwcStr := func(v float64) string {
			if !result.MWCAvailable {
				return "—"
			}
			return fmt.Sprintf("%.4f", v)
		}
		fmt.Fprintf(w, "  Global:\t%s\n", mwcStr(result.MWCGlobal))
		fmt.Fprintf(w, "  Checker:\t%s\n", mwcStr(result.MWCChecker))
		fmt.Fprintf(w, "  Cube:\t%s\n", mwcStr(result.MWCCube))
	} else {
		fmt.Fprintf(w, "  Global:\t%.3f\n", result.PRGlobal)
		fmt.Fprintf(w, "  Checker:\t%.3f\n", result.PRChecker)
		fmt.Fprintf(w, "  Cube:\t%.3f\n", result.PRCube)
		fmt.Fprintf(w, "  Snowie ER:\t%.3f\n", result.SnowieGlobal)
	}
	w.Flush()
	fmt.Println()

	// 4. Rolling
	rollingNs := []int{5, 10, 50, 100, 250, 500, 1000}
	fmt.Printf("── Rolling %s ──\n", metricLabel)
	fmt.Fprintln(w, "  N\tDecisions used\tValue")
	fmt.Fprintln(w, "  —\t——————————————\t—————")
	for _, n := range rollingNs {
		var val string
		if useMWC {
			if v, ok := result.MWCRolling[n]; ok {
				if result.MWCAvailable {
					val = fmt.Sprintf("%.4f", v)
				} else {
					val = "—"
				}
			} else {
				val = "n/a"
			}
		} else {
			if v, ok := result.PRRolling[n]; ok {
				val = fmt.Sprintf("%.3f", v)
			} else {
				val = "n/a"
			}
		}
		actualN := n
		if actualN > result.Totals.NumDecisions {
			actualN = result.Totals.NumDecisions
		}
		fmt.Fprintf(w, "  %d\t%d\t%s\n", n, actualN, val)
	}
	w.Flush()
	fmt.Println()

	// 5. Top blunders
	fmt.Printf("── Top %d Blunders ──\n", topN)
	fmt.Fprintln(w, "  Pos ID\tType\tError (EMG)\tMWC Loss\tDate\tPlayers")
	fmt.Fprintln(w, "  ——————\t————\t———————————\t————————\t————\t———————")
	limit := topN
	if len(result.TopBlunders) < limit {
		limit = len(result.TopBlunders)
	}
	for _, b := range result.TopBlunders[:limit] {
		dt := "checker"
		if b.DecisionType == 1 {
			dt = "cube"
		}
		errEMG := fmt.Sprintf("%.3f", float64(b.ErrorMP)/1000)
		mwcStr := "—"
		if result.MWCAvailable && b.MWCLoss != 0 {
			mwcStr = fmt.Sprintf("%.4f", b.MWCLoss)
		}
		date := b.MatchDate
		if date == "" {
			date = "—"
		}
		fmt.Fprintf(w, "  %d\t%s\t%s\t%s\t%s\t%s\n",
			b.PositionID, dt, errEMG, mwcStr, date, b.PlayerNames)
	}
	w.Flush()
	fmt.Println()

	// 6. Cube action breakdown
	if len(result.CubeActionBreakdown) > 0 {
		fmt.Println("── Cube Action Breakdown ──")
		fmt.Fprintln(w, "  Action\tDecisions\tBlunders\tBlunder %\tPR\tMWC")
		fmt.Fprintln(w, "  ——————\t—————————\t————————\t—————————\t——\t———")
		for _, ca := range result.CubeActionBreakdown {
			blunderPct := 0.0
			if ca.NumDecisions > 0 {
				blunderPct = 100 * float64(ca.BlunderCount) / float64(ca.NumDecisions)
			}
			mwcStr := "—"
			if result.MWCAvailable {
				mwcStr = fmt.Sprintf("%.4f", ca.MWC)
			}
			fmt.Fprintf(w, "  %s\t%d\t%d\t%.1f%%\t%.3f\t%s\n",
				ca.Action, ca.NumDecisions, ca.BlunderCount, blunderPct, ca.PR, mwcStr)
		}
		w.Flush()
		fmt.Println()
	}

	// 7. Error histogram
	if len(result.ErrorHistogram) > 0 {
		fmt.Println("── Error Histogram ──")
		fmt.Fprintln(w, "  Range (EMG)\tCount")
		fmt.Fprintln(w, "  ——————————\t—————")
		for _, b := range result.ErrorHistogram {
			var rangeStr string
			if b.MaxMP == -1 {
				rangeStr = fmt.Sprintf("≥%.3f", float64(b.MinMP)/1000)
			} else {
				rangeStr = fmt.Sprintf("%.3f–%.3f", float64(b.MinMP)/1000, float64(b.MaxMP)/1000)
			}
			fmt.Fprintf(w, "  %s\t%d\n", rangeStr, b.Count)
		}
		w.Flush()
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
		_, _ = fmt.Scanln(&response)
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
		matchCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	if *matchID == 0 {
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

	// Supported match file extensions
	supportedExts := map[string]bool{
		".xg":  true,
		".xgp": true,
		".sgf": true,
		".mat": true,
		".txt": true,
		".bgf": true,
	}

	// Find all supported match files
	var matchFiles []string

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

		// Check for supported extensions
		if supportedExts[strings.ToLower(filepath.Ext(path))] {
			matchFiles = append(matchFiles, path)
		}

		return nil
	}

	err := filepath.Walk(dirPath, walkFunc)
	if err != nil {
		return fmt.Errorf("failed to scan directory: %v", err)
	}

	if len(matchFiles) == 0 {
		fmt.Println("No match files found in directory (.xg, .sgf, .mat, .txt, .bgf)")
		return nil
	}

	fmt.Printf("Found %d match file(s) to import\n\n", len(matchFiles))

	// Import each file and collect results
	var results []BatchImportResult
	successCount := 0
	failCount := 0
	duplicateCount := 0
	totalPositions := 0

	for i, filePath := range matchFiles {
		relPath, _ := filepath.Rel(dirPath, filePath)
		fmt.Printf("[%d/%d] Importing: %s...", i+1, len(matchFiles), relPath)

		result := BatchImportResult{
			FilePath: relPath,
		}

		// Route to appropriate importer based on extension
		ext := strings.ToLower(filepath.Ext(filePath))
		var matchID int64
		switch ext {
		case ".xgp":
			posID, posErr := cli.db.ImportXGPPosition(filePath)
			if posErr != nil {
				fmt.Printf(" ERROR: %v\n", posErr)
				result.Error = posErr.Error()
				failCount++
			} else {
				result.Success = true
				result.Positions = 1
				totalPositions++
				successCount++
				fmt.Printf(" OK (Position ID: %d)\n", posID)
			}
			results = append(results, result)
			continue
		case ".xg":
			matchID, err = cli.db.ImportXGMatch(filePath)
		case ".sgf", ".mat", ".txt":
			matchID, err = cli.db.ImportGnuBGMatch(filePath)
		case ".bgf":
			matchID, err = cli.db.ImportBGFMatch(filePath)
		}

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

		// After each successful match import, checkpoint the WAL to keep file size bounded.
		if result.Success && result.MatchID > 0 {
			_, _ = cli.db.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
		}
	}

	// After all imports, update query planner statistics.
	_, _ = cli.db.db.Exec("ANALYZE")

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
		len(matchFiles), successCount, duplicateCount, failCount, totalPositions)

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
		editCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	// Check that at least one edit option is provided
	if *user == "" && *description == "" && !*clearUser && !*clearDescription {
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
	moveErrorMin := searchCmd.Float64("move-error-min", 0, "Minimum played move error (millipoints)")
	moveErrorMax := searchCmd.Float64("move-error-max", 0, "Maximum played move error (millipoints)")
	hasAnalysis := searchCmd.Bool("has-analysis", false, "Only positions with analysis")
	checkerOff1Min := searchCmd.Int("off1-min", 0, "Minimum checkers off for player 1")
	checkerOff2Min := searchCmd.Int("off2-min", 0, "Minimum checkers off for player 2")
	matchIDsFlag := searchCmd.String("match-ids", "", "Filter by match IDs (comma-separated, e.g. '1,3,5' or range '2,7')")
	tournamentIDsFlag := searchCmd.String("tournament-ids", "", "Filter by tournament IDs (comma-separated, e.g. '1,3' or range '1,5')")

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
		fmt.Println()
		fmt.Println("  # Search in specific matches")
		fmt.Println("  blunderdb search --db database.db --match-ids 2,5")
		fmt.Println()
		fmt.Println("  # Search in a tournament")
		fmt.Println("  blunderdb search --db database.db --tournament-ids 1")
	}

	if err := searchCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
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

	var moveErrorFilter string
	if *moveErrorMin > 0 || *moveErrorMax > 0 {
		if *moveErrorMin > 0 && *moveErrorMax > 0 {
			moveErrorFilter = fmt.Sprintf("E%f,%f", *moveErrorMin, *moveErrorMax)
		} else if *moveErrorMin > 0 {
			moveErrorFilter = fmt.Sprintf("E>%f", *moveErrorMin)
		} else {
			moveErrorFilter = fmt.Sprintf("E<%f", *moveErrorMax)
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

	// Use the core implementation to get analysis data in the same query, avoiding
	// per-row LoadAnalysis calls for errorMin and hasAnalysis filtering.
	positions, analysisMap, err := cli.db.loadPositionsByFiltersCore(SearchFilters{
		Filter:                  filter,
		IncludeCube:             includeCube,
		IncludeScore:            includeScore,
		PipCountFilter:          pipCountFilter,
		WinRateFilter:           winRateFilter,
		MoveErrorFilter:         moveErrorFilter,
		Player1CheckerOffFilter: player1CheckerOffFilter,
		Player2CheckerOffFilter: player2CheckerOffFilter,
		DecisionTypeFilter:      decisionTypeFilter,
		MatchIDsFilter:          *matchIDsFlag,
		TournamentIDsFilter:     *tournamentIDsFlag,
	})
	if err != nil {
		return fmt.Errorf("failed to search positions: %v", err)
	}

	// Apply errorMin / hasAnalysis using the analysis map from the JOIN (no extra DB queries).
	var filteredPositions []Position
	for _, pos := range positions {
		if *errorMin > 0 || *hasAnalysis {
			analysis := analysisMap[pos.ID]
			if analysis == nil {
				if *hasAnalysis {
					continue
				}
			} else if *errorMin > 0 {
				hasError := false
				if analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 1 {
					if analysis.CheckerAnalysis.Moves[1].EquityError != nil {
						if math.Round(*analysis.CheckerAnalysis.Moves[1].EquityError*1000)/1000 >= *errorMin {
							hasError = true
						}
					}
				}
				if analysis.DoublingCubeAnalysis != nil {
					if math.Round(analysis.DoublingCubeAnalysis.CubefulNoDoubleError*1000)/1000 >= *errorMin ||
						math.Round(analysis.DoublingCubeAnalysis.CubefulDoubleTakeError*1000)/1000 >= *errorMin ||
						math.Round(analysis.DoublingCubeAnalysis.CubefulDoublePassError*1000)/1000 >= *errorMin {
						hasError = true
					}
				}
				if !hasError {
					continue
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

		err = cli.db.ExportDatabase(ExportOptions{
			ExportPath:         *outputDB,
			Positions:          filteredPositions,
			Metadata:           metadata,
			IncludeAnalysis:    true,
			IncludeComments:    true,
			IncludePlayedMoves: true,
		})
		if err != nil {
			return fmt.Errorf("failed to export database: %v", err)
		}

		fmt.Println("Export completed successfully")
	}

	return nil
}
