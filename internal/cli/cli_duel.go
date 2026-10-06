package cli

import (
	"context"
	"flag"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/duel"
)

// runDuel is `blunderdb duel`: a Duel played without an interface, one Action
// per call (ADR-0072 rule 11). Every call is its own process, so it keeps
// nothing in memory: an Action on a Duel in suspense opens it in the same write, and
// the draft in the database is all there is of it. A Duel driven this way has
// neither Cadence nor decision times — a decision whose time was not watched
// is recorded as unknown, never as short.
//
// A Side is "external" (the CLI plays it) or "bot:<level>" (the engine plays
// it, within the call that gave it the trait). "external:<configuration>@<engine>"
// is an external Side that declares the Bot playing behind it, recorded in the
// Match's origin as declared, not attested.
func (cli *CLI) runDuel(args []string) error {
	if len(args) < 1 {
		cli.printDuelUsage()
		return fmt.Errorf("missing duel sub-command")
	}
	sub := strings.ToLower(args[0])
	if sub == "--help" || sub == "-h" || sub == "help" {
		cli.printDuelUsage()
		return nil
	}
	run, ok := cli.duelHandlers()[sub]
	if !ok {
		cli.printDuelUsage()
		return fmt.Errorf("unknown duel sub-command: %s", args[0])
	}
	return run(args[1:])
}

// duelHandlers returns the sub-command table of `blunderdb duel`.
func (cli *CLI) duelHandlers() map[string]func([]string) error {
	act := func(kind duel.PlayKind) func([]string) error {
		return func(args []string) error { return cli.runDuelAct(kind, args) }
	}
	return map[string]func([]string) error{
		"create":     cli.runDuelCreate,
		"show":       cli.runDuelShow,
		"list":       cli.runDuelList,
		"roll":       act(duel.PlayRoll),
		"move":       act(duel.PlayMove),
		"double":     act(duel.PlayDouble),
		"take":       act(duel.PlayTake),
		"pass":       act(duel.PlayPass),
		"resign":     act(duel.PlayResign),
		"stop":       cli.runDuelStop,
		"discard":    cli.runDuelDiscard,
		"forfeit":    cli.runDuelForfeit,
		"contribute": cli.runDuelContribute,
	}
}

// DuelSubcommands returns the sub-commands of `blunderdb duel`, sorted — the
// exported view cmd/cli-doc-gen walks.
func (cli *CLI) DuelSubcommands() []string {
	h := cli.duelHandlers()
	names := make([]string, 0, len(h))
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (cli *CLI) printDuelUsage() {
	fmt.Println("Usage: blunderdb duel <sub-command> [options]")
	fmt.Println()
	fmt.Println("Play a Duel one Action per call, without an interface. The Duel lives in the")
	fmt.Println("database: each call resumes it, plays one Action and writes it back. A Duel")
	fmt.Println("played this way has no Cadence and no decision times.")
	fmt.Println()
	fmt.Println("Sub-commands:")
	fmt.Println("  create    Start a Duel (match length or money, the two Sides, a starting position)")
	fmt.Println("  show      Show a Duel: score, what it awaits, the legal plays")
	fmt.Println("  list      List the Duels in suspense")
	fmt.Println("  roll      Roll the dice, without doubling")
	fmt.Println("  move      Play a checker play, in notation (24/18 13/11)")
	fmt.Println("  double    Offer the cube")
	fmt.Println("  take      Take a double")
	fmt.Println("  pass      Pass on a double")
	fmt.Println("  resign    Resign the game (single, gammon or backgammon)")
	fmt.Println("  stop      Stop a money session and keep its Match as it stands")
	fmt.Println("  discard   Throw the Duel away: nothing of it is written")
	fmt.Println("  forfeit   Give the match up (--side): the Match is written won by the other Side")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  blunderdb duel create --db database.db --length 5 --name1 Alice --name2 Bob")
	fmt.Println("  blunderdb duel show --db database.db --id 1")
	fmt.Println("  blunderdb duel move --db database.db --id 1 --play \"24/18 13/11\"")
	fmt.Println("  blunderdb duel stop --db database.db --id 1")
	fmt.Println()
	fmt.Println("Use 'blunderdb duel <sub-command> --help' for the options of a sub-command.")
}

// duelFlagSet builds the FlagSet shared by every sub-command.
func duelFlagSet(sub, summary string, examples ...string) (*flag.FlagSet, *string, *string) {
	fs := flag.NewFlagSet("duel "+sub, flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	format := fs.String("format", "text", "Output format: text or json")
	fs.Usage = func() {
		fmt.Printf("Usage: blunderdb duel %s [options]\n\n%s\n\nOptions:\n", sub, summary)
		fs.PrintDefaults()
		if len(examples) > 0 {
			fmt.Println()
			fmt.Println("Examples:")
			for _, ex := range examples {
				fmt.Println("  " + ex)
			}
		}
	}
	return fs, dbPath, format
}

// duelOpen parses fs, checks --db and --format, and opens the database and the
// Duel service over it.
func (cli *CLI) duelOpen(fs *flag.FlagSet, dbPath, format *string, args []string) (*duel.Service, error) {
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *dbPath == "" {
		fs.Usage()
		return nil, fmt.Errorf("missing required flag: --db")
	}
	if f := strings.ToLower(*format); f != "text" && f != "json" {
		return nil, fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return nil, err
	}
	store, _ := database.CurrentStore(cli.db)
	if store == nil {
		return nil, fmt.Errorf("no database open")
	}
	return duel.New(store, duel.Options{}), nil
}

func (cli *CLI) runDuelCreate(args []string) error {
	fs, dbPath, format := duelFlagSet("create", "Start a Duel and play on to the first Decision of an external Side.",
		"blunderdb duel create --db database.db --length 5 --name1 Alice --name2 Bob",
		"blunderdb duel create --db database.db --money --jacoby",
		"blunderdb duel create --db database.db --length 7 --side2 bot:normal",
		"blunderdb duel create --db database.db --length 5 --name1 Alice --side2 external:normal@v1.6.0",
		"blunderdb duel create --db database.db --length 3 --side1 bot:instant --side2 bot:instant")
	length := fs.Int("length", 0, "Match length in points, 1 to 25")
	money := fs.Bool("money", false, "A money session instead of a match")
	jacoby := fs.Bool("jacoby", false, "With --money: play the Jacoby rule")
	side1 := fs.String("side1", "external", "Player 1's Side: external, external:<configuration>@<engine> (an external Side declaring its Bot), or bot:<level> (instant, normal, thorough)")
	side2 := fs.String("side2", "external", "Player 2's Side: external, external:<configuration>@<engine> (an external Side declaring its Bot), or bot:<level> (instant, normal, thorough); two Bots play the whole match in this call")
	name1 := fs.String("name1", "", "Player 1's name")
	name2 := fs.String("name2", "", "Player 2's name")
	start := fs.String("start", "", "XGID of the Position the first game begins at (default: the opening position)")
	discard := fs.Bool("discard-at-end", false, "Throw the draft away when the match is won instead of writing the Match")
	combined := fs.Bool("combined-seed", false, "Roll nothing before each external Side has contributed to the seed (duel contribute)")
	svc, err := cli.duelOpen(fs, dbPath, format, args)
	if err != nil {
		return err
	}
	if *money == (*length > 0) {
		return fmt.Errorf("give --length or --money, one of the two")
	}
	set := duel.Settings{
		MatchLength: *length, Jacoby: *jacoby, DiscardAtEnd: *discard, CombinedSeed: *combined,
	}
	for i, v := range []struct{ spec, name string }{{*side1, *name1}, {*side2, *name2}} {
		if set.Sides[i], err = parseSide(v.spec, v.name); err != nil {
			return err
		}
	}
	if *start != "" {
		pos, err := domain.DecodeXGID(*start)
		if err != nil {
			return fmt.Errorf("invalid --start: %w", err)
		}
		set.Start = &pos
	}
	st, err := svc.Create(context.Background(), "", set)
	if err != nil {
		return fmt.Errorf("creating the duel: %w", err)
	}
	return printDuel(st, *format)
}

// parseSide reads "external", "external:<configuration>@<engine>" or
// "bot:<level>" into the Side a Duel records.
func parseSide(v, name string) (duel.SideSpec, error) {
	level, isBot := strings.CutPrefix(v, "bot:")
	declared, isDeclared := strings.CutPrefix(v, "external:")
	switch {
	case v == "external" || v == "":
		return duel.SideSpec{Kind: duel.SideExternal, Name: name}, nil
	case isBot:
		return duel.SideSpec{Kind: duel.SideBot, Level: level}, nil
	case isDeclared:
		if conf, engine, ok := strings.Cut(declared, "@"); ok && conf != "" && engine != "" {
			return duel.SideSpec{Kind: duel.SideExternal, Name: name,
				Declared: &duel.DeclaredBot{Configuration: conf, Engine: engine}}, nil
		}
	}
	return duel.SideSpec{}, fmt.Errorf("side %q: external, external:<configuration>@<engine>, or bot:<level> (%s)", v, strings.Join(duel.BotLevels, ", "))
}

func (cli *CLI) runDuelShow(args []string) error {
	fs, dbPath, format := duelFlagSet("show", "Show a Duel in suspense: its score, what it awaits and, when it awaits a play, the legal ones.",
		"blunderdb duel show --db database.db --id 1")
	id := fs.Int64("id", 0, "Duel id (required)")
	svc, err := cli.duelOpen(fs, dbPath, format, args)
	if err != nil {
		return err
	}
	if *id == 0 {
		return fmt.Errorf("missing required flag: --id")
	}
	st, err := svc.Get(context.Background(), "", *id)
	if err != nil {
		return fmt.Errorf("reading duel %d: %w", *id, err)
	}
	return printDuel(st, *format)
}

func (cli *CLI) runDuelList(args []string) error {
	fs, dbPath, format := duelFlagSet("list", "List the Duels in suspense, the most recently played first.",
		"blunderdb duel list --db database.db")
	svc, err := cli.duelOpen(fs, dbPath, format, args)
	if err != nil {
		return err
	}
	rows, err := svc.List(context.Background(), "")
	if err != nil {
		return err
	}
	if strings.ToLower(*format) == "json" {
		if rows == nil {
			rows = []duel.Summary{}
		}
		return printJSON(rows)
	}
	if len(rows) == 0 {
		fmt.Println("No Duel in suspense.")
		return nil
	}
	for _, r := range rows {
		fmt.Printf("%d\t%s\t%s\n", r.ID, r.UpdatedAt, r.Label)
	}
	return nil
}

// runDuelAct plays one Action of the Side the Duel awaits, or of --side.
func (cli *CLI) runDuelAct(kind duel.PlayKind, args []string) error {
	fs, dbPath, format := duelFlagSet(string(kind), "Play one Action, then play on to the next Decision of an external Side. The Side defaults to the one the Duel awaits.",
		"blunderdb duel "+string(kind)+" --db database.db --id 1")
	id := fs.Int64("id", 0, "Duel id (required)")
	side := fs.Int("side", 0, "The Side that plays, 1 or 2 (default: the one the Duel awaits)")
	revision := fs.Int64("revision", 0, "Refuse the Action unless the Duel is at this revision (default: no check)")
	var plan *string
	var level *int
	switch kind {
	case duel.PlayMove:
		plan = fs.String("play", "", "The play in notation, as show lists them (required)")
	case duel.PlayResign:
		level = fs.Int("level", 1, "1 single, 2 gammon, 3 backgammon")
	}
	svc, err := cli.duelOpen(fs, dbPath, format, args)
	if err != nil {
		return err
	}
	if *id == 0 {
		return fmt.Errorf("missing required flag: --id")
	}
	ctx := context.Background()
	// A Duel in suspense is opened by the gesture itself.
	rev := *revision
	cur, err := svc.Get(ctx, "", *id)
	if err != nil {
		return err
	}
	if len(cur.AwaitingContribution) > 0 {
		return fmt.Errorf("duel %d awaits a contribution to its seed from Side %d before any play (duel contribute)", *id, cur.AwaitingContribution[0]+1)
	}
	if cur.Awaiting == nil {
		return fmt.Errorf("duel %d awaits nothing", *id)
	}
	p := duel.Play{Side: cur.Awaiting.Side, Kind: kind}
	if *side != 0 {
		if *side != 1 && *side != 2 {
			return fmt.Errorf("--side is 1 or 2")
		}
		p.Side = *side - 1
	}
	if level != nil {
		p.Level = *level
	}
	if plan != nil {
		steps, err := stepsOf(cur.Awaiting, *plan)
		if err != nil {
			return err
		}
		p.Steps = steps
	}
	st, err := svc.Play(ctx, "", *id, rev, p)
	if err != nil {
		return fmt.Errorf("duel %d: %w", *id, err)
	}
	return printDuel(st, *format)
}

// stepsOf finds, among the legal plays of the awaited Decision, the one the
// notation names — whatever the order its checkers are written in.
func stepsOf(d *duel.Decision, notation string) ([]domain.CheckerStep, error) {
	if d.Kind != duel.DecideMove {
		return nil, fmt.Errorf("the Duel awaits a %s decision, not a checker play", d.Kind)
	}
	want := strings.Fields(notation)
	if len(want) == 0 {
		return nil, fmt.Errorf("--play is required: the legal plays are listed by `duel show`")
	}
	slices.Sort(want)
	for _, lp := range domain.LegalMoves(&d.Position) {
		got := strings.Fields(domain.Notation(lp.Steps, int(d.Position.PlayerOnRoll)))
		slices.Sort(got)
		if slices.Equal(got, want) {
			return lp.Steps, nil
		}
	}
	return nil, fmt.Errorf("%q is not a legal play of this roll: `duel show` lists them", notation)
}

// runDuelForfeit has a Side give the match up: the Duel ends, won by the
// other Side, and its Match is written as a match won.
func (cli *CLI) runDuelForfeit(args []string) error {
	fs, dbPath, format := duelFlagSet("forfeit", "Give the match up: the game in progress goes to the other Side for the points that bring it to the length (at money play, a single at the cube's value), and the Match is written won by the other Side.",
		"blunderdb duel forfeit --db database.db --id 1 --side 1")
	id := fs.Int64("id", 0, "Duel id (required)")
	side := fs.Int("side", 0, "The Side that gives the match up, 1 or 2 (required)")
	revision := fs.Int64("revision", 0, "Refuse unless the Duel is at this revision (default: no check)")
	svc, err := cli.duelOpen(fs, dbPath, format, args)
	if err != nil {
		return err
	}
	if *id == 0 {
		return fmt.Errorf("missing required flag: --id")
	}
	if *side != 1 && *side != 2 {
		return fmt.Errorf("--side is required, 1 or 2")
	}
	ctx := context.Background()
	// A Duel in suspense is opened by the gesture itself.
	rev := *revision
	st, err := svc.Forfeit(ctx, "", *id, rev, *side-1)
	if err != nil {
		return fmt.Errorf("duel %d: %w", *id, err)
	}
	return printDuel(st, *format)
}

func (cli *CLI) runDuelContribute(args []string) error {
	fs, dbPath, format := duelFlagSet("contribute", "Contribute to the seed of a Duel created with --combined-seed: once per external Side, before the first roll, which the last contribution starts.",
		"blunderdb duel contribute --db database.db --id 1 --side 1 --value \"my own randomness\"")
	id := fs.Int64("id", 0, "Duel id (required)")
	side := fs.Int("side", 0, "The contributing Side, 1 or 2 (required)")
	value := fs.String("value", "", fmt.Sprintf("The contribution, 1 to %d bytes of text (required)", duel.MaxContribution))
	revision := fs.Int64("revision", 0, "Refuse unless the Duel is at this revision (default: no check)")
	svc, err := cli.duelOpen(fs, dbPath, format, args)
	if err != nil {
		return err
	}
	if *id == 0 {
		return fmt.Errorf("missing required flag: --id")
	}
	if *side != 1 && *side != 2 {
		return fmt.Errorf("--side is required, 1 or 2")
	}
	ctx := context.Background()
	// A Duel in suspense is opened by the gesture itself.
	rev := *revision
	st, err := svc.Contribute(ctx, "", *id, rev, *side-1, *value)
	if err != nil {
		return fmt.Errorf("duel %d: %w", *id, err)
	}
	return printDuel(st, *format)
}

func (cli *CLI) runDuelStop(args []string) error { return cli.runDuelEnd("stop", true, args) }

func (cli *CLI) runDuelDiscard(args []string) error { return cli.runDuelEnd("discard", false, args) }

func (cli *CLI) runDuelEnd(sub string, keep bool, args []string) error {
	summary := "Throw the Duel away: nothing of it is written, and the Match is not made."
	if keep {
		summary = "Stop a money session and write its Match as it stands: an unfinished game keeps no winner. A match in points is refused: it is written only once won (suspend, forfeit or discard it)."
	}
	fs, dbPath, format := duelFlagSet(sub, summary, "blunderdb duel "+sub+" --db database.db --id 1")
	id := fs.Int64("id", 0, "Duel id (required)")
	revision := fs.Int64("revision", 0, "Refuse unless the Duel is at this revision (default: no check)")
	svc, err := cli.duelOpen(fs, dbPath, format, args)
	if err != nil {
		return err
	}
	if *id == 0 {
		return fmt.Errorf("missing required flag: --id")
	}
	ctx := context.Background()
	// A Duel in suspense is opened by the gesture itself.
	rev := *revision
	st, err := svc.Stop(ctx, "", *id, rev, keep)
	if err != nil {
		return fmt.Errorf("duel %d: %w", *id, err)
	}
	return printDuel(st, *format)
}

// printDuel writes a Duel as the CLI shows it.
func printDuel(st *duel.State, format string) error {
	if strings.ToLower(format) == "json" {
		return printJSON(st)
	}
	h := st.Header
	if h.MatchLength == 0 {
		fmt.Printf("Duel %d (revision %d): %s vs %s, money\n", st.ID, st.Revision, h.Player1, h.Player2)
	} else {
		fmt.Printf("Duel %d (revision %d): %s vs %s, to %d\n", st.ID, st.Revision, h.Player1, h.Player2, h.MatchLength)
	}
	fmt.Printf("Score: %d-%d   Actions: %d   Fingerprint: %s\n", st.Score[0], st.Score[1], len(st.Actions), st.Fingerprint)
	if e := st.Ended; e != nil {
		switch {
		case e.Discarded:
			fmt.Println("Ended: thrown away, nothing written.")
		default:
			fmt.Printf("Ended: Match %d written (seed %s)", e.MatchID, e.DiceSeed)
			if e.Forfeited != 0 {
				fmt.Printf(", forfeited by Side %d", e.Forfeited)
			}
			fmt.Println(".")
		}
		return nil
	}
	for _, side := range st.AwaitingContribution {
		fmt.Printf("Awaits: Side %d (%s), a contribution to the seed (duel contribute)\n", side+1, st.Sides[side].Name)
	}
	d := st.Awaiting
	if d == nil {
		return nil
	}
	pos := d.Position
	fmt.Printf("Awaits: Side %d (%s), a %s decision\n", d.Side+1, st.Sides[d.Side].Name, d.Kind)
	fmt.Println("Position:", domain.EncodeXGID(&pos))
	if d.Kind == duel.DecideMove {
		fmt.Printf("Dice: %d-%d. Legal plays:\n", pos.Dice[0], pos.Dice[1])
		for _, lp := range domain.LegalMoves(&pos) {
			fmt.Println("  " + domain.Notation(lp.Steps, int(pos.PlayerOnRoll)))
		}
	}
	return nil
}
