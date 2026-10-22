package storage

import "github.com/kevung/blunderdb/pkg/blunderdb/domain"

// StatsSummary is the part of a StatsResult that can be shown in public: the
// level of the filter's decisions, with their counts and intervals, and
// nothing that names a position, a match, a tournament, an opponent or a
// tag. It is built field by field from the StatsResult (SummaryOf), so a
// field added to StatsResult stays out of it until it is chosen here — a
// public proxy that filtered a StatsResult itself would leak it instead.
type StatsSummary struct {
	Totals       StatsTotals     `json:"Totals"`
	PRGlobal     float64         `json:"PRGlobal"`
	PRInterval   domain.Interval `json:"PRInterval"`
	PRChecker    float64         `json:"PRChecker"`
	PRCube       float64         `json:"PRCube"`
	MWCAvailable bool            `json:"MWCAvailable"`
	MWCGlobal    float64         `json:"MWCGlobal"`
	MWCChecker   float64         `json:"MWCChecker"`
	MWCCube      float64         `json:"MWCCube"`
	MWC7         domain.MWC7     `json:"MWC7"`
	MWC7Checker  domain.MWC7     `json:"MWC7Checker"`
	MWC7Cube     domain.MWC7     `json:"MWC7Cube"`
	// PerPhase and PerScore are StatsResult's breakdowns, each row with its
	// count and interval: a cell without an interval is greyed by the
	// caller, as in the Statistics tab.
	PerPhase []PhaseStats     `json:"PerPhase"`
	PerScore []ScoreCellStats `json:"PerScore"`
}

// SummaryOf projects r onto its public part (see StatsSummary). A nil r
// gives an empty summary.
func SummaryOf(r *StatsResult) *StatsSummary {
	s := &StatsSummary{PerPhase: []PhaseStats{}, PerScore: []ScoreCellStats{}}
	if r == nil {
		return s
	}
	s.Totals = r.Totals
	s.PRGlobal, s.PRInterval, s.PRChecker, s.PRCube = r.PRGlobal, r.PRInterval, r.PRChecker, r.PRCube
	s.MWCAvailable, s.MWCGlobal, s.MWCChecker, s.MWCCube = r.MWCAvailable, r.MWCGlobal, r.MWCChecker, r.MWCCube
	s.MWC7, s.MWC7Checker, s.MWC7Cube = r.MWC7, r.MWC7Checker, r.MWC7Cube
	s.PerPhase = append(s.PerPhase, r.PerPhase...)
	s.PerScore = append(s.PerScore, r.PerScore...)
	return s
}
