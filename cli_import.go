package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

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
