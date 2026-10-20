package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// runExport handles the export command
func (cli *CLI) runExport(args []string) error {
	exportCmd := flag.NewFlagSet("export", flag.ContinueOnError)

	// Define flags
	dbPath := exportCmd.String("db", "", "Path to the database file (required)")
	exportType := exportCmd.String("type", "", "Export type: database, positions, matches, mat (required)")
	outputFile := exportCmd.String("file", "", "Path to the output file (required)")
	outputDir := exportCmd.String("dir", "", "Output directory for .mat batch export (type=mat, multiple matches)")
	includeAnalysis := exportCmd.Bool("analysis", true, "Include analysis in database export (default: true)")
	includeComments := exportCmd.Bool("comments", true, "Include comments in database export (default: true)")
	includeFilterLibrary := exportCmd.Bool("filters", true, "Include filter library in database export (default: true)")
	includePlayedMoves := exportCmd.Bool("played-moves", true, "Include played moves in analysis (default: true)")
	includeMatches := exportCmd.Bool("matches", true, "Include matches in database export (default: true)")
	includeTranscriptions := exportCmd.Bool("transcriptions", true, "Include the transcription drafts of the exported matches in database export (default: true)")
	includeAliases := exportCmd.Bool("aliases", true, "Include the player and event aliases the exported matches write in database export (default: true)")
	includeCollections := exportCmd.Bool("collections", true, "Include collections in database export (default: true)")
	includeTournaments := exportCmd.Bool("tournaments", true, "Include tournaments in database export (default: true)")
	includeDecks := exportCmd.Bool("decks", true, "Include Anki decks in database export (default: true)")
	includeLessons := exportCmd.Bool("lessons", true, "Include lessons, with the collections and positions their steps show, in database export (default: true)")
	includeAnkiHistory := exportCmd.Bool("anki-history", true, "Carry the review history of the exported Anki decks (default: true)")
	collectionIDsStr := exportCmd.String("collection-ids", "", "Comma-separated list of collection IDs to export (empty = all)")
	matchIDsStr := exportCmd.String("match-ids", "", "Comma-separated list of match IDs to export (empty = all)")
	deckIDsStr := exportCmd.String("deck-ids", "", "Comma-separated list of Anki deck IDs to export (empty = all)")
	lessonIDsStr := exportCmd.String("lesson-ids", "", "Comma-separated list of lesson IDs to export (empty = all)")
	tournamentIDsStr := exportCmd.String("tournament-ids", "", "Comma-separated list of tournament IDs to export (empty = all)")
	watermark := exportCmd.String("watermark", "", "Mark the exported file with where it comes from, e.g. \"Cours de Jean Dupont - 12 mars 2026\"")
	watermarkNote := exportCmd.String("watermark-note", "", "Free text attached to the watermark (terms of use, contact)")
	password := exportCmd.String("password", "", "Protect the exported file with a password (produces a .dbx container)")
	incremental := exportCmd.Bool("incremental", false, "Database export: only what entered the library since the last incremental export to the same folder (or --sync-cursor); --file may then be a folder, where a dated file name is chosen")
	syncCursor := exportCmd.String("sync-cursor", "", "With --incremental: name of the cursor to read and advance (default: the destination folder)")
	format := exportCmd.String("format", "text", "Output format: text or json")
	planCounts := exportCmd.Bool("plan", false, "Print what a database or matches export would write, family by family, without writing a file")
	planIDs := exportCmd.String("plan-ids", "", "Print one page of the source ids behind a --plan count, for a family: positions, comments, matches, tournaments, directions, rencontres, collections, lessons, ankiDecks, transcriptions, filters")
	planOffset := exportCmd.Int("offset", 0, "With --plan-ids: index of the first id of the page")
	planLimit := exportCmd.Int("limit", 0, "With --plan-ids: ids per page (at most 1000, the default)")

	exportCmd.Usage = func() {
		fmt.Println("Usage: blunderdb export [options]")
		fmt.Println()
		fmt.Println("Export data from the database.")
		fmt.Println()
		fmt.Println("Options:")
		exportCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Export Types:")
		fmt.Println("  database   Export the whole library, as the server's export does: positions, matches,")
		fmt.Println("             tournaments, collections, lessons, Anki decks, transcriptions, aliases, filters.")
		fmt.Println("             Each family can be left out (--matches=false) or narrowed (--match-ids=1,2).")
		fmt.Println("  positions  Export positions to text file (JSON format)")
		fmt.Println("  matches    Export only matches to a new database")
		fmt.Println("  mat        Export match(es) as Jellyfish/gnubg .mat transcript(s)")
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
		fmt.Println("  # Export with specific collections only")
		fmt.Println("  blunderdb export --db database.db --type database --file export.db --collection-ids=1,2,3")
		fmt.Println()
		fmt.Println("  # Export without collections or Anki decks")
		fmt.Println("  blunderdb export --db database.db --type database --file export.db --collections=false --decks=false")
		fmt.Println()
		fmt.Println("  # Export with specific lessons, or without any")
		fmt.Println("  blunderdb export --db database.db --type database --file export.db --lesson-ids=2")
		fmt.Println("  blunderdb export --db database.db --type database --file export.db --lessons=false")
		fmt.Println()
		fmt.Println("  # Export with specific tournaments")
		fmt.Println("  blunderdb export --db database.db --type database --file export.db --tournament-ids=1,2")
		fmt.Println()
		fmt.Println("  # Mark the exported file with its origin, and protect it with a password")
		fmt.Println("  blunderdb export --db database.db --type database --file cours.db \\")
		fmt.Println("      --watermark \"Cours de Jean Dupont - 12 mars 2026\" --password secret")
		fmt.Println()
		fmt.Println("  # Send another machine what entered the library since the last sync to this folder")
		fmt.Println("  blunderdb export --db database.db --type database --incremental --file ~/Sync/blunderdb/")
		fmt.Println()
		fmt.Println("  # Count what an export would carry, without writing it")
		fmt.Println("  blunderdb export --db database.db --type database --tournament-ids=1 --plan")
		fmt.Println()
		fmt.Println("  # Export positions to text file")
		fmt.Println("  blunderdb export --db database.db --type positions --file positions.txt")
		fmt.Println()
		fmt.Println("  # Export only matches to a new database")
		fmt.Println("  blunderdb export --db database.db --type matches --file matches.db")
		fmt.Println()
		fmt.Println("  # Export one match as a .mat transcript")
		fmt.Println("  blunderdb export --db database.db --type mat --match-ids 5 --file game.mat")
		fmt.Println()
		fmt.Println("  # Export several (or all) matches as .mat files into a directory")
		fmt.Println("  blunderdb export --db database.db --type mat --match-ids 5,9,12 --dir out/")
		fmt.Println("  blunderdb export --db database.db --type mat --dir out/")
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

	formatLower := strings.ToLower(*format)
	if formatLower != "text" && formatLower != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}
	text := formatLower != "json"

	plan := exportPlanRequest{counts: *planCounts || *planIDs != "", family: *planIDs, offset: *planOffset, limit: *planLimit}
	if plan.counts && !planType(*exportType) {
		return fmt.Errorf("--plan and --plan-ids count a database export: use them with --type database or matches, not %q", *exportType)
	}

	// The .mat export accepts either --file (single match) or --dir (batch);
	// every other type writes one output file and requires --file.
	if strings.ToLower(*exportType) == "mat" {
		if *outputFile == "" && *outputDir == "" {
			exportCmd.Usage()
			return fmt.Errorf("type mat requires --file (single match) or --dir (multiple matches)")
		}
	} else if *outputFile == "" && !plan.counts {
		exportCmd.Usage()
		return fmt.Errorf("missing required flag: --file")
	}

	// Parse comma-separated ID lists
	var ids [5][]int64
	for i, f := range []struct{ name, raw string }{
		{"collection-ids", *collectionIDsStr},
		{"lesson-ids", *lessonIDsStr},
		{"deck-ids", *deckIDsStr},
		{"match-ids", *matchIDsStr},
		{"tournament-ids", *tournamentIDsStr},
	} {
		if f.raw == "" {
			continue
		}
		parsed, parseErr := parseIDList(f.raw)
		if parseErr != nil {
			return fmt.Errorf("invalid %s: %w", f.name, parseErr)
		}
		ids[i] = parsed
	}
	collectionIDs, lessonIDs, deckIDs, matchIDs, tournamentIDs := ids[0], ids[1], ids[2], ids[3], ids[4]

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	// Perform export based on type
	marking := exportMarking{watermark: *watermark, note: *watermarkNote, password: *password}

	switch strings.ToLower(*exportType) {
	case "database":
		// A list of ids names some of a family: it brings that family in.
		opts := ExportOptions{
			ExportPath:            *outputFile,
			WholeLibrary:          true,
			IncludeAnalysis:       *includeAnalysis,
			IncludeComments:       *includeComments,
			IncludeFilterLibrary:  *includeFilterLibrary,
			IncludePlayedMoves:    *includePlayedMoves,
			IncludeMatches:        *includeMatches || len(matchIDs) > 0,
			IncludeTranscriptions: *includeTranscriptions,
			IncludeAliases:        *includeAliases,
			IncludeCollections:    *includeCollections || len(collectionIDs) > 0,
			IncludeTournaments:    *includeTournaments || len(tournamentIDs) > 0,
			IncludeAnkiDecks:      *includeDecks || len(deckIDs) > 0,
			IncludeAnkiHistory:    *includeAnkiHistory,
			IncludeLessons:        *includeLessons || len(lessonIDs) > 0,
			LessonIDs:             lessonIDs,
			CollectionIDs:         collectionIDs,
			MatchIDs:              matchIDs,
			TournamentIDs:         tournamentIDs,
			DeckIDs:               deckIDs,
			Incremental:           *incremental,
			SyncCursor:            *syncCursor,
		}
		if plan.counts {
			return cli.printExportPlan(opts, plan, text)
		}
		return cli.exportDatabaseWithOptions(opts, marking, text)
	case "positions":
		return cli.exportPositions(*outputFile, text)
	case "matches":
		return cli.exportMatchesOnly(*outputFile, matchIDs, *includeTranscriptions, *includeAliases, marking, text, plan)
	case "mat":
		return cli.exportMatchesMAT(matchIDs, *outputFile, *outputDir, text)
	default:
		return fmt.Errorf("unknown export type: %s (must be 'database', 'positions', 'matches', or 'mat')", *exportType)
	}
}

// exportMatchesMAT writes one or more matches as Jellyfish/gnubg .mat
// transcripts. A .mat file holds exactly one match, so --file exports a single
// match to that path, while --dir writes one auto-named file per match (all
// matches when matchIDs is empty). Auto-names come from the same helper the GUI
// dialog uses; on a name collision within the batch the match id is appended.
func (cli *CLI) exportMatchesMAT(matchIDs []int64, outputFile, outputDir string, text bool) error {
	ids := matchIDs
	if len(ids) == 0 {
		matches, err := cli.db.GetAllMatches()
		if err != nil {
			return fmt.Errorf("failed to load matches: %w", err)
		}
		for _, m := range matches {
			ids = append(ids, m.ID)
		}
	}
	if len(ids) == 0 {
		return fmt.Errorf("no matches found in database")
	}

	// Single match to an explicit file.
	if outputFile != "" {
		if len(ids) != 1 {
			return fmt.Errorf("--file exports exactly one match; use --dir for %d matches", len(ids))
		}
		path := ensureMatExt(outputFile)
		if err := cli.db.ExportMatchMAT(ids[0], path); err != nil {
			return fmt.Errorf("failed to export match %d: %w", ids[0], err)
		}
		if !text {
			return printJSON(struct {
				Files []string `json:"files"`
			}{Files: []string{path}})
		}
		fmt.Printf("Successfully exported match %d to %s\n", ids[0], path)
		return nil
	}

	// Batch into a directory, one auto-named .mat per match.
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	used := make(map[string]bool)
	var files []string
	for _, id := range ids {
		name, err := cli.db.SuggestMatFilename(id)
		if err != nil {
			return fmt.Errorf("failed to build filename for match %d: %w", id, err)
		}
		path := filepath.Join(outputDir, name)
		if used[name] || fileExists(path) {
			name = matSuffixID(name, id)
			path = filepath.Join(outputDir, name)
		}
		used[name] = true
		if err := cli.db.ExportMatchMAT(id, path); err != nil {
			return fmt.Errorf("failed to export match %d: %w", id, err)
		}
		files = append(files, path)
		if text {
			fmt.Printf("Exported match %d -> %s\n", id, path)
		}
	}
	if !text {
		return printJSON(struct {
			Files []string `json:"files"`
		}{Files: files})
	}
	fmt.Printf("Successfully exported %d match(es) to %s\n", len(ids), outputDir)
	return nil
}

// ensureMatExt appends .mat unless the path already ends in it.
func ensureMatExt(path string) string {
	if !strings.HasSuffix(strings.ToLower(path), ".mat") {
		return path + ".mat"
	}
	return path
}

// matSuffixID inserts "_m<id>" before the .mat extension to disambiguate a
// filename collision in a batch export.
func matSuffixID(name string, id int64) string {
	ext := filepath.Ext(name)
	return fmt.Sprintf("%s_m%d%s", strings.TrimSuffix(name, ext), id, ext)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// exportMarking bundles the two optional mechanisms an export can carry: a signed statement
// of where the file comes from, and a password around it. Both are the producer's choice and
// neither makes the recipient's side write anything.
type exportMarking struct {
	watermark string
	note      string
	password  string
}

// exportDatabaseWithOptions writes opts' export, stamped with this database's version and
// the producer's marking.
func (cli *CLI) exportDatabaseWithOptions(opts ExportOptions, marking exportMarking, text bool) error {
	outputFile := opts.ExportPath
	if text {
		fmt.Printf("Exporting database to: %s\n", outputFile)
	}

	opts.Metadata = make(map[string]string)
	if version, err := cli.db.GetDatabaseVersion(); err == nil {
		opts.Metadata["database_version"] = version
	}
	opts.Watermark, opts.WatermarkNote, opts.Password = marking.watermark, marking.note, marking.password

	var report ingest.ExportReport
	// Ctrl-C cancels a large export in flight instead of waiting it out.
	err := withInterruptibleContext(func() {
		if text {
			fmt.Println("\nCancelling...")
		}
	}, func(ctx context.Context) error {
		var err error
		report, err = cli.db.ExportDatabaseCtx(ctx, opts)
		return err
	})
	if err != nil && errors.Is(err, context.Canceled) {
		return fmt.Errorf("export cancelled")
	}
	if err != nil {
		return fmt.Errorf("failed to export database: %w", err)
	}
	// A password wraps the file into a .dbx next to the path asked for.
	if report.Path != "" {
		outputFile = report.Path
	}

	// Get file size
	var size int64
	if info, err := os.Stat(outputFile); err == nil {
		size = info.Size()
	}

	if !text {
		return printJSON(struct {
			File   string              `json:"file"`
			Bytes  int64               `json:"bytes"`
			Report ingest.ExportReport `json:"report"`
		}{File: outputFile, Bytes: size, Report: report})
	}
	if size > 0 {
		fmt.Printf("Successfully exported database (%d bytes)\n", size)
	} else {
		fmt.Println("Successfully exported database")
	}
	printExportReport(report)

	return nil
}

// exportPlanRequest is what --plan and --plan-ids ask: the counts, or one page of a family's ids.
type exportPlanRequest struct {
	counts        bool
	family        string
	offset, limit int
}

// planType reports whether an export type writes a database, the only kind --plan counts.
func planType(exportType string) bool {
	t := strings.ToLower(exportType)
	return t == "database" || t == "matches"
}

// printExportPlan prints what opts' export would write, family by family, without writing it.
func (cli *CLI) printExportPlan(opts ExportOptions, req exportPlanRequest, text bool) error {
	if req.family != "" {
		page, err := cli.db.PlanExportIDs(opts, req.family, req.offset, req.limit)
		if err != nil {
			return fmt.Errorf("failed to plan the export: %w", err)
		}
		if !text {
			return printJSON(page)
		}
		fmt.Printf("%s %d-%d of %d:", page.Family, page.Offset+1, page.Offset+len(page.IDs), page.Total)
		for _, id := range page.IDs {
			fmt.Printf(" %d", id)
		}
		fmt.Println()
		return nil
	}
	plan, err := cli.db.PlanExport(opts)
	if err != nil {
		return fmt.Errorf("failed to plan the export: %w", err)
	}
	if !text {
		return printJSON(plan)
	}
	fmt.Println("The export would write:")
	for _, f := range []struct {
		name string
		n    int
	}{
		{"positions", plan.Positions}, {"comments", plan.Comments},
		{"matches", plan.Matches}, {"tournaments", plan.Tournaments},
		{"directions", plan.Directions}, {"slots", plan.Slots}, {"rencontres", plan.Rencontres},
		{"collections", plan.Collections}, {"lessons", plan.Lessons},
		{"Anki decks", plan.AnkiDecks}, {"transcriptions", plan.Transcriptions},
		{"aliases", plan.Aliases}, {"filters", plan.Filters},
	} {
		fmt.Printf("  %-15s %d\n", f.name, f.n)
	}
	for _, c := range plan.FrozenCollections {
		fmt.Printf("Living collection %d (%s) leaves frozen: its query names rows the file will not hold\n", c.ID, c.Name)
	}
	for _, d := range plan.DegradedAnkiDecks {
		fmt.Printf("Search deck %d (%s) leaves with part of its source dropped: it names rows the file will not hold\n", d.ID, d.Name)
	}
	return nil
}

// printExportReport prints the counts of a finished export.
func printExportReport(r ingest.ExportReport) {
	fmt.Printf("  positions %d, analyses %d, comments %d, matches %d, tournaments %d, collections %d,\n",
		r.Positions, r.Analyses, r.Comments, r.Matches, r.Tournaments, r.Collections)
	fmt.Printf("  lessons %d, Anki decks %d, transcriptions %d, aliases %d, filters %d\n",
		r.Lessons, r.AnkiDecks, r.Transcriptions, r.Aliases, r.Filters)
	if r.Directions > 0 || r.Rencontres > 0 {
		fmt.Printf("  directions %d, slots %d, rencontres %d\n", r.Directions, r.Slots, r.Rencontres)
	}
	if r.DegradedAnkiDecks > 0 {
		fmt.Printf("  %d search deck(s) left with part of their source dropped (see the log)\n", r.DegradedAnkiDecks)
	}
	if r.Skipped > 0 {
		fmt.Printf("  %d row(s) skipped (see the log)\n", r.Skipped)
	}
	if r.SyncHeld {
		fmt.Println("  the sync cursor did not advance: the next incremental export sends these rows again")
	}
}

// exportMatchesOnly exports matches — those of matchIDs, every match when it is empty — to a
// new database, each with the positions its moves were played in and, under transcriptions,
// the draft it was typed from; under aliases, the player and event aliases they write go too.
func (cli *CLI) exportMatchesOnly(outputFile string, matchIDs []int64, transcriptions, aliases bool, marking exportMarking, text bool, plan exportPlanRequest) error {
	if text && !plan.counts {
		fmt.Printf("Exporting matches to: %s\n", outputFile)
	}

	ids := matchIDs
	if len(ids) == 0 {
		matches, err := cli.db.GetAllMatches()
		if err != nil {
			return fmt.Errorf("failed to load matches: %w", err)
		}
		for _, m := range matches {
			ids = append(ids, m.ID)
		}
	}
	if len(ids) == 0 {
		return fmt.Errorf("no matches found in database")
	}

	if text && !plan.counts {
		fmt.Printf("Found %d match(es) to export\n", len(ids))
	}

	// Export with empty positions but with matches
	metadata := make(map[string]string)
	version, err := cli.db.GetDatabaseVersion()
	if err == nil {
		metadata["database_version"] = version
	}

	// The matches carry their own positions (ExportDatabase sets MatchPositions); the
	// exporter reads them from the database. Ctrl-C cancels it in flight.
	opts := ExportOptions{
		ExportPath:            outputFile,
		MatchIDs:              ids,
		Metadata:              metadata,
		IncludeAnalysis:       true,
		IncludeComments:       true,
		IncludePlayedMoves:    true,
		IncludeMatches:        true,
		IncludeTranscriptions: transcriptions,
		IncludeAliases:        aliases,
		Watermark:             marking.watermark,
		WatermarkNote:         marking.note,
		Password:              marking.password,
	}
	if plan.counts {
		return cli.printExportPlan(opts, plan, text)
	}
	var report ingest.ExportReport
	err = withInterruptibleContext(func() {
		if text {
			fmt.Println("\nCancelling...")
		}
	}, func(ctx context.Context) error {
		var err error
		report, err = cli.db.ExportDatabaseCtx(ctx, opts)
		return err
	})
	if err != nil && errors.Is(err, context.Canceled) {
		return fmt.Errorf("export cancelled")
	}
	if err != nil {
		return fmt.Errorf("failed to export matches: %w", err)
	}
	if report.Path != "" {
		outputFile = report.Path
	}

	// Get file size
	var size int64
	if info, err := os.Stat(outputFile); err == nil {
		size = info.Size()
	}

	if !text {
		return printJSON(struct {
			File    string              `json:"file"`
			Bytes   int64               `json:"bytes"`
			Matches int                 `json:"matches"`
			Report  ingest.ExportReport `json:"report"`
		}{File: outputFile, Bytes: size, Matches: report.Matches, Report: report})
	}
	if size > 0 {
		fmt.Printf("Successfully exported matches (%d bytes)\n", size)
	} else {
		fmt.Println("Successfully exported matches")
	}
	printExportReport(report)

	return nil
}

// exportPositions exports positions to a text file
func (cli *CLI) exportPositions(outputFile string, text bool) error {
	if text {
		fmt.Printf("Exporting positions to: %s\n", outputFile)
	}

	// Get all positions
	positions, err := cli.db.LoadAllPositions()
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	// Create output file
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
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

	if !text {
		return printJSON(struct {
			File      string `json:"file"`
			Positions int    `json:"positions"`
		}{File: outputFile, Positions: len(positions)})
	}
	fmt.Printf("Successfully exported %d positions\n", len(positions))
	return nil
}
