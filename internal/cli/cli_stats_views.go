package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/report"
)

// statsViewFlags are the flags every `stats` view shares: the database and the
// panel's filter.
type statsViewFlags struct {
	db, player, tournament, from, to, decisionType, format *string
}

func newStatsViewFlags(fs *flag.FlagSet) statsViewFlags {
	return statsViewFlags{
		db:           fs.String("db", "", "Path to the database file (required)"),
		player:       fs.String("player", "", "Only this player's decisions"),
		tournament:   fs.String("tournament", "", "Filter by tournament IDs, comma-separated"),
		from:         fs.String("from", "", "Start date filter YYYY-MM-DD"),
		to:           fs.String("to", "", "End date filter YYYY-MM-DD"),
		decisionType: fs.String("decision-type", "all", "Decision type: all, checker, or cube"),
		format:       fs.String("format", "text", "Output format: text or json"),
	}
}

// runStatsView parses a view's flags, opens the database and computes the
// stats of the filter, cancellable with Ctrl-C.
func (cli *CLI) runStatsView(name string, usage func(*flag.FlagSet), args []string, extra func(*flag.FlagSet)) (*database.StatsResult, statsViewFlags, *flag.FlagSet, error) {
	fs := flag.NewFlagSet("stats "+name, flag.ContinueOnError)
	f := newStatsViewFlags(fs)
	if extra != nil {
		extra(fs)
	}
	fs.Usage = func() { usage(fs) }
	if err := fs.Parse(args); err != nil {
		return nil, f, fs, err
	}
	if *f.db == "" {
		fs.Usage()
		return nil, f, fs, fmt.Errorf("missing required flag: --db")
	}
	filter, err := buildStatsFilter(*f.player, *f.tournament, *f.from, *f.to, *f.decisionType)
	if err != nil {
		return nil, f, fs, err
	}
	if err := cli.initDatabase(*f.db); err != nil {
		return nil, f, fs, err
	}
	var res *database.StatsResult
	err = withInterruptibleContext(func() { fmt.Fprintln(os.Stderr, "\nCancelling...") }, func(ctx context.Context) error {
		var err error
		res, err = cli.db.ComputeStatsCtx(ctx, filter)
		return err
	})
	if errors.Is(err, context.Canceled) {
		return nil, f, fs, fmt.Errorf("stats cancelled")
	}
	if err != nil {
		return nil, f, fs, fmt.Errorf("stats %s: %w", name, err)
	}
	return res, f, fs, nil
}

// runStatsProgression is the Progression tab: PR over time (per match, per
// tournament, rolling).
func (cli *CLI) runStatsProgression(args []string) error {
	res, f, _, err := cli.runStatsView("progression", func(fs *flag.FlagSet) {
		fmt.Println("Usage: blunderdb stats progression --db <file> [options]")
		fmt.Println()
		fmt.Println("The Progression tab of the Stats panel: the PR of each match in date order,")
		fmt.Println("each tournament's PR, and the rolling PR over the last N decisions.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb stats progression --db database.db --player \"Alice\"")
		fmt.Println("  blunderdb stats progression --db database.db --format json")
	}, args, nil)
	if err != nil {
		return err
	}
	if strings.ToLower(*f.format) == "json" {
		return printJSON(struct {
			PerMatch      []database.MatchStats      `json:"PerMatch"`
			PerTournament []database.TournamentStats `json:"PerTournament"`
			PRRolling     map[int]float64            `json:"PRRolling"`
			MWCRolling    map[int]float64            `json:"MWCRolling"`
			MWCAvailable  bool                       `json:"MWCAvailable"`
		}{res.PerMatch, res.PerTournament, res.PRRolling, res.MWCRolling, res.MWCAvailable})
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DATE\tPLAYER\tPR\tDECISIONS")
	for _, m := range res.PerMatch {
		fmt.Fprintf(w, "%s\t%s\t%.2f\t%d\n", m.Date, m.PlayerName, m.PR, m.NumDecisions)
	}
	w.Flush()
	if len(res.PRRolling) > 0 {
		fmt.Println()
		fmt.Println("Rolling PR (last N decisions):")
		for _, n := range sortedKeys(res.PRRolling) {
			fmt.Printf("  %d: %.2f\n", n, res.PRRolling[n])
		}
	}
	return nil
}

// runStatsBreakdown is the Ventilations tab and the cube directions: the same
// PR split by phase, plan of play, tag, score and cube action.
func (cli *CLI) runStatsBreakdown(args []string) error {
	res, f, _, err := cli.runStatsView("breakdown", func(fs *flag.FlagSet) {
		fmt.Println("Usage: blunderdb stats breakdown --db <file> [options]")
		fmt.Println()
		fmt.Println("The Breakdowns tab of the Stats panel: the PR of the filter split by game")
		fmt.Println("phase, plan of play, comment tag, score (away x away) and cube action, and")
		fmt.Println("the direction in which the cube decisions went wrong.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb stats breakdown --db database.db --player \"Alice\"")
		fmt.Println("  blunderdb stats breakdown --db database.db --format json")
	}, args, nil)
	if err != nil {
		return err
	}
	if strings.ToLower(*f.format) == "json" {
		return printJSON(struct {
			PerPhase            []database.PhaseStats      `json:"PerPhase"`
			PerGameType         []database.GameTypeStats   `json:"PerGameType"`
			PerTag              []database.TagStats        `json:"PerTag"`
			PerScore            []database.ScoreCellStats  `json:"PerScore"`
			CubeActionBreakdown []database.CubeActionStats `json:"CubeActionBreakdown"`
			CubeDirections      database.CubeDirections    `json:"CubeDirections"`
		}{res.PerPhase, res.PerGameType, res.PerTag, res.PerScore, res.CubeActionBreakdown, res.CubeDirections})
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	section := func(title string) { fmt.Fprintf(w, "\n%s\n", title) }
	fmt.Fprintln(w, "PHASE\tPR\tDECISIONS\tBLUNDERS")
	for _, r := range res.PerPhase {
		fmt.Fprintf(w, "%s\t%.2f\t%d\t%d\n", r.Phase, r.PR, r.NumDecisions, r.BlunderCount)
	}
	section("PLAN OF PLAY\tPR\tDECISIONS\tBLUNDERS")
	for _, r := range res.PerGameType {
		fmt.Fprintf(w, "%s\t%.2f\t%d\t%d\n", r.GameType, r.PR, r.NumDecisions, r.BlunderCount)
	}
	section("TAG\tPR\tDECISIONS\tBLUNDERS")
	for _, r := range res.PerTag {
		fmt.Fprintf(w, "%s\t%.2f\t%d\t%d\n", r.Tag, r.PR, r.NumDecisions, r.BlunderCount)
	}
	section("CUBE ACTION\tPR\tDECISIONS\tBLUNDERS")
	for _, r := range res.CubeActionBreakdown {
		fmt.Fprintf(w, "%s\t%.2f\t%d\t%d\n", r.Action, r.PR, r.NumDecisions, r.BlunderCount)
	}
	section("SCORE (AWAY x AWAY)\tPR\tDECISIONS\tBLUNDERS")
	for _, r := range res.PerScore {
		fmt.Fprintf(w, "%d-away vs %d-away\t%.2f\t%d\t%d\n", r.MoverAway, r.OpponentAway, r.PR, r.NumDecisions, r.BlunderCount)
	}
	return w.Flush()
}

// runStatsReport writes the self-contained HTML report of the filter, from the
// generator the app and the daemon use.
func (cli *CLI) runStatsReport(args []string) error {
	var htmlFlag *bool
	var out, lang *string
	fs := flag.NewFlagSet("stats report", flag.ContinueOnError)
	f := newStatsViewFlags(fs)
	htmlFlag = fs.Bool("html", false, "Write the report as one self-contained HTML file (the only format)")
	out = fs.String("output", "", "File to write (default: standard output)")
	lang = fs.String("lang", "en", "Report language: "+strings.Join(report.Languages(), ", "))
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb stats report --db <file> --html [options]")
		fmt.Println()
		fmt.Println("The HTML report of the Stats panel: the filter's indicators and its ten most")
		fmt.Println("expensive decisions with their diagrams, as one self-contained file (no image,")
		fmt.Println("style sheet or script outside it) that a browser prints to PDF.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb stats report --db database.db --html --output rapport.html")
		fmt.Println("  blunderdb stats report --db database.db --html --player \"Alice\" --lang fr")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *f.db == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --db")
	}
	if !*htmlFlag {
		fs.Usage()
		return fmt.Errorf("missing required flag: --html")
	}
	filter, err := buildStatsFilter(*f.player, *f.tournament, *f.from, *f.to, *f.decisionType)
	if err != nil {
		return err
	}
	if err := cli.initDatabase(*f.db); err != nil {
		return err
	}
	var html string
	err = withInterruptibleContext(func() { fmt.Fprintln(os.Stderr, "\nCancelling...") }, func(ctx context.Context) error {
		var err error
		html, err = cli.db.StatsReportHTMLCtx(ctx, filter, *lang, nil)
		return err
	})
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("report cancelled")
	}
	if err != nil {
		return fmt.Errorf("stats report: %w", err)
	}
	if *out == "" {
		fmt.Print(html)
		return nil
	}
	if err := os.WriteFile(*out, []byte(html), 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Report written to %s\n", *out)
	return nil
}

func sortedKeys(m map[int]float64) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}
