package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"text/tabwriter"
)

// listStudy prints, per plan of play, what was revised over a window and the
// Performance Rating on either side of it. Deliberately no "gain" column:
// nothing is controlled, so an effect is not something these data can claim.
func (cli *CLI) listStudy(days int, format string) error {
	rows, err := cli.db.StudyImpact(days)
	if err != nil {
		return fmt.Errorf("failed to compute the study figures: %w", err)
	}

	switch format {
	case "json":
		enc := json.NewEncoder(cli.stdout())
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	case "csv":
		w := csv.NewWriter(cli.stdout())
		defer w.Flush()
		if err := w.Write([]string{"game_type", "reviewed", "pr_before", "decisions_before", "pr_after", "decisions_after"}); err != nil {
			return err
		}
		for _, r := range rows {
			if err := w.Write([]string{
				r.GameType,
				strconv.Itoa(r.Reviewed),
				strconv.FormatFloat(r.PRBefore, 'f', 2, 64),
				strconv.Itoa(r.DecisionsBefore),
				strconv.FormatFloat(r.PRAfter, 'f', 2, 64),
				strconv.Itoa(r.DecisionsAfter),
			}); err != nil {
				return err
			}
		}
		return nil
	}

	if len(rows) == 0 {
		fmt.Fprintln(cli.stdout(), "Nothing to compare yet: no plan of play is computed, or no match is stored.")
		fmt.Fprintln(cli.stdout(), "`blunderdb repair` computes the plans; importing matches fills the rest.")
		return nil
	}

	fmt.Fprintf(cli.stdout(), "\nStudied and played, over the last %d days\n\n", days)
	w := tabwriter.NewWriter(cli.stdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PLAN\tREVISED\tPR BEFORE\t(n)\tPR SINCE\t(n)")
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%d\t%s\t%d\t%s\t%d\n",
			r.GameType, r.Reviewed,
			prCell(r.PRBefore, r.DecisionsBefore), r.DecisionsBefore,
			prCell(r.PRAfter, r.DecisionsAfter), r.DecisionsAfter)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "These are three counts read side by side, not an effect: nothing here")
	fmt.Fprintln(cli.stdout(), "controls for the opponents met, the format played or the luck of the dice.")
	return nil
}

// prCell renders a PR, or a dash when too few decisions back it; the count is
// still printed beside it (same rule as StudyImpactMinDecisions).
func prCell(pr float64, decisions int) string {
	if decisions < 10 {
		return "—"
	}
	return strconv.FormatFloat(pr, 'f', 2, 64)
}
