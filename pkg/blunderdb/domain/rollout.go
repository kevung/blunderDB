package domain

import (
	"sort"
	"time"
)

// RolloutKinds name what a stored rollout rolled: the plays of a position
// with dice, or the cube decision of one without.
const (
	RolloutKindMoves = "moves"
	RolloutKindCube  = "cube"
)

// RolloutAnalysis is a rollout written down on a Position: a second Analysis,
// beside the one an import or an evaluation wrote, carrying its own
// Configuration (ADR-0060). It never replaces an entry of CheckerAnalysis or
// DoublingCubeAnalysis (ADR-0013): it lives in PositionAnalysis.Rollouts, one
// per Signature.
//
// Equities are on the one scale that leaves the engine (ADR-0019); chances
// are percentages [0,100] like every other analysis field.
type RolloutAnalysis struct {
	AnalysisEngine string `json:"analysisEngine"`
	AnalysisDepth  string `json:"analysisDepth"`
	// Signature is the full line the rollout is reproduced from; two rollouts
	// with one Signature on one position are the same Configuration.
	Signature string          `json:"signature"`
	Kind      string          `json:"kind"`
	Settings  RolloutSettings `json:"settings"`
	// Games is the most games any candidate played; Stop says why it ended.
	Games int    `json:"games"`
	Stop  string `json:"stop"`
	// CubefulBias: the cube model plays inside the games, so the ranking is
	// more reliable than the absolute equity.
	CubefulBias  bool `json:"cubefulBias"`
	ExactBearoff bool `json:"exactBearoff"`
	// Candidates are the plays, best first; for a cube rollout, the three
	// actions in the order No double, Double/Take, Double/Pass.
	Candidates []RolloutCandidate `json:"candidates"`
	// BestCubeAction, in DoublingCubeAnalysis.BestCubeAction's vocabulary,
	// and the two gaps that decide it, for a cube rollout only.
	BestCubeAction string    `json:"bestCubeAction,omitempty"`
	JSDDouble      float64   `json:"jsdDouble,omitempty"`
	JSDTake        float64   `json:"jsdTake,omitempty"`
	Date           time.Time `json:"date"`
}

// RolloutSettings are the parameters of a stored rollout: every one that
// moves a number (the worker count does not, so it is not kept).
type RolloutSettings struct {
	Truncation int     `json:"truncation"`
	MinGames   int     `json:"minGames"`
	MaxGames   int     `json:"maxGames"`
	JSDLimit   float64 `json:"jsdLimit"`
	Ply        int     `json:"ply"`
	Candidates int     `json:"candidates"`
	Seed       uint64  `json:"seed"`
}

// RolloutCandidate is one rolled-out play or cube action.
type RolloutCandidate struct {
	Move   string  `json:"move"`
	Equity float64 `json:"equity"`
	// StdErr is the standard error of Equity; CI95 the half-width of its 95 %
	// interval; JSD the gap to the best in standard deviations of the
	// difference.
	StdErr                   float64 `json:"stdErr"`
	CI95                     float64 `json:"ci95"`
	Games                    int     `json:"games"`
	JSD                      float64 `json:"jsd"`
	PlayerWinChance          float64 `json:"playerWinChance"`
	PlayerGammonChance       float64 `json:"playerGammonChance"`
	PlayerBackgammonChance   float64 `json:"playerBackgammonChance"`
	OpponentWinChance        float64 `json:"opponentWinChance"`
	OpponentGammonChance     float64 `json:"opponentGammonChance"`
	OpponentBackgammonChance float64 `json:"opponentBackgammonChance"`
}

// HasRollout reports whether a carries a rollout of this Signature.
func (a *PositionAnalysis) HasRollout(signature string) bool {
	if a == nil {
		return false
	}
	for _, r := range a.Rollouts {
		if r.Signature == signature {
			return true
		}
	}
	return false
}

// MergeRollouts keeps every rollout either side knows, one per Signature.
// When both carry the same Signature, the one with more games stays (a rerun
// of the same Configuration is the same numbers or a longer series of them);
// on a tie, incoming. The result is ordered by date, newest first, then by
// Signature: a total order, so the same rollouts listed in another order by
// either side merge to the same list and compare equal to what is stored.
func MergeRollouts(existing, incoming []RolloutAnalysis) []RolloutAnalysis {
	if len(existing) == 0 && len(incoming) == 0 {
		return nil
	}
	out := make([]RolloutAnalysis, 0, len(existing)+len(incoming))
	index := map[string]int{}
	for _, r := range existing {
		if i, ok := index[r.Signature]; ok {
			if r.Games > out[i].Games {
				out[i] = r
			}
			continue
		}
		index[r.Signature] = len(out)
		out = append(out, r)
	}
	for _, r := range incoming {
		if i, ok := index[r.Signature]; ok {
			if r.Games >= out[i].Games {
				out[i] = r
			}
			continue
		}
		index[r.Signature] = len(out)
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Date.Equal(out[j].Date) {
			return out[i].Date.After(out[j].Date)
		}
		return out[i].Signature < out[j].Signature
	})
	return out
}

// AttachRollout adds r to a beside everything a already holds: no entry of
// CheckerAnalysis or DoublingCubeAnalysis moves (ADR-0013, ADR-0060), and
// AnalysisType stays empty on a position only rollouts analyse, so an import
// still reads it as a gap to fill.
func (a *PositionAnalysis) AttachRollout(r RolloutAnalysis) {
	a.Rollouts = MergeRollouts(a.Rollouts, []RolloutAnalysis{r})
}

// MergeImportedAnalysis is what an import writes over existing: the imported
// analysis only when existing has no primary analysis (ADR-0013), and in
// every case the rollouts of both sides (ADR-0060). changed reports whether
// the result differs from existing, so an unchanged position is not written.
func MergeImportedAnalysis(existing, imported *PositionAnalysis) (merged *PositionAnalysis, changed bool) {
	switch {
	case imported == nil:
		return existing, false
	case existing == nil:
		return imported, true
	}
	if !existing.HasPrimary() && imported.HasPrimary() {
		out := *imported
		out.Rollouts = MergeRollouts(existing.Rollouts, imported.Rollouts)
		return &out, true
	}
	rollouts := MergeRollouts(existing.Rollouts, imported.Rollouts)
	if sameRollouts(existing.Rollouts, rollouts) {
		return existing, false
	}
	out := *existing
	out.Rollouts = rollouts
	return &out, true
}

func sameRollouts(a, b []RolloutAnalysis) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Signature != b[i].Signature || a[i].Games != b[i].Games || !a[i].Date.Equal(b[i].Date) {
			return false
		}
	}
	return true
}

// BestRollout is the rollout that speaks for a when it has no other analysis:
// the one with the most games, the newest on a tie. Nil when a has none.
func (a *PositionAnalysis) BestRollout() *RolloutAnalysis {
	if a == nil {
		return nil
	}
	var best *RolloutAnalysis
	for i := range a.Rollouts {
		r := &a.Rollouts[i]
		if best == nil || r.Games > best.Games || (r.Games == best.Games && r.Date.After(best.Date)) {
			best = r
		}
	}
	return best
}

// HasPrimary reports whether a carries an analysis other than rollouts.
func (a *PositionAnalysis) HasPrimary() bool {
	if a == nil {
		return false
	}
	return a.DoublingCubeAnalysis != nil || (a.CheckerAnalysis != nil && len(a.CheckerAnalysis.Moves) > 0)
}

// ColumnSource is the analysis the indexed columns are derived from: a
// itself, unless a holds nothing but rollouts — then its BestRollout read as
// an ordinary analysis, so a position only a rollout analysed is still found
// by the search. A rollout never moves the columns of a position that has
// another analysis (ADR-0060).
func (a *PositionAnalysis) ColumnSource() *PositionAnalysis {
	if a == nil || a.HasPrimary() {
		return a
	}
	r := a.BestRollout()
	if r == nil {
		return a
	}
	view := *a
	if r.Kind == RolloutKindCube {
		view.AnalysisType = "DoublingCube"
		view.DoublingCubeAnalysis = r.cubeAnalysis()
	} else {
		view.AnalysisType = "CheckerMove"
		view.CheckerAnalysis = &CheckerAnalysis{Moves: r.checkerMoves()}
	}
	return &view
}

func (r *RolloutAnalysis) checkerMoves() []CheckerMove {
	moves := make([]CheckerMove, 0, len(r.Candidates))
	for i, c := range r.Candidates {
		var e *float64
		if i > 0 {
			d := r.Candidates[0].Equity - c.Equity
			e = &d
		}
		moves = append(moves, CheckerMove{
			Index: i, AnalysisDepth: r.AnalysisDepth, AnalysisEngine: r.AnalysisEngine,
			Move: c.Move, Equity: c.Equity, EquityError: e,
			PlayerWinChance: c.PlayerWinChance, PlayerGammonChance: c.PlayerGammonChance,
			PlayerBackgammonChance: c.PlayerBackgammonChance, OpponentWinChance: c.OpponentWinChance,
			OpponentGammonChance: c.OpponentGammonChance, OpponentBackgammonChance: c.OpponentBackgammonChance,
		})
	}
	return moves
}

func (r *RolloutAnalysis) cubeAnalysis() *DoublingCubeAnalysis {
	if len(r.Candidates) < 3 {
		return nil
	}
	nd, dt, dp := r.Candidates[0], r.Candidates[1], r.Candidates[2]
	best := max(nd.Equity, min(dt.Equity, dp.Equity))
	return &DoublingCubeAnalysis{
		AnalysisDepth: r.AnalysisDepth, AnalysisEngine: r.AnalysisEngine,
		PlayerWinChances: nd.PlayerWinChance, PlayerGammonChances: nd.PlayerGammonChance,
		PlayerBackgammonChances: nd.PlayerBackgammonChance, OpponentWinChances: nd.OpponentWinChance,
		OpponentGammonChances: nd.OpponentGammonChance, OpponentBackgammonChances: nd.OpponentBackgammonChance,
		CubefulNoDoubleEquity: nd.Equity, CubefulNoDoubleError: nd.Equity - best,
		CubefulDoubleTakeEquity: dt.Equity, CubefulDoubleTakeError: dt.Equity - best,
		CubefulDoublePassEquity: dp.Equity, CubefulDoublePassError: dp.Equity - best,
		BestCubeAction: r.BestCubeAction,
	}
}
