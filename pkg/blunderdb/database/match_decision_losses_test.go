package database

import (
	"math"
	"path/filepath"
	"testing"
)

// TestMatchDecisionLossesSumToMatchMWCLoss locks that the per-decision losses
// of each player add up to the MWC loss the match list shows for that player,
// on real analysed matches (one single game, the others several).
func TestMatchDecisionLossesSumToMatchMWCLoss(t *testing.T) {
	for _, file := range []string{
		"test.xg",
		"charlot1-charlot2_7p_2025-11-08-2305.xg",
		"HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg",
	} {
		t.Run(file, func(t *testing.T) {
			db := newTestDB(t)
			id, err := db.ImportXGMatch(filepath.Join("testdata", file))
			if err != nil {
				t.Fatalf("ImportXGMatch: %v", err)
			}
			matches, err := db.GetAllMatches()
			if err != nil {
				t.Fatalf("GetAllMatches: %v", err)
			}
			var shown *Match
			for i := range matches {
				if matches[i].ID == id {
					shown = &matches[i]
				}
			}
			if shown == nil {
				t.Fatalf("match %d not listed", id)
			}
			losses, err := db.GetMatchDecisionLosses(id)
			if err != nil {
				t.Fatalf("GetMatchDecisionLosses: %v", err)
			}
			var sum [2]float64
			var scored, games int
			lastGame := 0
			for _, d := range losses {
				if d.GameNumber != lastGame {
					games++
					lastGame = d.GameNumber
				}
				if d.MWCLoss != nil {
					sum[d.Player] += *d.MWCLoss
					scored++
				}
			}
			t.Logf("%s: %d decisions, %d scored, %d games, loss %.4f / %.4f", file, len(losses), scored, games, sum[0], sum[1])
			if scored == 0 {
				t.Fatalf("no decision scored")
			}
			if math.Abs(sum[0]-shown.MWCLoss) > 1e-9 || math.Abs(sum[1]-shown.MWCLoss2) > 1e-9 {
				t.Errorf("per-decision sums %v differ from the list (%v, %v)", sum, shown.MWCLoss, shown.MWCLoss2)
			}
		})
	}
}
