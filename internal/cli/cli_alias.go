package cli

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// runPlayers is `blunderdb players alias …`: the other spellings of a player.
func (cli *CLI) runPlayers(args []string) error {
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "merge":
			return cli.runPlayersMerge(args[1:])
		case "swap":
			return cli.runPlayersSwap(args[1:])
		}
	}
	return cli.runAliasCommand("players", "player", args)
}

// runPlayersMerge is `players merge`: the window's merge of the matches panel,
// through the same MergePlayers as the daemon's matches.mergePlayers. It
// records aliases, so the matches keep the names their files wrote.
func (cli *CLI) runPlayersMerge(args []string) error {
	fs := flag.NewFlagSet("players merge", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	into := fs.String("into", "", "The canonical name the others become spellings of (required)")
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb players merge --db FILE --into CANONICAL NAME [NAME...]")
		fmt.Println()
		fmt.Println("Make every NAME another spelling of CANONICAL (same as `players alias add`")
		fmt.Println("for each): the stats, the players table and the search read them as one")
		fmt.Println("person, every later import stores CANONICAL, and the matches already stored")
		fmt.Println("keep the names their files wrote.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb players merge --db base.db --into \"John Doe\" \"Doe J.\" \"J. Doe\"")
	}
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *into == "" || fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("usage: blunderdb players merge --db FILE --into CANONICAL NAME [NAME...]")
	}
	if err := cli.db.MergePlayers(fs.Args(), *into); err != nil {
		return err
	}
	fmt.Printf("%d name(s) merged into %q\n", fs.NArg(), strings.TrimSpace(*into))
	return nil
}

// runPlayersSwap is `players swap`: the window's inversion of a match's two
// players, through the same SwapMatchPlayers as the daemon's matches.swapPlayers.
func (cli *CLI) runPlayersSwap(args []string) error {
	fs := flag.NewFlagSet("players swap", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb players swap --db FILE MATCH_ID")
		fmt.Println()
		fmt.Println("Swap the two players of a match: the file named them in the wrong order.")
		fmt.Println("Every position of the match is rewritten from the other player's side.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb players swap --db base.db 42")
	}
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("usage: blunderdb players swap --db FILE MATCH_ID")
	}
	id, err := strconv.ParseInt(fs.Arg(0), 10, 64)
	if err != nil || id <= 0 {
		return fmt.Errorf("invalid match id %q", fs.Arg(0))
	}
	if err := cli.db.SwapMatchPlayers(id); err != nil {
		return err
	}
	fmt.Printf("players of match %d swapped\n", id)
	return nil
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
	if cmd == "players" {
		fmt.Println()
		fmt.Println("Also: `players merge --db FILE --into CANONICAL NAME...` and")
		fmt.Println("`players swap --db FILE MATCH_ID` (see their --help).")
	}
}
