package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	// Validate that the file has a .db extension to prevent arbitrary file deletion
	if !strings.HasSuffix(strings.ToLower(filePath), ".db") {
		return fmt.Errorf("only .db files can be deleted")
	}
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
	// Open the file dialog with position and match file types
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Import Position or Match File",
		Filters: []runtime.FileFilter{
			{DisplayName: "All Supported Files (*.txt, *.xg, *.sgf, *.mat, *.bgf)", Pattern: "*.txt;*.xg;*.sgf;*.mat;*.bgf"},
			{DisplayName: "Position Files (*.txt)", Pattern: "*.txt"},
			{DisplayName: "XG Match Files (*.xg)", Pattern: "*.xg"},
			{DisplayName: "GnuBG Match Files (*.sgf)", Pattern: "*.sgf"},
			{DisplayName: "Jellyfish Match Files (*.mat)", Pattern: "*.mat"},
			{DisplayName: "BGBlitz Match Files (*.bgf)", Pattern: "*.bgf"},
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

// OpenPositionFilesDialog opens a multi-file selection dialog for position and match files.
func (a *App) OpenPositionFilesDialog() ([]string, error) {
	return runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Import Position or Match Files",
		Filters: []runtime.FileFilter{
			{DisplayName: "All Supported Files (*.txt, *.xg, *.sgf, *.mat, *.bgf)", Pattern: "*.txt;*.xg;*.sgf;*.mat;*.bgf"},
			{DisplayName: "Position Files (*.txt)", Pattern: "*.txt"},
			{DisplayName: "XG Match Files (*.xg)", Pattern: "*.xg"},
			{DisplayName: "GnuBG Match Files (*.sgf)", Pattern: "*.sgf"},
			{DisplayName: "Jellyfish Match Files (*.mat)", Pattern: "*.mat"},
			{DisplayName: "BGBlitz Match Files (*.bgf)", Pattern: "*.bgf"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
}

// OpenPositionFolderDialog opens a directory selection dialog for importing all files within.
func (a *App) OpenPositionFolderDialog() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Folder to Import",
	})
}

// supportedImportExtensions lists file extensions supported for position/match import.
var supportedImportExtensions = map[string]bool{
	".txt": true,
	".xg":  true,
	".sgf": true,
	".mat": true,
	".bgf": true,
}

// IsDirectory returns true if the given path is a directory.
func (a *App) IsDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// CollectImportableFiles recursively walks dirPath and returns all supported position/match files.
func (a *App) CollectImportableFiles(dirPath string) ([]string, error) {
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip files/dirs we can't access
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if supportedImportExtensions[ext] {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}
	return files, nil
}

// ReadFileContent reads and returns the content of a file as a string.
func (a *App) ReadFileContent(filePath string) (*FileDialogResponse, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return &FileDialogResponse{FilePath: filePath, Error: err.Error()}, err
	}
	return &FileDialogResponse{
		FilePath: filePath,
		Content:  string(content),
	}, nil
}

func (a *App) ShowAlert(message string) {
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   "Alert",
		Message: message,
	})
}

// ShowQuestionDialog displays a question dialog with custom buttons and returns which button was clicked.
func (a *App) ShowQuestionDialog(title, message string, buttons []string, defaultButton string) (string, error) {
	return runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         title,
		Message:       message,
		Buttons:       buttons,
		DefaultButton: defaultButton,
	})
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}
