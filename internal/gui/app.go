package gui

import (
	"bytes"
	"compress/gzip"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// demoDBGz is a small, self-contained sample database (a couple of imported
// matches grouped into a tournament, with analysis) embedded gzip-compressed so
// the guided tours and first-time users have real content to explore. Regenerate
// with: ./blunderdb create -db demo.db && ./blunderdb import -db demo.db -type
// match -file <a.xg> ... ; sqlite3 demo.db 'PRAGMA journal_mode=DELETE; VACUUM;'
// ; gzip -9 -c demo.db > internal/gui/demo.db.gz
//
//go:embed demo.db.gz
var demoDBGz []byte

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// PrepareDemoDatabase decompresses the embedded sample database to a fresh
// temporary file and returns its path. The frontend then opens it through the
// normal open-database flow, so loading the demo behaves exactly like opening
// any other database (a fresh copy each time, never mutating the embedded data).
func (a *App) PrepareDemoDatabase() (string, error) {
	gz, err := gzip.NewReader(bytes.NewReader(demoDBGz))
	if err != nil {
		return "", fmt.Errorf("reading demo database: %w", err)
	}
	defer gz.Close()

	data, err := io.ReadAll(gz)
	if err != nil {
		return "", fmt.Errorf("decompressing demo database: %w", err)
	}

	f, err := os.CreateTemp("", "blunderdb-demo-*.db")
	if err != nil {
		return "", fmt.Errorf("creating temp demo database: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return "", fmt.Errorf("writing temp demo database: %w", err)
	}
	return f.Name(), nil
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
			{DisplayName: "All Supported Files (*.txt, *.xg, *.xgp, *.sgf, *.mat, *.bgf)", Pattern: "*.txt;*.xg;*.xgp;*.sgf;*.mat;*.bgf"},
			{DisplayName: "Position Files (*.txt)", Pattern: "*.txt"},
			{DisplayName: "XG Match Files (*.xg)", Pattern: "*.xg"},
			{DisplayName: "XG Position Files (*.xgp)", Pattern: "*.xgp"},
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
			{DisplayName: "All Supported Files (*.txt, *.xg, *.xgp, *.sgf, *.mat, *.bgf)", Pattern: "*.txt;*.xg;*.xgp;*.sgf;*.mat;*.bgf"},
			{DisplayName: "Position Files (*.txt)", Pattern: "*.txt"},
			{DisplayName: "XG Match Files (*.xg)", Pattern: "*.xg"},
			{DisplayName: "XG Position Files (*.xgp)", Pattern: "*.xgp"},
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
	".xgp": true,
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
	_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
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

// CopyImageToClipboard takes base64-encoded PNG data and copies it to the system clipboard.
func (a *App) CopyImageToClipboard(base64Data string) error {
	pngData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return fmt.Errorf("failed to decode base64 data: %w", err)
	}

	switch goruntime.GOOS {
	case "linux":
		// Try xclip first (supports MIME types)
		if path, err := exec.LookPath("xclip"); err == nil {
			cmd := exec.Command(path, "-selection", "clipboard", "-t", "image/png")
			cmd.Stdin = bytes.NewReader(pngData)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("xclip failed: %w", err)
			}
			return nil
		}
		// Try wl-copy for Wayland
		if path, err := exec.LookPath("wl-copy"); err == nil {
			cmd := exec.Command(path, "--type", "image/png")
			cmd.Stdin = bytes.NewReader(pngData)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("wl-copy failed: %w", err)
			}
			return nil
		}
		return fmt.Errorf("no clipboard tool found (install xclip or wl-copy)")

	case "darwin":
		// Use osascript to set clipboard to PNG via a temp file
		tmpFile, err := os.CreateTemp("", "blunderdb-clipboard-*.png")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())
		if _, err := tmpFile.Write(pngData); err != nil {
			tmpFile.Close()
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		tmpFile.Close()
		cmd := exec.Command("osascript", "-e", fmt.Sprintf(`set the clipboard to (read (POSIX file "%s") as «class PNGf»)`, tmpFile.Name()))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("osascript failed: %w", err)
		}
		return nil

	case "windows":
		// Write to temp file and use PowerShell
		tmpFile, err := os.CreateTemp("", "blunderdb-clipboard-*.png")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())
		if _, err := tmpFile.Write(pngData); err != nil {
			tmpFile.Close()
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		tmpFile.Close()
		script := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.Clipboard]::SetImage([System.Drawing.Image]::FromFile('%s'))`, tmpFile.Name())
		cmd := exec.Command("powershell", "-Command", script)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("powershell clipboard failed: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unsupported OS: %s", goruntime.GOOS)
	}
}
