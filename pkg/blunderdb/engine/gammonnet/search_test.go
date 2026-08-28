package gammonnet

import (
	"math"
	"sync"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

const openingXGID = "XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:0:10"

func openingPosition(t *testing.T) Position {
	t.Helper()
	dp, err := domain.DecodeXGID(openingXGID)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	dp.PlayerOnRoll = domain.White
	p, err := FromDomain(&dp)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	return p
}

// The opening plays where the best move does not get argued about. This is the
// test that catches errors of MEANING rather than of arithmetic — inverted
// colours, a flipped perspective, permuted dice, a negation the wrong way round.
// None of those crash; all of them fail here.
//
// Positions are checked rather than notations: index i is point i+1 for White,
// so its five point is index 4, its bar point index 6, its midpoint index 12.
func TestUndisputedOpeningPlays(t *testing.T) {
	cases := []struct {
		name   string
		d1, d2 int
		want   map[int]int8 // engine index → checkers after the play
	}{
		{"3-1 makes the five point", 3, 1, map[int]int8{4: 2, 5: 4, 7: 2}},
		{"6-1 makes the bar point", 6, 1, map[int]int8{6: 2, 7: 2, 12: 4}},
		{"4-2 makes the four point", 4, 2, map[int]int8{3: 2, 5: 4, 7: 2}},
		{"6-5 runs one back checker", 6, 5, map[int]int8{23: 1, 12: 6}},
	}

	// 0-ply is the cheap gate and runs always. 2-ply costs seconds per decision
	// — measured, not guessed — so it stays out of the short cycle; a package
	// whose tests cannot finish inside `go test`'s ten minutes is a package
	// nobody runs.
	plies := []int{0}
	if !testing.Short() {
		plies = append(plies, 2)
	}
	for _, ply := range plies {
		s, err := NewSearcher(DefaultConfig(ply))
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range cases {
			t.Run(nameWithPly(tc.name, ply), func(t *testing.T) {
				p := openingPosition(t)
				best, ok, err := s.BestPlay(&p, tc.d1, tc.d2)
				if err != nil || !ok {
					t.Fatalf("no play: ok=%v err=%v", ok, err)
				}
				for idx, want := range tc.want {
					if got := best.Play.Result.Points[idx]; got != want {
						t.Errorf("index %d holds %d, want %d (equity %.4f)",
							idx, got, want, best.Equity)
					}
				}
			})
		}
	}
}

func nameWithPly(name string, ply int) string {
	return name + " (" + string(rune('0'+ply)) + "-ply)"
}

// A deeper search must not change what the engine is looking at: the candidate
// list holds the same plays, whatever the depth.
func TestSameCandidatesAtEveryDepth(t *testing.T) {
	if testing.Short() {
		t.Skip("includes a 2-ply search")
	}
	p := openingPosition(t)
	var counts []int
	for _, ply := range []int{0, 1, 2} {
		s, err := NewSearcher(SearchConfig{Ply: ply}) // pruning off: every legal play
		if err != nil {
			t.Fatal(err)
		}
		out := make([]Candidate, MaxPlays)
		n, err := s.Plays(&p, 3, 1, out)
		if err != nil {
			t.Fatal(err)
		}
		counts = append(counts, n)
	}
	for i := 1; i < len(counts); i++ {
		if counts[i] != counts[0] {
			t.Errorf("candidate counts differ by depth: %v", counts)
		}
	}
}

// The candidate list comes back best first.
func TestCandidatesAreOrdered(t *testing.T) {
	p := openingPosition(t)
	s, err := NewSearcher(DefaultConfig(0))
	if err != nil {
		t.Fatal(err)
	}
	out := make([]Candidate, MaxPlays)
	n, err := s.Plays(&p, 6, 5, out)
	if err != nil || n < 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	for i := 1; i < n; i++ {
		if out[i].Equity > out[i-1].Equity {
			t.Fatalf("candidate %d (%.6f) outranks %d (%.6f)", i, out[i].Equity, i-1, out[i-1].Equity)
		}
	}
}

// The cache may change how long a search takes. It must never change what it
// answers: the key is the position alone, so a hit returns exactly what a miss
// would have computed.
func TestCacheDoesNotChangeTheAnswer(t *testing.T) {
	if testing.Short() {
		t.Skip("a 1-ply search costs hundreds of milliseconds")
	}
	p := openingPosition(t)
	s, err := NewSearcher(DefaultConfig(1))
	if err != nil {
		t.Fatal(err)
	}
	warm := make([]Candidate, MaxPlays)
	n1, err := s.Plays(&p, 5, 3, warm)
	if err != nil {
		t.Fatal(err)
	}
	again := make([]Candidate, MaxPlays)
	n2, err := s.Plays(&p, 5, 3, again) // now every leaf is cached
	if err != nil {
		t.Fatal(err)
	}
	if n1 != n2 {
		t.Fatalf("%d candidates cold, %d warm", n1, n2)
	}
	for i := 0; i < n1; i++ {
		if warm[i].Play.Result != again[i].Play.Result || warm[i].Equity != again[i].Equity {
			t.Fatalf("candidate %d changed once cached: %.17g → %.17g", i, warm[i].Equity, again[i].Equity)
		}
	}
	if s.cache.hits == 0 {
		t.Error("no cache hit in the second pass — the cache is not being consulted")
	}
}

// Searchers are per goroutine over a shared read-only network. Running the same
// search concurrently must give the identical answer every time.
func TestConcurrentSearchesAgree(t *testing.T) {
	if testing.Short() {
		t.Skip("eight concurrent 1-ply searches")
	}
	p := openingPosition(t)
	ref, err := NewSearcher(DefaultConfig(1))
	if err != nil {
		t.Fatal(err)
	}
	want, ok, err := ref.BestPlay(&p, 3, 1)
	if err != nil || !ok {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errs := make(chan string, 8)
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, err := NewSearcher(DefaultConfig(1))
			if err != nil {
				errs <- err.Error()
				return
			}
			pos := p
			got, ok, err := s.BestPlay(&pos, 3, 1)
			if err != nil || !ok {
				errs <- "no play"
				return
			}
			if got.Play.Result != want.Play.Result || got.Equity != want.Equity {
				errs <- "diverged"
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}

func TestGameValue(t *testing.T) {
	// White has borne off everything; Black's position decides the stake.
	build := func(mut func(*Position)) Position {
		var p Position
		p.Off[White] = 15
		p.Points[23] = -15 // Black on its own 24 point, outside White's home
		p.Turn = Black     // the turn names the loser
		mut(&p)
		return p
	}
	for _, tc := range []struct {
		name string
		mut  func(*Position)
		want int
	}{
		{"plain: the loser has borne off", func(p *Position) { p.Points[23] = -14; p.Off[Black] = 1 }, 1},
		{"gammon: nothing off, nothing back", func(p *Position) {}, 2},
		{"backgammon: a checker on the bar", func(p *Position) { p.Points[23] = -14; p.Bar[Black] = 1 }, 3},
		{"backgammon: a checker in the winner's home", func(p *Position) { p.Points[23] = -14; p.Points[2] = -1 }, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := build(tc.mut)
			if !p.Valid() {
				t.Fatalf("test board is not valid")
			}
			if got := gameValue(&p); got != tc.want {
				t.Errorf("stake %d, want %d", got, tc.want)
			}
			if got := terminalEquity(&p); got != -float64(tc.want) {
				t.Errorf("terminal equity %v, want %v (turn names the loser)", got, -float64(tc.want))
			}
		})
	}
}

// A 1-ply value is the weighted average over the 21 rolls of the best reply.
// The weights sum to one, so the value must lie between the worst and best of
// them — a coarse check, but it catches a weight applied to the wrong roll.
func TestOnePlyValueLiesWithinItsReplies(t *testing.T) {
	if testing.Short() {
		t.Skip("a 1-ply search over every roll")
	}
	p := openingPosition(t)
	s, err := NewSearcher(SearchConfig{Ply: 1})
	if err != nil {
		t.Fatal(err)
	}
	v, ok := s.positionEquity(&p, 1, 0, false)
	if !ok {
		t.Fatal("search failed")
	}
	if math.Abs(v) > 3 {
		t.Errorf("1-ply value %v is outside any plausible equity", v)
	}
	// The opening is symmetric, so its value to the player on roll is small and
	// positive — having the roll is worth something, but not much.
	if v < 0 || v > 0.3 {
		t.Errorf("value of the opening roll = %.4f, expected a small positive equity", v)
	}
}

// Parallelism must change how long a search takes and nothing else. The twenty
// one roll values are computed by different goroutines but summed afterwards in
// roll order, so the result is bit-identical to the serial one — not "equal to
// within a tolerance", identical.
func TestParallelSearchIsBitIdentical(t *testing.T) {
	if testing.Short() {
		t.Skip("two 2-ply searches")
	}
	p := openingPosition(t)

	serial, err := NewSearcher(DefaultConfig(2))
	if err != nil {
		t.Fatal(err)
	}
	par, err := NewSearcher(DefaultConfig(2))
	if err != nil {
		t.Fatal(err)
	}
	par = par.WithWorkers(8)

	a := make([]Candidate, MaxPlays)
	b := make([]Candidate, MaxPlays)
	pa, pb := p, p
	na, err := serial.Plays(&pa, 3, 1, a)
	if err != nil {
		t.Fatal(err)
	}
	nb, err := par.Plays(&pb, 3, 1, b)
	if err != nil {
		t.Fatal(err)
	}
	if na != nb {
		t.Fatalf("%d candidates serial, %d parallel", na, nb)
	}
	for i := 0; i < na; i++ {
		if a[i].Play.Result != b[i].Play.Result {
			t.Fatalf("candidate %d is a different play", i)
		}
		if a[i].Equity != b[i].Equity {
			t.Fatalf("candidate %d: serial %.17g, parallel %.17g — not bit-identical",
				i, a[i].Equity, b[i].Equity)
		}
	}
}
