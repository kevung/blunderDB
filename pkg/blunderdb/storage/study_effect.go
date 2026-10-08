package storage

import (
	"math"
	"sort"
)

// The before/after measure of ADR-0078: for each studied family of the study
// plan (ADR-0077), the family's MWC loss rate in real play before and after the
// day it was first studied, and the change with its 95 % interval. The
// thresholds are fixed there, before any result was read.
const (
	// StudyEffectMinDecisions is the fewest decisions of the family's plan
	// and kind each window needs before a direction is stated.
	StudyEffectMinDecisions = 30

	// The verdicts of a family's change.
	StudyEffectImproved     = "improved"
	StudyEffectWorse        = "worse"
	StudyEffectUndetermined = "undetermined"
	StudyEffectInsufficient = "insufficient"
)

// StudyEffectWindow is one side of the study day.
type StudyEffectWindow struct {
	// Decisions counts the counted match-play decisions of the family's plan
	// and kind; Errors the family's priced errors among them.
	Decisions int `json:"Decisions"`
	Errors    int `json:"Errors"`
	// Loss is Σℓ of those errors and Rate Loss / Decisions, MWC fraction.
	Loss float64 `json:"Loss"`
	Rate float64 `json:"Rate"`
	sq   float64
}

// StudyEffectFamily is one studied family, measured on either side of the day
// it was first studied.
type StudyEffectFamily struct {
	GameType string `json:"GameType"`
	Kind     string `json:"Kind"`
	Theme    string `json:"Theme"`
	// StudiedOn is the UTC day of the first study action on one of the
	// family's positions, "YYYY-MM-DD"; Studied how many of its positions have
	// been studied at all.
	StudiedOn string            `json:"StudiedOn"`
	Studied   int               `json:"Studied"`
	Before    StudyEffectWindow `json:"Before"`
	After     StudyEffectWindow `json:"After"`
	// Gain is Before.Rate − After.Rate (positive: less lost since), with its
	// 95 % interval.
	Gain    float64 `json:"Gain"`
	Low     float64 `json:"Low"`
	High    float64 `json:"High"`
	Verdict string  `json:"Verdict"`
}

// StudyEffect is the before/after measure of one filter.
type StudyEffect struct {
	MinDecisions int `json:"MinDecisions"`
	// Families are the studied families, those with a verdict first.
	Families []StudyEffectFamily `json:"Families"`
	// Unstudied counts the filter's families no study action has reached.
	Unstudied int `json:"Unstudied"`
}

// StudyEffectDecisions counts a filter's counted match-play decisions of one
// plan of play and kind played on one day.
type StudyEffectDecisions struct {
	GameType string
	Kind     string
	Day      string
	Count    int
}

// BuildStudyEffect measures every studied family (ADR-0078). rows are the
// filter's classified errors, each with its match day; decisions the counted
// match-play decisions by plan, kind and day; studied the UTC day of each
// position's first study action. It is pure and shared by every backend.
func BuildStudyEffect(rows []StudyPlanRow, decisions []StudyEffectDecisions, studied map[int64]string) *StudyEffect {
	type key struct{ gameType, kind, theme string }
	type acc struct {
		f       StudyEffectFamily
		members map[int64]bool
		errs    []StudyPlanRow
	}
	byKey := map[key]*acc{}
	var order []key
	for _, r := range rows {
		if r.Theme == RecurringThemeNone {
			continue
		}
		k := key{r.GameType, r.Kind, r.Theme}
		a, ok := byKey[k]
		if !ok {
			a = &acc{f: StudyEffectFamily{GameType: k.gameType, Kind: k.kind, Theme: k.theme}, members: map[int64]bool{}}
			byKey[k] = a
			order = append(order, k)
		}
		a.errs = append(a.errs, r)
		if a.members[r.PositionID] {
			continue
		}
		a.members[r.PositionID] = true
		if day, ok := studied[r.PositionID]; ok && day != "" {
			a.f.Studied++
			if a.f.StudiedOn == "" || day < a.f.StudiedOn {
				a.f.StudiedOn = day
			}
		}
	}
	out := &StudyEffect{MinDecisions: StudyEffectMinDecisions, Families: []StudyEffectFamily{}}
	for _, k := range order {
		a := byKey[k]
		f := a.f
		if f.StudiedOn == "" {
			out.Unstudied++
			continue
		}
		for _, d := range decisions {
			if d.GameType != f.GameType || d.Kind != f.Kind || d.Day == "" {
				continue
			}
			if w := f.window(d.Day); w != nil {
				w.Decisions += d.Count
			}
		}
		for _, r := range a.errs {
			if r.Loss == nil || r.Day == "" {
				continue
			}
			if w := f.window(r.Day); w != nil {
				w.Errors++
				w.Loss += *r.Loss
				w.sq += *r.Loss * *r.Loss
			}
		}
		f.measure()
		out.Families = append(out.Families, f)
	}
	sort.SliceStable(out.Families, func(i, j int) bool {
		a, b := out.Families[i], out.Families[j]
		if ra, rb := effectRank(a.Verdict), effectRank(b.Verdict); ra != rb {
			return ra < rb
		}
		if a.Gain != b.Gain {
			return a.Gain > b.Gain
		}
		if a.StudiedOn != b.StudiedOn {
			return a.StudiedOn < b.StudiedOn
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

// window is the side of the study day a match day falls on, nil on the day
// itself: whether that match came before the study is unknown.
func (f *StudyEffectFamily) window(day string) *StudyEffectWindow {
	if len(day) > 10 {
		day = day[:10]
	}
	switch {
	case day < f.StudiedOn:
		return &f.Before
	case day > f.StudiedOn:
		return &f.After
	}
	return nil
}

// measure turns the two windows into rates, the change and its interval
// (compound Poisson on each side, ADR-0078 rule 5), and the verdict.
func (f *StudyEffectFamily) measure() {
	var variance float64
	for _, w := range []*StudyEffectWindow{&f.Before, &f.After} {
		if w.Decisions > 0 {
			n := float64(w.Decisions)
			w.Rate = w.Loss / n
			variance += w.sq / (n * n)
		}
	}
	f.Gain = f.Before.Rate - f.After.Rate
	half := StudyPlanZ * math.Sqrt(variance)
	f.Low, f.High = f.Gain-half, f.Gain+half
	switch {
	case f.Before.Decisions < StudyEffectMinDecisions || f.After.Decisions < StudyEffectMinDecisions:
		f.Verdict = StudyEffectInsufficient
	case f.Low > 0:
		f.Verdict = StudyEffectImproved
	case f.High < 0:
		f.Verdict = StudyEffectWorse
	default:
		f.Verdict = StudyEffectUndetermined
	}
}

func effectRank(verdict string) int {
	switch verdict {
	case StudyEffectImproved:
		return 0
	case StudyEffectWorse:
		return 1
	case StudyEffectUndetermined:
		return 2
	}
	return 3
}
