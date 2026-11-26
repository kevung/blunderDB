package arkqr

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// FormatVersion is the first byte of every coded match. The models below —
// the kind table, the dice law, the play model — are frozen for it.
const FormatVersion = 1

// ErrFormat is returned for a coded match this build cannot read.
var ErrFormat = errors.New("arkqr: not a readable archived match")

// Match is what the codec archives: a match as a transcription document, its
// comment, and per Action the notes the library holds on the position it was
// played from. Analyses are not archived: they are computed again.
type Match struct {
	Doc     transcript.Document
	Comment string
	// Notes is parallel to Doc.Actions, or nil when no Action has any.
	Notes []Note
	// The source metadata of the match, which a transcription document does
	// not hold: the players' ratings and experience, the program that wrote
	// the file. Nil is "not stated".
	Elo           [2]*float64
	Experience    [2]*int
	EngineVersion string
}

// maxLen bounds every string and list the codec writes: what the decoder
// accepts, so an archive that encodes always decodes.
const maxLen = 1 << 16

// Note is what an Action's position carries besides the play: its comments,
// oldest first, and the study mark of the source tool.
type Note struct {
	Comments []string
	Flagged  bool
}

func (n Note) empty() bool { return len(n.Comments) == 0 && !n.Flagged }

var kinds = [...]transcript.Kind{
	transcript.KindChecker, transcript.KindDance, transcript.KindUnrecorded, transcript.KindDouble,
	transcript.KindTake, transcript.KindPass, transcript.KindResign, transcript.KindForfeit,
}

const (
	tokEnd   = len(kinds)
	tokStart = tokEnd + 1
)

// kindTable[previous token][token]: which Action follows which.
var kindTable = [tokStart + 1][tokEnd + 1]uint64{
	//  checker dance unrec double take pass resign forfeit end
	{3800, 100, 4, 60, 1, 1, 25, 2, 2},    // after a checker play
	{3600, 300, 4, 60, 1, 1, 5, 2, 2},     // after a dance
	{3000, 100, 300, 60, 1, 1, 30, 4, 30}, // after an unrecorded play
	{1, 1, 1, 1, 2800, 1200, 4, 2, 4},     // after a double
	{3900, 150, 4, 1, 1, 1, 4, 2, 2},      // after a take
	{4000, 4, 4, 4, 1, 1, 4, 4, 60},       // after a pass
	{4000, 4, 4, 4, 1, 1, 4, 4, 60},       // after a resignation
	{60, 1, 1, 1, 1, 1, 1, 1, 4000},       // after a forfeit
	{1, 1, 1, 1, 1, 1, 1, 1, 1},           // (end: never a context)
	{4000, 1, 1, 1, 1, 1, 1, 1, 8},        // at the start of the document
}

// dicePairs are the 21 unordered rolls, larger die first; diceFreq their law.
var dicePairs, diceFreq = func() ([21][2]int, []uint64) {
	var p [21][2]int
	f := make([]uint64, 0, 21)
	i := 0
	for a := 6; a >= 1; a-- {
		for b := a; b >= 1; b-- {
			p[i] = [2]int{a, b}
			if a == b {
				f = append(f, 1)
			} else {
				f = append(f, 2)
			}
			i++
		}
	}
	return p, f
}()

var resignFreq = []uint64{8, 3, 1}

// models holds the adaptive bits of one coded match.
type models struct {
	declared, sideSame, standard, order, beforeSame, legal, note *adaptive
	times                                                        [4]*adaptive
}

func newModels() *models {
	m := &models{
		declared: newAdaptive(255, 1), sideSame: newAdaptive(1, 255), standard: newAdaptive(1, 255),
		order: newAdaptive(1, 1), beforeSame: newAdaptive(1, 255), legal: newAdaptive(1, 255),
		note: newAdaptive(63, 1),
	}
	for i := range m.times {
		m.times[i] = newAdaptive(63, 1)
	}
	return m
}

// state is what both ends track to predict the next Action: the board, whose
// side it is read from, and who rolled and doubled last in the game.
type state struct {
	atStart             bool
	board               rel
	own                 int
	lastRoller, doubler int
}

func (s *state) newGame() { *s = state{atStart: true, lastRoller: -1, doubler: -1} }

func (s *state) before(side int) rel {
	switch {
	case s.atStart:
		return start
	case s.own == side:
		return s.board
	default:
		return s.board.swap()
	}
}

func (s *state) expectedSide(k transcript.Kind) int {
	switch {
	case k == transcript.KindTake || k == transcript.KindPass:
		if s.doubler >= 0 {
			return 1 - s.doubler
		}
	case !s.atStart && s.lastRoller >= 0:
		return 1 - s.lastRoller
	}
	return -1
}

// Encode codes a match. The document must be one a Replay can read: Actions
// of known kinds and sides, of the current transcription format.
func Encode(m Match) ([]byte, error) {
	doc := m.Doc
	if doc.FormatVersion != transcript.FormatVersion {
		return nil, fmt.Errorf("arkqr: transcription format %d, want %d", doc.FormatVersion, transcript.FormatVersion)
	}
	if doc.Header.MatchLength < 0 || doc.Header.MaxCube < 0 {
		return nil, fmt.Errorf("arkqr: negative match length or cube limit")
	}
	if m.Notes != nil && len(m.Notes) != len(doc.Actions) {
		return nil, fmt.Errorf("arkqr: %d notes for %d actions", len(m.Notes), len(doc.Actions))
	}
	for i, a := range doc.Actions {
		if a.Side != 0 && a.Side != 1 {
			return nil, fmt.Errorf("arkqr: action %d: side %d", i, a.Side)
		}
		if kindIndex(a.Kind) < 0 {
			return nil, fmt.Errorf("arkqr: action %d: unknown kind %q", i, a.Kind)
		}
	}
	if err := checkLimits(m); err != nil {
		return nil, err
	}
	ann := transcript.Replay(doc, 0)
	e := newEncoder()
	out := m
	codeMatch(e, &out, ann.Actions)
	return append([]byte{FormatVersion}, e.finish()...), nil
}

// Decode reads a match Encode wrote.
func Decode(data []byte) (Match, error) {
	if len(data) == 0 || data[0] != FormatVersion {
		return Match{}, fmt.Errorf("%w: format byte", ErrFormat)
	}
	d := newDecoder(data[1:])
	var m Match
	codeMatch(d, &m, nil)
	if d.failed {
		return Match{}, fmt.Errorf("%w: damaged stream", ErrFormat)
	}
	return m, nil
}

func checkLimits(m Match) error {
	h := m.Doc.Header
	for _, s := range []string{h.Player1, h.Player2, h.Event, h.Location, h.Round, h.Transcriber, h.VideoSource, m.Comment, m.EngineVersion} {
		if len(s) > maxLen {
			return fmt.Errorf("arkqr: a header text of %d bytes, at most %d", len(s), maxLen)
		}
	}
	for i, a := range m.Doc.Actions {
		if len(a.Steps) > maxLen {
			return fmt.Errorf("arkqr: action %d: %d steps", i, len(a.Steps))
		}
	}
	for i, n := range m.Notes {
		if len(n.Comments) > maxLen {
			return fmt.Errorf("arkqr: action %d: %d comments, at most %d", i, len(n.Comments), maxLen)
		}
		for _, c := range n.Comments {
			if len(c) > maxLen {
				return fmt.Errorf("arkqr: action %d: a comment of %d bytes, at most %d", i, len(c), maxLen)
			}
		}
	}
	return nil
}

func kindIndex(k transcript.Kind) int {
	for i, x := range kinds {
		if x == k {
			return i
		}
	}
	return -1
}

func boolBit(c coder, v bool) bool { return flag(c, 1, 1, v) }

// str codes a string as its length and its bytes.
func str(c coder, s string) string {
	n := int(uvar(c, uint64(len(s))))
	if d, ok := c.(*decoder); ok && (n > maxLen || d.failed) {
		d.failed = true
		return ""
	}
	b := make([]byte, n)
	for i := range n {
		var v uint64
		if !c.decoding() {
			v = uint64(s[i])
		}
		b[i] = byte(raw(c, 8, v))
	}
	return string(b)
}

func count(c coder, n int) int {
	v := int(uvar(c, uint64(n)))
	if d, ok := c.(*decoder); ok && (v > maxLen || d.failed) {
		d.failed = true
		return 0
	}
	return v
}

func codeHeader(c coder, m *Match) {
	h := &m.Doc.Header
	h.MatchLength = int(uvar(c, uint64(h.MatchLength)))
	h.Jacoby = boolBit(c, h.Jacoby)
	h.Beaver = boolBit(c, h.Beaver)
	h.MaxCube = int(uvar(c, uint64(h.MaxCube)))
	h.PostCrawford = boolBit(c, h.PostCrawford)
	h.Player1 = str(c, h.Player1)
	h.Player2 = str(c, h.Player2)
	h.Event = str(c, h.Event)
	h.Location = str(c, h.Location)
	h.Round = str(c, h.Round)
	h.Transcriber = str(c, h.Transcriber)
	h.VideoSource = str(c, h.VideoSource)
	m.Comment = str(c, m.Comment)
	if boolBit(c, !h.Date.IsZero()) {
		sec := svar(c, h.Date.Unix())
		nsec := svar(c, int64(h.Date.Nanosecond()))
		h.Date = time.Unix(sec, nsec).UTC()
	}
	if boolBit(c, m.Elo[0] != nil || m.Elo[1] != nil || m.Experience[0] != nil || m.Experience[1] != nil || m.EngineVersion != "") {
		for i := range 2 {
			if boolBit(c, m.Elo[i] != nil) {
				var v uint64
				if m.Elo[i] != nil {
					v = math.Float64bits(*m.Elo[i])
				}
				f := math.Float64frombits(raw(c, 64, v))
				m.Elo[i] = &f
			}
			if boolBit(c, m.Experience[i] != nil) {
				var v int64
				if m.Experience[i] != nil {
					v = int64(*m.Experience[i])
				}
				x := int(svar(c, v))
				m.Experience[i] = &x
			}
		}
		m.EngineVersion = str(c, m.EngineVersion)
	}
}

// codeMatch is the format: one function both ends run, the encoder reading
// m and info, the decoder filling m.
func codeMatch(c coder, m *Match, info []transcript.ActionInfo) {
	enc := !c.decoding()
	if !enc {
		m.Doc.FormatVersion = transcript.FormatVersion
	}
	codeHeader(c, m)
	md := newModels()
	var st state
	st.newGame()
	prev := tokStart
	hasNotes := false
	var notes []Note
	for i := 0; ; i++ {
		if d, ok := c.(*decoder); ok && d.failed {
			return
		}
		var a transcript.Action
		tok := tokEnd
		if enc && i < len(m.Doc.Actions) {
			a = m.Doc.Actions[i]
			tok = kindIndex(a.Kind)
		}
		tok = c.sym(kindTable[prev][:], tok)
		if tok == tokEnd {
			break
		}
		prev = tok
		a.Kind = kinds[tok]
		if md.declared.code(c, a.Score != nil) {
			var s [2]int
			if enc {
				s = *a.Score
			}
			s[0] = int(svar(c, int64(s[0])))
			s[1] = int(svar(c, int64(s[1])))
			a.Score = &s
			st.newGame()
		}
		if exp := st.expectedSide(a.Kind); exp < 0 {
			a.Side = c.sym(fair, a.Side)
		} else if md.sideSame.code(c, a.Side == exp) {
			a.Side = exp
		} else {
			a.Side = 1 - exp
		}
		var ai *transcript.ActionInfo
		if enc && i < len(info) {
			ai = &info[i]
		}
		codeAction(c, md, &st, &a, ai)
		codeTimes(c, md, &a)
		var n Note
		if enc && i < len(m.Notes) {
			n = m.Notes[i]
		}
		if md.note.code(c, !n.empty()) {
			n.Flagged = boolBit(c, n.Flagged)
			k := count(c, len(n.Comments))
			cs := make([]string, k)
			for j := range k {
				var s string
				if enc {
					s = n.Comments[j]
				}
				cs[j] = str(c, s)
			}
			n.Comments = nil
			if k > 0 {
				n.Comments = cs
			}
			hasNotes = true
		}
		if !enc {
			m.Doc.Actions = append(m.Doc.Actions, a)
			notes = append(notes, n)
		}
	}
	if boolBit(c, m.Doc.NextScore != nil) {
		var s [2]int
		if enc {
			s = *m.Doc.NextScore
		}
		s[0] = int(svar(c, int64(s[0])))
		s[1] = int(svar(c, int64(s[1])))
		m.Doc.NextScore = &s
	}
	if !enc {
		m.Doc.Cursor = len(m.Doc.Actions)
		if hasNotes {
			m.Notes = notes
		}
	}
}

func isRoll(k transcript.Kind) bool {
	return k == transcript.KindChecker || k == transcript.KindDance || k == transcript.KindUnrecorded
}

func validDice(d [2]int) bool { return d[0] >= 1 && d[0] <= 6 && d[1] >= 1 && d[1] <= 6 }

// standard reports whether an Action is coded by the models rather than
// spelled out, and for a checker play the board it was played from and the
// index of its play among the candidates.
func standard(a *transcript.Action, info *transcript.ActionInfo) (bool, rel, []play, int) {
	if a.BoardAfter != nil || a.BoardBefore != nil {
		return false, rel{}, nil, 0
	}
	switch {
	case a.Kind == transcript.KindResign:
		if a.Level < 1 || a.Level > 3 || a.Dice != [2]int{} || len(a.Steps) > 0 {
			return false, rel{}, nil, 0
		}
		return true, rel{}, nil, 0
	case a.Level != 0:
		return false, rel{}, nil, 0
	case !isRoll(a.Kind):
		return a.Dice == [2]int{} && len(a.Steps) == 0, rel{}, nil, 0
	case !validDice(a.Dice):
		return false, rel{}, nil, 0
	case a.Kind != transcript.KindChecker:
		return len(a.Steps) == 0, rel{}, nil, 0
	}
	if info == nil || !info.HasPosition || len(info.Inconsistencies) > 0 || len(a.Steps) == 0 {
		return false, rel{}, nil, 0
	}
	before := relOf(info.Before.Board, a.Side)
	after, ok := applyDomainSteps(before, a.Steps, a.Side)
	if !ok {
		return false, rel{}, nil, 0
	}
	ps := before.plays(a.Dice[0], a.Dice[1])
	for i, p := range ps {
		if p.after == after {
			return true, before, ps, i
		}
	}
	return false, rel{}, nil, 0
}

func applyDomainSteps(r rel, steps []domain.CheckerStep, side int) (rel, bool) {
	for _, s := range steps {
		from, ok1 := relPoint(side, s.From)
		to, ok2 := relPoint(side, s.To)
		if !ok1 || !ok2 || to == 25 {
			return r, false
		}
		// The hit is the board's to say: a source may leave the mark out.
		hit := to > 0 && r.opp[25-to] == 1
		var ok bool
		if r, ok = r.applyStep(step{from, to, hit}); !ok {
			return r, false
		}
	}
	return r, true
}

func domainSteps(steps []step, side int) []domain.CheckerStep {
	out := make([]domain.CheckerStep, len(steps))
	for i, s := range steps {
		out[i] = domain.CheckerStep{From: absPoint(side, s.from), To: absPoint(side, s.to), Hit: s.hit}
	}
	return out
}

func codeAction(c coder, md *models, st *state, a *transcript.Action, info *transcript.ActionInfo) {
	enc := !c.decoding()
	var std bool
	var before rel
	var ps []play
	var idx int
	if enc {
		std, before, ps, idx = standard(a, info)
	}
	std = md.standard.code(c, std)
	if st.atStart && isRoll(a.Kind) {
		st.atStart = false
		st.board, st.own = start, a.Side
	}
	if !std {
		codeSpelled(c, a)
		switch {
		case a.Kind == transcript.KindChecker:
			b := st.before(a.Side)
			after := b
			if a.BoardAfter != nil {
				after = relOf(*a.BoardAfter, a.Side)
			} else if r, ok := applyDomainSteps(b, a.Steps, a.Side); ok {
				after = r
			}
			st.played(a.Side, after)
		case isRoll(a.Kind):
			st.lastRoller = a.Side
		default:
			st.after(a.Kind, a.Side)
		}
		return
	}
	if isRoll(a.Kind) {
		di := 0
		if enc {
			di = pairIndex(a.Dice)
		}
		di = c.sym(diceFreq, di)
		p := dicePairs[di]
		if p[0] != p[1] && md.order.code(c, enc && a.Dice[0] < a.Dice[1]) {
			p[0], p[1] = p[1], p[0]
		}
		a.Dice = p
	}
	switch a.Kind {
	case transcript.KindChecker:
		exp := st.before(a.Side)
		if md.beforeSame.code(c, before == exp) {
			before = exp
		} else {
			before = rawRel(c, before)
		}
		if !enc {
			ps = before.plays(a.Dice[0], a.Dice[1])
			if len(ps) == 0 {
				if d, ok := c.(*decoder); ok {
					d.failed = true
				}
				return
			}
		}
		if len(ps) > 1 {
			idx = c.sym(playFreqs(ps, before.contact()), idx)
		}
		a.Steps = domainSteps(ps[idx].steps, a.Side)
		st.played(a.Side, ps[idx].after)
	case transcript.KindDance, transcript.KindUnrecorded:
		st.lastRoller = a.Side
	case transcript.KindResign:
		a.Level = 1 + c.sym(resignFreq, a.Level-1)
		st.after(a.Kind, a.Side)
	default:
		st.after(a.Kind, a.Side)
	}
}

func (s *state) played(side int, after rel) {
	s.board, s.own, s.lastRoller = after, side, side
	if after.my.count() == 0 {
		s.newGame()
	}
}

func (s *state) after(k transcript.Kind, side int) {
	switch k {
	case transcript.KindDouble:
		s.doubler = side
	case transcript.KindPass, transcript.KindResign, transcript.KindForfeit:
		s.newGame()
	}
}

func pairIndex(d [2]int) int {
	a, b := max(d[0], d[1]), min(d[0], d[1])
	for i, p := range dicePairs {
		if p == [2]int{a, b} {
			return i
		}
	}
	return 0
}

// rawRel spells a board out, five bits a point.
func rawRel(c coder, r rel) rel {
	for i := range 26 {
		r.my[i] = int8(raw(c, 5, uint64(r.my[i])))
	}
	for i := range 26 {
		r.opp[i] = int8(raw(c, 5, uint64(r.opp[i])))
	}
	return r
}

// codeSpelled writes every field of an Action the models do not predict.
func codeSpelled(c coder, a *transcript.Action) {
	a.Dice[0] = int(svar(c, int64(a.Dice[0])))
	a.Dice[1] = int(svar(c, int64(a.Dice[1])))
	n := count(c, len(a.Steps))
	steps := make([]domain.CheckerStep, n)
	for i := range n {
		var s domain.CheckerStep
		if !c.decoding() {
			s = a.Steps[i]
		}
		s.From = int(svar(c, int64(s.From)))
		s.To = int(svar(c, int64(s.To)))
		s.Hit = boolBit(c, s.Hit)
		steps[i] = s
	}
	if n > 0 {
		a.Steps = steps
	} else {
		a.Steps = nil
	}
	a.BoardAfter = optBoard(c, a.BoardAfter)
	a.BoardBefore = optBoard(c, a.BoardBefore)
	a.Level = int(svar(c, int64(a.Level)))
}

func optBoard(c coder, b *domain.Board) *domain.Board {
	if !boolBit(c, b != nil) {
		return nil
	}
	var x domain.Board
	if b != nil {
		x = *b
	}
	for i := range x.Points {
		p := &x.Points[i]
		col := 3
		switch p.Color {
		case domain.None:
			col = 0
		case domain.Black:
			col = 1
		case domain.White:
			col = 2
		}
		switch c.sym([]uint64{1, 1, 1, 1}, col) {
		case 0:
			p.Color = domain.None
		case 1:
			p.Color = domain.Black
		case 2:
			p.Color = domain.White
		default:
			p.Color = int(svar(c, int64(p.Color)))
		}
		p.Checkers = int(svar(c, int64(p.Checkers)))
	}
	x.Bearoff[0] = int(svar(c, int64(x.Bearoff[0])))
	x.Bearoff[1] = int(svar(c, int64(x.Bearoff[1])))
	return &x
}

func codeTimes(c coder, md *models, a *transcript.Action) {
	for i, p := range []**int64{&a.DecisionMS, &a.CubeDecisionMS, &a.RollTickMS, &a.TickMS} {
		if md.times[i].code(c, *p != nil) {
			var v int64
			if *p != nil {
				v = **p
			}
			v = svar(c, v)
			*p = &v
		} else {
			*p = nil
		}
	}
}
