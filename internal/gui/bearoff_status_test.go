package gui

import (
	"os"
	"path/filepath"
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
