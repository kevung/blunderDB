package main

import (
	"context"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
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
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:                "New Database File",
		Filters:              []runtime.FileFilter{{DisplayName: "Database Files (*.db)", Pattern: "*.db"}},
		CanCreateDirectories: true,
	})
	if err != nil || filePath == "" {
		return filePath, err
	}
	// Ensure .db extension is present
	if !strings.HasSuffix(strings.ToLower(filePath), ".db") {
		filePath += ".db"
	}
	return filePath, nil
}

func (a *App) OpenDatabaseDialog() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Open Database File",
		Filters: []runtime.FileFilter{{DisplayName: "Database Files (*.db)", Pattern: "*.db"}},
	})
}

func (a *App) OpenImportDatabaseDialog() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Import Database File",
		Filters: []runtime.FileFilter{{DisplayName: "Database Files (*.db)", Pattern: "*.db"}},
	})
}

func (a *App) OpenExportDatabaseDialog() (string, error) {
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:                "Export Database File",
		Filters:              []runtime.FileFilter{{DisplayName: "Database Files (*.db)", Pattern: "*.db"}},
		CanCreateDirectories: true,
	})
	if err != nil || filePath == "" {
		return filePath, err
	}
	// Ensure .db extension is present
	if !strings.HasSuffix(strings.ToLower(filePath), ".db") {
		filePath += ".db"
	}
	return filePath, nil
}

func (a *App) DeleteFile(filePath string) error {
	err := os.Remove(filePath)
	if err != nil {
		return err
	}
	return nil
}

type FileDialogResponse struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
	Error    string `json:"error,omitempty"` // Optional field to capture any errors
}

func (a *App) OpenPositionDialog() (*FileDialogResponse, error) {
	// Open the file dialog with both .txt and .xg file types
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Import Position or XG Match File",
		Filters: []runtime.FileFilter{
			{DisplayName: "All Supported Files (*.txt, *.xg)", Pattern: "*.txt;*.xg"},
			{DisplayName: "Position Files (*.txt)", Pattern: "*.txt"},
			{DisplayName: "XG Match Files (*.xg)", Pattern: "*.xg"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})

	if err != nil {
		return &FileDialogResponse{Error: err.Error()}, err
	}

	if filePath == "" {
		return &FileDialogResponse{Error: "No file selected"}, nil
	}

	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return &FileDialogResponse{Error: err.Error()}, err
	}

	return &FileDialogResponse{
		FilePath: filePath,
		Content:  string(content),
	}, nil
}

func (a *App) OpenXGFileDialog() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Import XG Match File",
		Filters: []runtime.FileFilter{{DisplayName: "XG Match Files (*.xg)", Pattern: "*.xg"}},
	})
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
