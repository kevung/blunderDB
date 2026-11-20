package database

import (
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// An imported match carries its result, the one its stored games give, and
// the list of matches reads it back.
func TestImportStoresTheMatchResult(t *testing.T) {
	d := NewDatabase()
	if err := d.SetupDatabase(filepath.Join(t.TempDir(), "r.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	for _, imp := range []struct {
		file string
		fn   func(string) (int64, error)
	}{
		{"testdata/test.xg", d.ImportXGMatch},
		{"testdata/test.mat", d.ImportGnuBGMatch},
		{"testdata/gnubg_roll_then_resign_1p.mat", d.ImportGnuBGMatch},
	} {
		id, err := imp.fn(imp.file)
		if err != nil {
			t.Fatalf("%s: %v", imp.file, err)
		}
		m, err := d.GetMatchByID(id)
		if err != nil {
			t.Fatal(err)
		}
		games, err := d.GetGamesByMatch(id)
		if err != nil {
			t.Fatal(err)
		}
		final, winner := domain.MatchResultOf(m.MatchLength, games)
		if m.FinalScore == nil || *m.FinalScore != *final || m.Winner != winner {
			t.Errorf("%s: stored %v won by %d, games give %v won by %d", imp.file, m.FinalScore, m.Winner, *final, winner)
		}
		if winner == domain.WinnerUnfinished {
			t.Errorf("%s: a finished match has no winner (%v, length %d)", imp.file, *final, m.MatchLength)
		}
	}
}
