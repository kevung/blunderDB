package arkqr

import (
	"math"
	"math/bits"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// The models format version 2 adds. Version 1 spends most of a typical
// archive on what is not the play: header strings at eight bits a byte, a
// program name spelled in full, ratings as raw floats, and timecodes written
// as absolute values although each follows from the one before. Like those
// of version 1, every number below is part of the format.

// adaptiveFreqs is a symbol whose law follows the symbols already coded
// with it, from a prior.
type adaptiveFreqs struct {
	f   []uint64
	tot uint64
	inc uint64
}

func newAdaptiveFreqs(prior []uint64, inc uint64) *adaptiveFreqs {
	a := &adaptiveFreqs{f: append([]uint64(nil), prior...), inc: inc}
	for _, v := range prior {
		a.tot += v
	}
	return a
}

func (a *adaptiveFreqs) code(c coder, v int) int {
	v = c.sym(a.f, v)
	a.f[v] += a.inc
	a.tot += a.inc
	if a.tot > 1<<16 {
		a.tot = 0
		for i := range a.f {
			a.f[i] = (a.f[i] + 1) / 2
			a.tot += a.f[i]
		}
	}
	return v
}

// gamma codes an unsigned integer as its bit length, under an adaptive law,
// then its bits below the leading one. Every uint64 is coded.
type gamma struct{ n *adaptiveFreqs }

// newGamma favours the lengths up to short.
func newGamma(short int) gamma {
	var p [65]uint64
	for i := range p {
		p[i] = 1
		if i <= short {
			p[i] = 8
		}
	}
	return gamma{newAdaptiveFreqs(p[:], 24)}
}

func (g gamma) code(c coder, v uint64) uint64 {
	n := g.n.code(c, bits.Len64(v))
	if n <= 1 {
		return uint64(n)
	}
	return 1<<(n-1) | raw(c, n-1, v)
}

func (g gamma) signed(c coder, v int64) int64 {
	z := g.code(c, uint64(v<<1^(v>>63)))
	return int64(z>>1) ^ -int64(z&1)
}

// textModel codes the strings of a match: a string already written is named
// by its rank; any other is its length and its bytes, each byte under an
// adaptive law chosen by the class of the byte before it.
type textModel struct {
	seen   []string
	repeat *adaptive
	length gamma
	byCtx  [textContexts]*adaptiveFreqs
}

const textContexts = 5

func newTextModel() *textModel {
	t := &textModel{repeat: newAdaptive(4, 1), length: newGamma(6)}
	for i := range t.byCtx {
		t.byCtx[i] = newAdaptiveFreqs(textPriors[i][:], textAdapt)
	}
	return t
}

// textAdapt is the weight a coded byte adds to its context.
const textAdapt = 32

// Byte classes of the text prior.
const (
	clLower = iota
	clUpper
	clDigit
	clSpace
	clPunct
	clLead
	clCont
	clOther
	nClasses
)

func byteClass(b byte) int {
	switch {
	case b >= 'a' && b <= 'z':
		return clLower
	case b >= 'A' && b <= 'Z':
		return clUpper
	case b >= '0' && b <= '9':
		return clDigit
	case b == ' ':
		return clSpace
	case strings.IndexByte("-_.,'/:()#", b) >= 0:
		return clPunct
	case b >= 0xC0:
		return clLead
	case b >= 0x80:
		return clCont
	}
	return clOther
}

// classWeight[context][class] is how often a byte of the class follows one of
// the context (per thousand), in names, places and file names.
var classWeight = [textContexts][nClasses]uint64{
	{250, 400, 150, 50, 50, 30, 1, 2}, // start, after a separator
	{800, 10, 10, 100, 40, 30, 30, 2}, // after a lower-case letter
	{600, 250, 20, 50, 40, 20, 1, 2},  // after a capital
	{50, 30, 600, 100, 200, 5, 1, 2},  // after a digit
	{2, 2, 2, 2, 2, 2, 1000, 1},       // after a UTF-8 lead byte
}

// letterWeight is the frequency of each letter in French and English text,
// per thousand.
var letterWeight = [26]uint64{
	80, 12, 32, 35, 150, 12, 15, 25, 75, 5, 6, 50, 28, // a-m
	70, 60, 25, 4, 65, 70, 65, 45, 12, 5, 4, 8, 3, // n-z
}

// textPriors[context][byte] is the law of a byte before any is seen: the
// weight of its class shared among its members by their own frequency.
var textPriors = func() (p [textContexts][256]uint64) {
	var classTotal [nClasses]uint64
	member := func(b byte) uint64 {
		switch {
		case b >= 'a' && b <= 'z':
			return letterWeight[b-'a']
		case b >= 'A' && b <= 'Z':
			return letterWeight[b-'A']
		case b == 0xC3:
			return 40 // the lead byte of é, è, à, ç, ü
		case b >= 0xC0:
			return 1
		case b >= 0x80 && b <= 0xBF:
			return 4
		}
		return 10
	}
	for b := range 256 {
		classTotal[byteClass(byte(b))] += member(byte(b))
	}
	for ctx := range p {
		for b := range 256 {
			cl := byteClass(byte(b))
			p[ctx][b] = max(1, 4096*classWeight[ctx][cl]*member(byte(b))/(1000*classTotal[cl]))
		}
	}
	return p
}()

// textContext is the class of the byte before: none or a separator, a
// lower-case letter or a byte inside a UTF-8 sequence, a capital, a digit, the
// lead byte of a UTF-8 sequence.
func textContext(prev int) int {
	switch {
	case prev < 0:
		return 0
	case prev >= 'a' && prev <= 'z', prev >= 0x80 && prev < 0xC0:
		return 1
	case prev >= 'A' && prev <= 'Z':
		return 2
	case prev >= '0' && prev <= '9':
		return 3
	case prev >= 0xC0:
		return 4
	}
	return 0
}

func (t *textModel) code(c coder, s string) string {
	d, decoding := c.(*decoder)
	if len(t.seen) > 0 {
		at := -1
		if !decoding {
			for i, x := range t.seen {
				if x == s {
					at = i
					break
				}
			}
		}
		if t.repeat.code(c, at >= 0) {
			at = c.sym(uniform(len(t.seen)), at)
			return t.seen[at]
		}
	}
	n := t.length.code(c, uint64(len(s)))
	if decoding && (n > maxLen || d.failed) {
		d.failed = true
		return ""
	}
	b := make([]byte, n)
	prev := -1
	for i := range b {
		v := 0
		if !decoding {
			v = int(s[i])
		}
		b[i] = byte(t.byCtx[textContext(prev)].code(c, v))
		prev = int(b[i])
	}
	s = string(b)
	if s != "" {
		t.seen = append(t.seen, s)
	}
	return s
}

// uniform is the flat law over n symbols.
func uniform(n int) []uint64 {
	f := make([]uint64, n)
	for i := range f {
		f[i] = 1
	}
	return f
}

// enginePrefixes are the beginnings of the program names the importers write.
var enginePrefixes = []string{"", "eXtreme Gammon, file format ", "eXtreme Gammon ", "GNU Backgammon ", "BGBlitz "}

var enginePrefixFreq = []uint64{4, 8, 2, 2, 1}

// engineName codes a program name as the longest known beginning and the rest.
func engineName(c coder, t *textModel, s string) string {
	k := 0
	for i, p := range enginePrefixes {
		if strings.HasPrefix(s, p) && len(p) > len(enginePrefixes[k]) {
			k = i
		}
	}
	k = c.sym(enginePrefixFreq, k)
	var rest string
	if !c.decoding() {
		rest = s[len(enginePrefixes[k]):]
	}
	return enginePrefixes[k] + t.code(c, rest)
}

// rating codes a rating; a whole one is its difference from base.
func rating(c coder, md *models, f float64, base int64) float64 {
	whole := f == math.Trunc(f) && math.Abs(f) < 1<<40 && math.Float64bits(f) != 1<<63
	if flag(c, 1, 4, whole) {
		return float64(base + md.small.signed(c, int64(f)-base))
	}
	return math.Float64frombits(raw(c, 64, math.Float64bits(f)))
}

// timeline predicts the timecodes of an Action from the ones before: a roll
// is read where the last Action ended, after the cube decision; the play ends
// after the decision.
type timeline struct {
	end   int64
	known bool
	had   [4]bool
}

// timeKind is the context of a duration's presence: a checker play, another
// roll, a cube action.
func timeKind(k transcript.Kind) int {
	switch {
	case k == transcript.KindChecker:
		return 0
	case isRoll(k):
		return 1
	}
	return 2
}

func codeTimes2(c coder, md *models, a *transcript.Action) {
	tl := &md.timeline
	fields := [4]**int64{&a.DecisionMS, &a.CubeDecisionMS, &a.RollTickMS, &a.TickMS}
	kc := timeKind(a.Kind)
	for i, p := range fields {
		ctx := 0
		if tl.had[i] {
			ctx = 1
		}
		if !md.timesAt[i][kc][ctx].code(c, *p != nil) {
			*p = nil
			tl.had[i] = false
			continue
		}
		tl.had[i] = true
		var v int64
		if *p != nil {
			v = **p
		}
		pred := int64(0)
		switch i {
		case 2:
			if tl.known {
				pred = tl.end
				if a.CubeDecisionMS != nil {
					pred += *a.CubeDecisionMS
				}
			}
		case 3:
			switch {
			case a.RollTickMS != nil && a.DecisionMS != nil:
				pred = *a.RollTickMS + *a.DecisionMS
			case tl.known:
				pred = tl.end
				if a.DecisionMS != nil {
					pred += *a.DecisionMS
				}
			}
		}
		v = pred + md.timeVal[i].signed(c, v-pred)
		*p = &v
	}
	switch {
	case a.TickMS != nil:
		tl.end, tl.known = *a.TickMS, true
	case a.RollTickMS != nil && a.DecisionMS != nil:
		tl.end, tl.known = *a.RollTickMS+*a.DecisionMS, true
	case a.RollTickMS != nil:
		tl.end, tl.known = *a.RollTickMS, true
	}
}

// diceLaw is the law of the roll of a checker play (dance false) or of a
// dance from board r: a roll that leaves no play is 32 times less likely than
// another for a play, 32 times more for a dance. Neither is excluded, so a
// transcription that contradicts the board still codes.
func diceLaw(r rel, dance bool) []uint64 {
	var blocked [7]bool
	for d := 1; d <= 6; d++ {
		blocked[d] = !r.canMove(d)
	}
	law := make([]uint64, len(dicePairs))
	for i, p := range dicePairs {
		w := uint64(1)
		if noPlay := blocked[p[0]] && blocked[p[1]]; noPlay == dance {
			w = 32
		}
		law[i] = w * diceFreq[i]
	}
	return law
}

// kindPrior is the version 2 law of the Action after token prev: version 1's
// table, with a dance made likely when the roller is on the bar and all but
// excluded when it is not.
func kindPrior(bar bool, prev int) []uint64 {
	f := append([]uint64(nil), kindTable[prev][:]...)
	const dance = 1
	if f[0] >= 3000 {
		if bar {
			f[dance] = 1500
		} else {
			f[dance] = 2
		}
	}
	return f
}

// codeDate codes a date as its day counted from the first day of the year 2000 (UTC), its second in the day and
// its nanosecond: a match dated without a time costs its day alone.
func codeDate(c coder, md *models, t time.Time) time.Time {
	const epoch = 946_684_800
	sec := t.Unix() - epoch
	day, in := sec/86_400, sec%86_400
	if in < 0 {
		day, in = day-1, in+86_400
	}
	day = md.small.signed(c, day-9_000) + 9_000
	in = int64(md.timeVal[0].code(c, uint64(in)))
	nsec := int64(md.timeVal[0].code(c, uint64(t.Nanosecond())))
	return time.Unix(epoch+day*86_400+in, nsec).UTC()
}
