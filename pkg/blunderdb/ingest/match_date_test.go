package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A match file without a readable date of play gives an undated match, never
// the time of the import.
func TestParseMatchDate_NoDateStaysUndated(t *testing.T) {
	for _, in := range []string{"", "  ", "someday", "17-05-2025", "1899-12-30 00:00:00"} {
		if got := parseMatchDate(in); !got.IsZero() {
			t.Errorf("parseMatchDate(%q) = %v, want the zero time", in, got)
		}
	}
	day := time.Date(2025, 5, 17, 0, 0, 0, 0, time.UTC)
	for _, in := range []string{"2025-05-17", "2025.05.17", "2025/05/17", "May 17, 2025", "2025-05-17 00:00:00"} {
		if got := parseMatchDate(in); !got.Equal(day) {
			t.Errorf("parseMatchDate(%q) = %v, want %v", in, got, day)
		}
	}
}

func TestMapGnuBGText_WithoutEventDateIsUndated(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "gnubg_roll_then_resign_1p.mat"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	cut := strings.Index(text, "\n")
	if !strings.Contains(text[:cut], "EventDate") {
		t.Fatalf("fixture's first line is no longer its date: %q", text[:cut])
	}
	g, err := MapGnuBGText(text[cut+1:])
	if err != nil {
		t.Fatal(err)
	}
	if !g.Match.MatchDate.IsZero() {
		t.Fatalf("undated .mat imported with date %v", g.Match.MatchDate)
	}
}
