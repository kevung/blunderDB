package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen/bearofftest"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// The bearoff sources after ADR-0027: nothing embedded, nothing downloaded.
// What the status has to report is therefore new — whether this machine has
// its tables yet, and which one is being made — and the empty case is a real
// state rather than an impossible one.

func isolate(t *testing.T) string {
	t.Helper()
	// Capture the shared cache path and table now: the cleanup below runs
	// after the test has finished, and a helper that calls t.Fatalf then would
	// panic rather than report.
	shared := bearofftest.DataDir(t)
	oneSided := bearofftest.OneSidedPath(t)
	dir := t.TempDir()
	race.SetDataDir(dir)
	race.SetExternalPath("")
	race.Invalidate()
	t.Cleanup(func() {
		race.SetDataDir(shared)
		race.SetExternalPath("")
		race.Invalidate()
		// Put the package-wide table back: TestMain loaded it for every test
		// here, and leaving it unloaded would break whatever runs next.
		_ = engine.LoadOneSided(oneSided)
	})
	return dir
}

func TestBearoffStatus_EmptyMachineReportsWhatIsMissing(t *testing.T) {
	dir := isolate(t)
	_ = engine.LoadOneSided("")

	st := (&App{}).BearoffStatus()
	if st.Ready {
		t.Error("an empty data dir cannot be ready")
	}
	if len(st.Missing) != 2 {
		t.Errorf("Missing = %v, want both default domains", st.Missing)
	}
	if st.DataDir != dir {
		t.Errorf("DataDir = %q, want %q", st.DataDir, dir)
	}
	if st.ActiveDomain != 0 || st.ActiveOrigin != "" {
		t.Errorf("no table means no active source, got %d / %q", st.ActiveDomain, st.ActiveOrigin)
	}
	if st.OneSidedReady {
		t.Error("the EPC has no table either")
	}
	if st.Generating != "" {
		t.Errorf("nothing is being generated, got %q", st.Generating)
	}
}

func TestBearoffStatus_WithBothTablesItIsReady(t *testing.T) {
	dir := isolate(t)
	for _, d := range bearoffgen.DefaultDomains() {
		src, err := os.ReadFile(bearofftest.Path(t, d))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, d.FileName()), src, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	race.Invalidate()
	if err := engine.LoadOneSided(filepath.Join(dir, "gnubg_os6.bd")); err != nil {
		t.Fatal(err)
	}

	st := (&App{}).BearoffStatus()
	if !st.Ready || len(st.Missing) != 0 {
		t.Errorf("both tables are there: Ready=%v Missing=%v", st.Ready, st.Missing)
	}
	if st.ActiveDomain != 6 {
		t.Errorf("ActiveDomain = %d, want the TS-06-06 that is present", st.ActiveDomain)
	}
	if !st.OneSidedReady {
		t.Error("the EPC table is loaded")
	}
}

// A corrupt table counts as missing: it will be regenerated rather than read.
func TestBearoffStatus_ACorruptTableCountsAsMissing(t *testing.T) {
	dir := isolate(t)
	src, err := os.ReadFile(bearofftest.TwoSidedPath(t))
	if err != nil {
		t.Fatal(err)
	}
	src[len(src)/2] ^= 0xFF
	if err := os.WriteFile(filepath.Join(dir, "gnubg_ts6x6.bd"), src, 0o644); err != nil {
		t.Fatal(err)
	}

	st := (&App{}).BearoffStatus()
	for _, m := range st.Missing {
		if m == "TS-06-06" {
			return
		}
	}
	t.Errorf("a corrupt TS-06-06 must be listed as missing, got %v", st.Missing)
}

// EnsureBearoffTables is what start-up calls: it must make both tables and
// leave the engine able to answer.
func TestEnsureBearoffTables_MakesWhatIsMissing(t *testing.T) {
	if testing.Short() {
		t.Skip("generating both default tables takes a few seconds")
	}
	dir := isolate(t)
	_ = engine.LoadOneSided("")

	app := &App{}
	app.EnsureBearoffTables()

	// The work is a goroutine; wait for it the way the panel does, by polling
	// the status.
	deadline := 120
	for i := 0; i < deadline; i++ {
		if st := app.BearoffStatus(); st.Ready && st.Generating == "" {
			break
		}
		sleepMillis(500)
	}

	st := app.BearoffStatus()
	if !st.Ready {
		t.Fatalf("both tables must exist after EnsureBearoffTables: %v", st.Missing)
	}
	if !st.OneSidedReady {
		t.Error("the EPC table must be loaded once generated")
	}
	if got := race.Resolve(); got == nil || got.Checkers() != 6 {
		t.Errorf("the two-sided table must resolve, got %v", got)
	}
	for _, d := range bearoffgen.DefaultDomains() {
		if _, err := os.Stat(filepath.Join(dir, d.FileName()+".part")); err == nil {
			t.Errorf("%s left a .part behind", d)
		}
	}
}

func sleepMillis(n int) {
	time.Sleep(time.Duration(n) * time.Millisecond)
}

// The plan is what the Bearoff tab renders. It must name every domain the user
// may ask for, price each one, and say which the machine cannot hold — an
// absent row answers nothing.
func TestBearoffPlan_PricesEveryDomainAndGreysWhatDoesNotFit(t *testing.T) {
	dir := isolate(t)
	for _, d := range bearoffgen.DefaultDomains() {
		src, err := os.ReadFile(bearofftest.Path(t, d))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, d.FileName()), src, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	plan := (&App{}).BearoffPlan(0, 4)
	if plan.Cores != 4 {
		t.Errorf("Cores = %d, want the 4 asked for", plan.Cores)
	}
	if plan.DefaultCores < 1 {
		t.Errorf("DefaultCores = %d, want at least one", plan.DefaultCores)
	}
	if plan.RateMeasured {
		t.Error("no rate was given, so none is measured")
	}
	if len(plan.Candidates) != len(bearoffgen.Candidates()) {
		t.Fatalf("%d candidates, want %d", len(plan.Candidates), len(bearoffgen.Candidates()))
	}

	var lastSize int64
	var lastSeconds float64
	for _, c := range plan.Candidates {
		if c.Size <= lastSize {
			t.Errorf("%s: size %d does not exceed the previous domain's", c.Domain, c.Size)
		}
		if c.Seconds <= lastSeconds {
			t.Errorf("%s: estimate %g s does not exceed the previous domain's", c.Domain, c.Seconds)
		}
		if !c.Fits && c.Reason == "" {
			t.Errorf("%s does not fit but says nothing about why", c.Domain)
		}
		lastSize, lastSeconds = c.Size, c.Seconds
	}

	// TS-06-06 is on disk here, and it is the one the fixture verifies.
	found := false
	for _, c := range plan.Candidates {
		if c.Checkers == 6 {
			found = true
			if !c.Present || c.Verdict != "verified" {
				t.Errorf("TS-06-06: Present=%v Verdict=%q, want a verified table", c.Present, c.Verdict)
			}
		}
	}
	if !found {
		t.Error("TS-06-06 is not among the candidates")
	}

	// The widest domain is 22 GB of table: no machine this test runs on has
	// that available, so it must be greyed rather than offered.
	last := plan.Candidates[len(plan.Candidates)-1]
	if plan.RAMAvailable > 0 && last.Fits {
		t.Errorf("%s (%d bytes of RAM) is offered on a machine with %d available", last.Domain, last.RAMNeeded, plan.RAMAvailable)
	}

	// The files block lists the two tables, and nothing else in the directory.
	if len(plan.Files) != 2 {
		t.Errorf("Files lists %d entries, want the two tables: %+v", len(plan.Files), plan.Files)
	}
	for _, f := range plan.Files {
		if f.Size <= 0 || f.Verdict != "verified" {
			t.Errorf("file %s: size %d verdict %q", f.Name, f.Size, f.Verdict)
		}
	}
}

// A paused run shows up as an interrupted domain with its percentage, so the
// tab can offer Reprendre / Supprimer at launch.
func TestBearoffPlan_ReportsAPausedRun(t *testing.T) {
	dir := isolate(t)
	d := bearoffgen.Domain{Kind: bearoffgen.TwoSidedKind, Points: 6, Checkers: 7}
	st := bearoffgen.NewTwoSidedState(d.Points, d.Checkers)
	st.Diagonal = bearoffgen.NumPositions(d.Points, d.Checkers) // about a quarter of the pairs
	if err := bearoffgen.WriteCheckpoint(dir, st); err != nil {
		t.Fatal(err)
	}

	plan := (&App{}).BearoffPlan(0, 2)
	var seen bool
	for _, c := range plan.Candidates {
		if c.Checkers != 7 {
			continue
		}
		seen = true
		if !c.Interrupted {
			t.Error("the paused domain does not report itself interrupted")
		}
		if c.Percent <= 0 || c.Percent >= 100 {
			t.Errorf("Percent = %g, want a partial run", c.Percent)
		}
	}
	if !seen {
		t.Fatal("TS-06-07 is not among the candidates")
	}
	// And a checkpoint is not a table: it must not appear in the file list.
	for _, f := range plan.Files {
		if strings.HasSuffix(f.Name, ".ckpt") {
			t.Errorf("the checkpoint %s is listed as a table", f.Name)
		}
	}

	if err := (&App{}).DiscardBearoffCheckpoint(d.Points, d.Checkers); err != nil {
		t.Fatal(err)
	}
	for _, c := range (&App{}).BearoffPlan(0, 2).Candidates {
		if c.Checkers == 7 && c.Interrupted {
			t.Error("the discarded checkpoint is still reported")
		}
	}
}

// A measured rate is what makes the estimate about this machine. A slower rate
// must predict a longer run.
func TestBearoffPlan_TheMeasuredRateMovesTheEstimate(t *testing.T) {
	isolate(t)
	fast := (&App{}).BearoffPlan(3e-10, 4)
	slow := (&App{}).BearoffPlan(6e-10, 4)
	if !fast.RateMeasured || !slow.RateMeasured {
		t.Fatal("a rate was given and is not reported as measured")
	}
	for i := range fast.Candidates {
		if slow.Candidates[i].Seconds <= fast.Candidates[i].Seconds {
			t.Fatalf("%s: the slower machine is not slower (%g vs %g s)",
				fast.Candidates[i].Domain, slow.Candidates[i].Seconds, fast.Candidates[i].Seconds)
		}
	}
}
