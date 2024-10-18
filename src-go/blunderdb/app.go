package main

import (
	"context"

    "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

func (a *App) SaveDatabaseDialog() (string, error) {
    return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
        Title:   "New Database File",
        Filters: []runtime.FileFilter{{DisplayName: "Database Files", Pattern: "*.db"}},
        CanCreateDirectories: true,
    })
}

func (a *App) OpenDatabaseDialog() (string, error) {
    return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
        Title:   "Open Database File",
        Filters: []runtime.FileFilter{{DisplayName: "Database Files", Pattern: "*.db"}},
    })
}

func (a *App) OpenPositionDialog() (string, error) {
    return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
        Title:   "Import Position File",
        Filters: []runtime.FileFilter{{DisplayName: "Position Files", Pattern: "*.txt"}},
    })
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}
