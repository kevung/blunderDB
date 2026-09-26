package engine

import (
	"bytes"
	"compress/zlib"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/klauspost/compress/zstd"
)

// This file holds the pure analysis-encoding helpers shared by the Database
// wrapper and both Storage backends: compression of the analysis JSON blob,
// derivation of the denormalised scalar columns, and float rounding for
// compact storage. They perform no database I/O.
//
// # Blob codec and format compatibility (ADR-0030)
//
// analysis.data holds one of three formats, told apart by content and never
// by a schema version: raw JSON (first byte '{'), zlib level 9 (CMF/FLG
// header), or zstd level 19 with the embedded dictionary (magic 0x28 0xB5
// 0x2F 0xFD). The signatures cannot collide, so any database, however old,
// still decodes. Each zstd frame carries its Dictionary_ID, so a second
// dictionary later needs no row migration — only registering its bytes in
// zstdDecoder's dict set.
//
// Every write is zstd. zlib/raw rows are read forever and upgraded
// opportunistically (RecompressAnalysisData on native-.db import, and
// sqlite.Storage.Vacuum) — never in a schema migration: a schema bump is for
// DDL, not for the bytes inside an unchanged BLOB column.
//
// analysis_dict.bin is trained offline by cmd/train-analysis-dict; at runtime
// the pure-Go klauspost decoder reads it, no zstd binary or cgo needed.

//go:embed analysis_dict.bin
var analysisZstdDict []byte

// zstdMagic is the four-byte signature every zstd frame starts with (RFC 8878
// §3.1.1). Checked first so the common case skips a failed zlib-header parse.
var zstdMagic = []byte{0x28, 0xB5, 0x2F, 0xFD}

// zstdEncoder and zstdDecoder are shared: klauspost documents EncodeAll and
// DecodeAll as concurrency-safe, and one instance per call is the memory
// blow-up upstream warns against. Internal concurrency is pinned to 1 —
// bounded memory over single-call latency, right for few-kilobyte blobs.
var (
	zstdEncoder *zstd.Encoder
	zstdDecoder *zstd.Decoder
)

func init() {
	enc, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(19)),
		zstd.WithEncoderDict(analysisZstdDict),
		zstd.WithEncoderConcurrency(1),
	)
	if err != nil {
		// The dictionary is a build-time asset: failure means a broken
		// binary. Fail loudly rather than fall back to an un-dictionaried codec.
		panic(fmt.Sprintf("engine: zstd encoder init: %v", err))
	}
	zstdEncoder = enc

	dec, err := zstd.NewReader(nil,
		zstd.WithDecoderDicts(analysisZstdDict),
		// Same cap as the zlib path, far below klauspost's 64 GiB default:
		// a crafted frame claiming gigabytes is refused before decoding.
		zstd.WithDecoderMaxMemory(MaxAnalysisBytes),
		zstd.WithDecoderConcurrency(1),
		zstd.WithDecoderLowmem(true),
	)
	if err != nil {
		panic(fmt.Sprintf("engine: zstd decoder init: %v", err))
	}
	zstdDecoder = dec
}

// CompressAnalysisData compresses raw JSON bytes with zstd level 19 and the
// embedded dictionary — the only format ever written.
func CompressAnalysisData(jsonData []byte) ([]byte, error) {
	return zstdEncoder.EncodeAll(jsonData, nil), nil
}

// MaxAnalysisBytes bounds what one analysis blob may inflate to. A real one is
// a few kilobytes (a rollout well under a megabyte); the blob may come from a
// third-party database, so a crafted decompression bomb must be refused.
const MaxAnalysisBytes = 16 << 20

// ErrAnalysisTooLarge is returned when a compressed analysis inflates past
// MaxAnalysisBytes.
var ErrAnalysisTooLarge = fmt.Errorf("analysis blob inflates past %d bytes", MaxAnalysisBytes)

// isZstdFrame reports whether data starts with the zstd frame magic number.
func isZstdFrame(data []byte) bool {
	return len(data) >= len(zstdMagic) && bytes.Equal(data[:len(zstdMagic)], zstdMagic)
}

// DecompressAnalysisData detects the blob's format (raw JSON, zlib, zstd)
// from its content. Inflation past MaxAnalysisBytes is an error.
func DecompressAnalysisData(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	if data[0] == '{' {
		return data, nil
	}
	if isZstdFrame(data) {
		out, err := zstdDecoder.DecodeAll(data, nil)
		if err != nil {
			if errors.Is(err, zstd.ErrDecoderSizeExceeded) {
				return nil, ErrAnalysisTooLarge
			}
			return nil, err
		}
		return out, nil
	}
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return data, nil
	}
	defer r.Close()
	// One byte past the cap tells an oversized stream from one that is exactly
	// the cap.
	out, err := io.ReadAll(io.LimitReader(r, MaxAnalysisBytes+1))
	if err != nil {
		return nil, err
	}
	if len(out) > MaxAnalysisBytes {
		return nil, ErrAnalysisTooLarge
	}
	return out, nil
}

// NeedsRecompression reports whether data is raw JSON or legacy zlib rather
// than zstd — an allocation-free check so a full-table pass skips current rows
// without decompressing them.
func NeedsRecompression(data []byte) bool {
	return len(data) > 0 && !isZstdFrame(data)
}

// RecompressAnalysisData recompresses raw JSON or zlib data to zstd and
// returns zstd data unchanged. It is the opportunistic upgrade path (native-.db
// import, sqlite.Storage.Vacuum), so no migration step is needed.

func RecompressAnalysisData(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	if isZstdFrame(data) {
		return data, nil
	}
	jsonData, err := DecompressAnalysisData(data)
	if err != nil {
		return nil, err
	}
	return CompressAnalysisData(jsonData)
}

// EncodeAnalysisForStorage marshals a PositionAnalysis to JSON and compresses it.
func EncodeAnalysisForStorage(a *domain.PositionAnalysis) ([]byte, error) {
	jsonData, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	return CompressAnalysisData(jsonData)
}

// DecodeAnalysisFromStorage decompresses (if needed) and unmarshals analysis data.
func DecodeAnalysisFromStorage(data []byte) (domain.PositionAnalysis, error) {
	var a domain.PositionAnalysis
	jsonData, err := DecompressAnalysisData(data)
	if err != nil {
		return a, err
	}
	err = json.Unmarshal(jsonData, &a)
	return a, err
}

// DecodeAnalysesConcurrently decodes a batch of stored analyses across all
// cores: decoding is pure computation and the largest cost of reading a
// library once queries are batched.
//
// A payload that cannot be decoded is reported in failed under its position
// id and absent from decoded, so a caller can tell "no analysis" from "an
// analysis nobody can read" and decide which of the two it tolerates.
func DecodeAnalysesConcurrently(raw map[int64][]byte) (decoded map[int64]*domain.PositionAnalysis, failed map[int64]error) {
	decoded = make(map[int64]*domain.PositionAnalysis, len(raw))
	failed = make(map[int64]error)
	if len(raw) == 0 {
		return decoded, failed
	}

	type job struct {
		id   int64
		data []byte
	}
	jobs := make([]job, 0, len(raw))
	for id, data := range raw {
		jobs = append(jobs, job{id: id, data: data})
	}

	results := make([]*domain.PositionAnalysis, len(jobs))
	errs := make([]error, len(jobs))
	workers := min(runtime.NumCPU(), len(jobs))
	var wg sync.WaitGroup
	var next atomic.Int64
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(next.Add(1)) - 1
				if i >= len(jobs) {
					return
				}
				a, err := DecodeAnalysisFromStorage(jobs[i].data)
				if err != nil {
					errs[i] = err
					continue
				}
				results[i] = &a
			}
		}()
	}
	wg.Wait()

	for i, j := range jobs {
		if errs[i] != nil {
			failed[j.id] = errs[i]
			continue
		}
		decoded[j.id] = results[i]
	}
	return decoded, failed
}

// AnalysisColumns holds the derived scalar columns computed from a
// PositionAnalysis. Win/gammon/backgammon rates follow the on-roll convention:
// "player1" is always the player on roll, "player2" the opponent.
type AnalysisColumns struct {
	BestCubeAction        string
	CubeError             int64 // equity loss × 1000 (millipoints); 0 if no played action
	BestMoveEquityError   int64 // equity loss × 1000 (millipoints); 0 if no played move
	IsForced              int64 // 1 if checker position with exactly 1 legal move, else 0
	IsCloseCube           int64 // 1 if cube decision meets gnuBG isCloseCubedecision
	Player1WinRate        int64
	Player1GammonRate     int64
	Player1BackgammonRate int64
	Player2WinRate        int64
	Player2GammonRate     int64
	Player2BackgammonRate int64
}

// closeCubeThreshold is the gnuBG isCloseCubedecision equity gap threshold.
const closeCubeThreshold = 0.16

// ComputeIsCloseCube returns 1 if the cube decision qualifies as "close" per
// the gnuBG isCloseCubedecision predicate (gnubg/eval.c:5088-5100). Take/Pass
// decisions always count as close. Returns 0 when dca is nil.
func ComputeIsCloseCube(dca *domain.DoublingCubeAnalysis, playedCubeAction string) int64 {
	if playedCubeAction == "Take" || playedCubeAction == "Pass" {
		return 1
	}
	if dca == nil {
		return 0
	}
	var rOptimal float64
	switch dca.BestCubeAction {
	case "No Double":
		rOptimal = dca.CubefulNoDoubleEquity
	case "Double, Take", "Double/Take":
		rOptimal = dca.CubefulDoubleTakeEquity
	case "Double, Pass", "Double/Pass":
		rOptimal = dca.CubefulDoublePassEquity
	default:
		rOptimal = dca.CubefulNoDoubleEquity
		if dca.CubefulDoubleTakeEquity > rOptimal {
			rOptimal = dca.CubefulDoubleTakeEquity
		}
		if dca.CubefulDoublePassEquity > rOptimal {
			rOptimal = dca.CubefulDoublePassEquity
		}
	}
	rDouble := dca.CubefulDoubleTakeEquity
	if rDouble > 1.0 {
		rDouble = 1.0
	}
	if rOptimal-rDouble < closeCubeThreshold {
		return 1
	}
	return 0
}

// CubeActionError returns the signed equity error of the played cube action
// relative to the best one, ok=false when the action is empty or unrecognised.
// It is the single source of cube-error attribution, shared by
// PopulateAnalysisColumns (analysis.cube_error) and the search move-error
// filters so they cannot drift apart.
//
// A doubling decision is scored as min(DoubleTakeError, DoublePassError); a
// pure Take/Pass is scored from the responder's side against the optimal
// response.
func CubeActionError(dca *domain.DoublingCubeAnalysis, playedCubeAction string) (float64, bool) {
	if dca == nil {
		return 0, false
	}
	switch CanonicalCubeAction(playedCubeAction) {
	case CubeNoDouble:
		return dca.CubefulNoDoubleError, true
	case CubeDouble:
		return math.Min(dca.CubefulDoubleTakeError, dca.CubefulDoublePassError), true
	case CubeTake:
		return math.Min(dca.CubefulDoubleTakeEquity, dca.CubefulDoublePassEquity) - dca.CubefulDoubleTakeEquity, true
	case CubePass:
		return math.Min(dca.CubefulDoubleTakeEquity, dca.CubefulDoublePassEquity) - dca.CubefulDoublePassEquity, true
	}
	return 0, false
}

// normaliseCubeLabel folds a cube label to lowercase and drops the separators
// its producers disagree about, so that "Double, Take", "Double/Take" and
// "double take" all reduce to the same string: the one place where a spelling
// becomes a meaning.
func normaliseCubeLabel(label string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '/', ',', '-', '_', '.':
			return -1
		}
		return unicode.ToLower(r)
	}, label)
}

// CubeVerdict is what the analysis engine ruled about a cube position, split
// into the two decisions it actually contains: whether the cube should be
// offered, and how it should be answered if it is.
//
// The two belong to different players and are scored separately, which is why
// they cannot stay welded into one string. ShouldPass is meaningful even when
// ShouldDouble is false: "No Double" says the offer would be wrong AND that
// taking is right should the cube come anyway.
type CubeVerdict struct {
	ShouldDouble bool
	ShouldPass   bool
}

// BestCubeVerdict decodes an engine's bestCubeAction — "No Double",
// "Double, Take", "Double, Pass", "Too good to double, take" — into its two
// rulings. ok is false on an empty or unrecognised label, so an unknown
// spelling is never silently read as "no double".
//
// "Too good" is the trap: it contains "double" while ruling AGAINST doubling
// (the player is too strong to cash and plays on for the gammon), so it is
// tested before the plain "double" case — the same ordering discipline that
// CanonicalCubeAction applies to "Double No".
func BestCubeVerdict(bestCubeAction string) (CubeVerdict, bool) {
	s := normaliseCubeLabel(bestCubeAction)
	pass := strings.Contains(s, "pass") || strings.Contains(s, "drop")
	switch {
	case s == "":
		return CubeVerdict{}, false
	case strings.Contains(s, "toogood"):
		return CubeVerdict{ShouldDouble: false, ShouldPass: pass}, true
	case strings.Contains(s, "nodouble") || strings.Contains(s, "doubleno"):
		return CubeVerdict{ShouldDouble: false, ShouldPass: false}, true
	case strings.Contains(s, "double"):
		return CubeVerdict{ShouldDouble: true, ShouldPass: pass}, true
	}
	return CubeVerdict{}, false
}

// The four cube actions a player can take, as returned by CanonicalCubeAction.
// CubeUnknown is the zero value, so an unrecognised label never silently passes
// for a real action.
const (
	CubeUnknown  = ""
	CubeNoDouble = "no double"
	CubeDouble   = "double"
	CubeTake     = "take"
	CubePass     = "pass"
)

// CanonicalCubeAction maps every spelling of a cube action met in the wild onto
// one of the four constants above, or CubeUnknown.
//
// Producers disagree on spelling: XG writes both "No Double" and "Double No",
// and the latter contains "double" but not "nodouble", so an ad-hoc Contains
// chain reads it as a DOUBLE. Recognition is therefore stated once, here.
//
// The doubler's combined labels ("Double/Take", "Double/Pass") are DOUBLES: they
// name what the doubler did, the response being the opponent's. The bare
// abbreviations dt/dp, however, are the responses take/pass — that is how they
// reach us from move.cube_action and from filter input.
func CanonicalCubeAction(action string) string {
	s := normaliseCubeLabel(action)

	switch {
	case s == "":
		return CubeUnknown
	// Before the "double" test: "doubleno" contains "double".
	case s == "nd" || strings.Contains(s, "nodouble") || strings.Contains(s, "doubleno"):
		return CubeNoDouble
	case strings.Contains(s, "double"): // double, double/take, double/pass, redouble
		return CubeDouble
	case s == "dt" || strings.Contains(s, "take"):
		return CubeTake
	case s == "dp" || strings.Contains(s, "pass") || strings.Contains(s, "drop"):
		return CubePass
	}
	return CubeUnknown
}

// IsResponseCubeAction reports whether a cube action is a pure take/pass
// response (the cube was offered to this player), as opposed to a doubling
// decision; the doubler's combined "Double/Take", "Double/Pass" are NOT
// responses.

func IsResponseCubeAction(action string) bool {
	switch CanonicalCubeAction(action) {
	case CubeTake, CubePass:
		return true
	}
	return false
}

// PopulateAnalysisColumns computes the scalar analysis columns from a
// PositionAnalysis. playedMove and playedCubeAction are the actions taken in
// this position (may be empty). Rates are stored × 100, equities × 1000.
func PopulateAnalysisColumns(a *domain.PositionAnalysis, playedMove, playedCubeAction string) AnalysisColumns {
	var c AnalysisColumns
	if a == nil {
		return c
	}

	if dca := a.DoublingCubeAnalysis; dca != nil {
		c.BestCubeAction = dca.BestCubeAction

		c.Player1WinRate = int64(math.Round(dca.PlayerWinChances * 100))
		c.Player1GammonRate = int64(math.Round(dca.PlayerGammonChances * 100))
		c.Player1BackgammonRate = int64(math.Round(dca.PlayerBackgammonChances * 100))
		c.Player2WinRate = int64(math.Round(dca.OpponentWinChances * 100))
		c.Player2GammonRate = int64(math.Round(dca.OpponentGammonChances * 100))
		c.Player2BackgammonRate = int64(math.Round(dca.OpponentBackgammonChances * 100))

		if raw, ok := CubeActionError(dca, playedCubeAction); ok {
			c.CubeError = int64(math.Round(math.Abs(raw) * 1000))
		}
	} else if ca := a.CheckerAnalysis; ca != nil && len(ca.Moves) > 0 {
		best := ca.Moves[0]
		c.Player1WinRate = int64(math.Round(best.PlayerWinChance * 100))
		c.Player1GammonRate = int64(math.Round(best.PlayerGammonChance * 100))
		c.Player1BackgammonRate = int64(math.Round(best.PlayerBackgammonChance * 100))
		c.Player2WinRate = int64(math.Round(best.OpponentWinChance * 100))
		c.Player2GammonRate = int64(math.Round(best.OpponentGammonChance * 100))
		c.Player2BackgammonRate = int64(math.Round(best.OpponentBackgammonChance * 100))
	}

	if playedMove != "" && a.CheckerAnalysis != nil {
		normPlayed := NormalizeMove(playedMove)
		for _, m := range a.CheckerAnalysis.Moves {
			if NormalizeMove(m.Move) == normPlayed && m.EquityError != nil {
				c.BestMoveEquityError = int64(math.Round(*m.EquityError * 1000))
				break
			}
		}
	}

	if a.CheckerAnalysis != nil && len(a.CheckerAnalysis.Moves) == 1 {
		c.IsForced = 1
	}

	c.IsCloseCube = ComputeIsCloseCube(a.DoublingCubeAnalysis, playedCubeAction)

	return c
}

// RoundToMillipoint rounds an equity value (equity points) to the nearest 0.001.
func RoundToMillipoint(v float64) float64 {
	return math.Round(v*1000) / 1000
}

// RoundToHundredthPercent rounds a rate (percent) to the nearest 0.01%.
func RoundToHundredthPercent(v float64) float64 {
	return math.Round(v*100) / 100
}

// RoundAnalysisForStorage rounds every float field of a PositionAnalysis for
// compact JSON storage: rates → 2 decimals, equities/errors → millipoints.
func RoundAnalysisForStorage(a *domain.PositionAnalysis) {
	if a == nil {
		return
	}
	roundDCA := func(dca *domain.DoublingCubeAnalysis) {
		dca.PlayerWinChances = RoundToHundredthPercent(dca.PlayerWinChances)
		dca.PlayerGammonChances = RoundToHundredthPercent(dca.PlayerGammonChances)
		dca.PlayerBackgammonChances = RoundToHundredthPercent(dca.PlayerBackgammonChances)
		dca.OpponentWinChances = RoundToHundredthPercent(dca.OpponentWinChances)
		dca.OpponentGammonChances = RoundToHundredthPercent(dca.OpponentGammonChances)
		dca.OpponentBackgammonChances = RoundToHundredthPercent(dca.OpponentBackgammonChances)
		dca.CubelessNoDoubleEquity = RoundToMillipoint(dca.CubelessNoDoubleEquity)
		dca.CubelessDoubleEquity = RoundToMillipoint(dca.CubelessDoubleEquity)
		dca.CubefulNoDoubleEquity = RoundToMillipoint(dca.CubefulNoDoubleEquity)
		dca.CubefulNoDoubleError = RoundToMillipoint(dca.CubefulNoDoubleError)
		dca.CubefulDoubleTakeEquity = RoundToMillipoint(dca.CubefulDoubleTakeEquity)
		dca.CubefulDoubleTakeError = RoundToMillipoint(dca.CubefulDoubleTakeError)
		dca.CubefulDoublePassEquity = RoundToMillipoint(dca.CubefulDoublePassEquity)
		dca.CubefulDoublePassError = RoundToMillipoint(dca.CubefulDoublePassError)
		dca.WrongPassPercentage = RoundToHundredthPercent(dca.WrongPassPercentage)
		dca.WrongTakePercentage = RoundToHundredthPercent(dca.WrongTakePercentage)
	}
	if a.DoublingCubeAnalysis != nil {
		roundDCA(a.DoublingCubeAnalysis)
	}
	for i := range a.AllCubeAnalyses {
		roundDCA(&a.AllCubeAnalyses[i])
	}
	if ca := a.CheckerAnalysis; ca != nil {
		for i := range ca.Moves {
			m := &ca.Moves[i]
			m.Equity = RoundToMillipoint(m.Equity)
			if m.EquityError != nil {
				rounded := RoundToMillipoint(*m.EquityError)
				m.EquityError = &rounded
			}
			m.PlayerWinChance = RoundToHundredthPercent(m.PlayerWinChance)
			m.PlayerGammonChance = RoundToHundredthPercent(m.PlayerGammonChance)
			m.PlayerBackgammonChance = RoundToHundredthPercent(m.PlayerBackgammonChance)
			m.OpponentWinChance = RoundToHundredthPercent(m.OpponentWinChance)
			m.OpponentGammonChance = RoundToHundredthPercent(m.OpponentGammonChance)
			m.OpponentBackgammonChance = RoundToHundredthPercent(m.OpponentBackgammonChance)
		}
	}
}

// NormalizeMove normalizes a move string for comparison: "5/2 5/4" and
// "5/4 5/2" are the same move with parts in different order.
func NormalizeMove(move string) string {
	parts := strings.Fields(move)
	sort.Strings(parts)
	return strings.Join(parts, " ")
}

// PlayedActionsFor decides which checker move and which cube action a
// position's derived columns are computed against: what the analysis states
// (XG/GnuBG files carry it), else what the match's `move` table recorded.
// An analysis blunderDB computed itself carries no played action, and without
// the fallback every error would be zero — a Performance Rating of 0.00.
//
// A position deduplicated by Zobrist hash may have been played several times;
// its one row of derived columns takes the first occurrence, so the fallback
// must be as deterministic as the primary: the earliest recorded move.

func PlayedActionsFor(analysisMoves, analysisCubeActions, matchMoves, matchCubeActions []string) (playedMove, playedCubeAction string) {
	first := func(ss []string) string {
		for _, s := range ss {
			if s != "" {
				return s
			}
		}
		return ""
	}
	playedMove = first(analysisMoves)
	if playedMove == "" {
		playedMove = first(matchMoves)
	}
	playedCubeAction = first(analysisCubeActions)
	if playedCubeAction == "" {
		playedCubeAction = first(matchCubeActions)
	}
	return playedMove, playedCubeAction
}
