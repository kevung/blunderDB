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
	// Estimated regime only: measured error bounds of the estimator on the
	// calibration oracle (win-probability units). Extrapolated beyond 11
	// checkers per side.
	Sigma float64 `json:"sigma,omitempty"`
	P99   float64 `json:"p99,omitempty"`
	// Exact regime only.
	Money *Money `json:"money,omitempty"`
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

	src := Resolve()
	if src.Covers(us, them) {
		entry, err := src.Lookup(us, them)
		if err != nil {
			// Covers() passed, so this is an I/O problem (truncated or
			// vanished file); degrade to the estimate rather than go mute.
			slog.Warn("two-sided lookup failed, falling back to estimate", "source", src.Origin(), "err", err)
		} else {
			money := MoneyFromEntry(entry, cubeStateFor(pos, onRoll))
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

// cubeStateFor maps the position's cube to the on-roll player's viewpoint.
func cubeStateFor(pos *domain.Position, onRoll int) CubeState {
	switch pos.Cube.Owner {
	case onRoll:
		return CubeOwned
	case domain.None:
		return CubeCentered
	default:
		return CubeAgainst
	}
}
