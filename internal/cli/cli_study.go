package cli

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
)

// runStudy handles the study command: the library-wide backlog of the
// reference player's blunders nobody has dealt with, and the "studied" mark
// that takes a position out of it. The mark is the user's own data, written
// only by `study mark` and withdrawn by `study unmark`.
func (cli *CLI) runStudy(args []string) error {
	if len(args) < 1 {
		cli.printStudyUsage()
		return fmt.Errorf("missing study sub-command")
	}
	sub := strings.ToLower(args[0])
	if sub == "--help" || sub == "-h" || sub == "help" {
		cli.printStudyUsage()
		return nil
	}
	run, ok := cli.studyHandlers()[sub]
	if !ok {
		cli.printStudyUsage()
		return fmt.Errorf("unknown study sub-command: %s", args[0])
	}
	return run(args[1:])
}

// studyHandlers returns the sub-command table of `blunderdb study`.
func (cli *CLI) studyHandlers() map[string]func([]string) error {
	return map[string]func([]string) error{
		"queue":  cli.runStudyQueue,
		"mark":   cli.runStudyMark,
		"unmark": cli.runStudyUnmark,
	}
}

// StudySubcommands returns the sub-commands of `blunderdb study`, sorted —
// the exported view cmd/cli-doc-gen walks.
func (cli *CLI) StudySubcommands() []string {
	h := cli.studyHandlers()
	names := make([]string, 0, len(h))
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (cli *CLI) printStudyUsage() {
	fmt.Println("Usage: blunderdb study <sub-command> [options]")
	fmt.Println()
	fmt.Println("The study backlog: your blunders, across every import, that nothing has dealt with yet")
	fmt.Println("(no comment, no Anki card, no collection, no \"studied\" mark), costliest first.")
	fmt.Println()
	fmt.Println("Sub-commands:")
	fmt.Println("  queue    List the backlog")
	fmt.Println("  mark     Mark a position studied: it leaves the backlog")
	fmt.Println("  unmark   Withdraw the mark: the position returns to the backlog")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  blunderdb study queue --db database.db --limit 20")
	fmt.Println("  blunderdb study mark --db database.db --id 1234")
	fmt.Println()
	fmt.Println("Use 'blunderdb study <sub-command> --help' for the options of a sub-command.")
}

func studyFlagSet(sub, summary string, examples ...string) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet("study "+sub, flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	fs.Usage = func() {
		fmt.Printf("Usage: blunderdb study %s [options]\n\n%s\n\nOptions:\n", sub, summary)
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

func (cli *CLI) runStudyQueue(args []string) error {
	fs, dbPath := studyFlagSet("queue",
		"List your unhandled blunders across the whole library, costliest first.\nYour name is the database's reference player (see `blunderdb players`).",
		"blunderdb study queue --db database.db",
		"blunderdb study queue --db database.db --limit 10 --format json")
	limit := fs.Int("limit", 0, "Maximum number of positions (default and ceiling: 50)")
	format := fs.String("format", "text", "Output format: text or json")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	entries, err := cli.db.StudyBacklog(*limit)
	if err != nil {
		return fmt.Errorf("study backlog: %w", err)
	}
	if strings.EqualFold(*format, "json") {
		return printJSON(entries)
	}
	if len(entries) == 0 {
		fmt.Println("Nothing left to study.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "#\tPosition\tKind\tCost\tMatch")
	for i, e := range entries {
		kind := "checker"
		if e.IsCube {
			kind = "cube"
		}
		fmt.Fprintf(w, "%d\t%d\t%s\t%.3f\t%s\n", i+1, e.PositionID, kind, float64(e.ErrorMP)/1000, e.Label)
	}
	return w.Flush()
}

func (cli *CLI) runStudyMark(args []string) error {
	return cli.setStudied(args, true)
}

func (cli *CLI) runStudyUnmark(args []string) error {
	return cli.setStudied(args, false)
}

func (cli *CLI) setStudied(args []string, studied bool) error {
	sub, summary := "mark", "Mark a position studied: it leaves the study backlog."
	if !studied {
		sub, summary = "unmark", "Withdraw a position's studied mark: it returns to the study backlog."
	}
	fs, dbPath := studyFlagSet(sub, summary, "blunderdb study "+sub+" --db database.db --id 1234")
	id := fs.Int64("id", 0, "Position ID (required)")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if err := requireID(fs, "id", *id); err != nil {
		return err
	}
	if err := cli.db.SetPositionStudied(*id, studied); err != nil {
		return fmt.Errorf("study %s: %w", sub, err)
	}
	if studied {
		fmt.Printf("Position %d marked studied.\n", *id)
	} else {
		fmt.Printf("Position %d is back in the study backlog.\n", *id)
	}
	return nil
}
