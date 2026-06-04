package engine

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// This file holds the pure analysis-encoding helpers shared by the Database
// wrapper and the SQLite Storage backend: zlib (de)compression of the analysis
// JSON blob, derivation of the denormalised scalar columns, and float
// rounding for compact storage. They perform no database I/O.

// CompressAnalysisData compresses raw JSON bytes using zlib (best compression).
func CompressAnalysisData(jsonData []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := zlib.NewWriterLevel(&buf, zlib.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(jsonData); err != nil {
		w.Close()
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecompressAnalysisData auto-detects zlib-compressed data vs raw JSON. If the
// first byte is '{' the data is returned as-is; otherwise zlib is attempted.
func DecompressAnalysisData(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	if data[0] == '{' {
		return data, nil
	}
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return data, nil
	}
	defer r.Close()
	return io.ReadAll(r)
}

// RecompressAnalysisData ensures data is in compressed form: raw JSON is
// compressed, already-compressed data is returned unchanged.
func RecompressAnalysisData(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	if data[0] != '{' {
		return data, nil
	}
	return CompressAnalysisData(data)
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

// AnalysisColumns holds the derived scalar columns computed from a
// PositionAnalysis. Win/gammon/backgammon rates follow the on-roll convention:
// "player1" is always the player on roll, "player2" the opponent.
type AnalysisColumns struct {
	BestCubeAction      string
	CubeError           int64 // equity loss × 1000 (millipoints); 0 if no played action
	BestMoveEquityError int64 // equity loss × 1000 (millipoints); 0 if no played move
	IsForced            int64 // 1 if checker position with exactly 1 legal move, else 0
	IsCloseCube         int64 // 1 if cube decision meets gnuBG isCloseCubedecision
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

// CubeActionError returns the equity error (in equity points, signed) of the
// given played cube action relative to the best action, and ok=false when the
// action is empty or unrecognized. This is the single source of truth for
// cube-error attribution, shared by PopulateAnalysisColumns (which feeds the
// denormalized analysis.cube_error column and the stats/SQL pre-filter) and by
// the search move-error filters, so they cannot drift apart.
//
// A doubling decision (Double / Double/Take / Double/Pass / Redouble) is scored
// by how much worse doubling is than the best action, i.e. the worse of the two
// opponent responses: min(DoubleTakeError, DoublePassError). A pure response
// (Take / Pass) is scored from the responder's perspective: how much worse the
// chosen response is than the optimal one. Matching is case-insensitive and
// tolerates the abbreviations (nd/dt/dp/drop) that appear in move.cube_action
// and in filter input.
func CubeActionError(dca *domain.DoublingCubeAnalysis, playedCubeAction string) (float64, bool) {
	if dca == nil || playedCubeAction == "" {
		return 0, false
	}
	s := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(playedCubeAction)), " ", "")
	switch {
	case s == "nd" || strings.Contains(s, "nodouble"):
		return dca.CubefulNoDoubleError, true
	case strings.Contains(s, "double"): // double, double/take, double/pass, redouble
		return math.Min(dca.CubefulDoubleTakeError, dca.CubefulDoublePassError), true
	case s == "dt" || strings.Contains(s, "take"):
		return math.Min(dca.CubefulDoubleTakeEquity, dca.CubefulDoublePassEquity) - dca.CubefulDoubleTakeEquity, true
	case s == "dp" || strings.Contains(s, "pass") || strings.Contains(s, "drop"):
		return math.Min(dca.CubefulDoubleTakeEquity, dca.CubefulDoublePassEquity) - dca.CubefulDoublePassEquity, true
	}
	return 0, false
}

// IsResponseCubeAction reports whether a cube action is a pure take/pass
// response (the cube was offered to this player), as opposed to a doubling
// decision such as Double / Double/Take / No Double. The doubler's combined
// actions ("Double/Take", "Double/Pass") are NOT responses. Used for board
// orientation and to decide whether to render the offered cube on the board.
func IsResponseCubeAction(action string) bool {
	s := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(action)), " ", "")
	if strings.Contains(s, "double") { // double, double/take, double/pass, nodouble, redouble
		return false
	}
	return s == "dt" || s == "dp" ||
		strings.Contains(s, "take") || strings.Contains(s, "pass") || strings.Contains(s, "drop")
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
