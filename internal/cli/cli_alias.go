package cli

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// PlayersSubcommands returns the sub-commands of `blunderdb players` that carry
// their own flags: the exported view cmd/cli-doc-gen walks.
func (cli *CLI) PlayersSubcommands() []string {
	return []string{"main", "matches", "merge", "swap"}
}

// runPlayers is `blunderdb players alias …`: the other spellings of a player.
func (cli *CLI) runPlayers(args []string) error {
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "merge":
			return cli.runPlayersMerge(args[1:])
		case "swap":
			return cli.runPlayersSwap(args[1:])
		case "main":
			return cli.runPlayersMain(args[1:])
		case "matches":
			return cli.runPlayersMatches(args[1:])
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
		fmt.Fprintln(cli.stdout(), "Usage: blunderdb players merge --db FILE --into CANONICAL NAME [NAME...]")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Make every NAME another spelling of CANONICAL (same as `players alias add`")
		fmt.Fprintln(cli.stdout(), "for each): the stats, the players table and the search read them as one")
		fmt.Fprintln(cli.stdout(), "person, every later import stores CANONICAL, and the matches already stored")
		fmt.Fprintln(cli.stdout(), "keep the names their files wrote.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Options:")
		fs.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		fmt.Fprintln(cli.stdout(), "  blunderdb players merge --db base.db --into \"John Doe\" \"Doe J.\" \"J. Doe\"")
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
	fmt.Fprintf(cli.stdout(), "%d name(s) merged into %q\n", fs.NArg(), strings.TrimSpace(*into))
	return nil
}

// runPlayersSwap is `players swap`: the window's inversion of a match's two
// players, through the same SwapMatchPlayers as the daemon's matches.swapPlayers.
func (cli *CLI) runPlayersSwap(args []string) error {
	fs := flag.NewFlagSet("players swap", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	fs.Usage = func() {
		fmt.Fprintln(cli.stdout(), "Usage: blunderdb players swap --db FILE MATCH_ID")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Swap the two players of a match: the file named them in the wrong order.")
		fmt.Fprintln(cli.stdout(), "No position or analysis changes: each player keeps their statistics, under the other seat.")
		fmt.Fprintln(cli.stdout(), "Refused while a transcription draft is open on the match, and for a match played here.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Options:")
		fs.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		fmt.Fprintln(cli.stdout(), "  blunderdb players swap --db base.db 42")
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
	fmt.Fprintf(cli.stdout(), "players of match %d swapped\n", id)
	return nil
}

// runPlayersMain is `players main`: the star of the window's player list,
// through the same SetMainPlayer as the daemon's players.main.set.
func (cli *CLI) runPlayersMain(args []string) error {
	fs := flag.NewFlagSet("players main", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	dryRun := fs.Bool("dry-run", false, "Show the matches the designation would swap, change nothing")
	clearMain := fs.Bool("clear", false, "Forget the Main player (no match is swapped back)")
	format := fs.String("format", "text", "Output format: text or json")
	fs.Usage = func() {
		fmt.Fprintln(cli.stdout(), "Usage: blunderdb players main --db FILE [--dry-run] [NAME]")
		fmt.Fprintln(cli.stdout(), "       blunderdb players main --db FILE --clear")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Without NAME, print the Main player. With NAME, make them the Main player:")
		fmt.Fprintln(cli.stdout(), "every match where they (or one of their aliases) sit second has its players")
		fmt.Fprintln(cli.stdout(), "swapped, all in one transaction, and every later import puts them first.")
		fmt.Fprintln(cli.stdout(), "A match with a transcription draft open, or played here, is left as it is.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Options:")
		fs.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		fmt.Fprintln(cli.stdout(), "  blunderdb players main --db base.db --dry-run \"John Doe\"")
		fmt.Fprintln(cli.stdout(), "  blunderdb players main --db base.db \"John Doe\"")
	}
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if fs.NArg() > 1 || (*clearMain && fs.NArg() > 0) {
		fs.Usage()
		return fmt.Errorf("usage: blunderdb players main --db FILE [--dry-run] [NAME] | --clear")
	}
	switch {
	case *clearMain:
		if err := cli.db.ClearMainPlayer(); err != nil {
			return err
		}
		fmt.Fprintln(cli.stdout(), "Main player forgotten")
		return nil
	case fs.NArg() == 0:
		name, err := cli.db.MainPlayer()
		if err != nil {
			return err
		}
		if strings.ToLower(*format) == "json" {
			return cli.printJSON(map[string]string{"name": name})
		}
		if name == "" {
			fmt.Fprintln(cli.stdout(), "No Main player.")
		} else {
			fmt.Fprintln(cli.stdout(), name)
		}
		return nil
	}
	var plan database.MainPlayerPlan
	var err error
	if *dryRun {
		plan, err = cli.db.PlanMainPlayer(fs.Arg(0))
	} else {
		plan, err = cli.db.SetMainPlayer(fs.Arg(0))
	}
	if err != nil {
		return err
	}
	if strings.ToLower(*format) == "json" {
		return cli.printJSON(plan)
	}
	verb := "swapped"
	if *dryRun {
		verb = "to swap"
	}
	fmt.Fprintf(cli.stdout(), "Main player: %s\n%d match(es) %s: %s\n", plan.Name, len(plan.Swap), verb, joinIDs(plan.Swap))
	if len(plan.Refused) > 0 {
		fmt.Fprintf(cli.stdout(), "%d match(es) left (draft open or played here): %s\n", len(plan.Refused), joinIDs(plan.Refused))
	}
	return nil
}

// runPlayersMatches is `players matches`: the ids a player's match count
// opens, through the same PlayerMatchIDs as the daemon's players.matchIds.
func (cli *CLI) runPlayersMatches(args []string) error {
	fs := flag.NewFlagSet("players matches", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	fs.Usage = func() {
		fmt.Fprintln(cli.stdout(), "Usage: blunderdb players matches --db FILE NAME")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Print the ids of the matches NAME plays, at either seat and under any alias.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Options:")
		fs.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		fmt.Fprintln(cli.stdout(), "  blunderdb players matches --db base.db \"John Doe\"")
	}
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("usage: blunderdb players matches --db FILE NAME")
	}
	ids, err := cli.db.PlayerMatchIDs(fs.Arg(0))
	if err != nil {
		return err
	}
	for _, id := range ids {
		fmt.Fprintln(cli.stdout(), id)
	}
	return nil
}

// runEvents is `blunderdb events alias …`: the other spellings of an event.
func (cli *CLI) runEvents(args []string) error { return cli.runAliasCommand("events", "event", args) }

func (cli *CLI) runAliasCommand(cmd, kind string, args []string) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		cli.printAliasUsage(cmd, kind)
		if len(args) == 0 {
			return fmt.Errorf("missing %s sub-command", cmd)
		}
		return nil
	}
	if strings.ToLower(args[0]) != "alias" {
		cli.printAliasUsage(cmd, kind)
		return fmt.Errorf("unknown %s sub-command: %s", cmd, args[0])
	}
	args = args[1:]
	if len(args) == 0 || isHelpArg(args[0]) {
		cli.printAliasUsage(cmd, kind)
		if len(args) == 0 {
			return fmt.Errorf("missing %s alias action", cmd)
		}
		return nil
	}
	action := strings.ToLower(args[0])
	fs := flag.NewFlagSet(cmd+" alias "+action, flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	format := fs.String("format", "text", "Output format of list and suggest: text or json")
	fs.Usage = func() { cli.printAliasUsage(cmd, kind) }
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
		fmt.Fprintf(cli.stdout(), "%q is now an alias of %q\n", strings.TrimSpace(fs.Arg(0)), strings.TrimSpace(fs.Arg(1)))
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
		fmt.Fprintf(cli.stdout(), "alias %q removed\n", strings.TrimSpace(fs.Arg(0)))
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
			return cli.printJSON(list)
		}
		w := tabwriter.NewWriter(cli.stdout(), 0, 0, 2, ' ', 0)
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
			return cli.printJSON(list)
		}
		cli.printAliasSuggestions(cmd, list)
		return nil
	}
	cli.printAliasUsage(cmd, kind)
	return fmt.Errorf("unknown %s alias action: %s", cmd, args[0])
}

func (cli *CLI) printAliasSuggestions(cmd string, list []database.AliasSuggestion) {
	if len(list) == 0 {
		fmt.Fprintln(cli.stdout(), "No suggestion.")
		return
	}
	for _, s := range list {
		for _, a := range s.Aliases {
			fmt.Fprintf(cli.stdout(), "blunderdb %s alias add --db FILE %q %q\n", cmd, a, s.Canonical)
		}
	}
}

func isHelpArg(s string) bool {
	s = strings.ToLower(s)
	return s == "--help" || s == "-h" || s == "help"
}

func (cli *CLI) printAliasUsage(cmd, kind string) {
	fmt.Fprintf(cli.stdout(), "Usage: blunderdb %s alias <add|list|remove|suggest> --db FILE [arguments]\n\n", cmd)
	fmt.Fprintf(cli.stdout(), "Record the other spellings of a %s's name. An import stores the canonical\n", kind)
	fmt.Fprintln(cli.stdout(), "name where the file writes an alias; the stats, the players table and the")
	fmt.Fprintln(cli.stdout(), "search read every spelling as one. The matches already stored keep the")
	fmt.Fprintln(cli.stdout(), "names their files wrote, and so do the match fingerprints.")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Actions:")
	fmt.Fprintln(cli.stdout(), "  add ALIAS CANONICAL  Make ALIAS a spelling of CANONICAL")
	fmt.Fprintln(cli.stdout(), "  list                 List the aliases")
	fmt.Fprintln(cli.stdout(), "  remove ALIAS         Forget an alias")
	fmt.Fprintln(cli.stdout(), "  suggest              Propose the names that differ only by case, accents,")
	fmt.Fprintln(cli.stdout(), "                       punctuation or word order (nothing is recorded)")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Options:")
	fmt.Fprintln(cli.stdout(), "  --db FILE        Path to the database file (required)")
	fmt.Fprintln(cli.stdout(), "  --format FORMAT  Output of list and suggest: text or json (default text)")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Examples:")
	fmt.Fprintf(cli.stdout(), "  blunderdb %s alias add --db base.db \"Doe J.\" \"John Doe\"\n", cmd)
	fmt.Fprintf(cli.stdout(), "  blunderdb %s alias list --db base.db --format json\n", cmd)
	fmt.Fprintf(cli.stdout(), "  blunderdb %s alias suggest --db base.db\n", cmd)
	if cmd == "players" {
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Also: `players merge --db FILE --into CANONICAL NAME...`,")
		fmt.Fprintln(cli.stdout(), "`players swap --db FILE MATCH_ID`, `players main --db FILE [NAME]` and")
		fmt.Fprintln(cli.stdout(), "`players matches --db FILE NAME` (see their --help).")
	}
}
