package document

import (
	"context"
	"errors"
	"testing"
	"time"
)

func checkProgress(t *testing.T, path string) {
	t.Helper()
	var got []Progress
	_, err := AnalyzeFile(path, AnalyzeOptions{Title: "x", Progress: func(p Progress) { got = append(got, p) }})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("no progress reported")
	}
	prev := 0
	for _, p := range got {
		if p.Page < prev || p.Page > p.Pages || p.Pages != got[0].Pages {
			t.Fatalf("progress not monotone or inconsistent: %+v after page %d", p, prev)
		}
		prev = p.Page
	}
	last := got[len(got)-1]
	if last.Page != last.Pages {
		t.Fatalf("last report %+v is not the total", last)
	}
}

func TestAnalyzeProgressPDF(t *testing.T)    { checkProgress(t, "testdata/xg-diagrammes.pdf") }
func TestAnalyzeProgressImages(t *testing.T) { checkProgress(t, "testdata/sans-pips") }

func TestAnalyzeProgressCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pages := 0
	_, err := AnalyzeFile("testdata/sans-pips", AnalyzeOptions{Context: ctx, Progress: func(Progress) {
		if pages++; pages == 1 {
			cancel()
		}
	}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if pages != 1 {
		t.Fatalf("%d reports after cancel, want 1", pages)
	}
}

func TestThrottledProgress(t *testing.T) {
	n := 0
	f := ThrottledProgress(func(Progress) { n++ }, time.Hour)
	for i := 1; i <= 10; i++ {
		f(Progress{Page: i, Pages: 10})
	}
	if n != 2 {
		t.Fatalf("%d calls, want first and last", n)
	}
}
