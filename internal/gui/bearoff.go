package gui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/mem"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// Bearoff tables are generated here, not downloaded and not embedded
// (ADR-0027). What used to be a 1.2 GB release asset and 8.2 MB compiled into
// every binary is now a few seconds of arithmetic on the machine that wants
// the answer, and the result is verified against gnubg's own fingerprint.
//
// Two default tables are made on first launch, in the background and in
// silence: the user did not ask for them and has nothing to decide. The Eval
// panel only speaks up when a position actually needs a table that is not
// ready yet — that is the moment the absence means something.
//
// Progress reaches the frontend through the same Wails events the download
// used, so the Bearoff tab did not have to learn a new vocabulary:
//
//	bearoff:progress {done, total, domain}   (throttled)
//	bearoff:done     {domain}
//	bearoff:error    {message}

// BearoffStatus is what the configuration dialog and the Eval panel show.
type BearoffStatus struct {
	// Ready is true when both default tables are present and verified.
	Ready bool `json:"ready"`
	// Generating names the domain being generated, empty when idle.
	Generating string `json:"generating"`
	// Missing lists the default domains this machine does not have yet.
	Missing []string `json:"missing"`
	// DataDir is where the tables live.
	DataDir string `json:"data_dir"`
	// ActiveDomain is the chequer count of the resolved two-sided source, 0
	// when there is none.
	ActiveDomain int `json:"active_domain"`
	// ActiveOrigin is where that source came from.
	ActiveOrigin string `json:"active_origin"`
	// ExternalPath is the user-configured .bd, if any.
	ExternalPath string `json:"external_path"`
	// OneSidedReady reports whether the EPC has its table.
	OneSidedReady bool `json:"one_sided_ready"`
}

// BearoffFile is one file in the data directory, as the tab lists it.
type BearoffFile struct {
	Name    string `json:"name"`
	Domain  string `json:"domain"`
	Size    int64  `json:"size"`
	Verdict string `json:"verdict"` // verified | unverified | corrupt
}

// BearoffCandidate is a domain the user may generate, with everything needed
// to decide before committing: what it will weigh, what it needs to be held in
// memory while it is made, how long it should take here, and whether this
// machine can do it at all.
type BearoffCandidate struct {
	Domain string `json:"domain"`
	// Kind is "two-sided" or "one-sided": the first widens the exact cube
	// verdict, the second how far from home the EPC can answer.
	Kind      string `json:"kind"`
	Points    int    `json:"points"`
	Checkers  int    `json:"checkers"`
	Size      int64  `json:"size"`
	RAMNeeded int64  `json:"ram_needed"`
	// Seconds is the estimate on the chosen core count. The measured ETA
	// replaces it as soon as the run reports progress.
	Seconds float64 `json:"seconds"`
	// Fits is false when RAMNeeded exceeds what the machine has available;
	// Reason says which of the two limits it hit.
	Fits   bool   `json:"fits"`
	Reason string `json:"reason"`
	// Present is true when the table is already there, with its verdict.
	Present bool   `json:"present"`
	Verdict string `json:"verdict"`
	// Interrupted reports a paused run, with how far it got.
	Interrupted bool    `json:"interrupted"`
	Percent     float64 `json:"percent"`
}

// BearoffPlan is what the Bearoff tab renders: the machine's limits, the files
// on disk, and every domain that can be asked for.
type BearoffPlan struct {
	Cores        int                `json:"cores"`
	DefaultCores int                `json:"default_cores"`
	RAMAvailable int64              `json:"ram_available"`
	RateMeasured bool               `json:"rate_measured"`
	DataDir      string             `json:"data_dir"`
	Files        []BearoffFile      `json:"files"`
	Candidates   []BearoffCandidate `json:"candidates"`
}

var (
	bearoffMu     sync.Mutex
	bearoffCancel context.CancelFunc
	bearoffActive string
	// bearoffPause distinguishes a pause from a cancellation: both cancel the
	// context, only a pause leaves a checkpoint behind.
	bearoffPause bool
)

// BearoffStatus reports the state of the bearoff sources.
func (a *App) BearoffStatus() BearoffStatus {
	dir := race.DataDir()
	st := BearoffStatus{
		DataDir:       dir,
		OneSidedReady: engine.OneSidedReady(),
	}
	for _, d := range bearoffgen.Missing(dir) {
		st.Missing = append(st.Missing, d.String())
	}
	st.Ready = len(st.Missing) == 0

	bearoffMu.Lock()
	st.Generating = bearoffActive
	bearoffMu.Unlock()

	if src := race.Resolve(); src != nil {
		st.ActiveDomain = src.Checkers()
		st.ActiveOrigin = src.Origin()
	}
	return st
}

// EnsureBearoffTables generates whatever default table is missing, in the
// background, and returns at once. Called on start-up: a first launch spends
// about six seconds of one core making the two tables while the user does
// whatever they opened the application to do.
//
// A second call while one is running is a no-op — not an error: start-up and a
// manual retry from the configuration dialog both go through here.
func (a *App) EnsureBearoffTables() {
	bearoffMu.Lock()
	if bearoffCancel != nil {
		bearoffMu.Unlock()
		return
	}
	dir := race.DataDir()
	missing := bearoffgen.Missing(dir)
	if len(missing) == 0 {
		bearoffMu.Unlock()
		a.loadOneSided(dir)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	bearoffCancel = cancel
	bearoffMu.Unlock()

	go func() {
		defer func() {
			bearoffMu.Lock()
			bearoffCancel, bearoffActive = nil, ""
			bearoffMu.Unlock()
			cancel()
		}()
		for _, d := range missing {
			if ctx.Err() != nil {
				return
			}
			if err := a.generateOne(ctx, dir, d, DefaultBearoffCores(0)); err != nil {
				if ctx.Err() == nil { // a real failure, not a cancellation
					a.emitBearoff("bearoff:error", map[string]string{"message": err.Error()})
				}
				return
			}
		}
	}()
}

// generateOne makes one table and wires the result back into the engine.
func (a *App) generateOne(ctx context.Context, dir string, d bearoffgen.Domain, workers int) error {
	bearoffMu.Lock()
	bearoffActive = d.String()
	bearoffMu.Unlock()
	started := time.Now()

	// Progress is throttled: the two-sided sweep calls back once per diagonal,
	// which is 924 times for TS-06-06 and would be 12 376 for TS-06-11.
	last := time.Now()
	progress := func(done, total int64) {
		if time.Since(last) < 200*time.Millisecond && done != total {
			return
		}
		last = time.Now()
		a.emitBearoff("bearoff:progress", map[string]any{
			"done": done, "total": total, "domain": d.String(),
		})
	}

	bearoffMu.Lock()
	pausable := bearoffPause || d.Kind == bearoffgen.TwoSidedKind
	bearoffMu.Unlock()
	if _, err := bearoffgen.GenerateWith(ctx, dir, d, bearoffgen.RunOptions{
		Workers:  workers,
		Progress: progress,
		Pausable: pausable,
	}); err != nil {
		return fmt.Errorf("generating %s: %w", d, err)
	}

	switch d.Kind {
	case bearoffgen.OneSidedKind:
		a.loadOneSided(dir)
	default:
		race.Invalidate()
		race.Resolve() // pick the new table up at once
	}
	// The rate this run measured, so the next estimate is about this machine
	// rather than the one the constant was fitted on. 0 for a domain too small
	// to say anything; the frontend persists whatever is non-zero.
	a.emitBearoff("bearoff:done", map[string]any{
		"domain": d.String(),
		"rate":   d.MeasuredRate(time.Since(started), workers),
	})
	return nil
}

// loadOneSided points the EPC at the widest one-sided table in dir.
func (a *App) loadOneSided(dir string) {
	d := bearoffgen.Domain{Kind: bearoffgen.OneSidedKind, Points: 6, Checkers: 15}
	path := filepath.Join(dir, d.FileName())
	if _, err := os.Stat(path); err != nil {
		return
	}
	if err := engine.LoadOneSided(path); err != nil {
		a.emitBearoff("bearoff:error", map[string]string{"message": err.Error()})
	}
}

// emitBearoff sends an event only when the Wails context exists: generation
// starts before the window does, and on a headless test there is no frontend
// at all.
func (a *App) emitBearoff(name string, payload any) {
	if a.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(a.ctx, name, payload)
}

// CancelBearoffGeneration stops an in-flight generation and keeps nothing: no
// checkpoint, so the next run starts clean. Pausing is the other button.
//
// The partial `.part` file is left where it is: it is the trace of an
// interrupted run, and never a candidate for Resolve.
func (a *App) CancelBearoffGeneration() {
	bearoffMu.Lock()
	bearoffPause = false
	if bearoffCancel != nil {
		bearoffCancel()
	}
	bearoffMu.Unlock()
}

// DiscardBearoffCheckpoint drops a paused run's state — the "Supprimer" of the
// "TS-06-09 interrompue à 43 %" line.
func (a *App) DiscardBearoffCheckpoint(points, checkers int) error {
	d := bearoffgen.Domain{Kind: bearoffgen.TwoSidedKind, Points: points, Checkers: checkers}
	return bearoffgen.RemoveCheckpoint(race.DataDir(), d)
}

// GenerateBearoffTable makes one table on demand, named by its domain — the
// Bearoff tab's own button, for a domain wider than the defaults. `cores` is
// how many to use, 0 for the default (every core but one, so the machine stays
// usable while a domain that takes minutes is made). A run started here is
// pausable: it picks up a checkpoint if there is one, and leaves one behind if
// it is paused.
func (a *App) GenerateBearoffTable(kind string, points, checkers, cores int) error {
	d := bearoffgen.Domain{Kind: bearoffgen.TwoSidedKind, Points: points, Checkers: checkers}
	if kind == "one-sided" {
		d = bearoffgen.Domain{Kind: bearoffgen.OneSidedKind, Points: points, Checkers: 15}
		cores = 1 // sequential by construction
	}

	bearoffMu.Lock()
	if bearoffCancel != nil {
		bearoffMu.Unlock()
		return fmt.Errorf("a bearoff table is already being generated")
	}
	ctx, cancel := context.WithCancel(context.Background())
	bearoffCancel, bearoffPause = cancel, false
	bearoffMu.Unlock()

	go func() {
		defer func() {
			bearoffMu.Lock()
			bearoffCancel, bearoffActive = nil, ""
			bearoffMu.Unlock()
			cancel()
		}()
		dir := race.DataDir()
		if err := a.generateOne(ctx, dir, d, DefaultBearoffCores(cores)); err != nil && ctx.Err() == nil {
			a.emitBearoff("bearoff:error", map[string]string{"message": err.Error()})
		}
	}()
	return nil
}

// PauseBearoffGeneration stops the run in flight and writes its state down, so
// the next Generate on the same domain continues rather than starts over.
func (a *App) PauseBearoffGeneration() {
	bearoffMu.Lock()
	bearoffPause = true
	if bearoffCancel != nil {
		bearoffCancel()
	}
	bearoffMu.Unlock()
}

// DefaultBearoffCores turns a user choice into a core count: 0 means every
// core but one, so a domain that takes minutes does not take the machine with
// it.
func DefaultBearoffCores(cores int) int {
	if cores > 0 {
		if max := runtime.NumCPU(); cores > max {
			return max
		}
		return cores
	}
	if n := runtime.NumCPU(); n > 1 {
		return n - 1
	}
	return 1
}

// BearoffPlan is everything the Bearoff tab needs to render itself: what is on
// disk, what can be generated, what it costs here. `rate` is the sweep rate
// measured on this machine (Config.GetBearoffRate, 0 when none), `cores` the
// user's choice.
func (a *App) BearoffPlan(rate float64, cores int) BearoffPlan {
	dir := race.DataDir()
	workers := DefaultBearoffCores(cores)
	available := availableRAM()

	plan := BearoffPlan{
		Cores:        workers,
		DefaultCores: DefaultBearoffCores(0),
		RAMAvailable: available,
		RateMeasured: rate > 0,
		DataDir:      dir,
	}

	for _, name := range bearoffFileNames(dir) {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		verdict, d, _ := bearoffgen.Verify(path)
		plan.Files = append(plan.Files, BearoffFile{
			Name:    name,
			Domain:  d.String(),
			Size:    info.Size(),
			Verdict: verdict.String(),
		})
	}

	domains := append(bearoffgen.Candidates(), bearoffgen.OneSidedCandidates()...)
	for _, d := range domains {
		// The one-sided sweep is strictly sequential, so its estimate must not
		// pretend the core count helps.
		w := workers
		kind := "two-sided"
		if d.Kind == bearoffgen.OneSidedKind {
			w, kind = 1, "one-sided"
		}
		c := BearoffCandidate{
			Domain:    d.String(),
			Kind:      kind,
			Points:    d.Points,
			Checkers:  d.Checkers,
			Size:      d.Size(),
			RAMNeeded: d.RAMNeeded(),
			Seconds:   d.EstimateDuration(rate, w).Seconds(),
			Fits:      true,
		}
		if verdict, got, err := bearoffgen.Verify(filepath.Join(dir, d.FileName())); err == nil && got == d {
			c.Present, c.Verdict = true, verdict.String()
		}
		// Only the two-sided sweep checkpoints; asking for a one-sided one is
		// harmless and always answers "none".
		if done, total, err := bearoffgen.CheckpointProgress(dir, d); err == nil && total > 0 {
			c.Interrupted = true
			c.Percent = 100 * float64(done) / float64(total)
		}
		// A domain that does not fit is offered greyed with the reason, not
		// hidden: "TS-06-13 needs 24 GB" is an answer, an absent row is not.
		if available > 0 && c.RAMNeeded > available {
			c.Fits, c.Reason = false, "ram"
		}
		plan.Candidates = append(plan.Candidates, c)
	}
	return plan
}

// bearoffFileNames lists the .bd files in dir, sorted. Checkpoints and the
// debris of dead runs are not tables and are not listed here.
func bearoffFileNames(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".bd" {
			continue
		}
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}

// availableRAM is what the machine can still hand out, 0 when it cannot be
// read — in which case nothing is greyed out rather than everything.
func availableRAM() int64 {
	v, err := mem.VirtualMemory()
	if err != nil || v == nil {
		return 0
	}
	return int64(v.Available)
}

// DeleteBearoffTable removes one generated table. Nothing else is touched, and
// the next launch regenerates it if it is a default.
func (a *App) DeleteBearoffTable(name string) error {
	dir := race.DataDir()
	p := filepath.Join(dir, filepath.Base(name)) // never escape the data dir
	race.Invalidate()                            // release the handle first: Windows cannot remove an open file
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(p + ".part"); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(p + ".ckpt"); err != nil && !os.IsNotExist(err) {
		return err
	}
	race.Resolve() // re-resolve so the panel downgrades immediately
	return nil
}

// OpenBearoffFileDialog lets the user pick a .bd table of their own — one
// generated elsewhere, or gnubg's, which blunderDB reads unchanged. It wins
// over the generated tables when its domain is wider.
func (a *App) OpenBearoffFileDialog() (string, error) {
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select two-sided bearoff database",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "gnubg bearoff database (*.bd)", Pattern: "*.bd"},
		},
	})
}
