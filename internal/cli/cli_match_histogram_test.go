package cli

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func TestHistogramLineNamesTheCountedBands(t *testing.T) {
	bounds := storage.DecisionTimeHistogramBounds
	counts := make([]int, len(bounds)+1)
	counts[0], counts[1], counts[len(bounds)] = 4, 11, 1
	if got, want := histogramLine(bounds, counts), "<2 s: 4, 2-5 s: 11, >120 s: 1"; got != want {
		t.Errorf("histogramLine = %q, want %q", got, want)
	}
	if got := histogramLine(bounds, make([]int, len(bounds)+1)); got != "" {
		t.Errorf("an empty histogram printed %q", got)
	}
	if got := histogramLine(bounds, nil); got != "" {
		t.Errorf("a histogram from an older server printed %q", got)
	}
}
