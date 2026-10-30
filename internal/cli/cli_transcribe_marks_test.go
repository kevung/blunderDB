package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// --marks writes the Database's lines as JSONL and counts the URL source in
// its summary; nothing but the manifest file is written.
func TestTranscribeMarks_WritesLocalSourcesOnly(t *testing.T) {
	c, dbPath := setupCLIWithDB(t)
	video := filepath.Join(t.TempDir(), "match.mp4")
	if err := os.WriteFile(video, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	save := func(fixture, source string, timed bool) {
		raw, err := os.ReadFile(filepath.Join("testdata", fixture))
		if err != nil {
			t.Fatal(err)
		}
		doc, err := transcript.FromMAT(string(raw))
		if err != nil {
			t.Fatal(err)
		}
		doc.Header.VideoSource = source
		if timed {
			tick := int64(2500)
			doc.Actions[0].TickMS = &tick
		}
		if _, err := c.db.MaterializeTranscription(doc.Header, doc.Actions); err != nil {
			t.Fatal(err)
		}
	}
	save("gnubg_selfplay_drops_7p.mat", video, true)
	save("charlot1-charlot2_7p_2025-11-08-2305.mat", "https://youtu.be/abc", true)
	c.db.Close()

	out := filepath.Join(t.TempDir(), "marks.jsonl")
	stdout, err := runTranscribeOnFile(t, "--db", dbPath, "--marks", out, "--format", "json")
	if err != nil {
		t.Fatalf("--marks: %v", err)
	}
	var sum struct {
		database.TranscriptionMarksSummary
		Written string `json:"written"`
	}
	if err := json.Unmarshal([]byte(stdout), &sum); err != nil || sum.URLSources != 1 || sum.Marks != 1 || sum.Written != out {
		t.Fatalf("summary %q (%v)", stdout, err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	var mark database.TranscriptionMark
	if len(lines) != 1 || json.Unmarshal([]byte(lines[0]), &mark) != nil || mark.Source != video || mark.TickMS == nil || *mark.TickMS != 2500 {
		t.Fatalf("manifest = %q", raw)
	}

	if _, err := runTranscribeOnFile(t, "--db", dbPath, "--marks", out, "--check"); err == nil {
		t.Error("--marks with --check must be refused")
	}
	if _, err := runTranscribeOnFile(t, "--marks", out); err == nil {
		t.Error("--marks without --db must be refused")
	}
}
