package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// statsFilterFlags declares the filter flags `stats effect` and `stats
// biases` share with `stats plan`.
type statsFilterFlags struct {
	dbPath, player, tournament, from, to, decisionType, format *string
}

func newStatsFilterFlags(fs *flag.FlagSet) statsFilterFlags {
	return statsFilterFlags{
		dbPath:       fs.String("db", "", "Path to the database file (required)"),
		player:       fs.String("player", "", "Only this player's decisions"),
		tournament:   fs.String("tournament", "", "Filter by tournament IDs, comma-separated"),
		from:         fs.String("from", "", "Start date filter YYYY-MM-DD"),
		to:           fs.String("to", "", "End date filter YYYY-MM-DD"),
		decisionType: fs.String("decision-type", "all", "Decision type: all, checker, or cube"),
		format:       fs.String("format", "text", "Output format: text or json"),
	}
}

// open parses the filter and opens the database.
func (f statsFilterFlags) open(cli *CLI, fs *flag.FlagSet) (StatsFilter, error) {
	if *f.dbPath == "" {
		fs.Usage()
		return StatsFilter{}, fmt.Errorf("missing required flag: --db")
	}
	filter, err := buildStatsFilter(*f.player, *f.tournament, *f.from, *f.to, *f.decisionType)
	if err != nil {
		return filter, err
	}
	return filter, cli.initDatabase(*f.dbPath)
}

// computeInterruptibly runs a stats computation Ctrl-C can cancel.
func computeInterruptibly(text bool, run func(ctx context.Context) error) error {
	err := withInterruptibleContext(func() {
		if text {
			fmt.Println("\nCancelling...")
		}
	}, run)
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("stats cancelled")
	}
	return err
}

func (cli *CLI) runStatsEffect(args []string) error {
	fs := flag.NewFlagSet("stats effect", flag.ContinueOnError)
	ff := newStatsFilterFlags(fs)
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb stats effect --db <file> [options]")
		fmt.Println()
		fmt.Println("Did studying a family change how much it costs in real play? For each")
		fmt.Println("recurring-error family (plan of play x theme) one of whose positions was")
		fmt.Println("studied — marked studied, reviewed in Anki or answered in a quiz — the")
		fmt.Println("family's MWC loss per decision of its plan and kind, in the matches played")
		fmt.Println("before the day of the first study action and in those played after it")
		fmt.Println("(ADR-0079). The gain is the before rate minus the after rate, with a 95%")
		fmt.Println("interval; a direction is stated only when both windows hold at least")
		fmt.Printf("%d decisions and the interval excludes zero.\n", storage.StudyEffectMinDecisions)
		fmt.Println("A change, not an effect: a family is studied because it cost, so part of")
		fmt.Println("any gain is regression to the mean.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb stats effect --db database.db --player \"Alice\"")
		fmt.Println("  blunderdb stats effect --db database.db --player \"Alice\" --format json")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	filter, err := ff.open(cli, fs)
	if err != nil {
		return err
	}
	text := strings.ToLower(*ff.format) != "json"
	var effect *storage.StudyEffect
	if err := computeInterruptibly(text, func(ctx context.Context) error {
		effect, err = cli.db.ComputeStudyEffectCtx(ctx, filter)
		return err
	}); err != nil {
		return fmt.Errorf("study effect: %w", err)
	}
	if !text {
		return printJSON(effect)
	}
	printStudyEffect(effect)
	return nil
}

// printStudyEffect writes one row per studied family. Rates are MWC
// percentage points per 100 decisions.
func printStudyEffect(e *storage.StudyEffect) {
	fmt.Printf("Before/after study — loss in MWC points per 100 decisions of the family's plan and kind;\n")
	fmt.Printf("95%% interval; at least %d decisions on each side.\n\n", e.MinDecisions)
	if len(e.Families) == 0 {
		fmt.Println("No family has been studied yet: mark positions studied, review them in Anki or")
		fmt.Println("answer them in a quiz.")
	} else {
		per100 := func(v float64) float64 { return 100 * 100 * v }
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PLAN\tKIND\tTHEME\tSTUDIED ON\tSTUDIED\tBEFORE\t(n)\tAFTER\t(n)\tGAIN\t95% INTERVAL\tVERDICT")
		for _, f := range e.Families {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%.2f\t%d\t%.2f\t%d\t%+.2f\t[%+.2f, %+.2f]\t%s\n",
				f.GameType, f.Kind, f.Theme, f.StudiedOn, f.Studied,
				per100(f.Before.Rate), f.Before.Decisions, per100(f.After.Rate), f.After.Decisions,
				per100(f.Gain), per100(f.Low), per100(f.High), f.Verdict)
		}
		w.Flush()
	}
	if e.Unstudied > 0 {
		fmt.Printf("\n%d families of the filter have not been studied yet.\n", e.Unstudied)
	}
	fmt.Println()
	fmt.Println("A change, not an effect: a family is studied because it cost, and nothing here")
	fmt.Println("controls for the opponents, the format or the dice.")
}

func (cli *CLI) runStatsBiases(args []string) error {
	fs := flag.NewFlagSet("stats biases", flag.ContinueOnError)
	ff := newStatsFilterFlags(fs)
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb stats biases --db <file> [options]")
		fmt.Println()
		fmt.Println("Which way do the decisions err? Three signed biases (ADR-0079), each the share")
		fmt.Println("of decisions erring one way minus the share erring the other, with a 95%")
		fmt.Println("interval:")
		fmt.Println("  take/pass  wrong takes minus wrong passes: your take rate minus the bot's")
		fmt.Println("  doubles    premature doubles minus missed ones, overall and by score")
		fmt.Println("  blots      plays leaving more blots than the best minus plays leaving fewer")
		fmt.Printf("A direction is stated only with at least %d decisions and an interval that\n", storage.BiasMinDecisions)
		fmt.Println("excludes zero: too_much, too_little, balanced or insufficient.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb stats biases --db database.db --player \"Alice\"")
		fmt.Println("  blunderdb stats biases --db database.db --player \"Alice\" --format json")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	filter, err := ff.open(cli, fs)
	if err != nil {
		return err
	}
	text := strings.ToLower(*ff.format) != "json"
	var biases *storage.DirectionalBiases
	if err := computeInterruptibly(text, func(ctx context.Context) error {
		biases, err = cli.db.ComputeDirectionalBiasesCtx(ctx, filter)
		return err
	}); err != nil {
		return fmt.Errorf("biases: %w", err)
	}
	if !text {
		return printJSON(biases)
	}
	printBiases(biases)
	return nil
}

// printBiases writes the three biases, then the doubling bias of each score
// cell. Biases are percentage points of the decisions.
func printBiases(b *storage.DirectionalBiases) {
	fmt.Printf("Signed biases — share erring one way minus share erring the other; 95%% interval;\n")
	fmt.Printf("at least %d decisions.\n\n", b.MinDecisions)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	row := func(name, plus, minus string, s storage.SignedBias) {
		fmt.Fprintf(w, "%s\t%d\t%s %d (%d mp)\t%s %d (%d mp)\t%+.1f%%\t[%+.1f%%, %+.1f%%]\t%s\n", name, s.Decisions,
			plus, s.Plus, s.PlusMP, minus, s.Minus, s.MinusMP, 100*s.Bias, 100*s.Low, 100*s.High, s.Verdict)
	}
	fmt.Fprintln(w, "BIAS\tDECISIONS\tTOO MUCH\tTOO LITTLE\tBIAS\t95% INTERVAL\tVERDICT")
	row("take/pass", "wrong takes", "wrong passes", b.TakePass)
	row("doubles", "premature", "missed", b.Doubles)
	row("blots", "bolder", "more cautious", b.Blots)
	w.Flush()
	if b.BlotsUnread > 0 {
		fmt.Printf("(%d checker plays could not be replayed and are left out of the blots bias.)\n", b.BlotsUnread)
	}
	if len(b.DoublesByScore) > 0 {
		fmt.Println()
		fmt.Println("Doubles by score (your away, opponent's away; 0-0 is money play):")
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SCORE\tDECISIONS\tPREMATURE\tMISSED\tBIAS\t95% INTERVAL\tVERDICT")
		for _, c := range b.DoublesByScore {
			fmt.Fprintf(w, "%d-%d\t%d\t%d\t%d\t%+.1f%%\t[%+.1f%%, %+.1f%%]\t%s\n", c.MoverAway, c.OpponentAway,
				c.Decisions, c.Plus, c.Minus, 100*c.Bias, 100*c.Low, 100*c.High, c.Verdict)
		}
		w.Flush()
	}
}
