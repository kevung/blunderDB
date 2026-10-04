package engine

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// This file is the binary payload of analysis.data (ADR-0070): the bytes a
// PositionAnalysis becomes before zstd compresses them. The envelope
// (header, compression, format detection) lives in analysiscodec.go.
//
// The payload carries every field of the struct — nothing is rebuilt from a
// column — so a decode gives back exactly what was encoded, floats bit for
// bit. It is smaller than JSON because it drops what JSON spells out on every
// row: field names, the decimal text of numbers, repeated labels and the
// RFC 3339 text of the timestamps.
//
// Primitives:
//   - uvarint / zigzag varint (encoding/binary) for integers;
//   - a string is one uvarint u: odd → table[u>>1], even → u>>1 bytes follow
//     and join the table. The table starts as binarySeedStrings, so the
//     engine labels, depths and cube verdicts cost one byte;
//   - a float is one uvarint u whose low two bits pick the mode: 0 → k/100,
//     1 → k/1000 (k the zigzag of u>>2; u = 1 alone is -0), 2 → the field's
//     derivation (the value a sibling field implies), 3 → the raw IEEE-754
//     bits in the next 8 bytes.
//     The encoder takes a mode only when it reproduces the exact bits;
//   - a slice is a uvarint c: 0 → nil, n+1 → n elements, so nil and empty
//     survive apart;
//   - a time is zigzag seconds, uvarint nanoseconds and zigzag UTC offset in
//     seconds, relative to the analysis's CreationDate for every other time.

// binarySeedStrings is the string table every version-1 payload starts with.
// It is part of the format: changing it (even appending) is a new version.
var binarySeedStrings = [...]string{
	"",
	"CheckerMove", "DoublingCube",
	"XG", "GNUbg", "gnubg", "BGBlitz", "gammonNet", "HedgeHog",
	"0-ply", "1-ply", "2-ply", "3-ply", "4-ply", "5-ply", "6-ply", "7-ply",
	"XG Roller", "XG Roller+", "XG Roller++", "Book", "Rollout",
	"No Double", "Double, Take", "Double, Pass",
	"Too good to double, pass", "Too good to double, take",
	"No Redouble", "Redouble, Take", "Redouble, Pass",
	"Double/Take", "Double/Pass", "Double No", "Take", "Pass", "Drop",
	domain.RolloutKindMoves, domain.RolloutKindCube,
}

// binEpoch is the base of CreationDate; every other time is relative to it.
var binEpoch = time.Unix(0, 0).UTC()

// maxBinaryElements bounds how many slice elements and table strings one
// payload may declare, so a crafted payload cannot turn a few bytes per
// element into gigabytes of structs. A real analysis has a few dozen.
const maxBinaryElements = 1 << 16

// maxTimeOffset bounds a stored UTC offset; RFC 3339 cannot spell more.
const maxTimeOffset = 100 * 3600

const (
	binFlagCube    = 1 << 0
	binFlagChecker = 1 << 1
	binFlagsKnown  = binFlagCube | binFlagChecker

	binBoolCubefulBias  = 1 << 0
	binBoolExactBearoff = 1 << 1
	binBoolsKnown       = binBoolCubefulBias | binBoolExactBearoff

	floatModeHundredth  = 0
	floatModeThousandth = 1
	floatModeDerived    = 2
	floatModeRaw        = 3
)

// ErrCorruptAnalysis reports a binary payload that does not decode.
var ErrCorruptAnalysis = errors.New("corrupt binary analysis payload")

// ---------------------------------------------------------------- encoding

type binWriter struct {
	buf   []byte
	table map[string]uint64
}

func newBinWriter() *binWriter {
	w := &binWriter{buf: make([]byte, 0, 512), table: make(map[string]uint64, len(binarySeedStrings)+16)}
	for i, s := range binarySeedStrings {
		w.table[s] = uint64(i)
	}
	return w
}

func (w *binWriter) uvarint(u uint64) { w.buf = binary.AppendUvarint(w.buf, u) }
func (w *binWriter) varint(v int64)   { w.buf = binary.AppendVarint(w.buf, v) }

func (w *binWriter) str(s string) {
	if i, ok := w.table[s]; ok {
		w.uvarint(i<<1 | 1)
		return
	}
	w.table[s] = uint64(len(w.table))
	w.uvarint(uint64(len(s)) << 1)
	w.buf = append(w.buf, s...)
}

// count writes a slice header: 0 for nil, n+1 for n elements.
func (w *binWriter) count(isNil bool, n int) {
	if isNil {
		w.uvarint(0)
		return
	}
	w.uvarint(uint64(n) + 1)
}

// scaled reports k such that float64(k)/scale has exactly v's bits.
func scaled(v, scale float64) (int64, bool) {
	x := v * scale
	if math.IsNaN(x) || math.Abs(x) > 1<<53 {
		return 0, false
	}
	k := int64(math.Round(x))
	if math.Float64bits(float64(k)/scale) != math.Float64bits(v) {
		return 0, false
	}
	return k, true
}

var negativeZero = math.Float64bits(math.Copysign(0, -1))

func zigzag(k int64) uint64 { return uint64(k<<1) ^ uint64(k>>63) }

// float writes v; derived, when non-nil, is the value the decoder will
// compute for this field from fields it has already read.
func (w *binWriter) float(v float64, derived *float64) {
	if derived != nil && math.Float64bits(*derived) == math.Float64bits(v) {
		w.uvarint(floatModeDerived)
		return
	}
	if math.Float64bits(v) == negativeZero {
		// k/1000 with k = 0 never comes out of the loop below (k/100 takes
		// zero first), so that code spells the -0 rounding leaves behind.
		w.uvarint(floatModeThousandth)
		return
	}
	if k, ok := scaled(v, 100); ok {
		w.uvarint(zigzag(k)<<2 | floatModeHundredth)
		return
	}
	if k, ok := scaled(v, 1000); ok {
		w.uvarint(zigzag(k)<<2 | floatModeThousandth)
		return
	}
	w.uvarint(floatModeRaw)
	w.buf = binary.LittleEndian.AppendUint64(w.buf, math.Float64bits(v))
}

func (w *binWriter) time(t time.Time, base time.Time) {
	_, off := t.Zone()
	_, baseOff := base.Zone()
	w.varint(t.Unix() - base.Unix())
	w.uvarint(uint64(t.Nanosecond()))
	w.varint(int64(off - baseOff))
}

// opponentWin is the derivation of an opponent's win chance (percent).
func opponentWin(playerWin float64) *float64 {
	v := RoundToHundredthPercent(100 - playerWin)
	return &v
}

func (w *binWriter) cube(d *domain.DoublingCubeAnalysis) {
	w.str(d.AnalysisDepth)
	w.str(d.AnalysisEngine)
	w.float(d.PlayerWinChances, nil)
	w.float(d.PlayerGammonChances, nil)
	w.float(d.PlayerBackgammonChances, nil)
	w.float(d.OpponentWinChances, opponentWin(d.PlayerWinChances))
	w.float(d.OpponentGammonChances, nil)
	w.float(d.OpponentBackgammonChances, nil)
	w.float(d.CubelessNoDoubleEquity, nil)
	w.float(d.CubelessDoubleEquity, nil)
	w.float(d.CubefulNoDoubleEquity, nil)
	w.float(d.CubefulNoDoubleError, nil)
	w.float(d.CubefulDoubleTakeEquity, nil)
	w.float(d.CubefulDoubleTakeError, nil)
	w.float(d.CubefulDoublePassEquity, nil)
	w.float(d.CubefulDoublePassError, nil)
	w.str(d.BestCubeAction)
	w.float(d.WrongPassPercentage, nil)
	w.float(d.WrongTakePercentage, nil)
}

// equityErrorFrom is the derivation of a move's equity error: its distance
// to the first (best) move, rounded as import rounds it.
func equityErrorFrom(best, equity float64) *float64 {
	v := RoundToMillipoint(best - equity)
	return &v
}

func (w *binWriter) move(i int, moves []domain.CheckerMove) {
	m := &moves[i]
	w.varint(int64(m.Index) - int64(i))
	w.str(m.AnalysisDepth)
	w.str(m.AnalysisEngine)
	w.str(m.Move)
	w.float(m.Equity, nil)
	if m.EquityError == nil {
		w.uvarint(0)
	} else {
		w.uvarint(1)
		w.float(*m.EquityError, equityErrorFrom(moves[0].Equity, m.Equity))
	}
	w.float(m.PlayerWinChance, nil)
	w.float(m.PlayerGammonChance, nil)
	w.float(m.PlayerBackgammonChance, nil)
	w.float(m.OpponentWinChance, opponentWin(m.PlayerWinChance))
	w.float(m.OpponentGammonChance, nil)
	w.float(m.OpponentBackgammonChance, nil)
}

func (w *binWriter) rollout(r *domain.RolloutAnalysis, base time.Time) {
	w.str(r.AnalysisEngine)
	w.str(r.AnalysisDepth)
	w.str(r.Signature)
	w.str(r.Kind)
	s := &r.Settings
	w.varint(int64(s.Truncation))
	w.varint(int64(s.MinGames))
	w.varint(int64(s.MaxGames))
	w.float(s.JSDLimit, nil)
	w.varint(int64(s.Ply))
	w.varint(int64(s.Candidates))
	w.uvarint(s.Seed)
	w.varint(int64(r.Games))
	w.str(r.Stop)
	var bools uint64
	if r.CubefulBias {
		bools |= binBoolCubefulBias
	}
	if r.ExactBearoff {
		bools |= binBoolExactBearoff
	}
	w.uvarint(bools)
	w.count(r.Candidates == nil, len(r.Candidates))
	for i := range r.Candidates {
		c := &r.Candidates[i]
		w.str(c.Move)
		w.float(c.Equity, nil)
		w.float(c.StdErr, nil)
		w.float(c.CI95, nil)
		w.varint(int64(c.Games))
		w.float(c.JSD, nil)
		w.float(c.PlayerWinChance, nil)
		w.float(c.PlayerGammonChance, nil)
		w.float(c.PlayerBackgammonChance, nil)
		w.float(c.OpponentWinChance, opponentWin(c.PlayerWinChance))
		w.float(c.OpponentGammonChance, nil)
		w.float(c.OpponentBackgammonChance, nil)
	}
	w.str(r.BestCubeAction)
	w.float(r.JSDDouble, nil)
	w.float(r.JSDTake, nil)
	w.time(r.Date, base)
}

// MarshalAnalysisBinary encodes a as the version-1 binary payload, before
// compression. Exported for the dictionary trainer and the tests.
func MarshalAnalysisBinary(a *domain.PositionAnalysis) []byte {
	w := newBinWriter()
	var flags uint64
	if a.DoublingCubeAnalysis != nil {
		flags |= binFlagCube
	}
	if a.CheckerAnalysis != nil {
		flags |= binFlagChecker
	}
	w.uvarint(flags)
	w.varint(int64(a.PositionID))
	w.str(a.XGID)
	w.str(a.Player1)
	w.str(a.Player2)
	w.str(a.AnalysisType)
	w.str(a.AnalysisEngineVersion)
	w.time(a.CreationDate, binEpoch)
	w.time(a.LastModifiedDate, a.CreationDate)
	if a.DoublingCubeAnalysis != nil {
		w.cube(a.DoublingCubeAnalysis)
	}
	w.count(a.AllCubeAnalyses == nil, len(a.AllCubeAnalyses))
	for i := range a.AllCubeAnalyses {
		w.cube(&a.AllCubeAnalyses[i])
	}
	if ca := a.CheckerAnalysis; ca != nil {
		w.count(ca.Moves == nil, len(ca.Moves))
		for i := range ca.Moves {
			w.move(i, ca.Moves)
		}
	}
	w.str(a.PlayedMove)
	w.str(a.PlayedCubeAction)
	w.count(a.PlayedMoves == nil, len(a.PlayedMoves))
	for _, s := range a.PlayedMoves {
		w.str(s)
	}
	w.count(a.PlayedCubeActions == nil, len(a.PlayedCubeActions))
	for _, s := range a.PlayedCubeActions {
		w.str(s)
	}
	w.count(a.Rollouts == nil, len(a.Rollouts))
	for i := range a.Rollouts {
		w.rollout(&a.Rollouts[i], a.CreationDate)
	}
	return w.buf
}

// ---------------------------------------------------------------- decoding

type binReader struct {
	buf      []byte
	pos      int
	table    []string
	elements int // slice elements and table strings declared so far
	err      error
}

func (r *binReader) fail(what string) {
	if r.err == nil {
		r.err = fmt.Errorf("%w: %s at byte %d", ErrCorruptAnalysis, what, r.pos)
	}
}

func (r *binReader) uvarint() uint64 {
	if r.err != nil {
		return 0
	}
	u, n := binary.Uvarint(r.buf[r.pos:])
	if n <= 0 {
		r.fail("bad varint")
		return 0
	}
	r.pos += n
	return u
}

func (r *binReader) varint() int64 {
	if r.err != nil {
		return 0
	}
	v, n := binary.Varint(r.buf[r.pos:])
	if n <= 0 {
		r.fail("bad varint")
		return 0
	}
	r.pos += n
	return v
}

func (r *binReader) int() int {
	v := r.varint()
	if int64(int(v)) != v {
		r.fail("integer out of range")
		return 0
	}
	return int(v)
}

func (r *binReader) charge(n uint64) bool {
	if n > maxBinaryElements || uint64(r.elements)+n > maxBinaryElements {
		r.fail("too many elements")
		return false
	}
	r.elements += int(n)
	return true
}

func (r *binReader) str() string {
	u := r.uvarint()
	if r.err != nil {
		return ""
	}
	if u&1 == 1 {
		i := u >> 1
		if i >= uint64(len(r.table)) {
			r.fail("string reference out of range")
			return ""
		}
		return r.table[i]
	}
	n := u >> 1
	if n > uint64(len(r.buf)-r.pos) {
		r.fail("string past end")
		return ""
	}
	if !r.charge(1) {
		return ""
	}
	s := string(r.buf[r.pos : r.pos+int(n)])
	r.pos += int(n)
	r.table = append(r.table, s)
	return s
}

// count reads a slice header and returns (n, isNil). Every element takes at
// least one byte, so n is checked against what is left before anything is
// allocated.
func (r *binReader) count() (int, bool) {
	c := r.uvarint()
	if r.err != nil || c == 0 {
		return 0, true
	}
	n := c - 1
	if n > uint64(len(r.buf)-r.pos) {
		r.fail("slice longer than payload")
		return 0, true
	}
	if !r.charge(n) {
		return 0, true
	}
	return int(n), false
}

func unzigzag(u uint64) int64 { return int64(u>>1) ^ -int64(u&1) }

func (r *binReader) float(derived func() float64) float64 {
	u := r.uvarint()
	if r.err != nil {
		return 0
	}
	switch u & 3 {
	case floatModeHundredth:
		return float64(unzigzag(u>>2)) / 100
	case floatModeThousandth:
		if u == floatModeThousandth {
			return math.Copysign(0, -1)
		}
		return float64(unzigzag(u>>2)) / 1000
	case floatModeDerived:
		if derived == nil || u != floatModeDerived {
			r.fail("derived float on a field without derivation")
			return 0
		}
		return derived()
	default:
		if u != floatModeRaw || len(r.buf)-r.pos < 8 {
			r.fail("bad raw float")
			return 0
		}
		v := math.Float64frombits(binary.LittleEndian.Uint64(r.buf[r.pos:]))
		r.pos += 8
		return v
	}
}

// time rebuilds a time the way encoding/json's time.Parse does from its
// RFC 3339 text, so a blob decodes to the same value whichever format holds
// it: offset 0 is UTC, an offset the local zone has at that instant is
// Local, any other a fixed zone.
func (r *binReader) time(base time.Time) time.Time {
	_, baseOff := base.Zone()
	sec := base.Unix() + r.varint()
	nsec := r.uvarint()
	off := int64(baseOff) + r.varint()
	if r.err != nil {
		return time.Time{}
	}
	if nsec >= 1e9 || off < -maxTimeOffset || off > maxTimeOffset {
		r.fail("bad time")
		return time.Time{}
	}
	t := time.Unix(sec, int64(nsec))
	if off == 0 {
		return t.UTC()
	}
	if _, localOff := t.Zone(); int64(localOff) == off {
		return t
	}
	return t.In(time.FixedZone("", int(off)))
}

func (r *binReader) cube(d *domain.DoublingCubeAnalysis) {
	d.AnalysisDepth = r.str()
	d.AnalysisEngine = r.str()
	d.PlayerWinChances = r.float(nil)
	d.PlayerGammonChances = r.float(nil)
	d.PlayerBackgammonChances = r.float(nil)
	d.OpponentWinChances = r.float(func() float64 { return *opponentWin(d.PlayerWinChances) })
	d.OpponentGammonChances = r.float(nil)
	d.OpponentBackgammonChances = r.float(nil)
	d.CubelessNoDoubleEquity = r.float(nil)
	d.CubelessDoubleEquity = r.float(nil)
	d.CubefulNoDoubleEquity = r.float(nil)
	d.CubefulNoDoubleError = r.float(nil)
	d.CubefulDoubleTakeEquity = r.float(nil)
	d.CubefulDoubleTakeError = r.float(nil)
	d.CubefulDoublePassEquity = r.float(nil)
	d.CubefulDoublePassError = r.float(nil)
	d.BestCubeAction = r.str()
	d.WrongPassPercentage = r.float(nil)
	d.WrongTakePercentage = r.float(nil)
}

// move decodes moves[i]; the equity-error derivation reads moves[0], whose
// equity is decoded before any equity error (its own included).
func (r *binReader) move(i int, moves []domain.CheckerMove) {
	m := &moves[i]
	m.Index = int(int64(i) + r.varint())
	m.AnalysisDepth = r.str()
	m.AnalysisEngine = r.str()
	m.Move = r.str()
	m.Equity = r.float(nil)
	switch r.uvarint() {
	case 0:
	case 1:
		e := r.float(func() float64 { return *equityErrorFrom(moves[0].Equity, m.Equity) })
		m.EquityError = &e
	default:
		r.fail("bad equity error presence")
	}
	m.PlayerWinChance = r.float(nil)
	m.PlayerGammonChance = r.float(nil)
	m.PlayerBackgammonChance = r.float(nil)
	m.OpponentWinChance = r.float(func() float64 { return *opponentWin(m.PlayerWinChance) })
	m.OpponentGammonChance = r.float(nil)
	m.OpponentBackgammonChance = r.float(nil)
}

func (r *binReader) rollout(ro *domain.RolloutAnalysis, base time.Time) {
	ro.AnalysisEngine = r.str()
	ro.AnalysisDepth = r.str()
	ro.Signature = r.str()
	ro.Kind = r.str()
	s := &ro.Settings
	s.Truncation = r.int()
	s.MinGames = r.int()
	s.MaxGames = r.int()
	s.JSDLimit = r.float(nil)
	s.Ply = r.int()
	s.Candidates = r.int()
	s.Seed = r.uvarint()
	ro.Games = r.int()
	ro.Stop = r.str()
	bools := r.uvarint()
	if bools&^binBoolsKnown != 0 {
		r.fail("unknown rollout flags")
	}
	ro.CubefulBias = bools&binBoolCubefulBias != 0
	ro.ExactBearoff = bools&binBoolExactBearoff != 0
	if n, isNil := r.count(); !isNil {
		ro.Candidates = make([]domain.RolloutCandidate, n)
		for i := range ro.Candidates {
			c := &ro.Candidates[i]
			c.Move = r.str()
			c.Equity = r.float(nil)
			c.StdErr = r.float(nil)
			c.CI95 = r.float(nil)
			c.Games = r.int()
			c.JSD = r.float(nil)
			c.PlayerWinChance = r.float(nil)
			c.PlayerGammonChance = r.float(nil)
			c.PlayerBackgammonChance = r.float(nil)
			c.OpponentWinChance = r.float(func() float64 { return *opponentWin(c.PlayerWinChance) })
			c.OpponentGammonChance = r.float(nil)
			c.OpponentBackgammonChance = r.float(nil)
			if r.err != nil {
				return
			}
		}
	}
	ro.BestCubeAction = r.str()
	ro.JSDDouble = r.float(nil)
	ro.JSDTake = r.float(nil)
	ro.Date = r.time(base)
}

func (r *binReader) strings() []string {
	n, isNil := r.count()
	if isNil {
		return nil
	}
	out := make([]string, n)
	for i := range out {
		out[i] = r.str()
		if r.err != nil {
			return nil
		}
	}
	return out
}

// UnmarshalAnalysisBinary decodes a version-1 binary payload. Hostile input
// yields an error wrapping ErrCorruptAnalysis, never a panic, and allocates
// in proportion to the payload and maxBinaryElements.
func UnmarshalAnalysisBinary(payload []byte) (domain.PositionAnalysis, error) {
	var a domain.PositionAnalysis
	r := &binReader{buf: payload, table: make([]string, len(binarySeedStrings), len(binarySeedStrings)+16)}
	copy(r.table, binarySeedStrings[:])

	flags := r.uvarint()
	if flags&^binFlagsKnown != 0 {
		r.fail("unknown flags")
	}
	a.PositionID = r.int()
	a.XGID = r.str()
	a.Player1 = r.str()
	a.Player2 = r.str()
	a.AnalysisType = r.str()
	a.AnalysisEngineVersion = r.str()
	a.CreationDate = r.time(binEpoch)
	a.LastModifiedDate = r.time(a.CreationDate)
	if flags&binFlagCube != 0 && r.err == nil {
		a.DoublingCubeAnalysis = &domain.DoublingCubeAnalysis{}
		r.cube(a.DoublingCubeAnalysis)
	}
	if n, isNil := r.count(); !isNil {
		a.AllCubeAnalyses = make([]domain.DoublingCubeAnalysis, n)
		for i := range a.AllCubeAnalyses {
			r.cube(&a.AllCubeAnalyses[i])
			if r.err != nil {
				break
			}
		}
	}
	if flags&binFlagChecker != 0 && r.err == nil {
		ca := &domain.CheckerAnalysis{}
		if n, isNil := r.count(); !isNil {
			ca.Moves = make([]domain.CheckerMove, n)
			for i := range ca.Moves {
				r.move(i, ca.Moves)
				if r.err != nil {
					break
				}
			}
		}
		a.CheckerAnalysis = ca
	}
	a.PlayedMove = r.str()
	a.PlayedCubeAction = r.str()
	a.PlayedMoves = r.strings()
	a.PlayedCubeActions = r.strings()
	if n, isNil := r.count(); !isNil {
		a.Rollouts = make([]domain.RolloutAnalysis, n)
		for i := range a.Rollouts {
			r.rollout(&a.Rollouts[i], a.CreationDate)
			if r.err != nil {
				break
			}
		}
	}
	if r.err == nil && r.pos != len(r.buf) {
		r.fail("trailing bytes")
	}
	if r.err != nil {
		return domain.PositionAnalysis{}, r.err
	}
	return a, nil
}
