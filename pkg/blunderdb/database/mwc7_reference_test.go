package database

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The 7-point MWC loss of an imported match is the match's MWC loss rescaled,
// the same in the list badge and in the match's detail, with an interval
// drawn from its games; on test.xg (seven points) it is the MWC loss itself,
// close to the total MWC cost eXtreme Gammon reports for each player.
func TestMWC7_ReferenceMatches(t *testing.T) {
	files := []string{
		"test.xg",
		"HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg",
		"charlot1-charlot2_7p_2025-11-08-2305.xg",
	}
	// eXtreme Gammon's total MWC cost per player of test.xg.
	xg := map[string]float64{"Kévin Unger": 0.3003, "Maxence Job": 0.5120}
	for _, f := range files {
		d := NewDatabase()
		if err := d.SetupDatabase(filepath.Join(t.TempDir(), "m1.db")); err != nil {
			t.Fatal(err)
		}
		if _, err := d.ImportXGMatch(filepath.Join("testdata", f)); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		matches, err := d.GetAllMatches()
		if err != nil || len(matches) != 1 {
			t.Fatalf("%s: %d matches, %v", f, len(matches), err)
		}
		m := matches[0]
		detail, err := d.GetMatchDetailStats(m.ID)
		if err != nil {
			t.Fatal(err)
		}
		k := math.Sqrt(7 / float64(m.MatchLength))
		for _, p := range []struct {
			name   string
			loss   float64
			badge  domain.MWC7
			detail domain.MWC7
		}{
			{m.Player1Name, m.MWCLoss, m.MWC7, detail.Player1.MWC7},
			{m.Player2Name, m.MWCLoss2, m.MWC7P2, detail.Player2.MWC7},
		} {
			e := p.detail
			t.Logf("%s %s (%dp): L=%.4f L7=%.4f [%.4f, %.4f] D=%.0f [%.0f, %.0f]",
				f, p.name, m.MatchLength, p.loss, e.Loss, e.Low, e.High, e.Elo, e.EloLow, e.EloHigh)
			if !e.Available || math.Abs(e.Loss-k*p.loss) > 1e-9 || math.Abs(p.badge.Loss-e.Loss) > 1e-9 {
				t.Errorf("%s %s: badge %+v, detail %+v, MWC loss %v", f, p.name, p.badge, e, p.loss)
			}
			if !e.HasInterval || e.Low > e.Loss || e.High < e.Loss || e.EloLow > e.Elo || e.EloHigh < e.Elo {
				t.Errorf("%s %s: interval %+v", f, p.name, e)
			}
			if want, ok := xg[p.name]; ok && f == "test.xg" && math.Abs(e.Loss-want) > 0.15*want {
				t.Errorf("%s %s: L7 %.4f, eXtreme Gammon reports %.4f", f, p.name, e.Loss, want)
			}
		}
		d.Close()
	}
}
