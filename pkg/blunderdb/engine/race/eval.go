package race

import (
	"log/slog"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Regime tells the user whether a displayed number was read from a two-sided
// database (exact) or estimated by convolution + calibrated correction.
type Regime string

const (
	RegimeExact     Regime = "exact"
	RegimeEstimated Regime = "estimated"
	// RegimeEvaluated is gammonNet playing the cube decision out — available
	// wherever the engine is, including at a match score, which
	// RegimeEstimated's convolution never offered (ADR-0012 extends
	// ADR-0009: what was refused was estimating a verdict from a snapshot,
	// not an engine that plays the trajectory out). See EvaluateWithEngine.
	RegimeEvaluated Regime = "evaluated"
)

// Eval is the race zone of the panel: win probability for the player on
// roll, and — exact regime only — the money cube analysis. The cube verdict
// is never estimated (ADR-0009).
type Eval struct {
	Regime Regime `json:"regime"`
	// OnRoll is the evaluated player (domain.Black or domain.White).
	OnRoll int `json:"on_roll"`
	// SourceCheckers is the per-player capacity of the database used
	// (exact regime), 0 when estimated.
	SourceCheckers int     `json:"source_checkers,omitempty"`
	WinProb        float64 `json:"win_prob"`
	// Evaluated regime only: gammon and backgammon chances, from the same
	// probability vector WinProb comes from — OnRoll's own (Win*) and the
	// opponent's (Lose*: OnRoll loses a gammon/backgammon), mirroring
	// domain.DoublingCubeAnalysis's Player*/Opponent* split. All four are
	// zero in the exact and estimated regimes — not a missing value, a fact
	// about the position: a two-sided database's domain (≤ 6 checkers a
	// side) already has the opponent bearing off 9+ checkers, where a
	// gammon is structurally impossible (ADR-0017), and the estimator
	// never models gammons at all.
	WinGammon      float64 `json:"win_gammon,omitempty"`
	WinBackgammon  float64 `json:"win_backgammon,omitempty"`
	LoseGammon     float64 `json:"lose_gammon,omitempty"`
	LoseBackgammon float64 `json:"lose_backgammon,omitempty"`
	// Estimated regime only: measured error bounds of the estimator on the
	// calibration oracle (win-probability units). Extrapolated beyond 11
	// checkers per side.
	Sigma float64 `json:"sigma,omitempty"`
	P99   float64 `json:"p99,omitempty"`
	// Exact regime only. The field keeps its name from before the
	// CubeVerdict rename (#190/C.3): "money" is still exactly right for
	// this field specifically, since the exact-regime table it carries is
	// always money-referential (MoneyFromEntry never reads a match score),
	// unlike the type itself, which the evaluated regime also uses at a
	// match score (internal/gui/gammonnet_eval.go's evaluateRaceRegime).
	Money *CubeVerdict `json:"money,omitempty"`
	// Evaluated regime only: the search depth that actually produced this
	// number — never the one requested, same discipline as
	// domain.CheckerMove/DoublingCubeAnalysis.AnalysisDepth (#125).
	Depth string `json:"depth,omitempty"`
}

// Result is the full EPC-panel payload: the per-player EPC blocks (always
// exact) plus the race zone when the position is a pure bearoff.
type Result struct {
	EPC
	Race *Eval `json:"race,omitempty"`
}

// Evaluate computes the panel payload for a position. The race zone is
// present only when both players have every checker in their home board and
// at least one checker left (pure bearoff, the panel's domain). The position
// is evaluated before the roll; dice on the position are ignored.
func Evaluate(pos *domain.Position) Result {
	bottom, bottomHome := computeSide(&pos.Board, domain.Black)
	top, topHome := computeSide(&pos.Board, domain.White)
	res := Result{EPC: EPC{Bottom: bottom, Top: top}}

	if !bottom.AllInHome || !top.AllInHome || bottom.CheckerCount == 0 || top.CheckerCount == 0 {
		return res
	}

	onRoll := pos.PlayerOnRoll
	us, them := bottomHome, topHome
	if onRoll == domain.White {
		us, them = topHome, bottomHome
	}

	// Resolve returns nil while the machine has no table yet — the first
	// launch, until the background generation finishes (ADR-0027). The exact
	// regime is simply unavailable then, and the estimate below answers, with
	// its own badge: the one thing that must not happen is an exact-looking
	// verdict pulled from nowhere (ADR-0009).
	src := Resolve()
	if src != nil && src.Covers(us, them) {
		entry, err := src.Lookup(us, them)
		if err != nil {
			// Covers() passed, so this is an I/O problem (truncated or
			// vanished file); degrade to the estimate rather than go mute.
			slog.Warn("two-sided lookup failed, falling back to estimate", "source", src.Origin(), "err", err)
		} else {
			money := MoneyFromEntry(entry, CubeStateFor(pos, onRoll))
			res.Race = &Eval{
				Regime:         RegimeExact,
				OnRoll:         onRoll,
				SourceCheckers: src.Checkers(),
				WinProb:        entry.WinProb,
				Money:          &money,
			}
			return res
		}
	}

	p, err := EstimatedWinProb(us, them)
	if err != nil {
		slog.Warn("win-probability estimation failed", "err", err)
		return res
	}
	res.Race = &Eval{
		Regime:  RegimeEstimated,
		OnRoll:  onRoll,
		WinProb: p,
		Sigma:   CorrectionSigma,
		P99:     CorrectionP99,
	}
	return res
}

// CubeStateFor maps the position's cube to the on-roll player's viewpoint.
// Exported (#197/C.10) so gui.evaluateRaceRegime (gammonnet_eval.go) can call
// this instead of keeping its own 3-line duplicate in sync by inspection —
// gammonnet's own internal tests import race, so race importing gammonnet
// back would cycle; gui already imports both with no such constraint.
func CubeStateFor(pos *domain.Position, onRoll int) CubeState {
	switch pos.Cube.Owner {
	case onRoll:
		return CubeOwned
	case domain.None:
		return CubeCentered
	default:
		return CubeAgainst
	}
}
