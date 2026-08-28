package ingest

import (
	"path/filepath"
	"testing"
)

// gnuBG SGF records the evaluation depth inside its evalcontext, after the
// outputs for a move and right after the version for a cube decision. Until
// gnubgparser v1.5.0 the cube parser read the `3` of `ver 3` — the SGF analysis
// FORMAT VERSION — as if it were the ply, so every cube analysis ever imported
// from a gnuBG SGF was labelled "3-ply" (kevung/gnubgparser#2).
//
// The two fixtures were analysed by gnuBG at 2-ply cubeful, with its usual
// 0-ply pre-filter over the whole candidate list. This pins that, on real
// files, so the label can never silently drift back to a constant.
//
// The cube counts are what THIS mapper produces, not what the parser reads:
// gnubgparser sees 94 and 207 cube analyses in these files, blunderDB turns 90
// and 200 of them into a DoublingCubeAnalysis. The two units are different and
// the test asserts its own.
func TestGnuBGAnalysisDepthComesFromTheEvalContext(t *testing.T) {
	for _, fixture := range []struct {
		name      string
		file      string
		wantCube  map[string]int
		wantMoves map[string]int
	}{
		{
			name:      "charlot1-charlot2 7pt",
			file:      "charlot1-charlot2_7p_2025-11-08-2305.sgf",
			wantCube:  map[string]int{"2-ply": 90},
			wantMoves: map[string]int{"0-ply": 1886, "2-ply": 1612},
		},
		{
			name:      "test.sgf",
			file:      "test.sgf",
			wantCube:  map[string]int{"2-ply": 200},
			wantMoves: map[string]int{"0-ply": 3530, "2-ply": 1987},
		},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			g, err := MapGnuBG(filepath.Join("..", "..", "..", "testdata", fixture.file))
			if err != nil {
				t.Fatalf("map: %v", err)
			}

			cube := map[string]int{}
			moves := map[string]int{}
			for gi := range g.Games {
				for mi := range g.Games[gi].Moves {
					for _, a := range g.Games[gi].Moves[mi].Analyses {
						if a == nil {
							continue
						}
						if a.DoublingCubeAnalysis != nil {
							cube[a.DoublingCubeAnalysis.AnalysisDepth]++
						}
						if a.CheckerAnalysis != nil {
							for _, m := range a.CheckerAnalysis.Moves {
								moves[m.AnalysisDepth]++
							}
						}
					}
				}
			}

			if !sameCounts(cube, fixture.wantCube) {
				t.Errorf("cube depths = %v, want %v\n"+
					"a constant here — 3-ply for every decision — is the format version being read as a ply",
					cube, fixture.wantCube)
			}
			if !sameCounts(moves, fixture.wantMoves) {
				t.Errorf("move depths = %v, want %v", moves, fixture.wantMoves)
			}
			if _, bad := cube["3-ply"]; bad {
				t.Error("a cube analysis is labelled 3-ply: the format version is being read as a depth again")
			}
		})
	}
}

func sameCounts(got, want map[string]int) bool {
	if len(got) != len(want) {
		return false
	}
	for k, v := range want {
		if got[k] != v {
			return false
		}
	}
	return true
}

// An unknown depth produces an empty label, never "0-ply". Zero is a real gnuBG
// depth, so defaulting a missing value to it would relabel a rollout as the
// shallowest search there is — the same lie in a different place.
func TestUnknownDepthIsBlankNotZeroPly(t *testing.T) {
	if got := translateGnuBGAnalysisDepth(0, false); got != "" {
		t.Errorf("unknown depth → %q, want empty", got)
	}
	if got := translateGnuBGAnalysisDepth(0, true); got != "0-ply" {
		t.Errorf("a known 0-ply → %q, want %q", got, "0-ply")
	}
	if got := translateGnuBGAnalysisDepth(2, true); got != "2-ply" {
		t.Errorf("a known 2-ply → %q, want %q", got, "2-ply")
	}
}
