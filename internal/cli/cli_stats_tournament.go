package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// runStatsTournament prints one player's review of a tournament (ADR-0081),
// the figures the Tournaments panel shows.
func (cli *CLI) runStatsTournament(args []string) error {
	fs := flag.NewFlagSet("stats tournament", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	id := fs.Int64("id", 0, "Tournament ID (required; `list --type tournaments` lists them)")
	player := fs.String("player", "", "Player to review (default: the one in the most matches of the tournament)")
	format := fs.String("format", "text", "Output format: text or json")
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb stats tournament --db <file> --id <n> [options]")
		fmt.Println()
		fmt.Println("One player's tournament set against their usual level (ADR-0081): L7 and PR")
		fmt.Printf("against the %d days before the tournament, by round, by decision rank within\n", storage.TournamentUsualDays)
		fmt.Println("the match (fatigue), at pressure scores (DMP, Crawford, post-Crawford) and by")
		fmt.Println("pace (quick or considered), with 95% intervals, then the tournament's error")
		fmt.Println("families. A verdict (worse, better, usual) needs an interval and at least")
		fmt.Printf("%d decisions on both sides, and a usual level of %d matches; otherwise it reads\n",
			storage.TournamentReviewMinDecisions, storage.TournamentUsualMinMatches)
		fmt.Println("insufficient.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb stats tournament --db database.db --id 3")
		fmt.Println("  blunderdb stats tournament --db database.db --id 3 --player \"Alice\" --format json")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" || *id == 0 {
		fs.Usage()
		return fmt.Errorf("missing required flags: --db and --id")
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}
	review, err := cli.db.GetTournamentReview(*id, *player)
	if err != nil {
		return statsErr("tournament review", err)
	}
	if strings.ToLower(*format) == "json" {
		return printJSON(review)
	}
	printTournamentReview(review)
	return nil
}

// fmtPRBand writes a PR and its interval, or a dash for the band when the
// figure has none.
func fmtPRBand(pr float64, iv domain.Interval, n int) string {
	if n == 0 {
		return "-"
	}
	if !iv.Available {
		return fmt.Sprintf("%.2f [-]", pr)
	}
	return fmt.Sprintf("%.2f [%.2f-%.2f]", pr, iv.Low, iv.High)
}

func fmtL7(e domain.MWC7) string {
	if !e.Available {
		return "-"
	}
	if !e.HasInterval {
		return fmt.Sprintf("%.1f%% [-]", 100*e.Loss)
	}
	return fmt.Sprintf("%.1f%% [%.1f-%.1f]", 100*e.Loss, 100*e.Low, 100*e.High)
}

func fmtVersus(c storage.Comparison, scale float64, unit string) string {
	if c.Verdict == storage.VerdictInsufficient {
		return c.Verdict
	}
	return fmt.Sprintf("%s (%+.2f%s [%+.2f, %+.2f])", c.Verdict, scale*c.Delta, unit, scale*c.Low, scale*c.High)
}

func printTournamentReview(r storage.TournamentReview) {
	fmt.Printf("Tournament %d %q", r.TournamentID, r.Name)
	if r.Date != "" {
		fmt.Printf(" (%s)", r.Date)
	}
	fmt.Printf(" - %s, %d matches, %d decisions\n", r.Player, r.Matches, r.Decisions)
	if r.Usual.Available {
		fmt.Printf("Usual level: %s to %s, %d matches, %d decisions\n", r.Usual.From, r.Usual.To, r.Usual.Matches, r.Usual.Decisions)
	} else {
		fmt.Printf("Usual level unknown (%d matches in the window, %d needed)\n", r.Usual.Matches, storage.TournamentUsualMinMatches)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "\tTOURNAMENT\tUSUAL\tVERSUS")
	fmt.Fprintf(w, "PR\t%s\t%s\t%s\n", fmtPRBand(r.PR, r.PRInterval, r.Decisions),
		fmtPRBand(r.Usual.PR, r.Usual.PRInterval, r.Usual.Decisions), fmtVersus(r.PRVersus, 1, ""))
	fmt.Fprintf(w, "L7\t%s\t%s\t%s\n", fmtL7(r.MWC7), fmtL7(r.Usual.MWC7), fmtVersus(r.MWC7Versus, 100, "%"))
	_ = w.Flush()

	fmt.Println()
	fmt.Println("By round:")
	w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ROUND\tOPPONENT\tDECISIONS\tPR\tL7\tVERSUS USUAL PR")
	for _, rr := range r.Rounds {
		fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\t%s\n", rr.Round, rr.Opponent, rr.Decisions,
			fmtPRBand(rr.PR, rr.PRInterval, rr.Decisions), fmtL7(rr.MWC7), fmtVersus(rr.Versus, 1, ""))
	}
	_ = w.Flush()

	for _, part := range []struct {
		title string
		cells []storage.ReviewCell
	}{{"By decision rank in the match", r.ByRank}, {"By score", r.ByPressure}, {"By pace", r.ByClock}} {
		fmt.Println()
		fmt.Println(part.title + ":")
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "CELL\tDECISIONS\tPR\tUSUAL\tUSUAL PR\tVERSUS")
		for _, c := range part.cells {
			fmt.Fprintf(w, "%s\t%d\t%s\t%d\t%s\t%s\n", c.Key, c.Decisions, fmtPRBand(c.PR, c.PRInterval, c.Decisions),
				c.UsualDecisions, fmtPRBand(c.UsualPR, c.UsualInterval, c.UsualDecisions), fmtVersus(c.Versus, 1, ""))
		}
		_ = w.Flush()
	}

	fmt.Println()
	if len(r.Families) == 0 {
		fmt.Printf("No error family holds in this tournament (%d to confirm).\n", r.Tentative)
		return
	}
	fmt.Println("Error families of the tournament:")
	for i, f := range r.Families {
		fmt.Printf("  %d. %s / %s / %s: %d errors, recoverable %.2f%% [%.2f, %.2f]\n", i+1, f.GameType, f.Kind, f.Theme,
			f.Errors, 100*f.Recoverable, 100*f.Low, 100*f.High)
	}
	if r.Tentative > 0 {
		fmt.Printf("  (%d more to confirm)\n", r.Tentative)
	}
}
