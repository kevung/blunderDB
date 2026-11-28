package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/jobs"
)

// longOpEnd drains ch up to the first event of prefix that ends a task.
func longOpEnd(t *testing.T, ch <-chan event, prefix string) (event, []string) {
	t.Helper()
	var seen []string
	deadline := time.After(60 * time.Second)
	for {
		select {
		case e := <-ch:
			seen = append(seen, e.name)
			if strings.HasPrefix(e.name, prefix+":") && e.name != prefix+":progress" {
				return e, seen
			}
		case <-deadline:
			t.Fatalf("no end event; saw %v", seen)
		}
	}
}

func longOpApp(t *testing.T) *App {
	t.Helper()
	a, _ := matchWithoutStats(t)
	return a
}

func jobState(t *testing.T, a *App, id int64) jobs.State {
	t.Helper()
	for _, j := range a.jobs.List() {
		if j.ID == id {
			return j.State
		}
	}
	t.Fatalf("task %d not listed", id)
	return ""
}

func TestStartExportDatabase_ReportsThenWrites(t *testing.T) {
	a := longOpApp(t)
	events := recordEvents(a, nil)
	out := filepath.Join(t.TempDir(), "out.db")
	id := a.StartExportDatabase(database.ExportOptions{ExportPath: out, AllPositions: true, IncludeMatches: true, Metadata: map[string]string{}})
	end, seen := longOpEnd(t, events, "export-db")
	if end.name != "export-db:done" || id == 0 {
		t.Fatalf("ended with %s %v (id %d)", end.name, end.payload, id)
	}
	if seen[0] != "export-db:progress" {
		t.Errorf("events %v: no progress before the end", seen)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatal(err)
	}
	if s := jobState(t, a, id); s != jobs.Done {
		t.Errorf("task state %s, want done", s)
	}
}

func TestStartExportDatabase_CancelLeavesNoFile(t *testing.T) {
	a := longOpApp(t)
	events := recordEvents(a, nil)
	out := filepath.Join(t.TempDir(), "out.db")
	// The task cancelled from the panel at its first report, as the user
	// would: the export stops between two batches.
	hook := *a.emitHook.Load()
	cancelled := false
	wrapped := func(name string, payload ...any) {
		if name == "export-db:progress" && !cancelled {
			cancelled = true
			_ = a.CancelJob(payload[0].(LongOpProgress).ID)
		}
		hook(name, payload...)
	}
	a.emitHook.Store(&wrapped)
	id := a.StartExportDatabase(database.ExportOptions{ExportPath: out, AllPositions: true, IncludeMatches: true, Metadata: map[string]string{}})
	if end, seen := longOpEnd(t, events, "export-db"); end.name != "export-db:cancelled" {
		t.Fatalf("events %v, last %v: want export-db:cancelled", seen, end.payload)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("a cancelled export left %s (%v)", out, err)
	}
	if s := jobState(t, a, id); s != jobs.Cancelled {
		t.Errorf("task state %s, want cancelled", s)
	}
}

func TestStartPaperArchiveExport_WritesThePage(t *testing.T) {
	a := longOpApp(t)
	events := recordEvents(a, nil)
	out := filepath.Join(t.TempDir(), "archive.html")
	a.StartPaperArchiveExport(nil, out)
	end, _ := longOpEnd(t, events, "paper-archive")
	if end.name != "paper-archive:done" {
		t.Fatalf("ended with %s %v", end.name, end.payload)
	}
	if !database.IsPaperArchivePage(out) {
		t.Fatalf("%s is not an archive page", out)
	}
	res := end.payload.(map[string]any)["result"].(PaperArchiveExportResult)
	if res.Path != out || res.Matches != 1 {
		t.Fatalf("result %+v", res)
	}
}

func TestStartLongOp_RefusesABadPathAsAnError(t *testing.T) {
	a := longOpApp(t)
	events := recordEvents(a, nil)
	a.StartMatchSheet([]int64{1}, database.MatchSheetOptions{}, "relative.pdf")
	if end, _ := longOpEnd(t, events, "print-sheet"); end.name != "print-sheet:error" {
		t.Fatalf("ended with %s %v", end.name, end.payload)
	}
}

func TestStartLongOp_WithoutDatabase(t *testing.T) {
	a := NewApp(nil)
	events := recordEvents(a, nil)
	if id := a.StartPaperArchiveImport([]string{"x.png"}); id != 0 {
		t.Fatalf("id %d without a library", id)
	}
	if end, _ := longOpEnd(t, events, "paper-archive"); end.name != "paper-archive:error" {
		t.Fatalf("ended with %s", end.name)
	}
}
