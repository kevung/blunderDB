package rollout

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// opening31 is the opening 3-1 at money, centred cube.
const opening31 = "XGID=-b----E-C---eE---c-e----B-:0:0:1:31:0:0:0:0:10"

func decode(t *testing.T, xgid string) domain.Position {
	t.Helper()
	pos, err := domain.DecodeXGID(xgid)
	if err != nil {
		t.Fatalf("decode %s: %v", xgid, err)
	}
	return pos
}

// small is a rollout cheap enough for a unit test: what it checks does not
// depend on the game count.
func small() Settings {
	return Settings{Truncation: 2, MinGames: 36, MaxGames: 72, Ply: 0, Candidates: 2, Seed: 7}
}

func TestDiceStratification(t *testing.T) {
	for _, games := range []int{36, 216} {
		var first, second [36]int
		for g := 0; g < games; g++ {
			d := newGameDice(1, g)
			first[d.first]++
			second[d.second]++
		}
		for i := range first {
			if first[i] != games/36 || second[i] != games/36 {
				t.Fatalf("%d games: roll %d drawn %d/%d times as first/second, want %d", games, i, first[i], second[i], games/36)
			}
		}
	}
	pairs := map[[2]int]bool{}
	for g := 0; g < stratumGames; g++ {
		d := newGameDice(1, g)
		pairs[[2]int{d.first, d.second}] = true
	}
	if len(pairs) != stratumGames {
		t.Fatalf("1296 games cover %d ordered pairs of opening rolls, want 1296", len(pairs))
	}
}

func TestDiceStreamIsAFunctionOfSeedAndGame(t *testing.T) {
	a, b := newGameDice(42, 1000), newGameDice(42, 1000)
	for ply := 0; ply < 50; ply++ {
		a1, a2 := a.roll(ply)
		b1, b2 := b.roll(ply)
		if a1 != b1 || a2 != b2 {
			t.Fatalf("ply %d: %d-%d then %d-%d from the same seed and game", ply, a1, a2, b1, b2)
		}
	}
}

// TestReproducibleAcrossWorkers holds the determinism contract: one worker or
// many, the same settings give the same numbers bit for bit — plays and cube.
func TestReproducibleAcrossWorkers(t *testing.T) {
	cases := []struct {
		name string
		xgid string
		ply  int
	}{
		{"moves", opening31, 0},
		// At 1 ply the plays come from policy.BestPlay, not from the luck pass.
		{"moves-1ply", opening31, 1},
		{"cube", "XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:0:10", 0},
		{"match", "XGID=-b----E-C---eE---c-e----B-:0:0:1:00:2:3:0:7:10", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pos := decode(t, tc.xgid)
			settings := func() Settings {
				s := small()
				s.Ply = tc.ply
				if tc.ply > 0 {
					s.MaxGames = 36
				}
				return s
			}
			var runs []*Result
			for _, workers := range []int{1, 7} {
				s := settings()
				s.Workers = workers
				r, err := Run(context.Background(), pos, s, Options{NoBearoffTable: true})
				if err != nil {
					t.Fatal(err)
				}
				r.Settings.Workers = 0
				runs = append(runs, r)
			}
			if !reflect.DeepEqual(runs[0], runs[1]) {
				t.Fatalf("1 worker and 7 workers disagree:\n%+v\n%+v", runs[0].Candidates, runs[1].Candidates)
			}
			// A caller's Exec playing the games backwards, as a shared
			// pool hands them out in whatever order its turns fall.
			backwards := func(n int, task func(int)) error {
				for i := n - 1; i >= 0; i-- {
					task(i)
				}
				return nil
			}
			viaExec, err := Run(context.Background(), pos, settings(), Options{NoBearoffTable: true, Exec: backwards})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(runs[0], viaExec) {
				t.Fatalf("own workers and a caller's Exec disagree:\n%+v\n%+v", runs[0].Candidates, viaExec.Candidates)
			}
			if runs[0].Games != settings().MaxGames {
				t.Fatalf("played %d games, want %d", runs[0].Games, settings().MaxGames)
			}
			s := settings()
			s.Seed++
			other, err := Run(context.Background(), pos, s, Options{NoBearoffTable: true})
			if err != nil {
				t.Fatal(err)
			}
			if reflect.DeepEqual(other.Candidates, runs[0].Candidates) {
				t.Fatal("a different seed gave the same numbers: the seed does not reach the dice")
			}
		})
	}
}

// TestExecSkippingAGameIsRefused: a batch summed short of a game would pass
// for a rollout of the settings it does not honour.
func TestExecSkippingAGameIsRefused(t *testing.T) {
	skip := func(n int, task func(int)) error {
		for i := 1; i < n; i++ {
			task(i)
		}
		return nil
	}
	if _, err := Run(context.Background(), decode(t, opening31), small(), Options{NoBearoffTable: true, Exec: skip}); err == nil {
		t.Fatal("a batch missing a game was summed")
	}
}

// TestJSDStopsEarly: the opening 3-1 point is far ahead of 13/10 10/9, so the
// rollout stops on the JSD rule long before its game budget.
func TestJSDStopsEarly(t *testing.T) {
	s := small()
	s.MaxGames = 1296
	s.JSDLimit = 3
	r, err := Run(context.Background(), decode(t, opening31), s, Options{Moves: []string{"13/10 10/9", "8/5 6/5"}})
	if err != nil {
		t.Fatal(err)
	}
	if r.Stop != StopJSD || r.Games >= s.MaxGames || r.Games < s.MinGames {
		t.Fatalf("stop %s after %d games, want a JSD stop between %d and %d", r.Stop, r.Games, s.MinGames, s.MaxGames)
	}
	if r.Candidates[0].Move != "8/5 6/5" || r.Candidates[1].JSD < s.JSDLimit {
		t.Fatalf("best %s, runner-up JSD %.2f: want 8/5 6/5 with JSD >= %g", r.Candidates[0].Move, r.Candidates[1].JSD, s.JSDLimit)
	}
	if c := r.Candidates[0]; math.Abs(c.CI95-z95*c.StdErr) > 1e-12 || c.StdErr <= 0 {
		t.Fatalf("CI95 %g is not 1.96 standard errors (%g)", c.CI95, c.StdErr)
	}
}

func TestCancelReturnsCompletedBatches(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := small()
	s.MaxGames = 1296
	r, err := Run(ctx, decode(t, opening31), s, Options{Progress: func(p Progress) {
		if p.Games >= 36 {
			cancel()
		}
	}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err %v, want context.Canceled", err)
	}
	if r == nil || r.Stop != StopCancelled || r.Games != 36 {
		t.Fatalf("cancelled after the first batch: got %+v", r)
	}
}

func TestCubeRolloutIsDecided(t *testing.T) {
	r, err := Run(context.Background(), decode(t, "XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:0:10"), small(), Options{NoBearoffTable: true})
	if err != nil {
		t.Fatal(err)
	}
	if r.Kind != KindCube || r.Cube == nil {
		t.Fatalf("kind %s, cube %v", r.Kind, r.Cube)
	}
	if r.Cube.Action != "No double" || r.Cube.DoublePass.Equity != 1 {
		t.Fatalf("opening position: %s with DP %g, want No double and DP +1", r.Cube.Action, r.Cube.DoublePass.Equity)
	}
	if !r.CubefulBias {
		t.Fatal("a cubeful rollout must carry its bias flag")
	}
}

func TestPresets(t *testing.T) {
	fast, _ := Preset("rapide")
	std, _ := Preset("standard")
	if fast.Truncation != 7 || fast.MaxGames != 216 || fast.MinGames != 108 || fast.JSDLimit != 3 {
		t.Fatalf("fast preset %+v", fast)
	}
	if std.MaxGames != 1296 || std.MinGames != 324 || std.JSDLimit != 3 {
		t.Fatalf("standard preset %+v", std)
	}
	if err := (Settings{MaxGames: 10, MinGames: 20}).Validate(); err == nil {
		t.Fatal("min games above max games accepted")
	}
	if got := domain.AnalysisDepthRank(fast.DepthLabel()); got <= domain.AnalysisDepthRank("XG Roller++") {
		t.Fatalf("depth label %q ranks %d, not above every ply and XG roller", fast.DepthLabel(), got)
	}
}

// fakeTwoSided writes a two-checker two-sided table in which every entry
// says the side on roll wins outright, and makes it the one race.Resolve
// finds.
func fakeTwoSided(t *testing.T) {
	t.Helper()
	const nPos = 28 // C(2+6, 6)
	hdr := "gnubg-TS-06-02-1"
	hdr += strings.Repeat("x", 39-len(hdr)) + "\n"
	raw := []byte(hdr)
	for i := 0; i < nPos*nPos*4; i++ {
		raw = append(raw, 0xff, 0xff)
	}
	path := filepath.Join(t.TempDir(), "fake.bd")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	race.SetDataDir(t.TempDir())
	race.SetExternalPath(path)
	t.Cleanup(func() {
		race.SetExternalPath("")
		race.SetDataDir("")
		race.Invalidate()
	})
}

// TestExactBearoffEndsTheGame: in a bearoff the table covers, every game
// stops at once on the table's value — here an outright win for the side on
// roll, so No double is +1 exactly, Double/Take +2, and the verdict a pass.
func TestExactBearoffEndsTheGame(t *testing.T) {
	fakeTwoSided(t)
	pos := decode(t, "XGID=-B----------------------b-:0:0:1:00:0:0:0:0:10")
	r, err := Run(context.Background(), pos, small(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !r.ExactBearoff {
		t.Fatal("the table was not picked up")
	}
	c := r.Cube
	if c.NoDouble.Equity != 1 || c.DoubleTake.Equity != 2 || c.NoDouble.StdErr != 0 || c.Action != "Double, pass" {
		t.Fatalf("ND %+.4f DT %+.4f sigma %g action %s: want +1, +2, 0, Double, pass", c.NoDouble.Equity, c.DoubleTake.Equity, c.NoDouble.StdErr, c.Action)
	}
}

// TestVarianceReductionKeepsTheMean: with and without the luck correction,
// the same games give the same mean within a few standard deviations, and
// the correction narrows the interval.
func TestVarianceReductionKeepsTheMean(t *testing.T) {
	heavy(t)
	s := small()
	s.MaxGames = 1296
	opt := Options{Moves: []string{"8/5 6/5"}, NoBearoffTable: true}
	with, err := Run(context.Background(), decode(t, opening31), s, opt)
	if err != nil {
		t.Fatal(err)
	}
	opt.withoutLuck = true
	without, err := Run(context.Background(), decode(t, opening31), s, opt)
	if err != nil {
		t.Fatal(err)
	}
	a, b := with.Candidates[0], without.Candidates[0]
	if d := math.Abs(a.Equity - b.Equity); d > 4*math.Hypot(a.StdErr, b.StdErr) {
		t.Fatalf("variance reduction moved the mean: %+.4f ± %.4f against %+.4f ± %.4f", a.Equity, a.StdErr, b.Equity, b.StdErr)
	}
	if a.StdErr >= b.StdErr {
		t.Fatalf("variance reduction did not reduce: σ %.4f with, %.4f without", a.StdErr, b.StdErr)
	}
}

// TestOwnedCubeRolloutScale: with a 2-cube the player on roll owns, No
// double is counted per unit of that cube, as the search reports it — a
// rollout counted in raw points would read twice as large.
func TestOwnedCubeRolloutScale(t *testing.T) {
	pos := decode(t, "XGID=-CCCBBB-----------bbbccc--:1:1:1:00:0:0:0:0:10")
	eval, err := gammonnet.EvaluatePosition(pos, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	s := small()
	r, err := Run(context.Background(), pos, s, Options{NoBearoffTable: true})
	if err != nil {
		t.Fatal(err)
	}
	want := eval.Cube.CubefulNoDoubleEquity
	if got := r.Cube.NoDouble.Equity; math.Abs(got-want) > 0.15 {
		t.Fatalf("owned 2-cube: rollout No double %+.4f, search %+.4f", got, want)
	}
	if r.Cube.DoublePass.Equity != 1 {
		t.Fatalf("Double/Pass %+.4f, want +1 (the cube's own value)", r.Cube.DoublePass.Equity)
	}
}

// TestMatchMoveRollout: at a match score the plays come out in normalised
// equity, on the search's scale and in its order for a clear decision.
func TestMatchMoveRollout(t *testing.T) {
	pos := decode(t, "XGID=-b----E-C---eE---c-e----B-:0:0:1:31:2:3:0:7:10")
	eval, err := gammonnet.EvaluatePosition(pos, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	searched := map[string]float64{}
	for _, m := range eval.Moves {
		searched[m.Move] = m.Equity
	}
	s := small()
	r, err := Run(context.Background(), pos, s, Options{Moves: []string{"13/10 10/9", "8/5 6/5"}, NoBearoffTable: true})
	if err != nil {
		t.Fatal(err)
	}
	if r.Candidates[0].Move != "8/5 6/5" {
		t.Fatalf("best %s, want 8/5 6/5", r.Candidates[0].Move)
	}
	for _, c := range r.Candidates {
		if math.Abs(c.Equity-searched[c.Move]) > 0.15 {
			t.Fatalf("%s: rollout %+.4f, search %+.4f — not the same scale", c.Move, c.Equity, searched[c.Move])
		}
	}
}

// A play named in another engine's dialect — a chained hop, steps in another
// order — is rolled out under the name it was given.
func TestNamedMovesInAnotherDialect(t *testing.T) {
	s := small()
	s.MaxGames, s.MinGames = 36, 36
	r, err := Run(context.Background(), decode(t, opening31), s, Options{Moves: []string{"13/9", "6/5 8/5"}, NoBearoffTable: true})
	if err != nil {
		t.Fatal(err)
	}
	names := []string{r.Candidates[0].Move, r.Candidates[1].Move}
	slices.Sort(names)
	if names[0] != "13/9" || names[1] != "6/5 8/5" {
		t.Fatalf("candidates %v, want the names given", names)
	}
}
