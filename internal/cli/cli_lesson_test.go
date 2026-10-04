package cli

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The lesson sub-commands write a Lesson, reorder and edit its steps, show
// it, and export it with what its steps show.
func TestCLI_LessonLifecycle(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	collID, _ := seedCollection(t, cli, "Primes", 2)
	run := func(args ...string) string {
		t.Helper()
		return captureStdout(t, func() {
			if err := cli.Run(append([]string{"lesson"}, append(args, "--db", dbPath)...)); err != nil {
				t.Fatalf("lesson %v: %v", args, err)
			}
		})
	}
	run("create", "--name", "Jouer contre une prime")
	textFile := filepath.Join(t.TempDir(), "step.txt")
	if err := os.WriteFile(textFile, []byte("Comptez.\nPuis jouez."), 0o600); err != nil {
		t.Fatal(err)
	}
	run("add-step", "--lesson", "1", "--title", "Intro", "--text", "Bienvenue")
	run("add-step", "--lesson", "1", "--title", "Série", "--text-file", textFile, "--collection", itoa64(collID))
	run("reorder", "--lesson", "1", "--steps", "2,1")
	run("edit-step", "--lesson", "1", "--step", "1", "--title", "Accueil")

	var l domain.Lesson
	if err := json.Unmarshal([]byte(run("show", "--id", "1", "--format", "json")), &l); err != nil {
		t.Fatal(err)
	}
	if len(l.Steps) != 2 || l.Steps[0].Title != "Série" || l.Steps[1].Title != "Accueil" ||
		l.Steps[1].Text != "Bienvenue" || l.Steps[0].CollectionID != collID || l.Steps[0].Text != "Comptez.\nPuis jouez." {
		t.Fatalf("lesson = %+v", l)
	}
	if out := run("list"); !strings.Contains(out, "Jouer contre une prime") {
		t.Fatalf("list:\n%s", out)
	}
	out := filepath.Join(t.TempDir(), "lecon.db")
	run("export", "--id", "1", "--out", out, "--watermark", "Cours")
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("export wrote nothing: %v", err)
	}
	run("remove-step", "--step", "2")
	run("delete", "--id", "1", "--confirm")
	if out := run("list"); !strings.Contains(out, "No lessons.") {
		t.Fatalf("list after delete:\n%s", out)
	}
}

// `export --type database` is the whole library: its Lessons leave with it.
func TestCLI_DatabaseExportCarriesLessons(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	seedCollection(t, cli, "Primes", 1)
	if err := cli.Run([]string{"lesson", "create", "--db", dbPath, "--name", "Sauvegardée"}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "backup.db")
	if err := cli.Run([]string{"export", "--db", dbPath, "--type", "database", "--file", out}); err != nil {
		t.Fatal(err)
	}
	raw, err := sql.Open("sqlite", out)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	var n int
	if err := raw.QueryRow(`SELECT COUNT(*) FROM lesson WHERE name = 'Sauvegardée'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("lessons in the exported file = %d, %v; want 1", n, err)
	}
}

// The reader's progress is written by `done` alone, withdrawn by `--undo`,
// and read back by `progress`; showing the lesson writes nothing.
func TestCLI_LessonProgress(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	run := func(args ...string) string {
		t.Helper()
		return captureStdout(t, func() {
			if err := cli.Run(append([]string{"lesson"}, append(args, "--db", dbPath)...)); err != nil {
				t.Fatalf("lesson %v: %v", args, err)
			}
		})
	}
	type row struct {
		StepID int64 `json:"stepId"`
		Done   bool  `json:"done"`
	}
	progress := func() []row {
		t.Helper()
		var rows []row
		if err := json.Unmarshal([]byte(run("progress", "--id", "1", "--format", "json")), &rows); err != nil {
			t.Fatal(err)
		}
		return rows
	}
	run("create", "--name", "Parcours")
	run("add-step", "--lesson", "1", "--title", "Un", "--text", "a")
	run("add-step", "--lesson", "1", "--title", "Deux", "--text", "b")
	run("show", "--id", "1")
	if rows := progress(); len(rows) != 2 || rows[0].Done || rows[1].Done {
		t.Fatalf("progress before any gesture = %+v", rows)
	}
	run("done", "--step", "2")
	if rows := progress(); rows[0].Done || !rows[1].Done {
		t.Fatalf("progress after done = %+v", rows)
	}
	if out := run("progress", "--id", "1"); !strings.Contains(out, "1 / 2 steps done") {
		t.Fatalf("progress text:\n%s", out)
	}
	run("done", "--step", "2", "--undo")
	if rows := progress(); rows[1].Done {
		t.Fatalf("progress after undo = %+v", rows)
	}
}
