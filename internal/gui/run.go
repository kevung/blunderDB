package gui

import (
	"context"
	"embed"
	"log/slog"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// Run starts the Wails GUI. The caller supplies the embedded frontend assets
// and icon, the initial window dimensions, the Database the gammonNet batch
// job (#129) runs against, any extra structs to bind for the frontend (the
// App struct is created and bound here), and startupFilePath — the database
// file path the OS handed this process on the command line, "" on an
// ordinary launch (#241; see App.StartupFilePath).
func Run(assets embed.FS, icon []byte, width, height int, db *database.Database, extraBinds []interface{}, startupFilePath string) error {
	app := NewApp(db)
	app.startupFilePath = startupFilePath
	return wails.Run(&options.App{
		Title:  "blunderDB",
		Width:  width,
		Height: height,
		// Below this, the toolbar wraps and panels lose their minimum usable
		// size — nothing in the layout copes with a window smaller than the
		// content it was designed for (#215).
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 240, G: 240, B: 240, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       shutdown(app, extraBinds),
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: false, // Must be false on Linux: gtk_drag_dest_unset() prevents GTK drag signals from firing (Wails v2 bug #4743)
		},
		Bind: append([]interface{}{app}, extraBinds...),
		Linux: &linux.Options{
			Icon:                icon,
			WindowIsTranslucent: false,
			WebviewGpuPolicy:    linux.WebviewGpuPolicyNever,
			ProgramName:         "blunderDB",
		},
		// WebView2 applies the OS DPI scaling itself; pin its own zoom to 1.0 so
		// it doesn't compound with the CSS `zoom`/`--ui-scale` interface scale and
		// leave blank space at DPI > 100% on Windows (issue #64).
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			IsZoomControlEnabled: true,
			ZoomFactor:           1.0,
		},
		Debug: options.Debug{
			OpenInspectorOnStartup: false,
		},
	})
}

// shutdownJobGrace bounds how long shutdown waits for a cancelled batch to
// actually stop. A gammonNet goroutine cancelled mid-position still finishes
// that position — the search has no internal checkpoint — so the wait is a
// small multiple of one 2-ply position, not of the batch. Exceeded, the wait
// gives up and the close goes ahead: an application that refuses to quit is
// worse than the warning line a late write produces.
const shutdownJobGrace = 5 * time.Second

// shutdown stops the background jobs and then closes anything bound to the frontend that
// owns resources — in practice the database. In that order, and the order is the point.
//
// Without the close, the process simply exits with the database still open, and SQLite never
// gets to tidy up: the `-wal` and `-shm` files it keeps beside a database are removed when
// the last connection closes cleanly, and they were being left behind on every run. The
// single-writer lock is dropped here too, rather than relying on the kernel doing it when
// the process dies.
//
// Without the stop, that close happened UNDER the running jobs: the gammonNet batch writes
// an analysis per position from its own goroutine, and the GUI opens the user's file on a
// single connection (CLAUDE.md, "Notes & Gotchas"), so quitting mid-batch closed the
// connection out from under a writer — a race, not merely a lost batch, and one that only
// ever showed up as a warning in the log. It takes the App rather than the binds alone
// because the jobs are the App's (ADR-0045 §8 asks for it, and every batch benefits).
func shutdown(app *App, binds []interface{}) func(ctx context.Context) {
	return func(context.Context) {
		app.stopBackgroundJobs(shutdownJobGrace)

		for _, bind := range binds {
			closer, ok := bind.(interface{ Close() error })
			if !ok {
				continue
			}
			if err := closer.Close(); err != nil {
				slog.Warn("closing on shutdown", "err", err)
			}
		}
	}
}
