package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// runPlayers is `blunderdb players alias …`: the other spellings of a player.
func (cli *CLI) runPlayers(args []string) error {
	return cli.runAliasCommand("players", "player", args)
}

// runEvents is `blunderdb events alias …`: the other spellings of an event.
func (cli *CLI) runEvents(args []string) error { return cli.runAliasCommand("events", "event", args) }

func (cli *CLI) runAliasCommand(cmd, kind string, args []string) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		printAliasUsage(cmd, kind)
		if len(args) == 0 {
			return fmt.Errorf("missing %s sub-command", cmd)
		}
		return nil
	}
	if strings.ToLower(args[0]) != "alias" {
		printAliasUsage(cmd, kind)
		return fmt.Errorf("unknown %s sub-command: %s", cmd, args[0])
	}
	args = args[1:]
	if len(args) == 0 || isHelpArg(args[0]) {
		printAliasUsage(cmd, kind)
		if len(args) == 0 {
			return fmt.Errorf("missing %s alias action", cmd)
		}
		return nil
	}
	action := strings.ToLower(args[0])
	fs := flag.NewFlagSet(cmd+" alias "+action, flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	format := fs.String("format", "text", "Output format of list and suggest: text or json")
	fs.Usage = func() { printAliasUsage(cmd, kind) }
	switch action {
	case "add":
		if err := cli.collectionOpen(fs, dbPath, args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 2 {
			return fmt.Errorf("usage: blunderdb %s alias add --db FILE ALIAS CANONICAL", cmd)
		}
		if err := cli.db.SetAlias(kind, fs.Arg(0), fs.Arg(1)); err != nil {
			return err
		}
		fmt.Printf("%q is now an alias of %q\n", strings.TrimSpace(fs.Arg(0)), strings.TrimSpace(fs.Arg(1)))
		return nil
	case "remove":
		if err := cli.collectionOpen(fs, dbPath, args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("usage: blunderdb %s alias remove --db FILE ALIAS", cmd)
		}
		removed, err := cli.db.RemoveAlias(kind, fs.Arg(0))
		if err != nil {
			return err
		}
		if !removed {
			return fmt.Errorf("%q is not an alias", fs.Arg(0))
		}
		fmt.Printf("alias %q removed\n", strings.TrimSpace(fs.Arg(0)))
		return nil
	case "list":
		if err := cli.collectionOpen(fs, dbPath, args[1:]); err != nil {
			return err
		}
		list, err := cli.db.ListAliases(kind)
		if err != nil {
			return err
		}
		if strings.ToLower(*format) == "json" {
			return printJSON(list)
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ALIAS\tCANONICAL")
		for _, a := range list {
			fmt.Fprintf(w, "%s\t%s\n", a.Alias, a.Canonical)
		}
		return w.Flush()
	case "suggest":
		if err := cli.collectionOpen(fs, dbPath, args[1:]); err != nil {
			return err
		}
		list, err := cli.db.SuggestAliases(kind)
		if err != nil {
			return err
		}
		if strings.ToLower(*format) == "json" {
			return printJSON(list)
		}
		printAliasSuggestions(cmd, list)
		return nil
	}
	printAliasUsage(cmd, kind)
	return fmt.Errorf("unknown %s alias action: %s", cmd, args[0])
}

func printAliasSuggestions(cmd string, list []database.AliasSuggestion) {
	if len(list) == 0 {
		fmt.Println("No suggestion.")
		return
	}
	for _, s := range list {
		for _, a := range s.Aliases {
			fmt.Printf("blunderdb %s alias add --db FILE %q %q\n", cmd, a, s.Canonical)
		}
	}
}

func isHelpArg(s string) bool {
	s = strings.ToLower(s)
	return s == "--help" || s == "-h" || s == "help"
}

func printAliasUsage(cmd, kind string) {
	fmt.Printf("Usage: blunderdb %s alias <add|list|remove|suggest> --db FILE [arguments]\n\n", cmd)
	fmt.Printf("Record the other spellings of a %s's name. An import stores the canonical\n", kind)
	fmt.Println("name where the file writes an alias; the stats, the players table and the")
	fmt.Println("search read every spelling as one. The matches already stored keep the")
	fmt.Println("names their files wrote, and so do the match fingerprints.")
	fmt.Println()
	fmt.Println("Actions:")
	fmt.Println("  add ALIAS CANONICAL  Make ALIAS a spelling of CANONICAL")
	fmt.Println("  list                 List the aliases")
	fmt.Println("  remove ALIAS         Forget an alias")
	fmt.Println("  suggest              Propose the names that differ only by case, accents,")
	fmt.Println("                       punctuation or word order (nothing is recorded)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --db FILE        Path to the database file (required)")
	fmt.Println("  --format FORMAT  Output of list and suggest: text or json (default text)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  blunderdb %s alias add --db base.db \"Doe J.\" \"John Doe\"\n", cmd)
	fmt.Printf("  blunderdb %s alias list --db base.db --format json\n", cmd)
	fmt.Printf("  blunderdb %s alias suggest --db base.db\n", cmd)
}
