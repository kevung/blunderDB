package race

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// The Training question generator, Bearoff half (ADR-0041).
//
// # What this is, and what it is not
//
// A question is a SEED plus k PLIES: the engine rolls k times from the seed,
// plays each roll, and the snapshot is the question. It is the only place in
// blunderDB where something resembling a game advances on its own, and
// ADR-0037 is the record that says why that is still not a play mode: no dice
// the user sees, no score that advances, no game to review, nothing kept. The
// plies exist to MAKE a position, and are thrown away with it. A reader who
// found "the engine playing" here and nowhere else was meant to find this
// paragraph.
//
// A uniform placement is not a position. « Fifteen chequers on the ace point »
// is a placement nothing plays into, and the gaps, low stacks and asymmetries
// a real bear-off has appear at their real frequency only if something like a
// game produced them. That claim is measured, not asserted:
// generate_histogram_test.go compares the chequer/wastage histograms of what
// this file produces against the bear-offs of ten real matches.
//
// # Why it lives in engine/race and not next to gammonNet
//
// A bear-off has no contact — Black stands on points 1..6, White on 19..24 —
// so its legal plays are a handful of one-sided moves, and ADR-0041 rule 1
// says the exact table is enough to CHOOSE between them. Nothing here needs
// the neural evaluator, and race cannot import it anyway (gammonnet imports
// race, not the other way round). The Evaluation exercise, whose domain is any
// position, will need gammonNet and therefore a different home; that is issue
// #322's problem, not this file's.
//
// # Parity
//
// The desktop binds this on *gui.App (GenerateBearoffQuestion), the way it
// binds ComputeCubeMatrix: a pure function of the engine over one position,
// with no storage behind it, so there is nothing for the Database wrapper to
// hold and databaseParity has nothing to say. The CLI and the daemon get no
// face in v1 and the absence is a decision, not an oversight: a generated
// question exists only because somebody is about to answer it under a clock,
// and neither a script nor an HTTP client has one (ADR-0041 consequences,
// « serve may expose it later, nothing in v1 requires it »).

// Seed sources (ADR-0041 rule 2). The interface names these three and nothing
// else; « seed » itself is internal vocabulary (CONTEXT.md § Training).
const (
	// SourcePool draws one of this exercise's canonical bear-in shapes.
	SourcePool = "pool"
	// SourceBoard takes the position as it stands, and never k = 0: the user
	// has just seen the seed, so the question is its neighbourhood.
	SourceBoard = "board"
	// SourceLibrary takes a position of the open library, at k = 0: it is
	// already real, and playing on from it would add nothing.
	SourceLibrary = "library"
)

// Refusal codes. A seed outside the domain is refused BY NAME and nothing
// starts (ADR-0041 rule 3): the codes travel to the interface, which owns the
// sentence in nine languages. No silent adaptation — playing on until contact
// breaks would hand the user a position they did not choose.
const (
	// RefusalNotBearoff — a chequer stands outside its own home board, or on
	// the bar. This is the contact case the rule names.
	RefusalNotBearoff = "notBearoff"
	// RefusalTooFewCheckers — the geometry is a bear-off, but a side is
	// outside the 4..15 the exercise trains. Distinct from notBearoff because
	// a refusal that does not name the real reason is not a named refusal.
	RefusalTooFewCheckers = "tooFewCheckers"
	// RefusalEmptyBoard — nothing on the board. The only refusal that still
	// produces a question: rule 3 sends an empty board back to the pool, and
	// says so.
	RefusalEmptyBoard = "emptyBoard"
	// RefusalNoTable — no one-sided table is loaded yet (ADR-0027 generates
	// them in the background on first launch). The exercise ASKS for the EPC,
	// so without the table there is no truth to grade against; saying so beats
	// generating a question nobody can answer.
	RefusalNoTable = "noTable"
	// RefusalUnknownSource — a source this exercise does not serve.
	RefusalUnknownSource = "unknownSource"
)

// The ply budgets of each source (ADR-0041 rule 2).
const (
	maxPoolPlies  = 10
	maxBoardPlies = 4
)

// questionDeadline is how long a walk may take before it gives up (ADR-0041
// rule 5: « past a deadline the generator falls back to k = 0 on a pool seed
// rather than wait »). Half the stated budget of generate_test.go's cost
// test, so the fallback lands before the red does. A walk costs tens of
// microseconds, so on a working machine this never fires; it is there for the
// machine that is not — swapping, suspended mid-walk — where a question late
// by a second is a clock the user watches stand still.
const questionDeadline = 50 * time.Millisecond

// The chequer bounds of the Bearoff domain (ADR-0041 rule 4). The floor is not
// raised: nothing says a four-chequer EPC is trivial.
const (
	minDomainCheckers = 4
	maxDomainCheckers = 15
)

// BearoffRequest asks for one question. Seed is required for the board and
// library sources and ignored for the pool.
type BearoffRequest struct {
	Source string           `json:"source"`
	Seed   *domain.Position `json:"seed,omitempty"`
}

// BearoffQuestion is one question, or the named reason there is none.
//
// Generated and Refusal are independent on purpose. Both set is the
// empty-board case of rule 3: a question was made from the pool AND the
// interface still says why the board was not used. Refusal alone is a refusal:
// nothing starts.
type BearoffQuestion struct {
	Generated bool   `json:"generated"`
	Refusal   string `json:"refusal,omitempty"`
	// Source is where the seed actually came from, which is not always what
	// was asked for — see above.
	Source string `json:"source"`
	// Plies is how many rolls were actually played from the seed. It can be
	// short of what was drawn: the walk stops rather than leave the domain.
	Plies    int             `json:"plies"`
	Position domain.Position `json:"position"`
	// EPC carries both sides' truth, computed here so the caller needs one
	// round trip and the clock — which starts when the question is DISPLAYED —
	// never contains a second one.
	EPC EPC `json:"epc"`
}

// GenerateBearoff makes one Bearoff question.
func GenerateBearoff(req BearoffRequest) BearoffQuestion {
	return generateBearoff(req, rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())))
}

// generateBearoff is the seeded core, so a test can replay a walk exactly.
func generateBearoff(req BearoffRequest, rng *rand.Rand) BearoffQuestion {
	return generateBearoffWithClock(req, rng, time.Now)
}

// generateBearoffWithClock is generateBearoff with the clock the deadline is
// read on, so a test can make a walk late without making it slow.
func generateBearoffWithClock(req BearoffRequest, rng *rand.Rand, now func() time.Time) BearoffQuestion {
	w := walk{rng: rng, now: now, deadline: now().Add(questionDeadline)}
	if !engine.OneSidedReady() {
		return BearoffQuestion{Source: req.Source, Refusal: RefusalNoTable}
	}

	switch req.Source {
	case SourcePool:
		return w.playOut(poolSeed(rng), rng.IntN(maxPoolPlies+1), SourcePool, "")

	case SourceBoard, SourceLibrary:
		if req.Seed == nil {
			return BearoffQuestion{Source: req.Source, Refusal: RefusalNotBearoff}
		}
		seed, refusal := seedFromBoard(&req.Seed.Board)
		seed.onRoll = rollerOf(req.Seed, rng)
		switch {
		case refusal == RefusalEmptyBoard:
			// Rule 3: an empty board is not a wrong position, it is no
			// position. It falls back to the pool AND keeps the sentence —
			// starting silently on something the user never put there would
			// be the adaptation the rule forbids.
			return w.playOut(poolSeed(rng), rng.IntN(maxPoolPlies+1), SourcePool, RefusalEmptyBoard)
		case refusal != "":
			return BearoffQuestion{Source: req.Source, Refusal: refusal}
		}
		plies := 0
		if req.Source == SourceBoard {
			plies = 1 + rng.IntN(maxBoardPlies)
		}
		return w.playOut(seed, plies, req.Source, "")

	default:
		return BearoffQuestion{Source: req.Source, Refusal: RefusalUnknownSource}
	}
}

// sideBoard is one player's own home board: sideBoard[i] is the number of
// chequers on their (i+1)-point, so index 0 is the ace point and index 5 the
// six point. It is the layout engine.ComputeEPC already reads, which is why
// there is no second convention here to keep in step with it.
type sideBoard [6]int

func (b sideBoard) checkers() int {
	total := 0
	for _, n := range b {
		total += n
	}
	return total
}

// highest is the highest occupied point, 1-based, and 0 when the side is out.
func (b sideBoard) highest() int {
	for p := 6; p >= 1; p-- {
		if b[p-1] > 0 {
			return p
		}
	}
	return 0
}

// bearoffSeed is a whole seed: both sides, and who rolls.
type bearoffSeed struct {
	black  sideBoard
	white  sideBoard
	onRoll int
}

// pool holds this exercise's canonical bear-in shapes (ADR-0041 rule 2).
//
// They are DATA OF THE EXERCISE, not a setting: adding one is a code change
// with its histogram check (generate_histogram_test.go), never a user-facing
// option. Each is fifteen chequers home — a bear-in that has just completed —
// and the plies do the rest. The ten cover the shapes a bear-off actually
// starts in: even, back-loaded, front-loaded, gapped, thin on the ace point,
// and — a fifth of them, which is what the real histogram asks for — buried
// low on it.
var pool = [10]sideBoard{
	{0, 1, 2, 4, 4, 4}, // wastage 7.5 — the tightest bear-in there is
	{1, 2, 2, 3, 3, 4}, // 8.3
	{0, 3, 2, 3, 3, 4}, // 8.5, ace point empty
	{1, 2, 3, 3, 3, 3}, // 8.9, flat
	{2, 2, 3, 3, 3, 2}, // 10.4
	{3, 2, 2, 2, 3, 3}, // 11.3, heavy ace point
	{2, 3, 4, 3, 2, 1}, // 13.1, front-loaded
	{4, 0, 3, 2, 4, 2}, // 13.5, gapped on the two point
	{4, 4, 3, 2, 1, 1}, // 18.6, chequers buried low
	{5, 3, 3, 2, 1, 1}, // 19.5, the same, worse
}

// rollerOf is who moves next in the question.
//
// A seed the user brought — from the board or from the library — carries a
// side on roll, and that is the one to keep: at k = 0 the library source hands
// the position back « as it is », and quietly turning every one of them over
// to Black would be exactly the silent adaptation rule 3 forbids, on the one
// field nobody would think to check.
//
// Only when the seed names nobody is the roller DRAWN (rule 4). That is the
// pool's case, and the case of a board somebody edited without choosing a
// side. Drawing is not a fallback here: it is what makes the pool's questions
// come from both sides of the board.
func rollerOf(seed *domain.Position, rng *rand.Rand) int {
	if seed != nil && (seed.PlayerOnRoll == domain.Black || seed.PlayerOnRoll == domain.White) {
		return seed.PlayerOnRoll
	}
	return rng.IntN(2)
}

// poolSeed draws a shape for each side INDEPENDENTLY, so the two sides are
// asymmetric the way a real bear-off is; the roller is drawn too (rule 4:
// « roller drawn »).
func poolSeed(rng *rand.Rand) bearoffSeed {
	return bearoffSeed{
		black:  pool[rng.IntN(len(pool))],
		white:  pool[rng.IntN(len(pool))],
		onRoll: rng.IntN(2),
	}
}

// seedFromBoard reads a seed off a full board, or names why it is not one.
//
// Only the GEOMETRY is judged here — where the chequers are, and how many.
// The cube and the score are not read from the seed and not refused for: they
// are what the exercise IS (rule 4, money play at a centred cube), the manual
// says so, and the EPC does not depend on them.
//
// The SIDE ON ROLL is different, and it is not set here: it belongs to the
// seed when the seed has one. See rollerOf.
func seedFromBoard(b *domain.Board) (bearoffSeed, string) {
	if b.Points[domain.WhiteBar].Checkers > 0 || b.Points[domain.BlackBar].Checkers > 0 {
		return bearoffSeed{}, RefusalNotBearoff
	}
	var seed bearoffSeed
	for i := 1; i <= domain.NumPoints; i++ {
		pt := b.Points[i]
		if pt.Checkers <= 0 {
			continue
		}
		switch pt.Color {
		case domain.Black:
			// Black bears off towards point 1: its home is 1..6, and its own
			// point number is the board index.
			if i > 6 {
				return bearoffSeed{}, RefusalNotBearoff
			}
			seed.black[i-1] += pt.Checkers
		case domain.White:
			// White bears off towards point 24: its home is 19..24, and its
			// own point n sits at board index 25-n.
			if i < 19 {
				return bearoffSeed{}, RefusalNotBearoff
			}
			seed.white[24-i] += pt.Checkers
		default:
			return bearoffSeed{}, RefusalNotBearoff
		}
	}
	if seed.black.checkers() == 0 && seed.white.checkers() == 0 {
		return bearoffSeed{}, RefusalEmptyBoard
	}
	if !inDomain(seed.black) || !inDomain(seed.white) {
		return bearoffSeed{}, RefusalTooFewCheckers
	}
	return seed, ""
}

func inDomain(b sideBoard) bool {
	n := b.checkers()
	return n >= minDomainCheckers && n <= maxDomainCheckers
}

// walk is one question's making: its dice, and the clock its deadline is read on.
type walk struct {
	rng      *rand.Rand
	now      func() time.Time
	deadline time.Time
}

// playOut rolls `plies` times from the seed and turns the snapshot into a
// question. It stops early rather than leave the domain, so what comes back is
// always a position the exercise can ask about — and Plies reports what was
// really played, not what was drawn.
//
// Past the deadline it drops the walk and falls back to a fresh pool seed at
// zero plies (rule 5), and Source says so: a question now beats a better one
// later, and a pool shape at k = 0 is one table lookup.
func (w walk) playOut(seed bearoffSeed, plies int, source, refusal string) BearoffQuestion {
	played := 0
	for ; played < plies; played++ {
		if w.now().After(w.deadline) {
			return question(poolSeed(w.rng), 0, SourcePool, refusal)
		}
		next, ok := onePly(seed, w.rng)
		if !ok {
			break
		}
		seed = next
	}
	return question(seed, played, source, refusal)
}

// question lays a walked seed out as the exercise's question, with its truth.
func question(seed bearoffSeed, played int, source, refusal string) BearoffQuestion {
	pos := domain.Position{
		Board:        boardOf(seed),
		Cube:         domain.Cube{Owner: domain.None, Value: 0},
		Score:        [2]int{domain.Unlimited, domain.Unlimited},
		PlayerOnRoll: seed.onRoll,
		DecisionType: domain.CheckerAction,
	}
	return BearoffQuestion{
		Generated: true,
		Refusal:   refusal,
		Source:    source,
		Plies:     played,
		Position:  pos,
		EPC:       ComputeEPC(&pos.Board),
	}
}

// onePly rolls once, plays the roll for the side on roll, and passes the turn.
// It reports false — and changes nothing — when the play would take the mover
// out of the domain, which is what bounds the walk from below.
func onePly(seed bearoffSeed, rng *rand.Rand) (bearoffSeed, bool) {
	d1, d2 := rng.IntN(6)+1, rng.IntN(6)+1
	var dice []int
	if d1 == d2 {
		dice = []int{d1, d1, d1, d1}
	} else if d1 > d2 {
		dice = []int{d1, d2}
	} else {
		dice = []int{d2, d1}
	}

	mover := seed.black
	if seed.onRoll == domain.White {
		mover = seed.white
	}
	results := legalPlays(mover, dice)
	if len(results) == 0 {
		// Nothing to play: the turn simply passes. It cannot leave the domain,
		// so it is not a stop.
		seed.onRoll = 1 - seed.onRoll
		return seed, true
	}
	best := bestPlay(results)
	if !inDomain(best) {
		return seed, false
	}
	if seed.onRoll == domain.White {
		seed.white = best
	} else {
		seed.black = best
	}
	seed.onRoll = 1 - seed.onRoll
	return seed, true
}

// legalPlays returns the distinct positions reachable from b with `dice`.
// Doubles come in as four equal dice.
//
// No opponent appears: in a bear-off the two home boards are disjoint (1..6
// against 19..24), so nothing blocks and nothing is hit. That is the whole
// reason this generator does not need the contact move generator.
//
// Nor does it need the rule that forbids wasting a die. In THIS domain that
// rule never binds: while a side has a chequer left, its highest occupied
// point can always be played — it moves down when the die is smaller, bears
// off when the die matches, and bears off when the die is bigger precisely
// because it is the highest. So every line plays every die, or ends because
// the side is out. The recursion below therefore keeps every leaf, and
// TestEveryDieIsPlayableWhileCheckersRemain is what makes that a measured fact
// rather than an assumption: widen the domain — a chequer on the bar, a
// chequer outside the home board — and it goes red before this does.
func legalPlays(b sideBoard, dice []int) []sideBoard {
	var out []sideBoard
	seen := make(map[sideBoard]bool, 16)

	var walk func(cur sideBoard, remaining []int)
	walk = func(cur sideBoard, remaining []int) {
		moved := false
		for i, d := range remaining {
			// `remaining` is kept in descending order, so equal dice sit next
			// to each other: playing the first of a pair explores everything
			// playing the second would.
			if i > 0 && remaining[i-1] == d {
				continue
			}
			rest := make([]int, 0, len(remaining)-1)
			rest = append(rest, remaining[:i]...)
			rest = append(rest, remaining[i+1:]...)
			for _, next := range withDie(cur, d) {
				moved = true
				walk(next, rest)
			}
		}
		if moved || cur == b {
			return
		}
		if !seen[cur] {
			seen[cur] = true
			out = append(out, cur)
		}
	}
	walk(b, dice)
	return out
}

// withDie returns every position reachable by playing one die.
func withDie(b sideBoard, die int) []sideBoard {
	high := b.highest()
	if high == 0 {
		return nil
	}
	var out []sideBoard
	for p := high; p >= 1; p-- {
		if b[p-1] == 0 {
			continue
		}
		switch {
		case p > die:
			next := b
			next[p-1]--
			next[p-1-die]++
			out = append(out, next)
		case p == die:
			next := b
			next[p-1]--
			out = append(out, next)
		case p == high:
			// A die larger than the highest occupied point bears off from it.
			next := b
			next[p-1]--
			out = append(out, next)
		}
	}
	return out
}

// bestPlay picks the play that minimises the expected number of rolls left,
// read straight off the exact one-sided table (ADR-0027). That is the whole
// move choice ADR-0041 rule 1 asks for in this domain: a bear-off has no
// contact, so there is nothing an evaluator would weigh that the table does
// not already answer exactly.
//
// Ties keep the first candidate, and the candidates are produced in a fixed
// order, so one seed and one dice stream always give the same walk.
func bestPlay(results []sideBoard) sideBoard {
	best := results[0]
	bestRolls := meanRolls(best)
	for _, candidate := range results[1:] {
		if r := meanRolls(candidate); r < bestRolls {
			best, bestRolls = candidate, r
		}
	}
	return best
}

// meanRolls is the exact expected number of rolls to bear the side off, 0 once
// it is out. A table miss returns +∞ so the candidate loses every comparison
// rather than silently winning them all with a zero.
func meanRolls(b sideBoard) float64 {
	if b.checkers() == 0 {
		return 0
	}
	result, err := engine.ComputeEPC([6]int(b))
	if err != nil || result == nil {
		return math.Inf(1)
	}
	return result.MeanRolls
}

// boardOf lays a seed out on a full board, in the exercise's own frame: cube
// centred, money, the rest of each side borne off (rule 4).
func boardOf(seed bearoffSeed) domain.Board {
	var b domain.Board
	// An empty point is colourless, as the parser writes it and as the
	// frontend's own empty board does. The zero value would say "Black", which
	// is true of nothing and reads as a colour to anyone who forgets to check
	// the chequer count first.
	for i := range b.Points {
		b.Points[i] = domain.Point{Color: domain.None}
	}
	for i := 0; i < 6; i++ {
		if n := seed.black[i]; n > 0 {
			b.Points[i+1] = domain.Point{Checkers: n, Color: domain.Black}
		}
		if n := seed.white[i]; n > 0 {
			b.Points[24-i] = domain.Point{Checkers: n, Color: domain.White}
		}
	}
	b.Bearoff[domain.Black] = maxDomainCheckers - seed.black.checkers()
	b.Bearoff[domain.White] = maxDomainCheckers - seed.white.checkers()
	return b
}
