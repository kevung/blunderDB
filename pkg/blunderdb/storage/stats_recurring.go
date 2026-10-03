package storage

import "sort"

// Recurring errors: the filter's errors grouped by what kind of mistake they
// are, so that "my errors in holding games" is one row and not a search.
//
// A group is a plan of play (the position's derived game_type) crossed with a
// theme. A checker theme is engine.ExplainChecker's token; a cube theme is the
// direction of the cube error (ClassifyCubeDirection). An error neither rule
// names confidently is grouped under RecurringThemeNone rather than guessed.

// RecurringThemeNone groups the errors no rule names: the cost is real, the
// reason is not one of the themes.
const RecurringThemeNone = "none"

// RecurringErrorGroup is one (plan of play, decision kind, theme) group.
type RecurringErrorGroup struct {
	// GameType is the domain.GameType token of the side on roll.
	GameType string `json:"GameType"`
	// Kind is "checker" or "cube": the two families have disjoint themes.
	Kind string `json:"Kind"`
	// Theme is an engine.Explanation theme (checker), a CubeCell* error cell
	// (cube), or RecurringThemeNone.
	Theme string `json:"Theme"`
	// Count is the number of decisions in the group; SumErrorMP their summed
	// error in millipoints of normalised equity, the stats' own column.
	Count      int   `json:"Count"`
	SumErrorMP int64 `json:"SumErrorMP"`
	// PRCost is what the group adds to the filter's Performance Rating: the
	// PR formula applied to the group's errors over ALL the filter's counted
	// decisions. The groups' PRCost sum to at most the filter's PR.
	PRCost float64 `json:"PRCost"`
	// PositionIDs are the distinct positions behind the group, in
	// decreasing order of error: what a click on the group opens.
	PositionIDs []int64 `json:"PositionIDs"`
}

// RecurringErrors is the grouped result for one filter.
type RecurringErrors struct {
	// NumDecisions is the filter's counted decisions, the PR denominator.
	NumDecisions int `json:"NumDecisions"`
	// ThresholdMP is the library's Error threshold (ADR-0046): a decision
	// costing less is not an error and joins no group.
	ThresholdMP int                   `json:"ThresholdMP"`
	Groups      []RecurringErrorGroup `json:"Groups"`
}

// RecurringErrorRow is one classified error, as a backend hands it to
// GroupRecurringErrors.
type RecurringErrorRow struct {
	PositionID int64
	GameType   string
	Kind       string
	Theme      string
	ErrorMP    int64
}

// GroupRecurringErrors folds classified errors into groups ranked by summed
// cost, the heaviest first. It is pure and shared by every backend so that the
// grouping and the ranking cannot differ between SQLite and PostgreSQL.
func GroupRecurringErrors(rows []RecurringErrorRow, numDecisions, thresholdMP int) *RecurringErrors {
	type key struct{ gameType, kind, theme string }
	type acc struct {
		g     RecurringErrorGroup
		worst map[int64]int64
	}
	byKey := map[key]*acc{}
	var order []key
	for _, r := range rows {
		k := key{r.GameType, r.Kind, r.Theme}
		a, ok := byKey[k]
		if !ok {
			a = &acc{g: RecurringErrorGroup{GameType: r.GameType, Kind: r.Kind, Theme: r.Theme}, worst: map[int64]int64{}}
			byKey[k] = a
			order = append(order, k)
		}
		a.g.Count++
		a.g.SumErrorMP += r.ErrorMP
		if cur, seen := a.worst[r.PositionID]; !seen || r.ErrorMP > cur {
			a.worst[r.PositionID] = r.ErrorMP
		}
	}

	out := &RecurringErrors{NumDecisions: numDecisions, ThresholdMP: thresholdMP, Groups: []RecurringErrorGroup{}}
	for _, k := range order {
		a := byKey[k]
		if numDecisions > 0 {
			a.g.PRCost = 500 * float64(a.g.SumErrorMP) / 1000 / float64(numDecisions)
		}
		ids := make([]int64, 0, len(a.worst))
		for id := range a.worst {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool {
			if a.worst[ids[i]] != a.worst[ids[j]] {
				return a.worst[ids[i]] > a.worst[ids[j]]
			}
			return ids[i] < ids[j]
		})
		a.g.PositionIDs = ids
		out.Groups = append(out.Groups, a.g)
	}
	// A total order, so that both backends and every run rank ties alike.
	sort.Slice(out.Groups, func(i, j int) bool {
		a, b := out.Groups[i], out.Groups[j]
		if a.SumErrorMP != b.SumErrorMP {
			return a.SumErrorMP > b.SumErrorMP
		}
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		if a.GameType != b.GameType {
			return a.GameType < b.GameType
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.Theme < b.Theme
	})
	return out
}
