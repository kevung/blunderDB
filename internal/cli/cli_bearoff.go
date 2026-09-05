package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// `blunderdb bearoff` is how a table is made outside the desktop application:
// on a server, on a build machine, on a volume the daemon reads (ADR-0027).
// Nothing here talks to a database — a bearoff table is arithmetic about the
// game, not about anyone's positions — so these sub-commands take no --db.
//
// The domains that matter are large: TS-06-11 is 1.2 GB and minutes of every
// core, TS-06-13 is hours and more memory than most machines have. So the
// commands say what a run will cost before it starts, and Ctrl-C PAUSES rather
// than throws the work away: the same checkpoint the desktop's Pause button
// writes, and the next `generate` on that domain continues from it.

func (cli *CLI) runBearoff(args []string) error {
	if len(args) < 1 {
		cli.printBearoffUsage()
		return fmt.Errorf("missing bearoff sub-command")
	}
	sub := strings.ToLower(args[0])
	if sub == "--help" || sub == "-h" || sub == "help" {
		cli.printBearoffUsage()
		return nil
	}
	run, ok := cli.bearoffHandlers()[sub]
	if !ok {
		cli.printBearoffUsage()
		return fmt.Errorf("unknown bearoff sub-command: %s", args[0])
	}
	return run(args[1:])
}

// bearoffHandlers returns the sub-command table of `blunderdb bearoff`.
func (cli *CLI) bearoffHandlers() map[string]func([]string) error {
	return map[string]func([]string) error{
		"generate": cli.runBearoffGenerate,
		"list":     cli.runBearoffList,
		"verify":   cli.runBearoffVerify,
		"delete":   cli.runBearoffDelete,
	}
}

// BearoffSubcommands returns the sub-commands of `blunderdb bearoff`, sorted —
// the exported view cmd/cli-doc-gen walks.
func (cli *CLI) BearoffSubcommands() []string {
	h := cli.bearoffHandlers()
	names := make([]string, 0, len(h))
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (cli *CLI) printBearoffUsage() {
	fmt.Println("Usage: blunderdb bearoff <sub-command> [options]")
	fmt.Println()
	fmt.Println("Generate and manage the bearoff databases. Nothing is downloaded")
	fmt.Println("and nothing is embedded: a table is computed here and checked")
	fmt.Println("against gnubg's own fingerprint.")
	fmt.Println()
	fmt.Println("Sub-commands:")
	fmt.Println("  generate    Compute a table (resumable: Ctrl-C pauses)")
	fmt.Println("  list        What the data directory holds, and what it costs")
	fmt.Println("  verify      Check a .bd file against its domain's fingerprint")
	fmt.Println("  delete      Remove a generated table and any paused run")
	fmt.Println()
	fmt.Println("Run 'blunderdb bearoff <sub-command> --help' for its options.")
}

// parseOneSided turns a point count into a one-sided domain — the table the
// EPC reads, whose width is how far from home a chequer may stand and still be
// answered for (ADR-0027 §9).
func parseOneSided(points int) (bearoffgen.Domain, error) {
	if points < 6 || points > 12 {
		return bearoffgen.Domain{}, fmt.Errorf("bad one-sided domain %d: the point count runs from 6 to 12", points)
	}
	return bearoffgen.Domain{Kind: bearoffgen.OneSidedKind, Points: points, Checkers: 15}, nil
}

// parseDomain turns "6x9" — the notation makebearoff uses — into a domain.
// "os" and "os8" name the one-sided table the EPC reads.
func parseDomain(spec string) (bearoffgen.Domain, error) {
	s := strings.ToLower(strings.TrimSpace(spec))
	if s == "os" || s == "one-sided" {
		return bearoffgen.Domain{Kind: bearoffgen.OneSidedKind, Points: 6, Checkers: 15}, nil
	}
	if rest, ok := strings.CutPrefix(s, "os"); ok {
		p, err := strconv.Atoi(rest)
		if err != nil {
			return bearoffgen.Domain{}, fmt.Errorf("bad domain %q: %w", spec, err)
		}
		return parseOneSided(p)
	}
	points, checkers, ok := strings.Cut(s, "x")
	if !ok {
		return bearoffgen.Domain{}, fmt.Errorf("bad domain %q: expected a form like 6x9, or 'os' for the one-sided table", spec)
	}
	p, err := strconv.Atoi(points)
	if err != nil {
		return bearoffgen.Domain{}, fmt.Errorf("bad domain %q: %w", spec, err)
	}
	c, err := strconv.Atoi(checkers)
	if err != nil {
		return bearoffgen.Domain{}, fmt.Errorf("bad domain %q: %w", spec, err)
	}
	if p != 6 {
		return bearoffgen.Domain{}, fmt.Errorf("bad domain %q: a bearoff table describes the six-point home board, so the point count is always 6", spec)
	}
	if c < 1 || c > 15 {
		return bearoffgen.Domain{}, fmt.Errorf("bad domain %q: the chequer count runs from 1 to 15", spec)
	}
	return bearoffgen.Domain{Kind: bearoffgen.TwoSidedKind, Points: p, Checkers: c}, nil
}

// bearoffDir is the data directory a sub-command works in: --data-dir when
// given, otherwise the one the application uses.
func bearoffDir(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return race.DataDir()
}

func humanBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(n)/(1<<20))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func humanDuration(d time.Duration) string {
	switch {
	case d < 90*time.Second:
		return fmt.Sprintf("%.0f s", d.Seconds())
	case d < 90*time.Minute:
		return fmt.Sprintf("%.0f min", d.Minutes())
	default:
		return fmt.Sprintf("%.1f h", d.Hours())
	}
}

func (cli *CLI) runBearoffGenerate(args []string) error {
	fs := flag.NewFlagSet("bearoff generate", flag.ContinueOnError)
	ts := fs.String("ts", "", "Two-sided domain to generate, as 6x9")
	osPoints := fs.Int("os", 0, "One-sided domain to generate, as a point count: 6 … 12")
	dataDir := fs.String("data-dir", "", "Where to write it (default: the application's data directory)")
	cores := fs.Int("cores", 0, "Cores to use (default: every core but one)")
	quiet := fs.Bool("quiet", false, "No progress line")
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb bearoff generate --ts <domain> [options]")
		fmt.Println()
		fmt.Println("Compute a bearoff table. The result is checked against gnubg's")
		fmt.Println("fingerprint for its domain before it is put in place, so a table")
		fmt.Println("this writes is the table makebearoff writes.")
		fmt.Println()
		fmt.Println("Ctrl-C PAUSES: the state is written beside the table and the next")
		fmt.Println("run on the same domain continues from it rather than starting")
		fmt.Println("over. `bearoff delete` throws a paused run away.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb bearoff generate --ts 6x9")
		fmt.Println("  blunderdb bearoff generate --ts 6x11 --cores 4 --data-dir /srv/bearoff")
		fmt.Println("  blunderdb bearoff generate --os 8       # the EPC beyond the home board")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if (*ts == "") == (*osPoints == 0) {
		fs.Usage()
		return fmt.Errorf("give exactly one of --ts and --os")
	}
	var domain bearoffgen.Domain
	var err error
	if *osPoints != 0 {
		domain, err = parseOneSided(*osPoints)
	} else {
		domain, err = parseDomain(*ts)
	}
	if err != nil {
		return err
	}
	dir := bearoffDir(*dataDir)
	workers := *cores
	if workers <= 0 {
		if workers = runtime.NumCPU() - 1; workers < 1 {
			workers = 1
		}
	}

	if verdict, got, err := bearoffgen.Verify(filepath.Join(dir, domain.FileName())); err == nil && got == domain && verdict != bearoffgen.Corrupt {
		fmt.Printf("%s is already in %s (%s).\n", domain, dir, verdict)
		return nil
	}

	resumed := ""
	if done, total, err := bearoffgen.CheckpointProgress(dir, domain); err == nil && total > 0 {
		resumed = fmt.Sprintf(", resuming at %.0f%%", 100*float64(done)/float64(total))
	}
	if domain.Kind == bearoffgen.OneSidedKind {
		// The one-sided sweep reads only positions below the one it is on, so
		// there is nothing to spread across cores.
		workers = 1
	}
	fmt.Printf("Generating %s in %s on %d core(s)%s\n", domain, dir, workers, resumed)
	size := humanBytes(domain.Size())
	if domain.Size() == 0 {
		size = "an unmeasured size"
	}
	fmt.Printf("  %s on disk, %s of memory, about %s\n",
		size, humanBytes(domain.RAMNeeded()),
		humanDuration(domain.EstimateDuration(0, workers)))

	// Ctrl-C pauses. The signal is caught rather than left to kill the
	// process: half an hour of arithmetic is worth writing down, and the run
	// is resumable by construction.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	started := time.Now()
	var lastPrint time.Time
	progress := func(done, total int64) {
		if *quiet || total == 0 {
			return
		}
		if time.Since(lastPrint) < time.Second && done != total {
			return
		}
		lastPrint = time.Now()
		pct := 100 * float64(done) / float64(total)
		eta := ""
		if done > 0 {
			remaining := time.Duration(float64(time.Since(started)) * float64(total-done) / float64(done))
			eta = fmt.Sprintf(", about %s left", humanDuration(remaining))
		}
		fmt.Printf("\r  %.1f%%%s          ", pct, eta)
	}

	path, err := bearoffgen.GenerateWith(ctx, dir, domain, bearoffgen.RunOptions{
		Workers:  workers,
		Progress: progress,
		Pausable: true,
	})
	if !*quiet {
		fmt.Println()
	}
	if err != nil {
		if ctx.Err() != nil {
			if done, total, perr := bearoffgen.CheckpointProgress(dir, domain); perr == nil && total > 0 {
				fmt.Printf("Paused at %.0f%%. Run the same command again to continue.\n", 100*float64(done)/float64(total))
				return nil
			}
			fmt.Println("Interrupted before anything could be saved.")
			return nil
		}
		return err
	}
	verdict, _, _ := bearoffgen.Verify(path)
	fmt.Printf("Wrote %s (%s, %s) in %s\n", path, humanBytes(domain.Size()), verdict, humanDuration(time.Since(started)))
	return nil
}

// bearoffListEntry is one row of `bearoff list --format json`.
type bearoffListEntry struct {
	Domain      string  `json:"domain"`
	File        string  `json:"file"`
	Present     bool    `json:"present"`
	Verdict     string  `json:"verdict,omitempty"`
	Size        int64   `json:"size"`
	RAMNeeded   int64   `json:"ram_needed"`
	Seconds     float64 `json:"estimate_seconds"`
	Interrupted bool    `json:"interrupted,omitempty"`
	Percent     float64 `json:"percent,omitempty"`
}

func (cli *CLI) runBearoffList(args []string) error {
	fs := flag.NewFlagSet("bearoff list", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "Where to look (default: the application's data directory)")
	cores := fs.Int("cores", 0, "Cores the estimate assumes (default: every core but one)")
	format := fs.String("format", "text", "Output format: text or json")
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb bearoff list [options]")
		fmt.Println()
		fmt.Println("List every domain that can be generated: what it weighs, what it")
		fmt.Println("needs in memory, roughly how long it takes here, and whether this")
		fmt.Println("machine already has it.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb bearoff list")
		fmt.Println("  blunderdb bearoff list --format json --cores 8")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	formatLower := strings.ToLower(*format)
	if formatLower != "text" && formatLower != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}
	dir := bearoffDir(*dataDir)
	workers := *cores
	if workers <= 0 {
		if workers = runtime.NumCPU() - 1; workers < 1 {
			workers = 1
		}
	}

	domains := append(bearoffgen.Candidates(), bearoffgen.OneSidedCandidates()...)
	rows := make([]bearoffListEntry, 0, len(domains))
	for _, d := range domains {
		row := bearoffListEntry{
			Domain:    d.String(),
			File:      d.FileName(),
			Size:      d.Size(),
			RAMNeeded: d.RAMNeeded(),
			Seconds:   d.EstimateDuration(0, workers).Seconds(),
		}
		if verdict, got, err := bearoffgen.Verify(filepath.Join(dir, d.FileName())); err == nil && got == d {
			row.Present, row.Verdict = true, verdict.String()
			if info, err := os.Stat(filepath.Join(dir, d.FileName())); err == nil {
				row.Size = info.Size()
			}
		}
		if done, total, err := bearoffgen.CheckpointProgress(dir, d); err == nil && total > 0 {
			row.Interrupted = true
			row.Percent = 100 * float64(done) / float64(total)
		}
		rows = append(rows, row)
	}

	if formatLower == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{"data_dir": dir, "cores": workers, "domains": rows})
	}

	fmt.Printf("Data directory: %s\n", dir)
	fmt.Printf("Estimates assume %d core(s).\n\n", workers)
	fmt.Printf("%-10s %-18s %10s %10s %10s  %s\n", "DOMAIN", "FILE", "SIZE", "MEMORY", "TIME", "STATE")
	for _, r := range rows {
		state := "—"
		if r.Present {
			state = r.Verdict
		}
		if r.Interrupted {
			state = fmt.Sprintf("paused at %.0f%%", r.Percent)
		}
		size, ram, est := humanBytes(r.Size), humanBytes(r.RAMNeeded), humanDuration(time.Duration(r.Seconds*float64(time.Second)))
		if r.Size == 0 {
			// A one-sided domain nobody has measured: say so rather than
			// print a zero that reads as "empty file".
			size = "?"
		}
		fmt.Printf("%-10s %-18s %10s %10s %10s  %s\n", r.Domain, r.File, size, ram, est, state)
	}
	return nil
}

func (cli *CLI) runBearoffVerify(args []string) error {
	fs := flag.NewFlagSet("bearoff verify", flag.ContinueOnError)
	format := fs.String("format", "text", "Output format: text or json")
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb bearoff verify <file.bd> [options]")
		fmt.Println()
		fmt.Println("Check a bearoff file against the SHA-256 gnubg produces for its")
		fmt.Println("domain. Three answers: verified (the same bytes as the reference),")
		fmt.Println("unverified (well formed, but no fingerprint is recorded for that")
		fmt.Println("domain), corrupt (the file contradicts its own header).")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb bearoff verify ~/.local/share/blunderDB/gnubg_ts6x6.bd")
	}
	// The file is positional, so it is taken before the flags are parsed.
	var path string
	rest := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		path, rest = args[0], args[1:]
	}
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if path == "" {
		fs.Usage()
		return fmt.Errorf("missing the file to verify")
	}
	formatLower := strings.ToLower(*format)
	if formatLower != "text" && formatLower != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}

	verdict, domain, err := bearoffgen.Verify(path)
	reason := ""
	if err != nil {
		reason = err.Error()
	}
	if formatLower == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if encErr := enc.Encode(map[string]any{
			"file": path, "domain": domain.String(), "verdict": verdict.String(), "reason": reason,
		}); encErr != nil {
			return encErr
		}
	} else if reason != "" {
		fmt.Printf("%s: %s (%s) — %s\n", path, verdict, domain, reason)
	} else {
		fmt.Printf("%s: %s (%s)\n", path, verdict, domain)
	}
	// A corrupt file is a failure the shell must see: this command exists to
	// be put in a script.
	if verdict == bearoffgen.Corrupt {
		return fmt.Errorf("%s does not verify", path)
	}
	return nil
}

func (cli *CLI) runBearoffDelete(args []string) error {
	fs := flag.NewFlagSet("bearoff delete", flag.ContinueOnError)
	ts := fs.String("ts", "", "Domain to delete, as 6x9, or os8 for a one-sided table (required)")
	dataDir := fs.String("data-dir", "", "Where to look (default: the application's data directory)")
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb bearoff delete --ts <domain> [options]")
		fmt.Println()
		fmt.Println("Remove a generated table, along with any paused run and any debris")
		fmt.Println("of an interrupted one. A default domain is regenerated on the next")
		fmt.Println("launch of the application; a wider one is not.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb bearoff delete --ts 6x11")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *ts == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --ts")
	}
	domain, err := parseDomain(*ts)
	if err != nil {
		return err
	}
	dir := bearoffDir(*dataDir)
	base := filepath.Join(dir, domain.FileName())

	removed := 0
	for _, p := range []string{base, base + ".part", base + ".ckpt"} {
		if err := os.Remove(p); err == nil {
			removed++
			fmt.Printf("Removed %s\n", p)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if removed == 0 {
		fmt.Printf("Nothing to remove for %s in %s\n", domain, dir)
	}
	return nil
}
