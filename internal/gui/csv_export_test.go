package gui

import (
	"os"
	"path/filepath"
	"testing"
)

// The file is the CSV the clipboard receives, byte for byte: no BOM, no
// line-ending rewrite.
func TestWriteCSVIsTheCopiedText(t *testing.T) {
	body := "Phase;Rang;id;Joueur\r\nGénérale;1;ha;Hugo Andrieu\r\n"
	path := filepath.Join(t.TempDir(), "open-classement-2026-09-24.csv")
	if err := writeCSV(path, body); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("file = %q, want the copied CSV %q", got, body)
	}
}

func TestWriteCSVRefusesAnEmptyPath(t *testing.T) {
	if err := writeCSV("", "a;b\n"); err == nil {
		t.Fatal("an empty path must be an error, not a file in the cwd")
	}
}

func TestCSVPathAddsTheExtension(t *testing.T) {
	for in, want := range map[string]string{
		"/x/classement":     "/x/classement.csv",
		"/x/classement.csv": "/x/classement.csv",
		"/x/classement.CSV": "/x/classement.CSV",
		"/x/v1.2":           "/x/v1.2.csv",
	} {
		if got := csvPath(in); got != want {
			t.Errorf("csvPath(%q) = %q, want %q", in, got, want)
		}
	}
}
