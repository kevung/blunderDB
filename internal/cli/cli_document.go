package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
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
	fmt.Fprintln(cli.stdout(), "page, label and two certainties: its checkers (sure, to-verify, uncertain) and its")
	fmt.Fprintln(cli.stdout(), "context (read, default, to-verify, uncertain). An entry is sure only when both are.")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Sub-commands:")
	fmt.Fprintln(cli.stdout(), "  analyze  Read the diagrams of a PDF or of page images into a relevé; nothing is written")
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
	fs := cli.documentFileFlagSet("analyze", "<file.pdf|page-images-folder|page.png>",
		"Read the diagrams of a document into a relevé, in the document's order, each with its page,\n"+
			"label, chapter (from the PDF's bookmarks) and state: sure when the diagram's own\n"+
			"cross-checks hold (15 checkers a side, the printed pip counts of both players), to-verify\n"+
			"or uncertain otherwise, with the doubt named. A PDF is read when its diagrams are drawn\n"+
			"with eXtreme Gammon's board font (XG's PDF prints), from its glyphs. A scanned book is\n"+
			"read by the structure of each diagram, from a scanned PDF (its pages composed from their\n"+
			"images; JBIG2 and JPEG 2000 images are not decoded) or from its pages as images (a folder\n"+
			"of PNG or JPEG files, one a page, in the order of their names). The printed pip counts\n"+
			"are read in the PDF's text layer and in the image; when the two disagree the entry is not\n"+
			"sure. Pages that cannot be read are named on standard error. Nothing is written to a\n"+
			"database: check the relevé, then import it with\n"+
			"'blunderdb document import'.",
		"blunderdb document analyze book.pdf -o releve.json",
		"blunderdb document analyze pages/ -o releve.json --title \"My book\"",
		"blunderdb document analyze book.pdf -o releve.json --title \"My book\" --author \"Its author\"")
	output := fs.String("o", "", "Write the relevé to this file (default: standard output)")
	title := fs.String("title", "", "Title of the document (default: the PDF's metadata, else the file or folder name)")
	author := fs.String("author", "", "Author of the document (default: the PDF's metadata)")
	path, err := documentFileFlags(fs, args, "document")
	if err != nil {
		return err
	}
	warn := func(s string) { fmt.Fprintln(os.Stderr, "blunderdb document analyze:", s) }
	r, err := document.AnalyzeFile(path, document.AnalyzeOptions{Title: *title, Author: *author, Warn: warn})
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
	printCertainties(out, s)
	fmt.Fprintf(out, "Positions:  %d distinct, %d repeated\n", s.Positions, s.Repeated)
	fmt.Fprintf(out, "Relevé:     %s\n", *output)
	return nil
}

// printCertainties names what the state of the entries rests on: how their
// context is known, the document's defaults, the entries whose checkers
// alone are certified and the complete ones, so none is taken for sure.
func printCertainties(out io.Writer, s document.Summary) {
	fmt.Fprintf(out, "Context:    read %d, default %d, to verify %d\n", s.ContextRead, s.ContextDefault, s.ContextToVerify)
	if d := s.Defaults; d != nil {
		state := "not confirmed: set \"confirmed\": true once the document's diagrams follow them"
		if d.Confirmed {
			state = "confirmed"
		}
		fmt.Fprintf(out, "Defaults:   %s (%s), %s\n", d.Description, d.Context, state)
		if len(d.Printed) > 0 {
			fmt.Fprintf(out, "Printed:    %s, read beside each diagram (learned from the document)\n", strings.Join(d.Printed, ", "))
		}
		if dir := d.Direction; dir != nil {
			state := "not confirmed: check it on a few diagrams, then set \"direction\": {\"confirmed\": true}"
			if dir.Confirmed {
				state = "confirmed"
			}
			fmt.Fprintf(out, "Direction:  %s, %s\n", dir.Description, state)
		}
	}
	if s.CheckersConfirmed > 0 {
		fmt.Fprintf(out, "Checkers:   %d confirmed, their context alone to verify (cube, dice, player on roll, score)\n", s.CheckersConfirmed)
	}
	if s.Complete > 0 {
		fmt.Fprintf(out, "Complete:   %d with 15 + 15 checkers read with no doubt and a certified context (imported with --accept-complete)\n", s.Complete)
	}
	if s.CompleteDirectionUnconfirmed > 0 {
		fmt.Fprintf(out, "Complete:   %d more held back until the direction of play is confirmed\n", s.CompleteDirectionUnconfirmed)
	}
}

func (cli *CLI) runDocumentCheck(args []string) error {
	fs := cli.documentFlagSet("check",
		"Validate a relevé (format, ranks, legal positions; version 1 is still read) and count its\n"+
			"entries by state and by how their context is known. Nothing is read from or written to a\n"+
			"database.",
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
	printCertainties(out, s)
	fmt.Fprintf(out, "Positions:  %d distinct, %d repeated\n", s.Positions, s.Repeated)
	switch {
	case len(s.Unverified) == 0:
		fmt.Fprintln(out, "Unverified: none")
	case s.Complete > 0 && len(s.UnverifiedComplete) == 0:
		fmt.Fprintf(out, "Unverified: rank %s (import refused without --accept-complete or --force)\n", joinRanks(s.Unverified))
	case s.Complete > 0:
		fmt.Fprintf(out, "Unverified: rank %s (import refused without --force; rank %s with --accept-complete)\n", joinRanks(s.Unverified), joinRanks(s.UnverifiedComplete))
	default:
		fmt.Fprintf(out, "Unverified: rank %s (import refused without --force)\n", joinRanks(s.Unverified))
	}
	switch {
	case len(s.Fallback) > 0 && !slices.Equal(s.Fallback, s.FallbackComplete):
		fmt.Fprintf(out, "Fallback:   rank %s imported with --fallback under the context its position carries (rank %s with --fallback --accept-complete), the fields to check named in its reference\n", joinRanks(s.Fallback), joinRanks(s.FallbackComplete))
	case len(s.Fallback) > 0:
		fmt.Fprintf(out, "Fallback:   rank %s imported with --fallback under the context its position carries, the fields to check named in its reference\n", joinRanks(s.Fallback))
	case len(s.FallbackComplete) > 0:
		fmt.Fprintf(out, "Fallback:   rank %s imported with --fallback --accept-complete under the context its position carries, the fields to check named in its reference\n", joinRanks(s.FallbackComplete))
	}
	return nil
}

func (cli *CLI) runDocumentImport(args []string) error {
	fs := cli.documentFlagSet("import",
		"Import the positions of a relevé, in the document's order, into a collection named after\n"+
			"the document (its title, author and edition as description; each position's page and\n"+
			"label as a comment). An entry is sure when its checkers are and its context is certified:\n"+
			"read, or under the document's defaults once \"confirmed\" is set on them. An entry neither\n"+
			"sure nor verified refuses the import unless --force; --accept-complete accepts as a whole\n"+
			"the entries of 15 + 15 checkers read with no doubt whose context is certified; --fallback\n"+
			"imports, rather than refuse, the entries whose checkers are certified and whose context is\n"+
			"to verify, under the context their position carries, each reference naming the fields to\n"+
			"check against the diagram (the document's defaults confirmed). A refused import writes\n"+
			"nothing. Importing the same relevé again adds nothing.",
		"blunderdb document import --db database.db releve.json",
		"blunderdb document import --db database.db releve.json --accept-complete",
		"blunderdb document import --db database.db releve.json --fallback",
		"blunderdb document import --db database.db releve.json --collection \"My book\" --force")
	dbPath := fs.String("db", "", "Path to the database file (required)")
	collection := fs.String("collection", "", "Name of the collection (default: the document's title)")
	acceptComplete := fs.Bool("accept-complete", false, "Import the entries of 15 + 15 checkers read with no doubt whose context is certified")
	fallback := fs.Bool("fallback", false, "Import the entries whose checkers are certified and whose context is to verify, under the context their position carries")
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
	res, err := cli.db.ImportDocumentReleve(string(data), *collection, *acceptComplete, *fallback, *force)
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
	if res.Complete > 0 {
		fmt.Fprintf(out, "Complete: %d entries of 15 + 15 checkers read with no doubt\n", res.Complete)
	}
	if res.Fallback > 0 {
		fmt.Fprintf(out, "Fallback: %d entries under the context their position carries, the fields to check named in their reference\n", res.Fallback)
	}
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
