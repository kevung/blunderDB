package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"
)

// listStudy prints, plan of play by plan of play, what was revised over a
// window and the Performance Rating on either side of it (issue #275, fiche
// I.19).
//
// It prints THREE NUMBERS SIDE BY SIDE and no fourth. There is no "gain"
// column and no arrow, because nothing here controls anything: the player may
// have met stronger opponents, changed format, or simply played more races
// this month. The rapprochement is the reader's, and a column claiming an
// effect would be a claim these data cannot carry.
func (cli *CLI) listStudy(days int, format string) error {
	rows, err := cli.db.StudyImpact(days)
	if err != nil {
		return fmt.Errorf("failed to compute the study figures: %w", err)
	}

	switch format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	case "csv":
		w := csv.NewWriter(os.Stdout)
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
		fmt.Println("Nothing to compare yet: no plan of play is computed, or no match is stored.")
		fmt.Println("`blunderdb repair` computes the plans; importing matches fills the rest.")
		return nil
	}

	fmt.Printf("\nStudied and played, over the last %d days\n\n", days)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
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
	fmt.Println()
	fmt.Println("These are three counts read side by side, not an effect: nothing here")
	fmt.Println("controls for the opponents met, the format played or the luck of the dice.")
	return nil
}

// prCell renders a PR, or a dash when there are too few decisions behind it to
// read. The COUNT is still printed beside it — hiding the sample would make
// the figure unauditable, which is the same rule storage.MinCellDecisions
// follows.
func prCell(pr float64, decisions int) string {
	if decisions < 10 {
		return "—"
	}
	return strconv.FormatFloat(pr, 'f', 2, 64)
}
