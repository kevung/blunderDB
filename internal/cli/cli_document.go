package cli

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/document"
)

// runDocument handles the document command: the Relevé of a document — the
// positions a book, an article or a web page shows, in its order — checked or
// imported into a Collection (ADR-0117).
func (cli *CLI) runDocument(args []string) error {
	if len(args) < 1 {
		cli.printDocumentUsage()
		return fmt.Errorf("missing document sub-command")
	}
	sub := strings.ToLower(args[0])
	if sub == "--help" || sub == "-h" || sub == "help" {
		cli.printDocumentUsage()
		return nil
	}
	run, ok := cli.documentHandlers()[sub]
	if !ok {
		cli.printDocumentUsage()
		return fmt.Errorf("unknown document sub-command: %s", args[0])
	}
	return run(args[1:])
}

// documentHandlers returns the sub-command table of `blunderdb document`.
func (cli *CLI) documentHandlers() map[string]func([]string) error {
	return map[string]func([]string) error{
		"analyze": cli.runDocumentAnalyze,
		"check":   cli.runDocumentCheck,
		"import":  cli.runDocumentImport,
	}
}

// DocumentSubcommands returns the sub-commands of `blunderdb document`,
// sorted — the exported view cmd/cli-doc-gen walks.
func (cli *CLI) DocumentSubcommands() []string {
	h := cli.documentHandlers()
	names := make([]string, 0, len(h))
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (cli *CLI) printDocumentUsage() {
	fmt.Fprintln(cli.stdout(), "Usage: blunderdb document <sub-command> [options] <file>")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "A relevé lists the positions a document shows, in its order, each with its")
	fmt.Fprintln(cli.stdout(), "page, label and state (sure, to-verify, uncertain).")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Sub-commands:")
	fmt.Fprintln(cli.stdout(), "  analyze  Read the diagrams of a PDF into a relevé; nothing is written to a database")
	fmt.Fprintln(cli.stdout(), "  check    Validate a relevé and count its entries; nothing is written")
	fmt.Fprintln(cli.stdout(), "  import   Import the positions of a relevé into a collection named after the document")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Use 'blunderdb document <sub-command> --help' for the options of a sub-command.")
}

// documentFlags parses flags given before or after the file's path, and
// returns that path.
func documentFlags(fs *flag.FlagSet, args []string) (string, error) {
	return documentFileFlags(fs, args, "relevé")
}

func documentFileFlags(fs *flag.FlagSet, args []string, what string) (string, error) {
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if fs.NArg() < 1 {
		fs.Usage()
		return "", fmt.Errorf("missing the %s file", what)
	}
	path := fs.Arg(0)
	if err := fs.Parse(fs.Args()[1:]); err != nil {
		return "", err
	}
	if fs.NArg() > 0 {
		fs.Usage()
		return "", fmt.Errorf("expected exactly one %s file", what)
	}
	return path, nil
}

func (cli *CLI) documentFlagSet(sub, summary string, examples ...string) *flag.FlagSet {
	return cli.documentFileFlagSet(sub, "<releve.json>", summary, examples...)
}

func (cli *CLI) documentFileFlagSet(sub, file, summary string, examples ...string) *flag.FlagSet {
	fs := flag.NewFlagSet("document "+sub, flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(cli.stdout(), "Usage: blunderdb document %s [options] %s\n\n%s\n\nOptions:\n", sub, file, summary)
		fs.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		for _, ex := range examples {
			fmt.Fprintln(cli.stdout(), "  "+ex)
		}
	}
	return fs
}

func readReleve(path string) (*document.Releve, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return document.Decode(data)
}

func (cli *CLI) runDocumentAnalyze(args []string) error {
	fs := cli.documentFileFlagSet("analyze", "<file.pdf>",
		"Read the diagrams of a PDF into a relevé, in the document's order, each with its page,\n"+
			"label, chapter (from the PDF's bookmarks) and state: sure when the diagram's own\n"+
			"cross-checks hold (15 checkers a side, the printed pip counts of both players), to-verify\n"+
			"or uncertain otherwise, with the doubt named. Only diagrams drawn with eXtreme Gammon's\n"+
			"board font (XG's PDF prints) are read; any other PDF is refused. Nothing is written to a\n"+
			"database: check the relevé, then import it with 'blunderdb document import'.",
		"blunderdb document analyze book.pdf -o releve.json",
		"blunderdb document analyze book.pdf -o releve.json --title \"My book\" --author \"Its author\"")
	output := fs.String("o", "", "Write the relevé to this file (default: standard output)")
	title := fs.String("title", "", "Title of the document (default: the PDF's metadata, else the file name)")
	author := fs.String("author", "", "Author of the document (default: the PDF's metadata)")
	path, err := documentFileFlags(fs, args, "PDF")
	if err != nil {
		return err
	}
	r, err := document.AnalyzePDFFile(path, document.AnalyzeOptions{Title: *title, Author: *author})
	if err != nil {
		return err
	}
	data, err := r.Encode()
	if err != nil {
		return err
	}
	if *output == "" {
		_, err := cli.stdout().Write(data)
		return err
	}
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		return err
	}
	s := document.Summarize(r)
	out := cli.stdout()
	fmt.Fprintf(out, "Document:   %s\n", s.Title)
	fmt.Fprintf(out, "Entries:    %d (sure %d, to verify %d, uncertain %d)\n", s.Entries, s.Sure, s.ToVerify, s.Uncertain)
	fmt.Fprintf(out, "Positions:  %d distinct, %d repeated\n", s.Positions, s.Repeated)
	fmt.Fprintf(out, "Relevé:     %s\n", *output)
	return nil
}

func (cli *CLI) runDocumentCheck(args []string) error {
	fs := cli.documentFlagSet("check",
		"Validate a relevé (format, ranks, legal positions) and count its entries by state.\nNothing is read from or written to a database.",
		"blunderdb document check releve.json",
		"blunderdb document check releve.json --format json")
	format := fs.String("format", "text", "Output format: text or json")
	path, err := documentFlags(fs, args)
	if err != nil {
		return err
	}
	r, err := readReleve(path)
	if err != nil {
		return err
	}
	s := document.Summarize(r)
	if *format == "json" {
		return cli.printJSON(s)
	}
	out := cli.stdout()
	fmt.Fprintf(out, "Document:   %s\n", s.Title)
	fmt.Fprintf(out, "Entries:    %d (sure %d, to verify %d, uncertain %d; verified %d)\n", s.Entries, s.Sure, s.ToVerify, s.Uncertain, s.Verified)
	fmt.Fprintf(out, "Positions:  %d distinct, %d repeated\n", s.Positions, s.Repeated)
	if len(s.Unverified) > 0 {
		fmt.Fprintf(out, "Unverified: rank %s (import refused without --force)\n", joinRanks(s.Unverified))
	} else {
		fmt.Fprintln(out, "Unverified: none")
	}
	return nil
}

func (cli *CLI) runDocumentImport(args []string) error {
	fs := cli.documentFlagSet("import",
		"Import the positions of a relevé, in the document's order, into a collection named after\n"+
			"the document (its title, author and edition as description; each position's page and\n"+
			"label as a comment). An entry neither sure nor verified refuses the import unless --force;\n"+
			"a refused import writes nothing. Importing the same relevé again adds nothing.",
		"blunderdb document import --db database.db releve.json",
		"blunderdb document import --db database.db releve.json --collection \"My book\" --force")
	dbPath := fs.String("db", "", "Path to the database file (required)")
	collection := fs.String("collection", "", "Name of the collection (default: the document's title)")
	force := fs.Bool("force", false, "Import entries neither sure nor verified")
	format := fs.String("format", "text", "Output format: text or json")
	path, err := documentFlags(fs, args)
	if err != nil {
		return err
	}
	if *dbPath == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --db")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}
	res, err := cli.db.ImportDocumentReleve(string(data), *collection, *force)
	if err != nil {
		return err
	}
	if *format == "json" {
		return cli.printJSON(res)
	}
	out := cli.stdout()
	verb := "Updated"
	if res.CreatedCollection {
		verb = "Created"
	}
	fmt.Fprintf(out, "%s collection %q (id %d)\n", verb, res.Collection, res.CollectionID)
	fmt.Fprintf(out, "Entries: %d, distinct positions: %d (%d new in the library), added to the collection: %d, repeated: %d\n",
		res.Entries, res.Positions, res.NewPositions, res.Added, res.Repeated)
	if res.Forced > 0 {
		fmt.Fprintf(out, "Forced: %d entries neither sure nor verified\n", res.Forced)
	}
	return nil
}

func joinRanks(ranks []int) string {
	s := make([]string, len(ranks))
	for i, r := range ranks {
		s[i] = fmt.Sprint(r)
	}
	return strings.Join(s, ", ")
}
