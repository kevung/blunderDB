package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// runEpc handles the epc command: EPC, win probability and money cube
// verdict for a bearoff position given as an XGID. Pure computation — no
// database is opened; the same engine/race code serves the GUI panel and the
// serve daemon (ADR-0009).
func (cli *CLI) runEpc(args []string) error {
	epcCmd := flag.NewFlagSet("epc", flag.ContinueOnError)

	format := epcCmd.String("format", "text", "Output format: text, json")
	tsPath := epcCmd.String("bearoff-ts", os.Getenv("BLUNDERDB_TS_PATH"),
		"Optional two-sided bearoff database (.bd) widening the embedded TS-06-06")

	epcCmd.Usage = func() {
		fmt.Println("Usage: blunderdb epc [options] <XGID>")
		fmt.Println()
		fmt.Println("Compute EPC, win probability and money cube verdict for a position.")
		fmt.Println("Win probability is exact inside the two-sided database domain and")
		fmt.Println("estimated (with its error bound) outside; the cube verdict is only")
		fmt.Println("ever shown when exact.")
		fmt.Println()
		fmt.Println("Options:")
		epcCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # EPC and race analysis of a bearoff position")
		fmt.Println("  blunderdb epc 'XGID=-BBBB----------------bbbb-:0:0:1:00:0:0:0:0:10'")
		fmt.Println()
		fmt.Println("  # With the downloaded/wider database")
		fmt.Println("  blunderdb epc --bearoff-ts ~/.local/share/blunderdb/gnubg_ts6x11.bd '<XGID>'")
	}

	if err := epcCmd.Parse(args); err != nil {
		return err
	}
	if epcCmd.NArg() != 1 {
		epcCmd.Usage()
		return fmt.Errorf("expected exactly one XGID argument")
	}

	pos, err := domain.DecodeXGID(epcCmd.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid XGID: %w", err)
	}
	if *tsPath != "" {
		race.SetExternalPath(*tsPath)
	}

	res := race.Evaluate(&pos)

	if *format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}

	printSide := func(label string, s race.Side) {
		fmt.Printf("%s: %d checkers", label, s.CheckerCount)
		if !s.AllInHome {
			fmt.Println(" (not all in home board — no EPC)")
			return
		}
		if s.EPC == nil {
			fmt.Println()
			return
		}
		fmt.Printf("  EPC %.2f  pips %d  wastage %.2f  rolls %.3f ± %.3f\n",
			s.EPC.EPC, s.EPC.PipCount, s.EPC.Wastage, s.EPC.MeanRolls, s.EPC.StdDev)
	}
	printSide("Bottom (X)", res.Bottom)
	printSide("Top    (O)", res.Top)

	if res.Race == nil {
		fmt.Println("Race analysis: not a pure bearoff position.")
		return nil
	}
	r := res.Race
	who := "X"
	if r.OnRoll == domain.White {
		who = "O"
	}
	switch r.Regime {
	case race.RegimeExact:
		fmt.Printf("Win probability (%s on roll): %.2f%% [exact, TS-06-%02d]\n",
			who, 100*r.WinProb, r.SourceCheckers)
		m := r.Money
		fmt.Printf("Money cube (%s): cubeless %+.3f  ND %+.3f  D/T %+.3f  D/P %+.3f\n",
			m.CubeState, m.Cubeless, m.NoDouble, m.DoubleTake, m.DoublePass)
		if m.Verdict != "" {
			fmt.Printf("Verdict: %s\n", m.Verdict)
		} else {
			fmt.Println("Verdict: cube is against the player on roll — no decision")
		}
	default:
		fmt.Printf("Win probability (%s on roll): %.2f%% ± %.2f%% [estimated, p99 %.2f%%]\n",
			who, 100*r.WinProb, 100*r.Sigma, 100*r.P99)
		fmt.Println("Cube verdict: unavailable (never estimated); provide a wider")
		fmt.Println("two-sided database with --bearoff-ts to widen the exact domain.")
	}
	return nil
}
