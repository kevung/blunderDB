package gui

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// --- IsDirectory --------------------------------------------------------

func TestIsDirectory(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	file := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	cases := []struct {
		name string
		path string
		want bool
	}{
		{"a directory", dir, true},
		{"a regular file", file, false},
		{"a non-existent path", filepath.Join(dir, "missing"), false},
		{"empty path", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := a.IsDirectory(c.path); got != c.want {
				t.Errorf("IsDirectory(%q) = %v, want %v", c.path, got, c.want)
			}
		})
	}
}

// --- PathExists -----------------------------------------------------------

func TestPathExists(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	file := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	cases := []struct {
		name string
		path string
		want bool
	}{
		{"empty path is never present", "", false},
		{"existing file", file, true},
		{"existing directory", dir, true},
		{"missing path", filepath.Join(dir, "nope"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := a.PathExists(c.path); got != c.want {
				t.Errorf("PathExists(%q) = %v, want %v", c.path, got, c.want)
			}
		})
	}
}

// --- CollectImportableFiles ------------------------------------------------

func TestCollectImportableFiles(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()

	files := []string{
		"position.txt",
		"match.xg",
		"match.xgp",
		"match.sgf",
		"match.mat",
		"match.bgf",
		"ignored.pdf",
		"ignored", // no extension
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}
	// Nested subdirectory: CollectImportableFiles walks recursively.
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "nested.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	got, err := a.CollectImportableFiles(dir)
	if err != nil {
		t.Fatalf("CollectImportableFiles: %v", err)
	}

	var gotNames []string
	for _, p := range got {
		gotNames = append(gotNames, filepath.Base(p))
	}
	sort.Strings(gotNames)

	want := []string{"match.bgf", "match.mat", "match.sgf", "match.xg", "match.xgp", "nested.txt", "position.txt"}
	if len(gotNames) != len(want) {
		t.Fatalf("CollectImportableFiles returned %v, want %v", gotNames, want)
	}
	for i := range want {
		if gotNames[i] != want[i] {
			t.Errorf("CollectImportableFiles()[%d] = %q, want %q (full: %v)", i, gotNames[i], want[i], gotNames)
		}
	}
}

func TestCollectImportableFilesIsCaseInsensitiveOnExtension(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "MATCH.XG"), []byte("x"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	got, err := a.CollectImportableFiles(dir)
	if err != nil {
		t.Fatalf("CollectImportableFiles: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected the uppercase-extension file to be collected: got %v", got)
	}
}

func TestCollectImportableFilesOnMissingDirectory(t *testing.T) {
	a := NewApp()
	// filepath.Walk reports the missing-root error to the walk callback (not
	// to Walk's own return value); CollectImportableFiles's callback swallows
	// every per-entry error unconditionally ("skip files/dirs we can't
	// access"), so Walk itself sees no error and CollectImportableFiles
	// returns (nil, nil) rather than surfacing "directory does not exist".
	dir := t.TempDir()
	missing := filepath.Join(dir, "does-not-exist")
	got, err := a.CollectImportableFiles(missing)
	if err != nil {
		t.Errorf("CollectImportableFiles(missing dir) error = %v, want nil (the per-entry error is swallowed)", err)
	}
	if len(got) != 0 {
		t.Errorf("CollectImportableFiles(missing dir) = %v, want empty", got)
	}
}

// --- ReadFileContent --------------------------------------------------------

func TestReadFileContent(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	file := filepath.Join(dir, "position.txt")
	content := "some position text\nwith multiple lines\n"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	resp, err := a.ReadFileContent(file)
	if err != nil {
		t.Fatalf("ReadFileContent: %v", err)
	}
	if resp.FilePath != file {
		t.Errorf("FilePath = %q, want %q", resp.FilePath, file)
	}
	if resp.Content != content {
		t.Errorf("Content = %q, want %q", resp.Content, content)
	}
	if resp.Error != "" {
		t.Errorf("Error = %q, want empty", resp.Error)
	}
}

func TestReadFileContentMissingFile(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.txt")

	resp, err := a.ReadFileContent(missing)
	if err == nil {
		t.Fatal("expected an error reading a missing file")
	}
	if resp.Error == "" {
		t.Error("expected FileDialogResponse.Error to be populated")
	}
	if resp.FilePath != missing {
		t.Errorf("FilePath = %q, want %q (kept even on error)", resp.FilePath, missing)
	}
}

// --- DeleteFile ---------------------------------------------------------

func TestDeleteFile(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	file := filepath.Join(dir, "database.db")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := a.DeleteFile(file); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("expected file to be removed, stat err = %v", err)
	}
}

func TestDeleteFileRejectsNonDbExtension(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	file := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := a.DeleteFile(file); err == nil {
		t.Error("expected DeleteFile to reject a non-.db path")
	}
	if _, err := os.Stat(file); err != nil {
		t.Errorf("file should survive a rejected delete: stat err = %v", err)
	}
}

func TestDeleteFileIsCaseInsensitiveOnExtension(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	file := filepath.Join(dir, "database.DB")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := a.DeleteFile(file); err != nil {
		t.Fatalf("DeleteFile should accept an uppercase .DB extension: %v", err)
	}
}

func TestDeleteFileMissingPathWithDbExtension(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	missing := filepath.Join(dir, "gone.db")
	if err := a.DeleteFile(missing); err == nil {
		t.Error("expected an error deleting a non-existent file")
	}
}
