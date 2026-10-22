package storage

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

// TestSummaryOfCarriesNoIdentifier pins the public projection: a StatsResult
// full of identifiers (positions, matches, tournaments, opponents, tags)
// gives a summary whose JSON holds none of them and keeps the level figures.
func TestSummaryOfCarriesNoIdentifier(t *testing.T) {
	r := &StatsResult{
		Totals:        StatsTotals{NumMatches: 3, NumDecisions: 120},
		PRGlobal:      4.2,
		PRChecker:     3.9,
		PRCube:        6.1,
		PerTournament: []TournamentStats{{ID: 7, Name: "Secret Open"}},
		PerMatch:      []MatchStats{{ID: 11, PlayerName: "Rival Player"}},
		TopBlunders:   []BlunderEntry{{PositionID: 99, PlayerNames: "A vs B"}},
		PerTag:        []TagStats{{Tag: "#private"}},
		PerGameType:   []GameTypeStats{{GameType: "blitz"}},
		PerPhase:      []PhaseStats{{Phase: "race", PR: 2.5, NumDecisions: 40}},
		PerScore:      []ScoreCellStats{{MoverAway: 2, OpponentAway: 1, PR: 8, NumDecisions: 5}},
	}
	b, err := json.Marshal(SummaryOf(r))
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"Secret Open", "Rival Player", "A vs B", "#private", "PositionID", "TopBlunders", "PerMatch", "PerTournament", "PerTag", `"ID"`} {
		if strings.Contains(string(b), leak) {
			t.Errorf("summary leaks %q: %s", leak, b)
		}
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(b, &top); err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(top))
	for k := range top {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	want := "MWC7,MWC7Checker,MWC7Cube,MWCAvailable,MWCChecker,MWCCube,MWCGlobal,PRChecker,PRCube,PRGlobal,PRInterval,PerPhase,PerScore,Totals"
	if got := strings.Join(keys, ","); got != want {
		t.Errorf("summary keys = %s, want %s", got, want)
	}
	s := SummaryOf(r)
	if s.PRGlobal != 4.2 || len(s.PerPhase) != 1 || s.PerPhase[0].Phase != "race" || len(s.PerScore) != 1 {
		t.Errorf("summary lost the level figures: %+v", s)
	}
	if e := SummaryOf(nil); e.PerPhase == nil || e.PerScore == nil {
		t.Errorf("SummaryOf(nil) = %+v, want empty, non-nil breakdowns", e)
	}
}
