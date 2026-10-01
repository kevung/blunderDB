package database

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/bgfparser"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/gnubgparser"
	"github.com/kevung/xgparser/xgparser"
)

// matchFixtures are every real match file under testdata/.
func matchFixtures(t *testing.T) []string {
	t.Helper()
	var files []string
	for _, pat := range []string{"testdata/*.xg", "testdata/*/*/*.xg", "testdata/*.sgf", "testdata/*.mat", "testdata/*.bgf"} {
		m, err := filepath.Glob(pat)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, m...)
	}
	if len(files) == 0 {
		t.Fatal("no match fixtures found")
	}
	return files
}

// TestMatchHashParity_DatabaseVsIngest checks that the two copies of the
// match hashes agree on every fixture: a divergence would deduplicate the same
// match differently depending on the import path.
func TestMatchHashParity_DatabaseVsIngest(t *testing.T) {
	for _, f := range matchFixtures(t) {
		t.Run(filepath.Base(f), func(t *testing.T) {
			var g *ingest.MatchGraph
			var wantHash, wantCanon string
			var err error
			switch strings.ToLower(filepath.Ext(f)) {
			case ".xg":
				g, err = ingest.MapXG(f)
				if err != nil {
					t.Fatal(err)
				}
				imp := xgparser.NewImport(f)
				segs, err := imp.GetFileSegments()
				if err != nil {
					t.Fatal(err)
				}
				m, err := xgparser.ParseXG(segs)
				if err != nil {
					t.Fatal(err)
				}
				wantHash, wantCanon = ComputeMatchHash(m), ComputeCanonicalMatchHashFromXG(m)
			case ".sgf", ".mat":
				g, err = ingest.MapGnuBG(f)
				if err != nil {
					t.Fatal(err)
				}
				var m *gnubgparser.Match
				if strings.HasSuffix(strings.ToLower(f), ".sgf") {
					m, err = gnubgparser.ParseSGFFile(f)
				} else {
					m, err = gnubgparser.ParseMATFile(f)
				}
				if err != nil {
					t.Fatal(err)
				}
				wantHash, wantCanon = ComputeGnuBGMatchHash(m), ComputeCanonicalMatchHashFromGnuBG(m)
			case ".bgf":
				g, err = ingest.MapBGF(f)
				if err != nil {
					t.Fatal(err)
				}
				m, err := bgfparser.ParseBGF(f)
				if err != nil {
					t.Fatal(err)
				}
				wantHash, wantCanon = ComputeBGFMatchHash(m), ComputeCanonicalMatchHashFromBGF(m)
			}
			if g.Match.MatchHash != wantHash {
				t.Errorf("match_hash: ingest %s, database %s", g.Match.MatchHash, wantHash)
			}
			if g.Match.CanonicalHash != wantCanon {
				t.Errorf("canonical_hash: ingest %s, database %s", g.Match.CanonicalHash, wantCanon)
			}
			t.Logf("REF %q %s %s", f, g.Match.MatchHash, g.Match.CanonicalHash)
		})
	}
}
