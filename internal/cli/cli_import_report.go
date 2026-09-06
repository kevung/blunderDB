package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The end-of-import report (issue #257, fiche I.1).
//
// An import used to end on a count of files. What the user actually wants to
// know next is what came in worth looking at: how they played, which decisions
// cost the most, how many positions the source tool had flagged, and how many
// nothing has judged yet. The same object is rendered here and by the
// interface, and travels verbatim under --format json, so a script sees what
// the panel shows.

// beginImportBatch opens a batch for the import about to run. A failure to
// open one is deliberately NOT fatal: the report is a convenience, and losing
// it must never cost the user the import itself. The zero id then flows
// through withoutFailing every call below.
func (cli *CLI) beginImportBatch(source, format string) int64 {
	id, err := cli.db.BeginImportBatch(source, format)
	if err != nil {
		return 0
	}
	return id
}

// finishImportBatch closes the batch and returns the measured report, or nil
// when there is nothing to report on. Like beginImportBatch it swallows its
// own failures: an import that succeeded must not be reported as failed
// because its summary could not be written.
func (cli *CLI) finishImportBatch(batchID int64, failures domain.ImportReport) *domain.ImportBatch {
	if batchID == 0 {
		return nil
	}
	if err := cli.db.FinishImportBatch(batchID, failures); err != nil {
		return nil
	}
	report, err := cli.db.ImportReport(batchID)
	if err != nil {
		return nil
	}
	return report
}

// printImportReport writes the report as the human-readable block that closes
// an import. It prints nothing at all when there is nothing to say — an import
// of one already-known match should not produce a page of zeros.
func printImportReport(b *domain.ImportBatch) {
	if b == nil {
		return
	}
	r := b.Report
	fmt.Println()
	fmt.Println("IMPORT REPORT")
	fmt.Println(strings.Repeat("-", 60))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Matches imported\t%d\n", r.MatchesImported)
	if r.MatchesEnriched > 0 {
		fmt.Fprintf(w, "Matches enriched in place\t%d\n", r.MatchesEnriched)
	}
	if r.MatchesSkipped > 0 {
		fmt.Fprintf(w, "Already in the database\t%d\n", r.MatchesSkipped)
	}
	if r.FilesFailed > 0 {
		fmt.Fprintf(w, "Files that could not be read\t%d\n", r.FilesFailed)
	}
	fmt.Fprintf(w, "New positions\t%d\n", r.PositionsSaved)
	if r.PositionsFlagged > 0 {
		fmt.Fprintf(w, "Flagged for study in the source tool\t%d\n", r.PositionsFlagged)
	}
	if r.PositionsWithoutAnalysis > 0 {
		fmt.Fprintf(w, "Positions with no analysis\t%d\n", r.PositionsWithoutAnalysis)
	}
	// A PR of zero on zero decisions is not a perfect game, it is the absence
	// of an analysis — so the line says which.
	if r.Decisions > 0 {
		who := r.Player
		if who == "" {
			who = "both players"
		}
		fmt.Fprintf(w, "PR over this import (%s)\t%.2f  (%d decisions)\n", who, r.PR, r.Decisions)
	} else {
		fmt.Fprintf(w, "PR over this import\tno analysis to score\n")
	}
	w.Flush()

	if len(r.WorstDecisions) > 0 {
		fmt.Println()
		fmt.Println("Worst decisions:")
		bw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for i, d := range r.WorstDecisions {
			kind := "checker"
			if d.IsCube {
				kind = "cube"
			}
			fmt.Fprintf(bw, "  %d.\tposition %d\t%s\t%.3f\t%s\n",
				i+1, d.PositionID, kind, float64(d.ErrorMP)/1000, d.Label)
		}
		bw.Flush()
	}

	if len(r.Failures) > 0 {
		fmt.Println()
		fmt.Println("Files that could not be read:")
		for _, f := range r.Failures {
			fmt.Printf("  %s: %s\n", f.Source, f.Reason)
		}
		if r.FilesFailed > len(r.Failures) {
			fmt.Printf("  … and %d more\n", r.FilesFailed-len(r.Failures))
		}
	}
}

// recordFailure adds a file the import could not read to the report the caller
// will hand FinishImportBatch. The count is exact; the list is capped, because
// a folder of a thousand unreadable files must not produce a thousand-line
// report.
func recordFailure(r *domain.ImportReport, source string, err error) {
	r.FilesFailed++
	if len(r.Failures) < domain.MaxImportFailures {
		r.Failures = append(r.Failures, domain.ImportFailure{Source: source, Reason: err.Error()})
	}
}

// listImports serves `blunderdb list --type imports`: the recorded batches,
// most recent first, or the full report of one of them with --batch.
//
// The list shows the counts stored at the end of each import; the single-batch
// form measures the rest again, so a batch whose positions have since been
// analysed reports what is true now rather than what was true then.
func (cli *CLI) listImports(limit int, batchID int64, format string) error {
	if batchID != 0 {
		b, err := cli.db.ImportReport(batchID)
		if err != nil {
			return fmt.Errorf("import batch %d: %w", batchID, err)
		}
		if format == "json" {
			return printJSON(b)
		}
		fmt.Printf("Import %d — %s (%s)\n", b.ID, b.Source, b.Format)
		fmt.Printf("Started %s", b.StartedAt)
		if b.FinishedAt != "" {
			fmt.Printf(", finished %s", b.FinishedAt)
		} else {
			fmt.Print(", never finished")
		}
		fmt.Println()
		printImportReport(b)
		return nil
	}

	batches, err := cli.db.ListImportBatches(limit, 0)
	if err != nil {
		return err
	}
	if format == "json" {
		return printJSON(batches)
	}
	if len(batches) == 0 {
		fmt.Println("No import has been recorded in this database yet.")
		return nil
	}
	fmt.Printf("Found %d import(s):\n\n", len(batches))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tStarted\tFormat\tSource\tImported\tSkipped\tEnriched\tFailed\tPositions")
	fmt.Fprintln(w, "--\t-------\t------\t------\t--------\t-------\t--------\t------\t---------")
	for _, b := range batches {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\n",
			b.ID, b.StartedAt, b.Format, b.Source,
			b.Report.MatchesImported, b.Report.MatchesSkipped,
			b.Report.MatchesEnriched, b.Report.FilesFailed, b.Report.PositionsSaved)
	}
	w.Flush()
	fmt.Println("\nUse --batch <id> for the full report of one import.")
	return nil
}
