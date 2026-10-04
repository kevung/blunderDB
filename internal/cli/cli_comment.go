package cli

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// runComment is `blunderdb comment`: the comments of one position, each signed
// by its author, so that several people can annotate the same file. The daemon
// serves the same family as /v1/comments.*.
func (cli *CLI) runComment(args []string) error {
	if len(args) < 1 {
		cli.printCommentUsage()
		return fmt.Errorf("missing comment sub-command")
	}
	sub := strings.ToLower(args[0])
	if sub == "--help" || sub == "-h" || sub == "help" {
		cli.printCommentUsage()
		return nil
	}
	run, ok := cli.commentHandlers()[sub]
	if !ok {
		cli.printCommentUsage()
		return fmt.Errorf("unknown comment sub-command: %s", args[0])
	}
	return run(args[1:])
}

// commentHandlers returns the sub-command table of `blunderdb comment`.
func (cli *CLI) commentHandlers() map[string]func([]string) error {
	return map[string]func([]string) error{
		"add":  cli.runCommentAdd,
		"list": cli.runCommentList,
	}
}

// CommentSubcommands returns the sub-commands of `blunderdb comment`, sorted:
// the exported view cmd/cli-doc-gen walks.
func (cli *CLI) CommentSubcommands() []string {
	h := cli.commentHandlers()
	names := make([]string, 0, len(h))
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (cli *CLI) printCommentUsage() {
	fmt.Println("Usage: blunderdb comment <sub-command> [options]")
	fmt.Println()
	fmt.Println("Comments on a position. A position holds any number of them, each signed")
	fmt.Println("by the person who wrote it.")
	fmt.Println()
	fmt.Println("Sub-commands:")
	fmt.Println("  add       Add a comment to a position, signed with --author")
	fmt.Println("  list      List the comments of a position, or of the whole database")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  blunderdb comment add --db database.db --position 412 --text \"Cube too early\" --author Alice")
	fmt.Println("  blunderdb comment list --db database.db --position 412")
	fmt.Println("  blunderdb comment list --db database.db --author Alice --format json")
	fmt.Println()
	fmt.Println("Use 'blunderdb comment <sub-command> --help' for the options of a sub-command.")
}

func commentFlagSet(sub, summary string, examples ...string) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet("comment "+sub, flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	fs.Usage = func() {
		fmt.Printf("Usage: blunderdb comment %s [options]\n\n%s\n\nOptions:\n", sub, summary)
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		for _, ex := range examples {
			fmt.Println("  " + ex)
		}
	}
	return fs, dbPath
}

func (cli *CLI) runCommentAdd(args []string) error {
	fs, dbPath := commentFlagSet("add", "Add a comment to a position.",
		"blunderdb comment add --db database.db --position 412 --text \"Cube too early\" --author Alice")
	position := fs.Int64("position", 0, "Position id (required)")
	text := fs.String("text", "", "Comment text (required)")
	author := fs.String("author", "", "Who signs the comment (empty: unsigned)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --db")
	}
	if *position == 0 {
		return fmt.Errorf("missing required flag: --position")
	}
	if strings.TrimSpace(*text) == "" {
		return fmt.Errorf("missing required flag: --text")
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}
	cli.db.SetCommentAuthor(*author)
	if err := cli.db.AddComment(*position, *text); err != nil {
		return fmt.Errorf("add comment: %w", err)
	}
	fmt.Printf("Comment added to position %d.\n", *position)
	return nil
}

func (cli *CLI) runCommentList(args []string) error {
	fs, dbPath := commentFlagSet("list", "List the comments of a position, or of the whole database.",
		"blunderdb comment list --db database.db --position 412",
		"blunderdb comment list --db database.db --author Alice --format json")
	position := fs.Int64("position", 0, "Only the comments of this position id (0: every position)")
	author := fs.String("author", "", "Only the comments signed by this author (whole name, any case)")
	format := fs.String("format", "text", "Output format: text or json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --db")
	}
	if f := strings.ToLower(*format); f != "text" && f != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}
	var (
		all []CommentEntry
		err error
	)
	if *position != 0 {
		all, err = cli.db.GetCommentsByPosition(*position)
	} else {
		all, err = cli.db.GetAllComments()
	}
	if err != nil {
		return fmt.Errorf("list comments: %w", err)
	}
	// The whole name, any case — Unicode case — as the search's au"…" token
	// and --comment-author compare it, so the two never disagree on who
	// signed what.
	want := storage.NormalizeCommentAuthor(*author)
	entries := all[:0:0]
	for _, e := range all {
		if want == "" || strings.EqualFold(e.Author, want) {
			entries = append(entries, e)
		}
	}
	if strings.ToLower(*format) == "json" {
		return printJSON(entries)
	}
	if len(entries) == 0 {
		fmt.Println("No comments.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPosition\tAuthor\tCreated\tText")
	fmt.Fprintln(w, "--\t--------\t------\t-------\t----")
	for _, e := range entries {
		fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%s\n", e.ID, e.PositionID, e.Author, e.CreatedAt, strings.ReplaceAll(e.Text, "\n", " "))
	}
	return w.Flush()
}
