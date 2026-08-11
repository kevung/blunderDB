package ingest

import (
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestClassifyBGFCubeAction cables classifyBGFCubeAction (used by
// buildBGFTextCubeAnalysis, itself reached from MapBGFTextPosition) onto three
// real BGBlitz text-position fixtures that carry cube-decision rows in both
// languages the format exports: French ("Pas de double", "Double / Prendre",
// "Double / Refuser") and English ("No Double", "Double / Pass",
// "Double / Take"). Before this test, testdata/bgf_positions/02_NDT_FR.txt,
// 04_DP_EN.txt and 06_RT_FR.txt were orphaned fixtures with 0 references
// anywhere in the repo, and classifyBGFCubeAction had 0% coverage.
//
// Each fixture's three cube-decision rows classify into exactly one of
// nodbl/take/pass; buildBGFTextCubeAnalysis reads the classified EMG value
// into CubefulNoDoubleEquity/CubefulDoubleTakeEquity/CubefulDoublePassEquity.
// A misclassification (e.g. a French "Prendre" row not recognised as "take")
// would leave the corresponding field at its zero value instead of the
// fixture's actual EMG number, so asserting the exact populated values proves
// classifyBGFCubeAction actually ran and classified correctly — not just that
// parsing didn't error.
func TestClassifyBGFCubeAction(t *testing.T) {
	cases := []struct {
		fixture string
		// Expected EMG-equity cubeful values, read straight off the fixture's
		// "Videau:"/"Cube Action:" table.
		wantNoDouble, wantDoubleTake, wantDoublePass float64
		wantBestAction                               string
	}{
		{
			// 02_NDT_FR.txt: "Pas de double" 0.287 EMG (nodbl), "Double / Prendre"
			// 0.164 EMG (take), "Double / Refuser" 1.000 EMG (pass). No Double is
			// the header's stated cube action, and is indeed the highest EMG.
			fixture:        "02_NDT_FR.txt",
			wantNoDouble:   0.287,
			wantDoubleTake: 0.164,
			wantDoublePass: 1.000,
			wantBestAction: "No Double",
		},
		{
			// 04_DP_EN.txt: "No Double" 0.767 EMG (nodbl), "Double / Take" 1.287
			// EMG (take), "Double / Pass" 1.000 EMG (pass). Header says
			// "Double / Reject", matching Double,Pass (min(take,pass)=pass=1.000 > nodbl).
			fixture:        "04_DP_EN.txt",
			wantNoDouble:   0.767,
			wantDoubleTake: 1.287,
			wantDoublePass: 1.000,
			wantBestAction: "Double, Pass",
		},
		{
			// 06_RT_FR.txt: "Pas de double" 0.685 EMG (nodbl), "Double / Prendre"
			// 0.760 EMG (take), "Double / Refuser" 1.000 EMG (pass). Header says
			// "Double / Prendre", matching Double,Take (min(take,pass)=take=0.760 > nodbl).
			fixture:        "06_RT_FR.txt",
			wantNoDouble:   0.685,
			wantDoubleTake: 0.760,
			wantDoublePass: 1.000,
			wantBestAction: "Double, Take",
		},
	}

	for _, c := range cases {
		t.Run(c.fixture, func(t *testing.T) {
			path := filepath.Join("..", "..", "..", "testdata", "bgf_positions", c.fixture)
			graphs, err := MapBGFTextPosition(path)
			if err != nil {
				t.Fatalf("MapBGFTextPosition(%s): %v", path, err)
			}
			if len(graphs) != 1 {
				t.Fatalf("expected exactly 1 position, got %d", len(graphs))
			}
			g := graphs[0]

			var cubeAnalysis *domain.PositionAnalysis
			for _, an := range g.Analyses {
				if an.AnalysisType == "DoublingCube" {
					cubeAnalysis = an
				}
			}
			if cubeAnalysis == nil {
				t.Fatalf("expected a DoublingCube analysis fragment, got %d analyses: %+v", len(g.Analyses), g.Analyses)
			}
			cube := cubeAnalysis.DoublingCubeAnalysis
			if cube == nil {
				t.Fatal("DoublingCubeAnalysis is nil")
			}

			if cube.CubefulNoDoubleEquity != c.wantNoDouble {
				t.Errorf("CubefulNoDoubleEquity = %v, want %v (classifyBGFCubeAction should have tagged the \"no double\" row)", cube.CubefulNoDoubleEquity, c.wantNoDouble)
			}
			if cube.CubefulDoubleTakeEquity != c.wantDoubleTake {
				t.Errorf("CubefulDoubleTakeEquity = %v, want %v (classifyBGFCubeAction should have tagged the \"take\" row)", cube.CubefulDoubleTakeEquity, c.wantDoubleTake)
			}
			if cube.CubefulDoublePassEquity != c.wantDoublePass {
				t.Errorf("CubefulDoublePassEquity = %v, want %v (classifyBGFCubeAction should have tagged the \"pass\" row)", cube.CubefulDoublePassEquity, c.wantDoublePass)
			}
			if cube.BestCubeAction != c.wantBestAction {
				t.Errorf("BestCubeAction = %q, want %q", cube.BestCubeAction, c.wantBestAction)
			}
		})
	}
}

// TestClassifyBGFCubeActionDirect exercises the classifier on literal action
// strings pulled from the three fixtures, isolating language handling from
// the parsing/equity plumbing checked above.
func TestClassifyBGFCubeActionDirect(t *testing.T) {
	cases := []struct {
		action string
		want   string
	}{
		{"Pas de double", "nodbl"},          // 02_NDT_FR.txt, 06_RT_FR.txt
		{"Double / Prendre", "take"},        // 02_NDT_FR.txt, 06_RT_FR.txt
		{"Double / Refuser", "pass"},        // 02_NDT_FR.txt, 06_RT_FR.txt
		{"No Double", "nodbl"},              // 04_DP_EN.txt
		{"Double / Take", "take"},           // 04_DP_EN.txt
		{"Double / Pass", "pass"},           // 04_DP_EN.txt
		{"  PAS DE DOUBLE  ", "nodbl"},      // case/whitespace insensitive
		{"garbage/unrecognised", "unknown"}, // contains "/" but no known pattern
		{"garbage unrecognised", "nodbl"},   // no "/" at all falls back to nodbl
	}
	for _, c := range cases {
		t.Run(c.action, func(t *testing.T) {
			if got := classifyBGFCubeAction(c.action); got != c.want {
				t.Errorf("classifyBGFCubeAction(%q) = %q, want %q", c.action, got, c.want)
			}
		})
	}
}
