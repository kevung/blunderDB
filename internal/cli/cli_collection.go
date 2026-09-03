package cli

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

// runCollection handles the collection command. Collections are the GUI's
// hand-picked sets of positions (the Collections panel) and the server's
// /v1/collections.* family; this command closes the CLI side of the parity
// invariant for the operations a script has a use for — listing, reading,
// creating, renaming, deleting and exporting a collection. Ordering positions
// within a collection, or moving a position between two of them, stays a
// drag-and-drop gesture in the GUI (and a JSON call on the server).
//
// The sub-command table is a map for the same reason handlers() is: the
// parity test walks it, so a sub-command wired here is known to exist.
func (cli *CLI) runCollection(args []string) error {
	if len(args) < 1 {
		cli.printCollectionUsage()
		return fmt.Errorf("missing collection sub-command")
	}
	sub := strings.ToLower(args[0])
	if sub == "--help" || sub == "-h" || sub == "help" {
		cli.printCollectionUsage()
		return nil
	}
	run, ok := cli.collectionHandlers()[sub]
	if !ok {
		cli.printCollectionUsage()
		return fmt.Errorf("unknown collection sub-command: %s", args[0])
	}
	return run(args[1:])
}

// collectionHandlers returns the sub-command table of `blunderdb collection`.
func (cli *CLI) collectionHandlers() map[string]func([]string) error {
	return map[string]func([]string) error{
		"list":   cli.runCollectionList,
		"show":   cli.runCollectionShow,
		"create": cli.runCollectionCreate,
		"rename": cli.runCollectionRename,
		"delete": cli.runCollectionDelete,
		"export": cli.runCollectionExport,
	}
}

// CollectionSubcommands returns the sub-commands of `blunderdb collection`,
// sorted — the exported view cmd/cli-doc-gen walks.
func (cli *CLI) CollectionSubcommands() []string {
	h := cli.collectionHandlers()
	names := make([]string, 0, len(h))
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (cli *CLI) printCollectionUsage() {
	fmt.Println("Usage: blunderdb collection <sub-command> [options]")
	fmt.Println()
	fmt.Println("Manage collections (hand-picked sets of positions).")
	fmt.Println()
	fmt.Println("Sub-commands:")
	fmt.Println("  list      List collections (id, name, number of positions)")
	fmt.Println("  show      List the positions of one collection (id, index, XGID)")
	fmt.Println("  create    Create an empty collection")
	fmt.Println("  rename    Rename a collection")
	fmt.Println("  delete    Delete a collection (its positions stay in the database)")
	fmt.Println("  export    Export one or more collections to a new database file")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  blunderdb collection list --db database.db")
	fmt.Println("  blunderdb collection show --db database.db --id 3 --format json")
	fmt.Println("  blunderdb collection create --db database.db --name \"Blitz openings\"")
	fmt.Println("  blunderdb collection rename --db database.db --id 3 --name \"Openings\"")
	fmt.Println("  blunderdb collection delete --db database.db --id 3 --confirm")
	fmt.Println("  blunderdb collection export --db database.db --id 3,4 --out openings.db")
	fmt.Println()
	fmt.Println("Use 'blunderdb collection <sub-command> --help' for the options of a sub-command.")
}

// collectionFlagSet builds the FlagSet shared by every sub-command: the
// database path, plus the usage banner naming the sub-command.
func collectionFlagSet(sub, summary string, examples ...string) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet("collection "+sub, flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	fs.Usage = func() {
		fmt.Printf("Usage: blunderdb collection %s [options]\n\n%s\n\nOptions:\n", sub, summary)
		fs.PrintDefaults()
		if len(examples) > 0 {
			fmt.Println()
			fmt.Println("Examples:")
			for _, ex := range examples {
				fmt.Println("  " + ex)
			}
		}
	}
	return fs, dbPath
}

// collectionOpen parses fs, checks --db and opens the database. Every
// sub-command starts with this.
func (cli *CLI) collectionOpen(fs *flag.FlagSet, dbPath *string, args []string) error {
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --db")
	}
	return cli.initDatabase(*dbPath)
}

// ── list ─────────────────────────────────────────────────────────────────────

func (cli *CLI) runCollectionList(args []string) error {
	fs, dbPath := collectionFlagSet("list", "List the collections of the database.",
		"blunderdb collection list --db database.db",
		"blunderdb collection list --db database.db --format csv")
	format := fs.String("format", "text", "Output format: text, json or csv")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	return cli.listCollections(*format)
}

func (cli *CLI) listCollections(format string) error {
	collections, err := cli.db.GetAllCollections()
	if err != nil {
		return fmt.Errorf("failed to get collections: %w", err)
	}

	switch strings.ToLower(format) {
	case "json":
		if collections == nil {
			collections = []Collection{}
		}
		return printJSON(collections)
	case "csv":
		w := csv.NewWriter(os.Stdout)
		defer w.Flush()
		if err := w.Write([]string{"id", "name", "positions", "description", "created_at"}); err != nil {
			return fmt.Errorf("write csv header: %w", err)
		}
		for _, c := range collections {
			if err := w.Write([]string{
				strconv.FormatInt(c.ID, 10), c.Name, strconv.Itoa(c.PositionCount), c.Description, c.CreatedAt,
			}); err != nil {
				return fmt.Errorf("write csv row: %w", err)
			}
		}
		return w.Error()
	case "text":
		if len(collections) == 0 {
			fmt.Println("No collections found in database")
			return nil
		}
		fmt.Printf("Found %d collection(s):\n\n", len(collections))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tName\tPositions\tDescription")
		fmt.Fprintln(w, "--\t----\t---------\t-----------")
		for _, c := range collections {
			fmt.Fprintf(w, "%d\t%s\t%d\t%s\n", c.ID, c.Name, c.PositionCount, c.Description)
		}
		return w.Flush()
	default:
		return fmt.Errorf("unknown format: %s (must be 'text', 'json', or 'csv')", format)
	}
}

// ── show ─────────────────────────────────────────────────────────────────────

// collectionPositionRow is one line of `collection show`: the position's
// database id, its 1-based index in the database (the number the GUI's
// status bar shows, so a user can jump to it), and its XGID — the one the
// imported analysis carries when there is one, otherwise the one the GUI
// generates from the board (see xgidOf).
type collectionPositionRow struct {
	ID           int64  `json:"id"`
	Index        int    `json:"index"`
	XGID         string `json:"xgid"`
	Score        [2]int `json:"score"`
	DecisionType string `json:"decision_type"`
}

func (cli *CLI) runCollectionShow(args []string) error {
	fs, dbPath := collectionFlagSet("show", "List the positions of one collection.",
		"blunderdb collection show --db database.db --id 3",
		"blunderdb collection show --db database.db --id 3 --format json")
	id := fs.Int64("id", 0, "Collection ID (required)")
	format := fs.String("format", "text", "Output format: text, json or csv")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *id == 0 {
		fs.Usage()
		return fmt.Errorf("missing required flag: --id")
	}
	return cli.showCollection(*id, *format)
}

func (cli *CLI) showCollection(id int64, format string) error {
	collection, err := cli.db.GetCollectionByID(id)
	if err != nil || collection == nil {
		return fmt.Errorf("collection with ID %d not found", id)
	}
	positions, err := cli.db.GetCollectionPositions(id)
	if err != nil {
		return fmt.Errorf("failed to get collection positions: %w", err)
	}
	indexMap, err := cli.db.GetPositionIndexMap()
	if err != nil {
		return fmt.Errorf("failed to get position index map: %w", err)
	}

	rows := make([]collectionPositionRow, 0, len(positions))
	for _, pos := range positions {
		row := collectionPositionRow{ID: pos.ID, Index: indexMap[pos.ID], Score: pos.Score, DecisionType: "cube"}
		if pos.DecisionType == CheckerAction {
			row.DecisionType = "checker"
		}
		if analysis, err := cli.db.LoadAnalysis(pos.ID); err == nil && analysis != nil {
			row.XGID = analysis.XGID
		}
		if row.XGID == "" {
			row.XGID = xgidOf(&pos)
		}
		rows = append(rows, row)
	}

	switch strings.ToLower(format) {
	case "json":
		return printJSON(struct {
			Collection *Collection             `json:"collection"`
			Positions  []collectionPositionRow `json:"positions"`
		}{collection, rows})
	case "csv":
		w := csv.NewWriter(os.Stdout)
		defer w.Flush()
		if err := w.Write([]string{"id", "index", "xgid", "score", "decision_type"}); err != nil {
			return fmt.Errorf("write csv header: %w", err)
		}
		for _, r := range rows {
			if err := w.Write([]string{
				strconv.FormatInt(r.ID, 10), strconv.Itoa(r.Index), r.XGID,
				fmt.Sprintf("%d-%d", r.Score[0], r.Score[1]), r.DecisionType,
			}); err != nil {
				return fmt.Errorf("write csv row: %w", err)
			}
		}
		return w.Error()
	case "text":
		fmt.Printf("Collection %d: %s\n", collection.ID, collection.Name)
		if collection.Description != "" {
			fmt.Printf("  Description: %s\n", collection.Description)
		}
		fmt.Printf("  Positions: %d\n\n", len(rows))
		if len(rows) == 0 {
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tIndex\tScore\tType\tXGID")
		fmt.Fprintln(w, "--\t-----\t-----\t----\t----")
		for _, r := range rows {
			fmt.Fprintf(w, "%d\t%d\t%d-%d\t%s\t%s\n", r.ID, r.Index, r.Score[0], r.Score[1], r.DecisionType, orDash(r.XGID))
		}
		return w.Flush()
	default:
		return fmt.Errorf("unknown format: %s (must be 'text', 'json', or 'csv')", format)
	}
}

// ── create / rename / delete ─────────────────────────────────────────────────

func (cli *CLI) runCollectionCreate(args []string) error {
	fs, dbPath := collectionFlagSet("create", "Create an empty collection.",
		"blunderdb collection create --db database.db --name \"Blitz openings\"")
	name := fs.String("name", "", "Collection name (required)")
	description := fs.String("description", "", "Collection description")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --name")
	}
	id, err := cli.db.CreateCollection(strings.TrimSpace(*name), *description)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	fmt.Printf("Successfully created collection %q (ID: %d)\n", strings.TrimSpace(*name), id)
	return nil
}

func (cli *CLI) runCollectionRename(args []string) error {
	fs, dbPath := collectionFlagSet("rename", "Rename a collection (and optionally change its description).",
		"blunderdb collection rename --db database.db --id 3 --name \"Openings\"")
	id := fs.Int64("id", 0, "Collection ID (required)")
	name := fs.String("name", "", "New collection name (required)")
	description := fs.String("description", "", "New description (empty keeps the current one)")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *id == 0 {
		fs.Usage()
		return fmt.Errorf("missing required flag: --id")
	}
	if strings.TrimSpace(*name) == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --name")
	}
	current, err := cli.db.GetCollectionByID(*id)
	if err != nil || current == nil {
		return fmt.Errorf("collection with ID %d not found", *id)
	}
	newDescription := current.Description
	if *description != "" {
		newDescription = *description
	}
	if err := cli.db.UpdateCollection(*id, strings.TrimSpace(*name), newDescription); err != nil {
		return fmt.Errorf("failed to rename collection: %w", err)
	}
	fmt.Printf("Successfully renamed collection %d: %q -> %q\n", *id, current.Name, strings.TrimSpace(*name))
	return nil
}

func (cli *CLI) runCollectionDelete(args []string) error {
	fs, dbPath := collectionFlagSet("delete", "Delete a collection. Its positions stay in the database.",
		"blunderdb collection delete --db database.db --id 3 --confirm")
	id := fs.Int64("id", 0, "Collection ID (required)")
	confirm := fs.Bool("confirm", false, "Confirm deletion without prompting")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *id == 0 {
		fs.Usage()
		return fmt.Errorf("missing required flag: --id")
	}
	current, err := cli.db.GetCollectionByID(*id)
	if err != nil || current == nil {
		return fmt.Errorf("collection with ID %d not found", *id)
	}
	fmt.Printf("Collection ID: %d\n  Name: %s\n  Positions: %d\n", current.ID, current.Name, current.PositionCount)
	if !*confirm {
		fmt.Print("\nDelete this collection? (yes/no): ")
		var response string
		_, _ = fmt.Scanln(&response)
		if strings.ToLower(response) != "yes" && strings.ToLower(response) != "y" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}
	if err := cli.db.DeleteCollection(*id); err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	fmt.Printf("Successfully deleted collection %d (%s)\n", current.ID, current.Name)
	return nil
}

// ── export ───────────────────────────────────────────────────────────────────

// runCollectionExport writes the positions of one or more collections to a
// new database file through Database.ExportCollections — the same call the
// GUI's collection export dialog makes, so the file carries the collections'
// membership, the analyses and comments asked for, and a watermark when the
// producer marks it (see the export command and ADR-0007).
func (cli *CLI) runCollectionExport(args []string) error {
	fs, dbPath := collectionFlagSet("export", "Export one or more collections to a new database file.",
		"blunderdb collection export --db database.db --id 3 --out openings.db",
		"blunderdb collection export --db database.db --id 3,4 --out openings.db --comments=false",
		"blunderdb collection export --db database.db --id 3 --out cours.db --watermark \"Cours du 12 mars\"")
	ids := fs.String("id", "", "Collection ID(s) to export, comma-separated (required)")
	out := fs.String("out", "", "Path of the database file to write (required)")
	includeAnalysis := fs.Bool("analysis", true, "Include analyses")
	includeComments := fs.Bool("comments", true, "Include comments")
	watermark := fs.String("watermark", "", "Mark the exported file with where it comes from")
	watermarkNote := fs.String("watermark-note", "", "Free text attached to the watermark (terms of use, contact)")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *ids == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --id")
	}
	if *out == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --out")
	}
	collectionIDs, err := parseIDList(*ids)
	if err != nil {
		return fmt.Errorf("invalid --id: %w", err)
	}
	if len(collectionIDs) == 0 {
		return fmt.Errorf("missing required flag: --id")
	}
	for _, id := range collectionIDs {
		if c, err := cli.db.GetCollectionByID(id); err != nil || c == nil {
			return fmt.Errorf("collection with ID %d not found", id)
		}
	}

	metadata := make(map[string]string)
	if version, err := cli.db.GetDatabaseVersion(); err == nil {
		metadata["database_version"] = version
	}

	fmt.Printf("Exporting %d collection(s) to: %s\n", len(collectionIDs), *out)
	if err := cli.db.ExportCollections(*out, collectionIDs, metadata, *includeAnalysis, *includeComments, *watermark, *watermarkNote); err != nil {
		return fmt.Errorf("failed to export collections: %w", err)
	}
	if info, err := os.Stat(*out); err == nil {
		fmt.Printf("Successfully exported collections (%d bytes)\n", info.Size())
	} else {
		fmt.Println("Successfully exported collections")
	}
	return nil
}

// xgidOf renders a Position as an XGID the way the GUI's clipboard does when
// no imported analysis carries one (generateXGID in
// frontend/src/services/positionService.js): the match length is taken as the
// larger away score — a Position does not retain the real one — and the
// Crawford flag is raised whenever a player is 1-away. Keep the two in step.
func xgidOf(pos *Position) string {
	board := make([]byte, 26)
	for i := 0; i < 26; i++ {
		p := pos.Board.Points[i]
		switch {
		case p.Checkers <= 0:
			board[i] = '-'
		case p.Color == Black:
			board[i] = byte('A' + p.Checkers - 1)
		default:
			board[i] = byte('a' + p.Checkers - 1)
		}
	}
	cubeOwner := 0
	switch pos.Cube.Owner {
	case Black:
		cubeOwner = 1
	case White:
		cubeOwner = -1
	}
	dice := "00"
	if pos.DecisionType == CheckerAction {
		dice = fmt.Sprintf("%d%d", pos.Dice[0], pos.Dice[1])
	}
	matchLength, score1, score2, crawford := 0, 0, 0, 0
	if pos.Score[0] != -1 && pos.Score[1] != -1 {
		matchLength = max(pos.Score[0], pos.Score[1])
		score1, score2 = matchLength-pos.Score[0], matchLength-pos.Score[1]
		if pos.Score[0] == 1 || pos.Score[1] == 1 {
			crawford = 1
		}
	}
	turn := 1
	if pos.PlayerOnRoll != Black {
		turn = -1
	}
	return fmt.Sprintf("%s:%d:%d:%d:%s:%d:%d:%d:%d:0",
		board, pos.Cube.Value, cubeOwner, turn, dice, score1, score2, crawford, matchLength)
}

// printJSON writes v to stdout, indented, the way every --format json output
// of the CLI is written.
func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
