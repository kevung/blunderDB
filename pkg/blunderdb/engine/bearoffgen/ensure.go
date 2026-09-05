package bearoffgen

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
)

// EnsureDefaults generates whatever default table `dir` is missing, and points
// the engine at the one-sided one. It is what every mode calls to be able to
// answer: the GUI in the background at start-up, the CLI and the daemon
// synchronously, because a command that printed "unavailable" and exited would
// be worse than one that took six seconds once.
//
// loadOneSided is the engine's loader, passed in rather than imported: this
// package generates tables and knows nothing about who reads them, and the
// engine already imports nothing from here.
func EnsureDefaults(ctx context.Context, dir string, loadOneSided func(path string) error) error {
	for _, d := range Missing(dir) {
		slog.Info("generating a bearoff table", "domain", d.String(), "dir", dir)
		if _, err := Generate(ctx, dir, d, nil); err != nil {
			return fmt.Errorf("generating %s: %w", d, err)
		}
	}
	if loadOneSided == nil {
		return nil
	}
	d := Domain{Kind: OneSidedKind, Points: 6, Checkers: osCheckers}
	return loadOneSided(filepath.Join(dir, d.FileName()))
}

var ensureOnce sync.Once

// EnsureDefaultsOnce is EnsureDefaults, run at most once per process. The CLI
// calls it from any command that may need a table; the cost is paid by the
// first one that does, and the failure is logged rather than fatal — a
// command that only needed the estimate still works.
func EnsureDefaultsOnce(dir string, loadOneSided func(path string) error) {
	ensureOnce.Do(func() {
		if err := EnsureDefaults(context.Background(), dir, loadOneSided); err != nil {
			slog.Warn("could not prepare the bearoff tables; the exact regime will be unavailable", "err", err)
		}
	})
}
