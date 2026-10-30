package database

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

var updateMarksGolden = flag.Bool("update-marks-golden", false,
	"rewrite testdata/transcription_marks.golden.jsonl from the current manifest")

const marksGolden = "testdata/transcription_marks.golden.jsonl"

// materializeWithVideo writes testdata/gnubg_selfplay_drops_7p.mat as a
// match whose video is source and whose first two Actions carry Repères; the
// third has none.
func materializeWithVideo(t *testing.T, d *Database, fixture, source string) int64 {
	t.Helper()
	text, err := os.ReadFile(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := transcript.FromMAT(string(text))
	if err != nil {
		t.Fatal(err)
	}
	doc.Header.VideoSource = source
	for i, a := range doc.Actions {
		if i >= 2 {
			break
		}
		roll, tick := int64(1000*(i+1)), int64(1000*(i+1)+400)
		a.RollTickMS, a.TickMS = &roll, &tick
		doc.Actions[i] = a
	}
	saved, err := d.MaterializeTranscription(doc.Header, doc.Actions)
	if err != nil {
		t.Fatalf("MaterializeTranscription: %v", err)
	}
	return saved.MatchID
}

// Only a match whose video is a file on this disk gives lines; a URL is
// counted, a missing file is named, a move without a Repère is skipped.
func TestTranscriptionMarks_LocalSourceOnly(t *testing.T) {
	d := newTestDB(t)
	video := filepath.Join(t.TempDir(), "match.mp4")
	if err := os.WriteFile(video, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	local := materializeWithVideo(t, d, "gnubg_selfplay_drops_7p.mat", video)
	other := materializeWithVideo(t, d, "charlot1-charlot2_7p_2025-11-08-2305.mat", "https://www.youtube.com/watch?v=abc")
	missing := filepath.Join(t.TempDir(), "absent.mp4")

	marks, sum, err := d.TranscriptionMarks(0)
	if err != nil {
		t.Fatal(err)
	}
	if sum.URLSources != 1 || len(sum.Missing) != 0 || sum.Matches != 1 {
		t.Errorf("summary = %+v, want 1 URL source and 1 match", sum)
	}
	if len(marks) != 2 || sum.Marks != 2 {
		t.Fatalf("got %d marks, want 2 (the third move has no Repère)", len(marks))
	}
	for _, m := range marks {
		if m.MatchID != local || m.Source != video || (m.RollTickMS == nil && m.TickMS == nil) {
			t.Errorf("unexpected mark %+v", m)
		}
	}

	if err := d.SetMatchVideoSource(other, missing); err != nil {
		t.Fatal(err)
	}
	if _, sum, err = d.TranscriptionMarks(0); err != nil || sum.URLSources != 0 || len(sum.Missing) != 1 || sum.Missing[0] != missing {
		t.Errorf("a local source absent from the disk must be named, not exported: %+v, %v", sum, err)
	}

	// --match narrows to one match; a URL match alone gives nothing.
	one, sum, err := d.TranscriptionMarks(local)
	if err != nil || len(one) != 2 || sum.URLSources != 0 {
		t.Errorf("TranscriptionMarks(%d) = %d marks, %+v, %v", local, len(one), sum, err)
	}

	// The manifest writes nothing.
	again, _, _ := d.TranscriptionMarks(0)
	if len(again) != len(marks) {
		t.Error("a second read differs from the first")
	}
}

// The line format is stable: one JSON object per mark, byte for byte.
func TestTranscriptionMarks_Golden(t *testing.T) {
	d := newTestDB(t)
	video := filepath.Join(t.TempDir(), "match.mp4")
	if err := os.WriteFile(video, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	materializeWithVideo(t, d, "gnubg_selfplay_drops_7p.mat", video)
	marks, _, err := d.TranscriptionMarks(0)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, m := range marks {
		if err := enc.Encode(m); err != nil {
			t.Fatal(err)
		}
	}
	got := strings.ReplaceAll(buf.String(), strings.ReplaceAll(video, `\`, `\\`), "/videos/match.mp4")
	if *updateMarksGolden {
		if err := os.WriteFile(marksGolden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(marksGolden)
	if err != nil {
		t.Fatalf("%v (run with -update-marks-golden)", err)
	}
	if got != string(want) {
		t.Errorf("manifest differs from %s (run with -update-marks-golden if intended)\n got: %s\nwant: %s", marksGolden, got, want)
	}
}
