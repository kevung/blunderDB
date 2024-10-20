package main

import (
	"context"
    "io/ioutil"
    "encoding/json"
    "github.com/wailsapp/wails/v2/pkg/runtime"
    "blunderdb/backend"
    // "fmt"
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

type FileDialogResponse struct {
    FilePath string `json:"file_path"`
    Content  string `json:"content"`
    Error    string `json:"error,omitempty"` // Optional field to capture any errors
}

func (a *App) OpenPositionDialog() (*FileDialogResponse, error) {
    // Open the file dialog
    filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
        Title:   "Import Position File",
        Filters: []runtime.FileFilter{{DisplayName: "Position Files", Pattern: "*.txt"}},
    })

    if err != nil {
        return &FileDialogResponse{Error: err.Error()}, err
    }

    if filePath == "" {
        return &FileDialogResponse{Error: "No file selected"}, nil
    }

    // Read the file content
    content, err := ioutil.ReadFile(filePath)
    if err != nil {
        return &FileDialogResponse{Error: err.Error()}, err
    }

    return &FileDialogResponse{
        FilePath: filePath,
        Content:  string(content),
    }, nil
}

func (a *App) SaveImportedPosition(positionJSON string) error {
    var position backend.Position
    err := json.Unmarshal([]byte(positionJSON), &position)
    if err != nil {
        return err
    }

    db, errDb := backend.SetupDatabase()
    if errDb != nil {
        return errDb
    }

    return backend.SavePosition(db, position) // Ensure SavePosition replaces SaveGameState
}

func (a *App) ShowAlert(message string) {
    runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
        Type:    runtime.InfoDialog,
        Title:   "Alert",
        Message: message,
    })
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}
