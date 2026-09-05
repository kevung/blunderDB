package gui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

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

var (
	bearoffMu     sync.Mutex
	bearoffCancel context.CancelFunc
	bearoffActive string
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
			if err := a.generateOne(ctx, dir, d); err != nil {
				if ctx.Err() == nil { // a real failure, not a cancellation
					a.emitBearoff("bearoff:error", map[string]string{"message": err.Error()})
				}
				return
			}
		}
	}()
}

// generateOne makes one table and wires the result back into the engine.
func (a *App) generateOne(ctx context.Context, dir string, d bearoffgen.Domain) error {
	bearoffMu.Lock()
	bearoffActive = d.String()
	bearoffMu.Unlock()

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

	if _, err := bearoffgen.Generate(ctx, dir, d, progress); err != nil {
		return fmt.Errorf("generating %s: %w", d, err)
	}

	switch d.Kind {
	case bearoffgen.OneSidedKind:
		a.loadOneSided(dir)
	default:
		race.Invalidate()
		race.Resolve() // pick the new table up at once
	}
	a.emitBearoff("bearoff:done", map[string]string{"domain": d.String()})
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
	runtime.EventsEmit(a.ctx, name, payload)
}

// CancelBearoffGeneration stops an in-flight generation. The partial file is
// left where it is: it is the trace of an interrupted run, and never a
// candidate for Resolve.
func (a *App) CancelBearoffGeneration() {
	bearoffMu.Lock()
	if bearoffCancel != nil {
		bearoffCancel()
	}
	bearoffMu.Unlock()
}

// GenerateBearoffTable makes one table on demand, named by its domain — the
// Bearoff tab's own button, for a domain wider than the defaults.
func (a *App) GenerateBearoffTable(kind string, points, checkers int) error {
	d := bearoffgen.Domain{Kind: bearoffgen.TwoSidedKind, Points: points, Checkers: checkers}
	if kind == "one-sided" {
		d = bearoffgen.Domain{Kind: bearoffgen.OneSidedKind, Points: points, Checkers: 15}
	}

	bearoffMu.Lock()
	if bearoffCancel != nil {
		bearoffMu.Unlock()
		return fmt.Errorf("a bearoff table is already being generated")
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
		dir := race.DataDir()
		if err := a.generateOne(ctx, dir, d); err != nil && ctx.Err() == nil {
			a.emitBearoff("bearoff:error", map[string]string{"message": err.Error()})
		}
	}()
	return nil
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
	race.Resolve() // re-resolve so the panel downgrades immediately
	return nil
}

// OpenBearoffFileDialog lets the user pick a .bd table of their own — one
// generated elsewhere, or gnubg's, which blunderDB reads unchanged. It wins
// over the generated tables when its domain is wider.
func (a *App) OpenBearoffFileDialog() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select two-sided bearoff database",
		Filters: []runtime.FileFilter{
			{DisplayName: "gnubg bearoff database (*.bd)", Pattern: "*.bd"},
		},
	})
}
