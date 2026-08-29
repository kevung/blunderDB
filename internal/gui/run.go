package gui

import (
	"context"
	"embed"
	"log/slog"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// Run starts the Wails GUI. The caller supplies the embedded frontend assets
// and icon, the initial window dimensions, the Database the gammonNet batch
// job (#129) runs against, and any extra structs to bind for the frontend
// (the App struct is created and bound here).
func Run(assets embed.FS, icon []byte, width, height int, db *database.Database, extraBinds []interface{}) error {
	app := NewApp(db)
	return wails.Run(&options.App{
		Title:  "blunderDB",
		Width:  width,
		Height: height,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 240, G: 240, B: 240, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       shutdown(extraBinds),
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

// shutdown closes anything bound to the frontend that owns resources — in practice the
// database.
//
// Without it the process simply exits with the database still open, and SQLite never gets
// to tidy up: the `-wal` and `-shm` files it keeps beside a database are removed when the
// last connection closes cleanly, and they were being left behind on every run. The
// single-writer lock is dropped here too, rather than relying on the kernel doing it when
// the process dies.
func shutdown(binds []interface{}) func(ctx context.Context) {
	return func(context.Context) {
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
