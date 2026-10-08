package storage

import (
	"math"
	"slices"
	"strconv"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The thresholds of the tournament review (ADR-0081). They are fixed there,
// before any result was read, and change only with a new ADR.
const (
	// TournamentUsualDays is the window of the usual level: the days before
	// the tournament.
	TournamentUsualDays = 365
	// TournamentUsualMinMatches is the fewest matches of that window the
	// usual level needs.
	TournamentUsualMinMatches = 5
	// TournamentReviewMinDecisions is the fewest counted decisions each side
	// of a comparison needs before the review states a verdict.
	TournamentReviewMinDecisions = 20
	// TournamentRankSpan is the width of a decision-rank slice: the player's
	// counted decisions 1–30, 31–60, … of a match.
	TournamentRankSpan = 30
	// TournamentRankSlices is how many slices there are, the last open.
	TournamentRankSlices = 4
	// TournamentFamilies is the most error families the review names.
	TournamentFamilies = 3
)

// The verdicts of a Comparison.
const (
	VerdictWorse        = "worse"
	VerdictBetter       = "better"
	VerdictUsual        = "usual"
	VerdictInsufficient = "insufficient"
)

// The cells of the pressure and clock breakdowns, in display order.
const (
	PressureDMP          = "dmp"
	PressureCrawford     = "crawford"
	PressurePostCrawford = "post_crawford"
	PressureOther        = "other"
	ClockQuick           = "quick"
	ClockConsidered      = "considered"
)

// Comparison is a tournament figure set against the usual level: Delta is
// tournament − usual, Low and High its 95 % interval. Verdict is worse when
// Low > 0, better when High < 0, usual otherwise, and insufficient when a
// side lacks an interval or TournamentReviewMinDecisions decisions — Delta
// and its bounds then mean nothing.
type Comparison struct {
	Verdict string  `json:"verdict"`
	Delta   float64 `json:"delta"`
	Low     float64 `json:"low"`
	High    float64 `json:"high"`
}

// UsualLevel is the player's level over the TournamentUsualDays before the
// tournament, its own matches excluded (ADR-0081). Available needs a dated
// tournament and TournamentUsualMinMatches matches; From and To bound the
// window, To excluded.
type UsualLevel struct {
	Available  bool            `json:"available"`
	From       string          `json:"from"`
	To         string          `json:"to"`
	Matches    int             `json:"matches"`
	Decisions  int             `json:"decisions"`
	PR         float64         `json:"pr"`
	PRInterval domain.Interval `json:"pr_interval"`
	MWC7       domain.MWC7     `json:"mwc7"`
}

// ReviewCell is one slice of a breakdown: the tournament's PR there and the
// usual level's in the same slice, each with its interval over matches.
type ReviewCell struct {
	Key            string          `json:"key"`
	Decisions      int             `json:"decisions"`
	PR             float64         `json:"pr"`
	PRInterval     domain.Interval `json:"pr_interval"`
	UsualDecisions int             `json:"usual_decisions"`
	UsualPR        float64         `json:"usual_pr"`
	UsualInterval  domain.Interval `json:"usual_interval"`
	Versus         Comparison      `json:"versus"`
}

// RoundReview is one match of the tournament, in tournament order. Its PR
// interval resamples the games; Versus sets it against the usual PR.
type RoundReview struct {
	Round      int             `json:"round"`
	MatchID    int64           `json:"match_id"`
	Opponent   string          `json:"opponent"`
	Date       string          `json:"date"`
	Length     int             `json:"length"`
	Decisions  int             `json:"decisions"`
	PR         float64         `json:"pr"`
	PRInterval domain.Interval `json:"pr_interval"`
	MWC7       domain.MWC7     `json:"mwc7"`
	Versus     Comparison      `json:"versus"`
}

// TournamentReview is one player's review of one tournament (ADR-0081).
type TournamentReview struct {
	TournamentID int64  `json:"tournament_id"`
	Name         string `json:"name"`
	Date         string `json:"date"`
	Player       string `json:"player"`
	// Matches counts the tournament's matches the player played.
	Matches    int             `json:"matches"`
	Decisions  int             `json:"decisions"`
	PR         float64         `json:"pr"`
	PRInterval domain.Interval `json:"pr_interval"`
	MWC7       domain.MWC7     `json:"mwc7"`
	Usual      UsualLevel      `json:"usual"`
	PRVersus   Comparison      `json:"pr_versus"`
	MWC7Versus Comparison      `json:"mwc7_versus"`
	Rounds     []RoundReview   `json:"rounds"`
	ByRank     []ReviewCell    `json:"by_rank"`
	ByPressure []ReviewCell    `json:"by_pressure"`
	ByClock    []ReviewCell    `json:"by_clock"`
	// Families are at most TournamentFamilies families of the study plan
	// restricted to the tournament; Tentative counts those short of its
	// evidence.
	Families  []StudyPlanFamily `json:"families"`
	Tentative int               `json:"tentative"`
}

// ReviewMatch is one of the player's matches as a backend hands it to
// BuildTournamentReview: Seat is the player's (0 is player 1), Decisions the
// whole match as MatchDecisionLosses lists it.
type ReviewMatch struct {
	MatchID   int64
	Opponent  string
	Date      string
	Length    int
	Seat      int
	Decisions []DecisionLoss
}

// RankKeys are the keys of the decision-rank cells, in display order.
func RankKeys() []string {
	keys := make([]string, TournamentRankSlices)
	for i := range keys {
		lo := i*TournamentRankSpan + 1
		if i == TournamentRankSlices-1 {
			keys[i] = strconv.Itoa(lo) + "+"
		} else {
			keys[i] = strconv.Itoa(lo) + "-" + strconv.Itoa(lo+TournamentRankSpan-1)
		}
	}
	return keys
}

// PressureKeys and ClockKeys are the keys of those cells, in display order.
func PressureKeys() []string {
	return []string{PressureDMP, PressureCrawford, PressurePostCrawford, PressureOther}
}

// ClockKeys are the keys of the clock cells, in display order.
func ClockKeys() []string { return []string{ClockQuick, ClockConsidered} }

// PressureOf names the pressure cell of a decision from its stored away
// scores, the Crawford rule carried inside them (CONTEXT.md, « Away score »):
// DMP when both players need one point, Crawford in the Crawford game,
// post-Crawford after it, other scores otherwise.
func PressureOf(away [2]int) string {
	one0, one1 := domain.PointsAway(away[0]) == 1, domain.PointsAway(away[1]) == 1
	switch {
	case one0 && one1:
		return PressureDMP
	case away[0] == 1 || away[1] == 1:
		return PressureCrawford
	case away[0] == 0 || away[1] == 0:
		return PressurePostCrawford
	}
	return PressureOther
}

// prPool pools per-unit PR sums — millipoints over decisions — in the order
// the units first arrive, so every run sums alike.
type prPool struct {
	order []int64
	errMP map[int64]int64
	n     map[int64]int64
}

func (p *prPool) add(unit, errMP int64) {
	if p.errMP == nil {
		p.errMP, p.n = map[int64]int64{}, map[int64]int64{}
	}
	if _, seen := p.n[unit]; !seen {
		p.order = append(p.order, unit)
	}
	p.errMP[unit] += errMP
	p.n[unit]++
}

// result is the PR, its interval over the units, and the decisions.
func (p *prPool) result() (float64, domain.Interval, int) {
	var pool domain.RatioPool
	var sumErr, n int64
	for _, u := range p.order {
		pool.Add(float64(p.errMP[u]), float64(p.n[u]))
		sumErr += p.errMP[u]
		n += p.n[u]
	}
	pr := 0.0
	if n > 0 {
		pr = 500 * float64(sumErr) / 1000 / float64(n)
	}
	return pr, pool.Interval(500.0 / 1000.0), int(n)
}

// breakdown is a set of keyed cells, each pooling over matches.
type breakdown map[string]*prPool

func (b breakdown) add(key string, match, errMP int64) {
	p := b[key]
	if p == nil {
		p = &prPool{}
		b[key] = p
	}
	p.add(match, errMP)
}

func (b breakdown) result(key string) (float64, domain.Interval, int) {
	if p := b[key]; p != nil {
		return p.result()
	}
	return 0, domain.Interval{}, 0
}

// side gathers the tournament, or the usual level, match by match.
type side struct {
	overall               prPool
	mwc7                  domain.MWC7Pool
	rank, pressure, clock breakdown
	matches               int
}

func newSide() *side {
	return &side{rank: breakdown{}, pressure: breakdown{}, clock: breakdown{}}
}

// round is one match's own figures: PR over its games, MWC7.
type round struct {
	pr        float64
	interval  domain.Interval
	decisions int
	mwc7      domain.MWC7
}

// medianDurations is the median of seat's known durations per decision type
// over its counted decisions of the match (ADR-0078, rule 4).
func medianDurations(decisions []DecisionLoss, seat int) map[string]float64 {
	durations := map[string][]int64{}
	for _, d := range decisions {
		if d.Player == seat && d.ErrorMP != nil && d.DurationMS != nil {
			durations[d.DecisionType] = append(durations[d.DecisionType], *d.DurationMS)
		}
	}
	median := map[string]float64{}
	for t, ds := range durations {
		slices.Sort(ds)
		m := len(ds) / 2
		if len(ds)%2 == 1 {
			median[t] = float64(ds[m])
		} else {
			median[t] = float64(ds[m-1]+ds[m]) / 2
		}
	}
	return median
}

// add folds one match into the side and returns its own figures.
func (s *side) add(m ReviewMatch) round {
	var games prPool
	var gameOrder []int
	gameLoss := map[int]float64{}
	seenGame := map[int]bool{}
	loss, priced := 0.0, false
	median := medianDurations(m.Decisions, m.Seat)
	rankKeys := RankKeys()
	rank := 0
	for _, d := range m.Decisions {
		if d.ErrorMP == nil {
			continue
		}
		// Every game with a counted decision is a unit of the match's MWC7,
		// as in the match review, the player's loss there 0 or more.
		if !seenGame[d.GameNumber] {
			seenGame[d.GameNumber] = true
			gameOrder = append(gameOrder, d.GameNumber)
		}
		if d.Player != m.Seat {
			continue
		}
		e := *d.ErrorMP
		games.add(int64(d.GameNumber), e)
		s.overall.add(m.MatchID, e)
		s.rank.add(rankKeys[min(rank/TournamentRankSpan, TournamentRankSlices-1)], m.MatchID, e)
		rank++
		s.pressure.add(PressureOf(d.Away), m.MatchID, e)
		if med, ok := median[d.DecisionType]; ok && d.DurationMS != nil {
			key := ClockConsidered
			if float64(*d.DurationMS) < med {
				key = ClockQuick
			}
			s.clock.add(key, m.MatchID, e)
		}
		if d.MWCLoss != nil {
			loss += *d.MWCLoss
			gameLoss[d.GameNumber] += *d.MWCLoss
			priced = true
		}
	}
	var r round
	r.pr, r.interval, r.decisions = games.result()
	if r.decisions > 0 {
		s.matches++
	}
	if priced {
		slices.Sort(gameOrder)
		perGame := make([]float64, len(gameOrder))
		for i, g := range gameOrder {
			perGame[i] = gameLoss[g]
		}
		r.mwc7 = domain.MatchMWC7(loss, m.Length, perGame)
		s.mwc7.Add(r.mwc7, m.Length)
	}
	return r
}

// halfWidth is a figure's standard error read from the upper half of its
// 95 % interval: the lower bound is floored at 0 and says nothing.
func halfWidth(value, high float64) float64 {
	return (high - value) / StudyPlanZ
}

// Compare sets a tournament figure against the usual one (ADR-0081, rules 4
// and 5). Each side is its value, the upper bound of its interval, whether it
// has one, and its counted decisions.
func Compare(t, tHigh float64, tHas bool, tN int, u, uHigh float64, uHas bool, uN int) Comparison {
	if !tHas || !uHas || tN < TournamentReviewMinDecisions || uN < TournamentReviewMinDecisions {
		return Comparison{Verdict: VerdictInsufficient}
	}
	st, su := halfWidth(t, tHigh), halfWidth(u, uHigh)
	half := StudyPlanZ * math.Sqrt(st*st+su*su)
	c := Comparison{Delta: t - u, Low: t - u - half, High: t - u + half, Verdict: VerdictUsual}
	switch {
	case c.Low > 0:
		c.Verdict = VerdictWorse
	case c.High < 0:
		c.Verdict = VerdictBetter
	}
	return c
}

func cells(keys []string, t, u breakdown, usual bool) []ReviewCell {
	out := make([]ReviewCell, 0, len(keys))
	for _, k := range keys {
		c := ReviewCell{Key: k}
		c.PR, c.PRInterval, c.Decisions = t.result(k)
		c.UsualPR, c.UsualInterval, c.UsualDecisions = u.result(k)
		c.Versus = Comparison{Verdict: VerdictInsufficient}
		if usual {
			c.Versus = Compare(c.PR, c.PRInterval.High, c.PRInterval.Available, c.Decisions,
				c.UsualPR, c.UsualInterval.High, c.UsualInterval.Available, c.UsualDecisions)
		}
		out = append(out, c)
	}
	return out
}

// BuildTournamentReview assembles a TournamentReview (ADR-0081) from the
// player's matches of the tournament, in tournament order, those of the usual
// window, and the study plan of the player's errors in the tournament. It is
// pure and shared by every backend; usual.Available is decided here from the
// matches, so a backend only states the window.
func BuildTournamentReview(r TournamentReview, matches, usualMatches []ReviewMatch, plan *StudyPlan) TournamentReview {
	t, u := newSide(), newSide()
	r.Rounds = []RoundReview{}
	rounds := make([]round, 0, len(matches))
	for _, m := range matches {
		rounds = append(rounds, t.add(m))
	}
	for _, m := range usualMatches {
		u.add(m)
	}
	r.Matches = t.matches
	r.PR, r.PRInterval, r.Decisions = t.overall.result()
	r.MWC7 = t.mwc7.Result()

	r.Usual.Matches = u.matches
	r.Usual.PR, r.Usual.PRInterval, r.Usual.Decisions = u.overall.result()
	r.Usual.MWC7 = u.mwc7.Result()
	r.Usual.Available = r.Usual.From != "" && u.matches >= TournamentUsualMinMatches

	r.PRVersus, r.MWC7Versus = Comparison{Verdict: VerdictInsufficient}, Comparison{Verdict: VerdictInsufficient}
	if r.Usual.Available {
		r.PRVersus = Compare(r.PR, r.PRInterval.High, r.PRInterval.Available, r.Decisions,
			r.Usual.PR, r.Usual.PRInterval.High, r.Usual.PRInterval.Available, r.Usual.Decisions)
		r.MWC7Versus = Compare(r.MWC7.Loss, r.MWC7.High, r.MWC7.HasInterval, r.Decisions,
			r.Usual.MWC7.Loss, r.Usual.MWC7.High, r.Usual.MWC7.HasInterval, r.Usual.Decisions)
	}
	for i, m := range matches {
		f := rounds[i]
		rr := RoundReview{Round: i + 1, MatchID: m.MatchID, Opponent: m.Opponent, Date: m.Date, Length: m.Length,
			Decisions: f.decisions, PR: f.pr, PRInterval: f.interval, MWC7: f.mwc7,
			Versus: Comparison{Verdict: VerdictInsufficient}}
		if r.Usual.Available {
			rr.Versus = Compare(f.pr, f.interval.High, f.interval.Available, f.decisions,
				r.Usual.PR, r.Usual.PRInterval.High, r.Usual.PRInterval.Available, r.Usual.Decisions)
		}
		r.Rounds = append(r.Rounds, rr)
	}
	r.ByRank = cells(RankKeys(), t.rank, u.rank, r.Usual.Available)
	r.ByPressure = cells(PressureKeys(), t.pressure, u.pressure, r.Usual.Available)
	r.ByClock = cells(ClockKeys(), t.clock, u.clock, r.Usual.Available)

	r.Families = []StudyPlanFamily{}
	if plan != nil {
		r.Families = append(r.Families, plan.Families[:min(len(plan.Families), TournamentFamilies)]...)
		r.Tentative = len(plan.Tentative)
	}
	return r
}
