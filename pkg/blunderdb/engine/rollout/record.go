package rollout

import (
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// Record is r as it is written down on a Position: a second Analysis with its
// own Configuration (ADR-0060 §8), dated at. Chances become percentages, as
// every stored analysis carries them.
func (r *Result) Record(at time.Time) domain.RolloutAnalysis {
	rec := domain.RolloutAnalysis{
		AnalysisEngine: r.EngineVersion,
		AnalysisDepth:  r.Settings.DepthLabel(),
		Signature:      r.Signature,
		Kind:           domain.RolloutKindMoves,
		Settings: domain.RolloutSettings{
			Truncation: r.Settings.Truncation, MinGames: r.Settings.MinGames, MaxGames: r.Settings.MaxGames,
			JSDLimit: r.Settings.JSDLimit, Ply: r.Settings.Ply, Candidates: r.Settings.Candidates, Seed: r.Settings.Seed,
		},
		Games:        r.Games,
		Stop:         string(r.Stop),
		CubefulBias:  r.CubefulBias,
		ExactBearoff: r.ExactBearoff,
		Date:         at.UTC(),
	}
	if r.Cube != nil {
		rec.Kind = domain.RolloutKindCube
		rec.Candidates = []domain.RolloutCandidate{
			candidateRecord(NoDouble, r.Cube.NoDouble),
			candidateRecord(DoubleTake, r.Cube.DoubleTake),
			candidateRecord(DoublePass, r.Cube.DoublePass),
		}
		rec.BestCubeAction = cubeActionLabel(r.Cube)
		rec.JSDDouble, rec.JSDTake = r.Cube.JSDDouble, r.Cube.JSDTake
		return rec
	}
	for _, c := range r.Candidates {
		rec.Candidates = append(rec.Candidates, candidateRecord(c.Move, c.Estimate))
	}
	return rec
}

func candidateRecord(move string, e Estimate) domain.RolloutCandidate {
	p := e.Chances
	return domain.RolloutCandidate{
		Move: move, Equity: e.Equity, StdErr: e.StdErr, CI95: e.CI95, Games: e.Games, JSD: e.JSD,
		PlayerWinChance:          100 * p[gammonnet.PWin],
		PlayerGammonChance:       100 * p[gammonnet.PWinGammon],
		PlayerBackgammonChance:   100 * p[gammonnet.PWinBackgammon],
		OpponentWinChance:        100 * (1 - p[gammonnet.PWin]),
		OpponentGammonChance:     100 * p[gammonnet.PLoseGammon],
		OpponentBackgammonChance: 100 * p[gammonnet.PLoseBackgammon],
	}
}

// cubeActionLabel speaks DoublingCubeAnalysis.BestCubeAction's vocabulary,
// the one the imports and gammonNet's own evaluation write, so the search
// reads a rollout's verdict as it reads theirs.
func cubeActionLabel(c *CubeResult) string {
	switch gammonnet.Verdict(c.NoDouble.Equity, c.DoubleTake.Equity, c.DoublePass.Equity) {
	case gammonnet.DoubleTake:
		return "Double, Take"
	case gammonnet.DoublePass:
		return "Double, Pass"
	case gammonnet.TooGood:
		if c.DoubleTake.Equity <= c.DoublePass.Equity {
			return "Too good to double, take"
		}
		return "Too good to double, pass"
	}
	return "No Double"
}
