package transcript

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// The gesture fuzzer drives an Editor the way the session layer does — every
// gesture followed by the Replay's jump (SeekCursor) — through byte-chosen
// sequences of the gestures of fonctionnel.md §2, and holds the properties no
// sequence may break:
//
//   - nothing panics, Replay, Build and RenderMAT included;
//   - a refused gesture leaves the document as it was, and Apply never moves
//     the document it was given (the undo stack keeps it);
//   - the Cursor and the Entry's slot stay within [0, len(Actions)];
//   - undo and redo walk back exactly the documents the gestures left, and
//     undo^n then redo^n gives back the document it started from;
//   - the durable part (header, Actions) survives its JSON round trip;
//   - a consistent document survives RenderMAT → FromMAT with the same plays.

// fuzzOp is one byte-coded step of the scenario.
const (
	opDie = iota
	opClearDice
	opSelect
	opValidate
	opDance
	opDouble
	opTake
	opPass
	opResign
	opBack
	opForward
	opCorrect
	opInsertBefore
	opInsertAfter
	opDelete
	opFlip
	opSetLength
	opSwap
	opSetScore
	opUndo
	opRedo
	opEnterLegal
	opEnterIllegal
	opTurn
	opTurn2
	opTurn3
	opOpening
	opCreate
	opHeader
	opCount
)

// byteSource hands out the fuzz input one byte at a time; exhausted, it says so.
type byteSource struct {
	data []byte
	i    int
}

func (s *byteSource) next() (byte, bool) {
	if s.i >= len(s.data) {
		return 0, false
	}
	b := s.data[s.i]
	s.i++
	return b, true
}

// byteOr is next() with a default once the input is exhausted.
func (s *byteSource) byteOr(d byte) byte {
	if b, ok := s.next(); ok {
		return b
	}
	return d
}

// gestureModel mirrors the Editor's stack with deep copies, so an alias between
// the stack and the live document shows as a difference.
type gestureModel struct {
	t      *testing.T
	ed     *Editor
	past   []Document
	future []Document
	trace  []string
}

func (m *gestureModel) fail(format string, args ...any) {
	m.t.Helper()
	m.t.Fatalf("%s\ntrace:\n%v", fmt.Sprintf(format, args...), m.trace)
}

// apply sends one gesture through the Editor and checks refusal and purity.
func (m *gestureModel) apply(g Gesture) error {
	m.t.Helper()
	m.trace = append(m.trace, fmt.Sprintf("%s die=%d cand=%d side=%d/%v lvl=%d len=%d/%v at=%d score=%v steps=%v board=%v",
		g.Kind, g.Die, g.Candidate, g.Side, g.HasSide, g.Level, g.MatchLength, g.HasLength, g.At, g.Score, g.Steps, g.BoardAfter != nil))
	before := m.ed.Doc.clone()
	err := m.ed.Apply(g)
	if err != nil {
		if !sameDoc(m.ed.Doc, before) {
			m.fail("refused %s (%v) altered the document", g.Kind, err)
		}
		m.settle()
		return err
	}
	if top := m.ed.past[len(m.ed.past)-1]; !sameDoc(top, before) {
		m.fail("%s moved the document it was given", g.Kind)
	}
	m.past = append(m.past, before)
	m.future = nil
	m.settle()
	return nil
}

func (m *gestureModel) undo() {
	m.t.Helper()
	m.trace = append(m.trace, "undo")
	cur := m.ed.Doc.clone()
	ok := m.ed.Undo()
	if ok != (len(m.past) > 0) {
		m.fail("Undo = %v with %d gestures to undo", ok, len(m.past))
	}
	if ok {
		want := m.past[len(m.past)-1]
		m.past = m.past[:len(m.past)-1]
		m.future = append(m.future, cur)
		if !sameDoc(m.ed.Doc, want) {
			m.fail("undo gave %s, want %s", dump(m.ed.Doc), dump(want))
		}
	}
	m.settle()
}

func (m *gestureModel) redo() {
	m.t.Helper()
	m.trace = append(m.trace, "redo")
	cur := m.ed.Doc.clone()
	ok := m.ed.Redo()
	if ok != (len(m.future) > 0) {
		m.fail("Redo = %v with %d gestures to redo", ok, len(m.future))
	}
	if ok {
		want := m.future[len(m.future)-1]
		m.future = m.future[:len(m.future)-1]
		m.past = append(m.past, cur)
		if !sameDoc(m.ed.Doc, want) {
			m.fail("redo gave %s, want %s", dump(m.ed.Doc), dump(want))
		}
	}
	m.settle()
}

// settle is what the session does after every gesture (database.stateOf): replay,
// jump to the Inconsistency, replay again — then the invariants.
func (m *gestureModel) settle() {
	m.t.Helper()
	ann := m.ed.Replay(m.ed.From())
	n := len(m.ed.Doc.Actions)
	if ann.Cursor < 0 || ann.Cursor > n {
		m.fail("Replay put the Cursor at %d of %d", ann.Cursor, n)
	}
	if ann.Cursor != m.ed.Doc.Cursor {
		m.ed.SeekCursor(ann.Cursor)
		ann = m.ed.Replay(m.ed.Doc.Cursor)
	}
	m.check(ann)
}

func (m *gestureModel) check(ann Annotated) {
	m.t.Helper()
	doc := m.ed.Doc
	n := len(doc.Actions)
	if doc.Cursor < 0 || doc.Cursor > n {
		m.fail("Cursor %d out of [0, %d]", doc.Cursor, n)
	}
	if doc.Entry != nil && (doc.Entry.At < 0 || doc.Entry.At > n) {
		m.fail("Entry.At %d out of [0, %d]", doc.Entry.At, n)
	}
	if len(ann.Actions) != n {
		m.fail("Replay annotated %d Actions of %d", len(ann.Actions), n)
	}
	checkDurableRoundTrip(m.t, doc, m.trace)

	// The save and the .mat pane read these; they must not panic on any draft.
	parts := Build(doc)
	_ = ingest.RenderMAT(MatchParts(doc))
	_ = parts
	_ = Candidates(doc)
}

// checkDurableRoundTrip holds what a row keeps: the JSON of the document decodes
// to the same header and Actions.
func checkDurableRoundTrip(t *testing.T, doc Document, trace []string) {
	t.Helper()
	blob, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v\ntrace: %v", err, trace)
	}
	var back Document
	if err := json.Unmarshal(blob, &back); err != nil {
		t.Fatalf("unmarshal: %v\ntrace: %v", err, trace)
	}
	if back.FormatVersion != doc.FormatVersion || back.Cursor != doc.Cursor {
		t.Fatalf("round trip: version/cursor %d/%d, want %d/%d\ntrace: %v",
			back.FormatVersion, back.Cursor, doc.FormatVersion, doc.Cursor, trace)
	}
	if !reflect.DeepEqual(back.Header, doc.Header) {
		t.Fatalf("round trip header:\n got %+v\nwant %+v\ntrace: %v", back.Header, doc.Header, trace)
	}
	if !reflect.DeepEqual(normActions(back.Actions), normActions(doc.Actions)) {
		t.Fatalf("round trip actions:\n got %s\nwant %s\ntrace: %v", dumpActions(back.Actions), dumpActions(doc.Actions), trace)
	}
}

// normActions makes an empty Steps and a nil one the same: JSON cannot tell
// them apart, and neither does any reader of an Action.
func normActions(as []Action) []Action {
	out := make([]Action, len(as))
	for i, a := range as {
		out[i] = a.clone()
		if len(out[i].Steps) == 0 {
			out[i].Steps = nil
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// sameDoc compares every field of two documents, the unexported pending board
// included, blind only to nil against empty slices, which no reader tells apart.
func sameDoc(a, b Document) bool {
	return reflect.DeepEqual(normDoc(a), normDoc(b))
}

func normDoc(d Document) Document {
	out := d.clone()
	out.Actions = normActions(d.Actions)
	if out.Entry != nil && len(out.Entry.Steps) == 0 {
		out.Entry.Steps = nil
	}
	return out
}

func dump(doc Document) string {
	e := "nil"
	if doc.Entry != nil {
		e = fmt.Sprintf("%+v", *doc.Entry)
	}
	return fmt.Sprintf("{cursor=%d return=%d/%v entry=%s pending=%v actions=%s}",
		doc.Cursor, doc.Return, doc.HasReturn, e, doc.pendingBoard != nil, dumpActions(doc.Actions))
}

func dumpActions(as []Action) string {
	s := "["
	for _, a := range as {
		s += fmt.Sprintf(" %d:%s%v%v", a.Side, a.Kind, a.Dice, a.Steps)
		if a.Level != 0 {
			s += fmt.Sprintf("L%d", a.Level)
		}
		if a.Score != nil {
			s += fmt.Sprintf("S%v", *a.Score)
		}
		if a.BoardAfter != nil {
			s += "B"
		}
	}
	return s + " ]"
}

// illegalBoard is a candidate's result with one checker of the mover carried to
// another point: a board no play of the roll reaches, as a hand on the board
// would leave it.
func illegalBoard(b domain.Board, mover int, from, to byte) *domain.Board {
	out := b
	var own []int
	for i := 1; i <= domain.NumPoints; i++ {
		if out.Points[i].Checkers > 0 && out.Points[i].Color == mover {
			own = append(own, i)
		}
	}
	if len(own) == 0 {
		return &out
	}
	src := own[int(from)%len(own)]
	dst := 1 + int(to)%domain.NumPoints
	if dst == src || (out.Points[dst].Checkers > 0 && out.Points[dst].Color != mover) {
		return &out
	}
	out.Points[src].Checkers--
	if out.Points[src].Checkers == 0 {
		out.Points[src].Color = domain.None
	}
	out.Points[dst].Checkers++
	out.Points[dst].Color = mover
	return &out
}

// playTurn types a roll and picks a candidate, dancing when there is none — the
// shape of a real transcription, so that the fuzzer reaches deep into games.
func (m *gestureModel) playTurn(s *byteSource) {
	d1, d2 := 1+int(s.byteOr(0))%6, 1+int(s.byteOr(1))%6
	if m.apply(Gesture{Kind: GestureEnterDie, Die: d1}) != nil {
		return
	}
	if m.apply(Gesture{Kind: GestureEnterDie, Die: d2}) != nil {
		return
	}
	cands := Candidates(m.ed.Doc)
	if len(cands) == 0 {
		// A closed board is a dance.
		if m.apply(Gesture{Kind: GestureValidate}) != nil {
			_ = m.apply(Gesture{Kind: GestureDance})
		}
		return
	}
	if m.apply(Gesture{Kind: GestureSelectCandidate, Candidate: int(s.byteOr(0)) % len(cands)}) != nil {
		return
	}
	_ = m.apply(Gesture{Kind: GestureValidate})
}

func (m *gestureModel) side(s *byteSource) (int, bool) {
	b := s.byteOr(0)
	return int(b>>1) & 1, b&1 == 1
}

func runGestureScenario(t *testing.T, data []byte) {
	src := &byteSource{data: data}
	start := New(int(src.byteOr(7)) % 16)
	m := &gestureModel{t: t, ed: NewEditor(start)}
	m.settle()

	for steps := 0; steps < 200; steps++ {
		b, ok := src.next()
		if !ok {
			break
		}
		switch int(b) % opCount {
		case opDie:
			_ = m.apply(Gesture{Kind: GestureEnterDie, Die: int(src.byteOr(3)) % 8})
		case opClearDice:
			_ = m.apply(Gesture{Kind: GestureClearDice})
		case opSelect:
			_ = m.apply(Gesture{Kind: GestureSelectCandidate, Candidate: int(src.byteOr(0))%40 - 2})
		case opValidate:
			_ = m.apply(Gesture{Kind: GestureValidate})
		case opDance:
			_ = m.apply(Gesture{Kind: GestureDance})
		case opDouble, opTake, opPass:
			kind := map[int]GestureKind{opDouble: GestureDouble, opTake: GestureTake, opPass: GesturePass}[int(b)%opCount]
			side, has := m.side(src)
			_ = m.apply(Gesture{Kind: kind, Side: side, HasSide: has})
		case opResign:
			side, has := m.side(src)
			_ = m.apply(Gesture{Kind: GestureResign, Side: side, HasSide: has, Level: int(src.byteOr(1)) % 5})
		case opBack:
			_ = m.apply(Gesture{Kind: GestureCursorBack})
		case opForward:
			_ = m.apply(Gesture{Kind: GestureCursorForward})
		case opCorrect:
			_ = m.apply(Gesture{Kind: GestureCorrect})
		case opInsertBefore, opInsertAfter:
			kind := GestureInsertBefore
			if int(b)%opCount == opInsertAfter {
				kind = GestureInsertAfter
			}
			side, has := m.side(src)
			_ = m.apply(Gesture{Kind: kind, Side: side, HasSide: has})
		case opDelete:
			_ = m.apply(Gesture{Kind: GestureDelete})
		case opFlip:
			_ = m.apply(Gesture{Kind: GestureFlipSide})
		case opSetLength:
			r := src.byteOr(0)
			_ = m.apply(Gesture{Kind: GestureSetLength, MatchLength: int(src.byteOr(7)) % 26, HasLength: r&1 == 0,
				HasRules: r&2 != 0, Jacoby: r&4 != 0, Beaver: r&8 != 0})
		case opSwap:
			_ = m.apply(Gesture{Kind: GestureSwapPlayers})
		case opSetScore:
			at := int(src.byteOr(0))%(len(m.ed.Doc.Actions)+2) - 1
			g := Gesture{Kind: GestureSetScore, At: at}
			if r := src.byteOr(0); r&1 == 0 {
				sc := [2]int{int(src.byteOr(0))%30 - 2, int(src.byteOr(0))%30 - 2}
				g.Score = &sc
			}
			_ = m.apply(g)
		case opUndo:
			for k := int(src.byteOr(1)) % 6; k >= 0; k-- {
				m.undo()
			}
		case opRedo:
			for k := int(src.byteOr(1)) % 6; k >= 0; k-- {
				m.redo()
			}
		case opEnterLegal:
			cands := Candidates(m.ed.Doc)
			if len(cands) == 0 {
				_ = m.apply(Gesture{Kind: GestureEnterPlay})
				break
			}
			c := cands[int(src.byteOr(0))%len(cands)]
			g := Gesture{Kind: GestureEnterPlay, Steps: c.Steps}
			if src.byteOr(0)&1 == 1 {
				board := c.Result.Board
				g.BoardAfter = &board
			}
			_ = m.apply(g)
		case opEnterIllegal:
			pos := EntryPosition(m.ed.Doc)
			mover := domain.Black
			if m.ed.Doc.Entry != nil {
				mover = m.ed.Doc.Entry.Side
			}
			steps := []domain.CheckerStep{{From: int(src.byteOr(0)) % 26, To: int(src.byteOr(0)) % 26}}
			g := Gesture{Kind: GestureEnterPlay, Steps: steps,
				BoardAfter: illegalBoard(pos.Board, mover, src.byteOr(0), src.byteOr(0))}
			_ = m.apply(g)
		case opTurn, opTurn2, opTurn3:
			m.playTurn(src)
		case opOpening:
			// The two opening dice, player 1's then player 2's, and the first
			// candidate of the side they name.
			d1 := 1 + int(src.byteOr(0))%6
			d2 := 1 + int(src.byteOr(1))%6
			_ = m.apply(Gesture{Kind: GestureEnterDie, Die: d1})
			_ = m.apply(Gesture{Kind: GestureEnterDie, Die: d2})
			_ = m.apply(Gesture{Kind: GestureSelectCandidate})
			_ = m.apply(Gesture{Kind: GestureValidate})
		case opCreate:
			r := src.byteOr(0)
			_ = m.apply(Gesture{Kind: GestureCreate, MatchLength: int(src.byteOr(7)) % 26, HasLength: r&1 == 0,
				HasRules: r&2 != 0, Jacoby: r&4 != 0, Beaver: r&8 != 0})
		case opHeader:
			_ = m.apply(Gesture{Kind: GestureSetHeader, Header: Header{
				Player1: fmt.Sprintf("P%d", src.byteOr(0)), Player2: fmt.Sprintf("Q%d", src.byteOr(0)),
			}})
		}
	}

	checkMATRoundTrip(t, m.ed.Doc, m.trace)

	// undo^n then redo^n: back to the start, then to the very same end.
	final := m.ed.Doc.clone()
	n := len(m.past)
	for i := 0; i < n; i++ {
		if !m.ed.Undo() {
			m.fail("undo %d of %d refused", i+1, n)
		}
	}
	if m.ed.Undo() {
		m.fail("undo past the start of the session")
	}
	for i := 0; i < n; i++ {
		if !m.ed.Redo() {
			m.fail("redo %d of %d refused", i+1, n)
		}
	}
	if !sameDoc(m.ed.Doc, final) {
		m.fail("undo^%d redo^%d gave %s, want %s", n, n, dump(m.ed.Doc), dump(final))
	}
}

// playShape is what a .mat keeps of an Action that produced a Move.
type playShape struct {
	Side  int
	Kind  Kind
	Dice  [2]int
	Cube  domain.Cube
	After domain.Board
}

type gameShape struct {
	InitialScore [2]int
	Winner       int
	PointsWon    int
	Crawford     bool
	Finished     bool
	Plays        []playShape
}

// matShape is the match as the .mat format can carry it: games, their scores and
// results, and the Moves with the board each left.
func matShape(doc Document) (int, []gameShape) {
	ann := Replay(doc, 0)
	games := make([]gameShape, len(ann.Games))
	for i, g := range ann.Games {
		games[i] = gameShape{InitialScore: g.InitialScore, Winner: g.Winner, PointsWon: g.PointsWon,
			Crawford: g.Crawford, Finished: g.Finished}
	}
	for _, info := range ann.Actions {
		if info.GameIndex < 0 || info.GameIndex >= len(games) {
			continue
		}
		if !info.HasPosition {
			continue
		}
		p := playShape{Side: info.Side, Kind: info.Kind, Cube: info.Before.Cube, After: info.After}
		if info.Kind == KindChecker || info.Kind == KindDance {
			p.Dice = info.Before.Dice
		}
		games[info.GameIndex].Plays = append(games[info.GameIndex].Plays, p)
	}
	// A game without a play or a winner, closed by the next game's declared score,
	// has no line in a .mat: there is nothing in it for the file to say.
	kept := games[:0]
	for _, g := range games {
		if len(g.Plays) > 0 || g.Winner >= 0 {
			kept = append(kept, g)
		}
	}
	return doc.Header.MatchLength, kept
}

// checkMATRoundTrip exports a consistent draft to .mat and reads it back: the
// same games, scores, results and plays. An inconsistent draft is exported as
// played (ADR-0044) and is out of the format's reach, so it is not held to this.
func checkMATRoundTrip(t *testing.T, doc Document, trace []string) {
	t.Helper()
	ann := Replay(doc, 0)
	if ann.Inconsistent() || len(ann.Games) == 0 {
		return
	}
	text := ingest.RenderMAT(MatchParts(doc))
	back, err := FromMAT(text)
	if err != nil {
		t.Fatalf("FromMAT of the rendered draft: %v\n%s\ntrace: %v", err, text, trace)
	}
	wl, wg := matShape(doc)
	gl, gg := matShape(back)
	if wl != gl || !reflect.DeepEqual(wg, gg) {
		t.Fatalf(".mat round trip:\n got len %d %+v\nwant len %d %+v\ndoc %s\nback %s\n%s\ntrace: %v",
			gl, gg, wl, wg, dumpActions(doc.Actions), dumpActions(back.Actions), text, trace)
	}
}

// seedScenarios are byte programs worth starting from: a match played for a few
// turns, a cube, a correction, a resignation, bursts of undo and redo.
func seedScenarios() [][]byte {
	turn := byte(opTurn)
	return [][]byte{
		{7, opOpening, 6, 3, turn, 1, 2, 0, turn, 5, 5, 1, turn, 3, 4, 0},
		{7, opOpening, 4, 2, turn, 6, 6, 0, opDouble, 0, opTake, 0, turn, 2, 1, 0, opUndo, 3, opRedo, 3},
		{5, opOpening, 3, 1, turn, 6, 5, 0, opDouble, 0, opPass, 0, opOpening, 2, 6, turn, 1, 1, 0},
		{3, opOpening, 5, 2, turn, 4, 4, 0, opBack, opBack, opDie, 6, opDie, 6, opSelect, 1, opForward, opValidate},
		{7, opOpening, 1, 6, turn, 2, 2, 0, opResign, 3, 2, opOpening, 3, 5, opSetScore, 3, 0, 9, 9},
		{1, opOpening, 6, 5, turn, 3, 3, 0, opBack, opDelete, opInsertBefore, 0, opFlip, opSetLength, 0, 0},
		{0, opOpening, 4, 1, turn, 6, 2, 0, opDouble, 0, opTake, 0, opSwap, opEnterIllegal, 1, 2, 3, 4, opValidate},
		{7, opOpening, 2, 2, opOpening, 5, 3, turn, 1, 1, 0, opEnterLegal, 0, 1, opUndo, 5, opRedo, 5, opCreate, 0, 3},
	}
}

func FuzzTranscriptGestures(f *testing.F) {
	for _, s := range seedScenarios() {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 400 {
			return
		}
		runGestureScenario(t, data)
	})
}

// TestTranscriptGesturesSeeded runs the scenario on pseudo-random programs from
// fixed seeds, so the properties hold on every push, not only under -fuzz.
func TestTranscriptGesturesSeeded(t *testing.T) {
	n := 24
	if testing.Short() {
		n = 10
	}
	for seed := 0; seed < n; seed++ {
		data := make([]byte, 160)
		x := uint32(seed*2654435761 + 1)
		for i := range data {
			x ^= x << 13
			x ^= x >> 17
			x ^= x << 5
			data[i] = byte(x)
		}
		t.Run(fmt.Sprint(seed), func(t *testing.T) { runGestureScenario(t, data) })
	}
}

// TestGameStartIsTheOpeningRoll: a game's first play is typed as the opening roll —
// player 1's die, then player 2's, the higher one naming the side — and what no
// opening can be is marked, never refused: a roll of doubles, a cube before the
// first play.
func TestGameStartIsTheOpeningRoll(t *testing.T) {
	doc := runSteps(t, New(7), []step{
		{name: "player 1's die", g: die(2)}, {name: "player 2's die", g: die(5)},
		{name: "candidate", g: candidate(0)}, {name: "play", g: confirm()},
	})
	if a := doc.Actions[0]; a.Side != domain.White || a.Kind != KindChecker || a.Dice != [2]int{2, 5} {
		t.Fatalf("a 2 against a 5 gives %s, want player 2's 52", dumpActions(doc.Actions))
	}
	ann := Replay(doc, 0)
	if !ann.Actions[0].OpensGame || ann.Inconsistent() || ann.Next.Side != domain.Black {
		t.Errorf("the first play: %+v, next %+v", ann.Actions[0], ann.Next)
	}

	doubles := runSteps(t, New(7), []step{
		{name: "die 3", g: die(3)}, {name: "die 3 again", g: die(3)},
		{name: "candidate", g: candidate(0)}, {name: "play", g: confirm()},
	})
	if !hasInconsistency(Replay(doubles, 0).Actions[0], InconsistentDice) {
		t.Errorf("a first play rolled 33 carries no Inconsistency")
	}
	if !Build(doubles).Inconsistent {
		t.Errorf("the export does not say the draft is inconsistent")
	}

	// Re-editing the cell with the same two dice, in either order, keeps the camp:
	// only a different roll decides the opening again.
	for _, order := range [][2]int{{2, 5}, {5, 2}} {
		again := runSteps(t, doc, []step{
			{name: "back", g: Gesture{Kind: GestureCursorBack}},
			{name: "die", g: die(order[0])}, {name: "die", g: die(order[1])},
			{name: "play", g: confirm()},
		})
		if a := again.Actions[0]; a.Side != domain.White || a.Dice != [2]int{2, 5} {
			t.Errorf("re-edited with %v: %+v, want player 2's 52 untouched", order, a)
		}
	}
	// `s` on a first play gives the opening to the other camp, dice with it.
	flipped := runSteps(t, doc, []step{{name: "back", g: Gesture{Kind: GestureCursorBack}},
		{name: "flip", g: Gesture{Kind: GestureFlipSide}}})
	if a := flipped.Actions[0]; a.Side != domain.Black || a.Dice != [2]int{5, 2} {
		t.Errorf("flipped first play = %+v", a)
	}

	cube := runSteps(t, New(7), []step{{name: "double", g: Gesture{Kind: GestureDouble}}})
	if !hasInconsistency(Replay(cube, 0).Actions[0], ImpossibleCube) {
		t.Errorf("a double before the first play carries no Inconsistency")
	}
}
