package storage

import (
	"context"
	"math"
	"sort"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The study plan of ADR-0077: the recurring-error families ranked by the
// winning chances studying them would recover, kept apart from the families
// whose sample is too thin to rank. The thresholds are fixed there, before any
// result was read, and change only with a new ADR.
const (
	// StudyPlanMinErrors is the fewest priced errors a family needs to enter
	// the plan.
	StudyPlanMinErrors = 5
	// StudyPlanZ is the normal quantile of the plan's two-sided 95 % interval.
	StudyPlanZ = 1.96
)

// StudyPlanPosition is one position of a family, with what its worst error
// there exceeded the reference player by.
type StudyPlanPosition struct {
	PositionID int64 `json:"PositionID"`
	// MatchID and Label name a match the error was played in, for the study
	// queue; Label is spelt the way the import report spells a match.
	MatchID int64  `json:"MatchID"`
	Label   string `json:"Label"`
	ErrorMP int64  `json:"ErrorMP"`
	// Excess is ℓ − d of the position's worst error, MWC fraction.
	Excess float64 `json:"Excess"`
}

// StudyPlanFamily is one (plan of play, decision kind, theme) family.
type StudyPlanFamily struct {
	GameType string `json:"GameType"`
	Kind     string `json:"Kind"`
	Theme    string `json:"Theme"`
	// Errors is the family's priced errors; Avoidable how many of them the
	// reference player would rarely make (ADR-0076).
	Errors    int `json:"Errors"`
	Avoidable int `json:"Avoidable"`
	// MeanLoss and MeanDifficulty are per error, MWC fraction.
	MeanLoss       float64 `json:"MeanLoss"`
	MeanDifficulty float64 `json:"MeanDifficulty"`
	// Recoverable is Σ(ℓ − d): frequency × mean excess loss, MWC fraction.
	Recoverable float64 `json:"Recoverable"`
	// Low and High bound Recoverable's 95 % interval (compound Poisson).
	Low  float64 `json:"Low"`
	High float64 `json:"High"`
	// Positions are the distinct positions, the largest excess first.
	Positions []StudyPlanPosition `json:"Positions"`
}

// StudyPlan is the plan of one filter.
type StudyPlan struct {
	NumDecisions int `json:"NumDecisions"`
	ThresholdMP  int `json:"ThresholdMP"`
	MinErrors    int `json:"MinErrors"`
	// Families are the families with enough evidence, ranked by Low.
	Families []StudyPlanFamily `json:"Families"`
	// Tentative are the themed families short of the evidence, by Recoverable.
	Tentative []StudyPlanFamily `json:"Tentative"`
	// Unthemed counts the priced errors no rule names; Unpriced the errors
	// without an MWC loss or a difficulty (money play, missing options).
	Unthemed int `json:"Unthemed"`
	Unpriced int `json:"Unpriced"`
	// UnthemedPositions and UnpricedPositions are the distinct positions of
	// those errors, ascending, so the counts outside the plan open them.
	UnthemedPositions []int64 `json:"UnthemedPositions"`
	UnpricedPositions []int64 `json:"UnpricedPositions"`
}

// StudyPlanRow is one classified error with its price, as a backend hands it
// to BuildStudyPlan and BuildStudyEffect. Loss is nil without an MWC (money
// play); Difficulty is nil too when the options carry no usable costs, and the
// error is then unpriced for the plan. Day is its match's date.
type StudyPlanRow struct {
	RecurringErrorRow
	MatchID    int64
	Label      string
	Day        string
	Loss       *float64
	Difficulty *float64
	Avoidable  bool
}

// BuildStudyPlan folds priced errors into families and ranks them (ADR-0077).
// It is pure and shared by every backend.
func BuildStudyPlan(rows []StudyPlanRow, numDecisions, thresholdMP int) *StudyPlan {
	type key struct{ gameType, kind, theme string }
	type acc struct {
		f              StudyPlanFamily
		loss, diff, sq float64
		worst          map[int64]StudyPlanPosition
	}
	plan := &StudyPlan{NumDecisions: numDecisions, ThresholdMP: thresholdMP, MinErrors: StudyPlanMinErrors,
		Families: []StudyPlanFamily{}, Tentative: []StudyPlanFamily{}}
	byKey := map[key]*acc{}
	var order []key
	unthemed, unpriced := map[int64]bool{}, map[int64]bool{}
	for _, r := range rows {
		if r.Loss == nil || r.Difficulty == nil {
			plan.Unpriced++
			unpriced[r.PositionID] = true
			continue
		}
		if r.Theme == RecurringThemeNone {
			plan.Unthemed++
			unthemed[r.PositionID] = true
			continue
		}
		k := key{r.GameType, r.Kind, r.Theme}
		a, ok := byKey[k]
		if !ok {
			a = &acc{f: StudyPlanFamily{GameType: k.gameType, Kind: k.kind, Theme: k.theme},
				worst: map[int64]StudyPlanPosition{}}
			byKey[k] = a
			order = append(order, k)
		}
		excess := *r.Loss - *r.Difficulty
		a.f.Errors++
		if r.Avoidable {
			a.f.Avoidable++
		}
		a.loss += *r.Loss
		a.diff += *r.Difficulty
		a.f.Recoverable += excess
		a.sq += excess * excess
		if cur, seen := a.worst[r.PositionID]; !seen || excess > cur.Excess {
			a.worst[r.PositionID] = StudyPlanPosition{PositionID: r.PositionID, MatchID: r.MatchID,
				Label: r.Label, ErrorMP: r.ErrorMP, Excess: excess}
		}
	}
	for _, k := range order {
		a := byKey[k]
		n := float64(a.f.Errors)
		a.f.MeanLoss = a.loss / n
		a.f.MeanDifficulty = a.diff / n
		half := StudyPlanZ * math.Sqrt(a.sq)
		a.f.Low, a.f.High = a.f.Recoverable-half, a.f.Recoverable+half
		positions := make([]StudyPlanPosition, 0, len(a.worst))
		for _, p := range a.worst {
			positions = append(positions, p)
		}
		sort.Slice(positions, func(i, j int) bool {
			if positions[i].Excess != positions[j].Excess {
				return positions[i].Excess > positions[j].Excess
			}
			return positions[i].PositionID < positions[j].PositionID
		})
		a.f.Positions = positions
		if a.f.Errors >= StudyPlanMinErrors && a.f.Low > 0 {
			plan.Families = append(plan.Families, a.f)
		} else {
			plan.Tentative = append(plan.Tentative, a.f)
		}
	}
	sortStudyFamilies(plan.Families, true)
	sortStudyFamilies(plan.Tentative, false)
	plan.UnthemedPositions = sortedIDs(unthemed)
	plan.UnpricedPositions = sortedIDs(unpriced)
	return plan
}

// sortedIDs is the set's ids in ascending order, never nil.
func sortedIDs(set map[int64]bool) []int64 {
	ids := make([]int64, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// sortStudyFamilies ranks families in a total order, by the interval's lower
// bound first when byLow, so both backends and every run rank ties alike.
func sortStudyFamilies(fs []StudyPlanFamily, byLow bool) {
	sort.Slice(fs, func(i, j int) bool {
		a, b := fs[i], fs[j]
		if byLow && a.Low != b.Low {
			return a.Low > b.Low
		}
		if a.Recoverable != b.Recoverable {
			return a.Recoverable > b.Recoverable
		}
		if a.Errors != b.Errors {
			return a.Errors > b.Errors
		}
		if a.GameType != b.GameType {
			return a.GameType < b.GameType
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.Theme < b.Theme
	})
}

// StudyPositionIDs are the positions of the plan's families picked by rank,
// in the plan's order and, inside a family, the largest excess first: rank 0
// takes the StudyWorstGroups first families, rank n>0 the n-th alone. A rank
// past the end yields nothing.
func (p *StudyPlan) StudyPositionIDs(rank int) []int64 {
	seen := map[int64]bool{}
	ids := []int64{}
	for _, f := range p.pick(rank) {
		for _, pos := range f.Positions {
			if !seen[pos.PositionID] {
				seen[pos.PositionID] = true
				ids = append(ids, pos.PositionID)
			}
		}
	}
	return ids
}

func (p *StudyPlan) pick(rank int) []StudyPlanFamily {
	if p == nil {
		return nil
	}
	fs := p.Families
	switch {
	case rank > 0 && rank <= len(fs):
		return fs[rank-1 : rank]
	case rank > 0:
		return nil
	case len(fs) > StudyWorstGroups:
		return fs[:StudyWorstGroups]
	}
	return fs
}

// QueueEntries turns the families picked by rank (as StudyPositionIDs) into a
// study queue, one entry per distinct position, in the plan's order.
func (p *StudyPlan) QueueEntries(rank int) []domain.StudyQueueEntry {
	seen := map[int64]bool{}
	out := []domain.StudyQueueEntry{}
	for _, f := range p.pick(rank) {
		for _, pos := range f.Positions {
			if seen[pos.PositionID] {
				continue
			}
			seen[pos.PositionID] = true
			out = append(out, domain.StudyQueueEntry{PositionID: pos.PositionID, MatchID: pos.MatchID,
				Reason: domain.StudyPlan, Label: pos.Label, ErrorMP: int(pos.ErrorMP), IsCube: f.Kind == "cube"})
		}
	}
	return out
}

// StudyPlanIDs are the positions to study from a filter's plan: the families
// picked by rank, then, when size is above zero, a random draw of at most size
// of them — StudyIDs for the plan.
func StudyPlanIDs(ctx context.Context, st Stores, scope string, filter StatsFilter, rank, size int) ([]int64, error) {
	plan, err := st.Stats().StudyPlan(ctx, scope, filter)
	if err != nil {
		return nil, err
	}
	ids := plan.StudyPositionIDs(rank)
	if size > 0 {
		ids = DrawStudyQuiz(ids, size, nil)
	}
	return ids, nil
}
