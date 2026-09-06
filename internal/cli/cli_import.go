package cli

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

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// runImport handles the import command
func (cli *CLI) runImport(args []string) error {
	importCmd := flag.NewFlagSet("import", flag.ContinueOnError)

	// Define flags
	dbPath := importCmd.String("db", "", "Path to the database file (required)")
	importType := importCmd.String("type", "", "Import type: match, position, batch (required)")
	inputFile := importCmd.String("file", "", "Path to the file to import (for match/position)")
	inputDir := importCmd.String("dir", "", "Path to directory for batch import (for batch)")
	recursive := importCmd.Bool("recursive", true, "Recursively scan subdirectories for batch import")
	format := importCmd.String("format", "text", "Output format: text or json")
	watchFolder := importCmd.Bool("watch", false,
		"With --type batch: keep running and import each match file as it appears in --dir (Ctrl-C to stop)")
	watchEvery := importCmd.Duration("watch-every", 0,
		"How often --watch looks at the folder (default 10s, floor 2s)")
	failOnError := importCmd.Bool("fail-on-error", false,
		"Exit non-zero when any item failed to import (position/batch); by default only a total failure (nothing imported, duplicates aside) is an error")

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
		fmt.Println()
		fmt.Println("  # Batch import, machine-readable, failing the run if any file errored")
		fmt.Println("  blunderdb import --db database.db --type batch --dir ./matches/ --format json --fail-on-error")
		fmt.Println()
		fmt.Println("  # Import the folder as it stands, then keep importing what appears in it")
		fmt.Println("  blunderdb import --db database.db --type batch --dir ~/XG/Matches")
		fmt.Println("  blunderdb import --db database.db --type batch --dir ~/XG/Matches --watch")
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

	formatLower := strings.ToLower(*format)
	if formatLower != "text" && formatLower != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
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
		return cli.importMatch(*inputFile, formatLower)
	case "position":
		if *inputFile == "" {
			importCmd.Usage()
			return fmt.Errorf("missing required flag: --file")
		}
		// Verify input file exists
		if _, err := os.Stat(*inputFile); os.IsNotExist(err) {
			return fmt.Errorf("input file does not exist: %s", *inputFile)
		}
		return cli.importPosition(*inputFile, formatLower, *failOnError)
	case "batch":
		if *inputDir == "" {
			importCmd.Usage()
			return fmt.Errorf("missing required flag: --dir")
		}
		// Verify directory exists
		if info, err := os.Stat(*inputDir); os.IsNotExist(err) || !info.IsDir() {
			return fmt.Errorf("directory does not exist or is not a directory: %s", *inputDir)
		}
		if *watchFolder {
			return cli.importWatch(*inputDir, formatLower, *watchEvery, *failOnError)
		}
		return cli.importBatch(*inputDir, *recursive, formatLower, *failOnError)
	default:
		return fmt.Errorf("unknown import type: %s (must be 'match', 'position', or 'batch')", *importType)
	}
}

// importMatchResult is the --format json shape for `import --type match`,
// covering both a whole match and a lone XGP position (Type tells them apart;
// the fields the other kind doesn't use are omitted rather than zero-valued).
type importMatchResult struct {
	Type        string `json:"type"` // "match" or "xgp_position"
	MatchID     int64  `json:"match_id,omitempty"`
	PositionID  int64  `json:"position_id,omitempty"`
	Player1     string `json:"player1,omitempty"`
	Player2     string `json:"player2,omitempty"`
	Event       string `json:"event,omitempty"`
	Location    string `json:"location,omitempty"`
	MatchLength int32  `json:"match_length,omitempty"`
	Games       int    `json:"games,omitempty"`
	// Report is the end-of-import report (#257), the same object the
	// interface's panel shows. Omitted when the batch could not be recorded.
	Report *domain.ImportReport `json:"report,omitempty"`
}

// reportOf unwraps a batch's report, nil-safe: every caller here treats a
// missing batch as "no report", never as an error.
func reportOf(b *domain.ImportBatch) *domain.ImportReport {
	if b == nil {
		return nil
	}
	return &b.Report
}

// importMatch imports a match file (XG, SGF, MAT, TXT, BGF) or XGP position file
func (cli *CLI) importMatch(filePath, format string) error {
	if format != "json" {
		fmt.Printf("Importing match from: %s\n", filePath)
	}

	// Verify file extension and route to appropriate importer
	ext := strings.ToLower(filepath.Ext(filePath))
	batchID := cli.beginImportBatch(filePath, strings.TrimPrefix(ext, "."))
	var failures domain.ImportReport
	var matchID int64
	var err error

	switch ext {
	case ".xgp":
		// XGP files are single-position files, not match files
		posID, posErr := cli.db.ImportXGPPosition(filePath)
		if posErr != nil {
			recordFailure(&failures, filePath, posErr)
			cli.finishImportBatch(batchID, failures)
			return fmt.Errorf("failed to import XGP position: %w", posErr)
		}
		report := cli.finishImportBatch(batchID, failures)
		if format == "json" {
			return printJSON(importMatchResult{Type: "xgp_position", PositionID: posID, Report: reportOf(report)})
		}
		fmt.Printf("Successfully imported XGP position (ID: %d)\n", posID)
		printImportReport(report)
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
		// A duplicate is not a failure to READ the file, so it is not recorded
		// as one: the batch's own counter already saw it as a skip.
		if !errors.Is(err, ErrDuplicateMatch) {
			recordFailure(&failures, filePath, err)
		}
		cli.finishImportBatch(batchID, failures)
		if errors.Is(err, ErrDuplicateMatch) {
			return fmt.Errorf("this match has already been imported to the database")
		}
		return fmt.Errorf("failed to import match: %w", err)
	}

	// Fetch match details (best-effort; a lookup failure does not undo the import).
	match, matchErr := cli.db.GetMatchByID(matchID)
	report := cli.finishImportBatch(batchID, failures)

	if format == "json" {
		result := importMatchResult{Type: "match", MatchID: matchID, Report: reportOf(report)}
		if matchErr == nil && match != nil {
			result.Player1 = match.Player1Name
			result.Player2 = match.Player2Name
			result.Event = match.Event
			result.Location = match.Location
			result.MatchLength = match.MatchLength
			result.Games = match.GameCount
		}
		return printJSON(result)
	}

	fmt.Printf("Successfully imported match (ID: %d)\n", matchID)

	if matchErr == nil && match != nil {
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

	printImportReport(report)
	return nil
}

// importPositionResult is the --format json shape for `import --type position`.
type importPositionResult struct {
	Imported   int   `json:"imported"`
	Failed     int   `json:"failed"`
	PositionID int64 `json:"position_id,omitempty"` // set only for a single BGBlitz position file
}

// importPosition imports a position file. It fails the run (regardless of
// --fail-on-error) when nothing at all was imported — a file that produced
// zero positions is never a silent success (#176) — and additionally fails
// when failOnError is set and any individual line errored despite others
// succeeding.
func (cli *CLI) importPosition(filePath, format string, failOnError bool) error {
	if format != "json" {
		fmt.Printf("Importing positions from: %s\n", filePath)
	}

	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Check if this is a BGBlitz position text file
	// BGBlitz text files contain "Position-ID:" which is unique to BGBlitz format
	contentStr := string(content)
	if strings.Contains(contentStr, "Position-ID:") {
		// BGBlitz position text file
		posID, err := cli.db.ImportBGFPosition(filePath)
		if err != nil {
			return fmt.Errorf("failed to import BGBlitz position: %w", err)
		}
		if format == "json" {
			return printJSON(importPositionResult{Imported: 1, PositionID: posID})
		}
		fmt.Printf("Successfully imported BGBlitz position (ID: %d)\n", posID)
		return nil
	}

	// Parse positions (assuming position JSON format, one per line)
	lines := strings.Split(contentStr, "\n")
	imported := 0
	failed := 0

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Try to parse as position JSON
		var pos Position
		if err := json.Unmarshal([]byte(line), &pos); err != nil {
			slog.Warn("parsing line", "line", i+1, "err", err)
			failed++
			continue
		}

		// A position file delivers positions on their own, with no match to
		// attach them to, so they are individually imported (ADR-0001). The
		// flag is sticky, so a position a match already brought in gains it.
		pos.IndividuallyImported = true

		// Save position
		_, err := cli.db.SavePosition(&pos)
		if err != nil {
			slog.Warn("importing line", "line", i+1, "err", err)
			failed++
			continue
		}
		imported++
	}

	if format == "json" {
		if err := printJSON(importPositionResult{Imported: imported, Failed: failed}); err != nil {
			return err
		}
	} else {
		fmt.Printf("Successfully imported %d positions\n", imported)
		if failed > 0 {
			fmt.Printf("Failed to import %d positions\n", failed)
		}
	}

	if imported == 0 {
		return fmt.Errorf("no positions were imported from %s (%d error(s))", filePath, failed)
	}
	if failOnError && failed > 0 {
		return fmt.Errorf("%d of %d position(s) failed to import from %s", failed, imported+failed, filePath)
	}

	return nil
}

// BatchImportResult represents the result of a single file import
type BatchImportResult struct {
	FilePath  string `json:"file_path"`
	Success   bool   `json:"success"`
	MatchID   int64  `json:"match_id,omitempty"`
	Error     string `json:"error,omitempty"`
	Player1   string `json:"player1,omitempty"`
	Player2   string `json:"player2,omitempty"`
	Games     int    `json:"games,omitempty"`
	Positions int    `json:"positions,omitempty"`
}

// importBatchResult is the --format json shape for `import --type batch`.
type importBatchResult struct {
	Files             []BatchImportResult `json:"files"`
	Total             int                 `json:"total"`
	Success           int                 `json:"success"`
	Duplicates        int                 `json:"duplicates"`
	Failed            int                 `json:"failed"`
	PositionsImported int                 `json:"positions_imported"`
	// Report is the end-of-import report (#257) over the whole batch.
	Report *domain.ImportReport `json:"report,omitempty"`
}

// importBatch imports all .xg files from a directory. Like importPosition, it
// fails the run when nothing at all was imported (every file either errored
// or was a duplicate), and additionally fails when failOnError is set and any
// file errored despite others succeeding (#176). A duplicate is not a
// failure: re-running a batch import over a directory that was already
// imported, with no new file added, stays a success — only a batch where
// NOTHING was recognised (every file failed) is an error.
func (cli *CLI) importBatch(dirPath string, recursive bool, format string, failOnError bool) error {
	if format != "json" {
		fmt.Printf("Batch importing from: %s (recursive: %v)\n\n", dirPath, recursive)
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
		if ingest.IsImportable(path) {
			matchFiles = append(matchFiles, path)
		}

		return nil
	}

	err := filepath.Walk(dirPath, walkFunc)
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(matchFiles) == 0 {
		return fmt.Errorf("no match files found in directory %s (%s)", dirPath, strings.Join(ingest.ImportableExtensions(), ", "))
	}

	text := format != "json"
	if text {
		fmt.Printf("Found %d match file(s) to import\n\n", len(matchFiles))
	}

	batchID := cli.beginImportBatch(dirPath, "mixed")
	var failures domain.ImportReport

	// Import each file and collect results
	var results []BatchImportResult
	successCount := 0
	failCount := 0
	duplicateCount := 0
	totalPositions := 0

	for i, filePath := range matchFiles {
		relPath, _ := filepath.Rel(dirPath, filePath)
		if text {
			fmt.Printf("[%d/%d] Importing: %s...", i+1, len(matchFiles), relPath)
		}

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
				if text {
					fmt.Printf(" ERROR: %v\n", posErr)
				}
				result.Error = posErr.Error()
				recordFailure(&failures, relPath, posErr)
				failCount++
			} else {
				result.Success = true
				result.Positions = 1
				totalPositions++
				successCount++
				if text {
					fmt.Printf(" OK (Position ID: %d)\n", posID)
				}
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
				if text {
					fmt.Println(" DUPLICATE")
				}
				result.Error = "duplicate"
				duplicateCount++
			} else {
				if text {
					fmt.Printf(" ERROR: %v\n", err)
				}
				result.Error = err.Error()
				recordFailure(&failures, relPath, err)
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

			if text {
				fmt.Printf(" OK (ID: %d, %d positions)\n", matchID, result.Positions)
			}
		}

		results = append(results, result)

		// After each successful match import, checkpoint the WAL to keep file size bounded.
		if result.Success && result.MatchID > 0 {
			_ = cli.db.Checkpoint()
		}
	}

	// After all imports, update query planner statistics.
	cli.db.RefreshSearchStatistics()

	report := cli.finishImportBatch(batchID, failures)

	if text {
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
		printImportReport(report)
	} else {
		if err := printJSON(importBatchResult{
			Files:             results,
			Total:             len(matchFiles),
			Success:           successCount,
			Duplicates:        duplicateCount,
			Failed:            failCount,
			PositionsImported: totalPositions,
			Report:            reportOf(report),
		}); err != nil {
			return err
		}
	}

	// Nothing recognised at all (every file failed) is always an error;
	// --fail-on-error additionally fails a partial one. A directory whose
	// files are all already in the database imported nothing NEW, but that
	// is the nominal night of a script re-importing the same folder: it is
	// a success, the duplicate count says so.
	if successCount == 0 && duplicateCount == 0 {
		return fmt.Errorf("no file was imported from %s (%d failure(s) out of %d file(s))",
			dirPath, failCount, len(matchFiles))
	}
	if failOnError && failCount > 0 {
		return fmt.Errorf("%d of %d file(s) failed to import from %s", failCount, len(matchFiles), dirPath)
	}

	return nil
}
