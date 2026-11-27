package cli

import (
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// runArchive handles the archive command: the paper archive of matches
// (ADR-0119), QR codes on printable A4 sheets, and its reimport.
func (cli *CLI) runArchive(args []string) error {
	if len(args) < 1 {
		cli.printArchiveUsage()
		return fmt.Errorf("missing archive sub-command")
	}
	sub := strings.ToLower(args[0])
	if sub == "--help" || sub == "-h" || sub == "help" {
		cli.printArchiveUsage()
		return nil
	}
	run, ok := cli.archiveHandlers()[sub]
	if !ok {
		cli.printArchiveUsage()
		return fmt.Errorf("unknown archive sub-command: %s", args[0])
	}
	return run(args[1:])
}

func (cli *CLI) archiveHandlers() map[string]func([]string) error {
	return map[string]func([]string) error{
		"export": cli.runArchiveExport,
		"import": cli.runArchiveImport,
	}
}

// ArchiveSubcommands returns the sub-commands of `blunderdb archive`, sorted —
// the exported view cmd/cli-doc-gen walks.
func (cli *CLI) ArchiveSubcommands() []string {
	h := cli.archiveHandlers()
	names := make([]string, 0, len(h))
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (cli *CLI) printArchiveUsage() {
	out := cli.stdout()
	fmt.Fprintln(out, "Usage: blunderdb archive <sub-command> [options]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "A paper archive prints matches as QR codes on A4 sheets, one match a code,")
	fmt.Fprintln(out, "and reads them back from the page or from a scan or photograph of it.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Sub-commands:")
	fmt.Fprintln(out, "  export  Write matches as a printable HTML archive page")
	fmt.Fprintln(out, "  import  Import the matches of archive pages, scans or photographs")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Use 'blunderdb archive <sub-command> --help' for the options of a sub-command.")
}

func (cli *CLI) archiveFlagSet(sub, args, summary string, examples ...string) *flag.FlagSet {
	fs := flag.NewFlagSet("archive "+sub, flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(cli.stdout(), "Usage: blunderdb archive %s [options] %s\n\n%s\n\nOptions:\n", sub, args, summary)
		fs.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		for _, ex := range examples {
			fmt.Fprintln(cli.stdout(), "  "+ex)
		}
	}
	return fs
}

func (cli *CLI) runArchiveExport(args []string) error {
	fs := cli.archiveFlagSet("export", "",
		"Write matches as a self-contained HTML page of A4 sheets of QR codes (error correction M):\n"+
			"one match a code when it fits, a parity code every five that rebuilds one unreadable code,\n"+
			"a white gutter on both fold lines. The matches are embedded too, so the page imports without\n"+
			"reading a code. Analyses are not archived: they are computed again after an import.",
		"blunderdb archive export --db database.db --output archive.html",
		"blunderdb archive export --db database.db --match 12,15 --output archive.html --origin \"Club de Lyon\"")
	dbPath := fs.String("db", "", "Path to the database file (required)")
	output := fs.String("output", "", "Path of the HTML page to write (required)")
	matches := fs.String("match", "", "Comma-separated match ids (default: the whole library)")
	origin := fs.String("origin", "", "Origin line printed at the foot of each sheet")
	format := fs.String("format", "text", "Output format: text or json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" || *output == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --db and --output")
	}
	ids, err := parseIDList(*matches)
	if err != nil {
		return err
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}
	res, err := cli.db.ExportPaperArchive(ids, *output, *origin)
	if err != nil {
		return err
	}
	if *format == "json" {
		return cli.printJSON(res)
	}
	out := cli.stdout()
	fmt.Fprintf(out, "Archived %d matches in %d QR codes (%d bytes of matches) to %s\n", res.Matches, res.Codes, res.Bytes, *output)
	for _, s := range res.Skipped {
		fmt.Fprintf(out, "Skipped match %d: %s\n", s.MatchID, s.Reason)
	}
	return nil
}

func (cli *CLI) runArchiveImport(args []string) error {
	fs := cli.archiveFlagSet("import", "<file>...",
		"Import the matches of paper archives: the exported HTML page (its embedded matches, no code\n"+
			"reading), a PDF the interface printed (an archive, a match sheet or a position list, its\n"+
			"codes attached; any other PDF is refused), or scans and photographs of its sheets (PNG,\n"+
			"JPEG), several codes an image, any number of images in any order. A lost code is rebuilt from its parity code. Matches go\n"+
			"through the ordinary import: an import batch, and a match already held is a duplicate.\n"+
			"The reimport codes of a printed position list (sheet --reimport-code) bring their\n"+
			"positions back, each saved on its own.",
		"blunderdb archive import --db database.db archive.html",
		"blunderdb archive import --db database.db page1.jpg page2.jpg",
		"blunderdb archive import --db database.db match-sheet.pdf")
	dbPath := fs.String("db", "", "Path to the database file (required)")
	format := fs.String("format", "text", "Output format: text or json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	paths := fs.Args()
	if *dbPath == "" || len(paths) == 0 {
		fs.Usage()
		return fmt.Errorf("missing required flag --db or archive file")
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}
	return cli.importArchive(paths, *format)
}

// importArchive imports archive files inside an import batch, as `import`
// does for every other format.
func (cli *CLI) importArchive(paths []string, format string) error {
	batchID := cli.beginImportBatch(strings.Join(paths, ", "), "archive")
	var failures domain.ImportReport
	res, err := cli.db.ImportPaperArchive(paths)
	if err != nil {
		recordFailure(&failures, paths[0], err)
		cli.finishImportBatch(batchID, failures)
		return fmt.Errorf("failed to import archive: %w", err)
	}
	for _, f := range res.Failed {
		recordFailure(&failures, paths[0], fmt.Errorf("%s", f))
	}
	report := cli.finishImportBatch(batchID, failures)
	if format == "json" {
		return cli.printJSON(res)
	}
	out := cli.stdout()
	fmt.Fprintf(out, "Read %d archive codes (%d rebuilt from parity)\n", res.Codes, res.Repaired)
	fmt.Fprintf(out, "Imported %d matches, %d already in the library\n", len(res.Imported), res.Duplicates)
	if res.Positions+res.KnownPositions > 0 {
		fmt.Fprintf(out, "Imported %d positions, %d already in the library\n", res.Positions, res.KnownPositions)
	}
	if len(res.Missing) > 0 {
		fmt.Fprintf(out, "Unreadable codes: %s\n", joinRanks(res.Missing))
	}
	for _, f := range res.Failed {
		fmt.Fprintf(out, "Failed: %s\n", f)
	}
	cli.printImportReport(report)
	return nil
}
