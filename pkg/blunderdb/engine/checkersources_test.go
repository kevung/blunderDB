package engine

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func sourceMoves(eng, depth string, equities ...float64) []domain.CheckerMove {
	notations := []string{"8/5 6/5", "24/23 13/10", "13/10 6/5", "24/21 13/11"}
	moves := make([]domain.CheckerMove, len(equities))
	for i, e := range equities {
		moves[i] = domain.CheckerMove{Move: notations[i], AnalysisEngine: eng, AnalysisDepth: depth, Equity: e}
	}
	RankCheckerMovesDepthFirst(moves)
	return moves
}

func TestMergeCheckerSourcesKeepsEachEngine(t *testing.T) {
	xg := sourceMoves("XG", "XG Roller++", 0.30, 0.20, 0.10)
	gnu := sourceMoves("GNUbg", "2-ply", 0.25, 0.28)

	got := MergeCheckerSources(xg, nil, gnu, nil)
	if len(got) != 2 || got[0].AnalysisEngine != "XG" || got[1].AnalysisEngine != "GNUbg" {
		t.Fatalf("sources = %+v, want XG then GNUbg", got)
	}
	if got[1].AnalysisDepth != "2-ply" || len(got[1].Moves) != 2 || got[1].Moves[1].Equity != 0.25 {
		t.Errorf("GNUbg source = %+v, want its own two moves", got[1])
	}

	// Re-importing a source rewrites it with itself.
	merged := MergeCheckerMoves(xg, gnu)
	RankCheckerMovesDepthFirst(merged)
	again := MergeCheckerSources(merged, got, gnu, nil)
	if !reflect.DeepEqual(again, got) {
		t.Errorf("re-import of GNUbg changed the sources:\n got  %+v\n want %+v", again, got)
	}

	// Saving back the merged view brings no new source.
	if back := MergeCheckerSources(merged, got, merged, nil); !reflect.DeepEqual(back, got) {
		t.Errorf("save-back changed the sources:\n got  %+v\n want %+v", back, got)
	}

	// A shallower run of one engine is a source of its own.
	xg2 := sourceMoves("XG", "2-ply", 0.31, 0.21)
	three := MergeCheckerSources(merged, got, xg2, nil)
	if len(three) != 3 || three[0].AnalysisDepth != "XG Roller++" || three[1].AnalysisDepth != "2-ply" {
		t.Errorf("sources = %+v, want XG Roller++, XG 2-ply, GNUbg", three)
	}
}

func TestMergeCheckerSourcesSingleSourceIsNil(t *testing.T) {
	xg := sourceMoves("XG", "3-ply", 0.30, 0.20)
	if got := MergeCheckerSources(nil, nil, xg, nil); got != nil {
		t.Errorf("first source = %+v, want nil", got)
	}
	if got := MergeCheckerSources(xg, nil, xg, nil); got != nil {
		t.Errorf("same source again = %+v, want nil", got)
	}
	a := &domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{Moves: xg}}
	srcs := CheckerSources(a)
	if len(srcs) != 1 || srcs[0].AnalysisEngine != "XG" || srcs[0].AnalysisDepth != "3-ply" || !reflect.DeepEqual(srcs[0].Moves, xg) {
		t.Errorf("CheckerSources of a single-source analysis = %+v", srcs)
	}
}

// A merged view written before the sources existed may already mix engines:
// read as sources, each engine's part is ranked against its own best move.
func TestCheckerSourcesOfMixedViewRanksEachEngine(t *testing.T) {
	mixed := []domain.CheckerMove{
		{Move: "a", AnalysisEngine: "XG", AnalysisDepth: "XG Roller++", Equity: 0.3},
		{Move: "b", AnalysisEngine: "GNUbg", AnalysisDepth: "2-ply", Equity: 0.2},
		{Move: "c", AnalysisEngine: "XG", AnalysisDepth: "3-ply", Equity: 0.1},
	}
	srcs := CheckerSourcesOf(mixed)
	if len(srcs) != 2 || srcs[0].AnalysisDepth != "XG Roller++" || len(srcs[0].Moves) != 2 {
		t.Fatalf("sources = %+v", srcs)
	}
	if g := srcs[1].Moves[0]; g.Index != 0 || g.EquityError != nil {
		t.Errorf("GNUbg's best move = %+v, want index 0 and no error", g)
	}
}

func TestBinaryPayloadCarriesCheckerSources(t *testing.T) {
	xg := sourceMoves("XG", "XG Roller++", 0.30, 0.20)
	gnu := sourceMoves("GNUbg", "2-ply", 0.25, 0.28)
	a := domain.PositionAnalysis{
		CreationDate:       time.Unix(1_700_000_000, 0).UTC(),
		LastModifiedDate:   time.Unix(1_700_000_000, 0).UTC(),
		CheckerAnalysis:    &domain.CheckerAnalysis{Moves: xg},
		AllCheckerAnalyses: MergeCheckerSources(xg, nil, gnu, nil),
	}
	RoundAnalysisForStorage(&a)
	got, err := UnmarshalAnalysisBinary(MarshalAnalysisBinary(&a))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.AllCheckerAnalyses, a.AllCheckerAnalyses) {
		t.Errorf("sources after a round trip:\n got  %+v\n want %+v", got.AllCheckerAnalyses, a.AllCheckerAnalyses)
	}

	// Without sources, the payload is the one written before the flag.
	a.AllCheckerAnalyses = nil
	payload := MarshalAnalysisBinary(&a)
	if payload[0]&binFlagCheckerSources != 0 {
		t.Error("a payload without sources sets their flag")
	}
	b := a
	b.AllCheckerAnalyses = []domain.CheckerSourceAnalysis{}
	if bytes.Equal(MarshalAnalysisBinary(&b), payload) {
		t.Error("an empty set of sources encodes as none")
	}
}

// Only a blob carrying sources needs a reader of version 2.
func TestBlobVersionFollowsCheckerSources(t *testing.T) {
	xg := sourceMoves("XG", "XG Roller++", 0.30, 0.20)
	gnu := sourceMoves("GNUbg", "2-ply", 0.25, 0.28)
	a := domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{Moves: xg}}
	blob, _ := EncodeAnalysisForStorage(&a)
	if blob[1] != binFormatVersion {
		t.Errorf("blob without sources is version %d", blob[1])
	}
	a.AllCheckerAnalyses = MergeCheckerSources(xg, nil, gnu, nil)
	blob, _ = EncodeAnalysisForStorage(&a)
	if blob[1] != binFormatVersionSources {
		t.Errorf("blob with sources is version %d", blob[1])
	}
	back, err := DecodeAnalysisFromStorage(blob)
	if err != nil || len(back.AllCheckerAnalyses) != 2 {
		t.Fatalf("decoded sources = %+v, %v", back.AllCheckerAnalyses, err)
	}
	blob[1] = binFormatVersion
	if _, err := DecodeAnalysisFromStorage(blob); err == nil {
		t.Error("a version-1 blob carrying sources was accepted")
	}
}
