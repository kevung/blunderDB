package engine

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// BearoffDatabase holds the loaded one-sided 6-point bearoff database.
type BearoffDatabase struct {
	nPoints   int
	nCheckers int
	nPos      int // C(nPoints+nCheckers, nPoints)
	index     []bearoffIndex
	data      []byte
}

type bearoffIndex struct {
	offset uint32
	nz     uint8
	ioff   uint8
	nzg    uint8
	ioffg  uint8
}

// EPCResult holds the computed EPC values for a one-sided position.
type EPCResult struct {
	EPC       float64 `json:"epc"`
	MeanRolls float64 `json:"meanRolls"`
	StdDev    float64 `json:"stdDev"`
	PipCount  int     `json:"pipCount"`
	Wastage   float64 `json:"wastage"`
}

// Average pips per roll in backgammon:
// (2*3 + 3*4 + 4*5 + 4*6 + 6*7 + 5*8 + 4*9 + 2*10 + 2*11 + 1*12 + 1*16 + 1*20 + 1*24) / 36
// = 294/36 = 8.16667
const avgPipsPerRoll = 294.0 / 36.0

var (
	bearoffMu       sync.RWMutex
	globalBearoffDB *BearoffDatabase
)

// LoadOneSided points the EPC at a one-sided table on disk, replacing whatever
// was loaded. An empty path unloads it.
//
// The table is no longer compiled into the binary (ADR-0027): it is generated
// on the machine that needs it, which means there is a moment — the first
// launch, until the background generation finishes — when there is none. The
// EPC answers that it cannot compute rather than pretending; see OneSidedReady.
func LoadOneSided(path string) error {
	bearoffMu.Lock()
	defer bearoffMu.Unlock()
	if path == "" {
		globalBearoffDB = nil
		return nil
	}
	db, err := loadBearoffDatabaseFrom(path)
	if err != nil {
		return err
	}
	globalBearoffDB = db
	return nil
}

// OneSidedReady reports whether a one-sided table is loaded. The Eval panel
// asks before telling the user anything: silence while the table is being
// generated, a message only when a position actually needs it.
func OneSidedReady() bool {
	bearoffMu.RLock()
	defer bearoffMu.RUnlock()
	return globalBearoffDB != nil
}

// PipCounts returns the pip counts for both players from a Board.
// pip1 is Black's pip count (player on roll when PlayerOnRoll==0),
// pip2 is White's pip count. The pip count is purely positional and
// does not require the bearoff database.
func PipCounts(b domain.Board) (pip1, pip2 int) {
	for i, pt := range b.Points {
		if pt.Checkers <= 0 || pt.Color < 0 {
			continue
		}
		if pt.Color == domain.Black {
			pip1 += pt.Checkers * i
		} else {
			pip2 += pt.Checkers * (25 - i)
		}
	}
	return
}

// combination computes C(n, k) using iterative multiplication.
func combination(n, k int) int {
	if k > n {
		return 0
	}
	if k == 0 || k == n {
		return 1
	}
	if k > n-k {
		k = n - k
	}
	result := 1
	for i := 0; i < k; i++ {
		result = result * (n - i) / (i + 1)
	}
	return result
}

// loadBearoffDatabaseFrom reads a one-sided gnubg table from disk.
func loadBearoffDatabaseFrom(path string) (*BearoffDatabase, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read bearoff database %s: %w", path, err)
	}

	if len(raw) < 40 {
		return nil, fmt.Errorf("bearoff database too small")
	}

	// Parse header: gnubg-OS-06-15-1-1-0
	// Positions 6-7: "OS" (one-sided)
	// Position 9-10: nPoints (06)
	// Position 12-13: nCheckers (15)
	// Position 15: fGammon (1)
	// Position 17: fCompressed (1)
	// Position 19: fND (0)
	//
	// The point count is READ, not assumed to be six. A table over seven points
	// or more is what lets the EPC answer for a side whose farthest chequer is
	// outside the home board, and it differs from the six-point one in nothing
	// but its width — same header, same index, same runs (ADR-0027 §9).
	if string(raw[:9]) != "gnubg-OS-" {
		return nil, fmt.Errorf("%s is not a one-sided bearoff database", path)
	}
	points, err := strconv.Atoi(string(raw[9:11]))
	if err != nil || points < 1 || points > 24 {
		return nil, fmt.Errorf("%s: unreadable point count %q", path, raw[9:11])
	}
	checkers, err := strconv.Atoi(string(raw[12:14]))
	if err != nil || checkers < 1 || checkers > 15 {
		return nil, fmt.Errorf("%s: unreadable chequer count %q", path, raw[12:14])
	}

	db := &BearoffDatabase{
		nPoints:   points,
		nCheckers: checkers,
	}
	db.nPos = combination(db.nPoints+db.nCheckers, db.nPoints)

	headerSize := 40
	indexEntrySize := 8 // with gammon
	indexSize := db.nPos * indexEntrySize

	if len(raw) < headerSize+indexSize {
		return nil, fmt.Errorf("bearoff database too small for index")
	}

	// Parse index
	db.index = make([]bearoffIndex, db.nPos)
	for i := 0; i < db.nPos; i++ {
		base := headerSize + i*indexEntrySize
		db.index[i] = bearoffIndex{
			offset: binary.LittleEndian.Uint32(raw[base : base+4]),
			nz:     raw[base+4],
			ioff:   raw[base+5],
			nzg:    raw[base+6],
			ioffg:  raw[base+7],
		}
	}

	// Store data section
	db.data = raw[headerSize+indexSize:]

	return db, nil
}

// positionBearoff converts a checker arrangement on nPoints points to a
// combinatorial index. anBoard[i] = checkers on point i+1 (point 1 = bearing
// off next), and must be at least nPoints long.
// This implements the GNUbg PositionBearoff function using combinatorial number system.
func positionBearoff(anBoard []int, nPoints, nCheckers int) int {
	// Encode as combination index using "stars and bars"
	// Total bits = nCheckers + nPoints, with nPoints bits set
	j := nPoints - 1
	for i := 0; i < nPoints; i++ {
		j += anBoard[i]
	}

	// Build the bit pattern and convert to combinatorial index
	// using PositionF equivalent (combination-based ranking)
	fBits := uint32(1) << uint(j)
	for i := 0; i < nPoints-1; i++ {
		j -= anBoard[i] + 1
		fBits |= uint32(1) << uint(j)
	}

	return positionF(fBits, nCheckers+nPoints, nPoints)
}

// positionF converts a bit pattern to a combinatorial index.
// It counts the combination rank of a bit string of length n with k bits set.
func positionF(fBits uint32, n, k int) int {
	index := 0
	for n > 0 {
		n--
		if fBits&(1<<uint(n)) != 0 {
			if k > 0 {
				index += combination(n, k)
			}
			k--
		}
	}
	return index
}

// getDistribution reads the bearoff probability distribution for a given position index.
// Returns the probability of bearing off all checkers in exactly i rolls (i=0..31).
func (db *BearoffDatabase) getDistribution(posID int) ([]float64, error) {
	if posID < 0 || posID >= db.nPos {
		return nil, fmt.Errorf("position ID %d out of range [0, %d)", posID, db.nPos)
	}

	idx := db.index[posID]

	// Calculate byte offset into data section
	byteOffset := int(idx.offset) * 2

	totalShorts := int(idx.nz) + int(idx.nzg)
	if byteOffset+totalShorts*2 > len(db.data) {
		return nil, fmt.Errorf("data offset out of range for position %d", posID)
	}

	// Read bearoff probabilities
	probs := make([]float64, 32)
	for i := 0; i < int(idx.nz); i++ {
		off := byteOffset + i*2
		val := binary.LittleEndian.Uint16(db.data[off : off+2])
		probs[int(idx.ioff)+i] = float64(val) / 65535.0
	}

	return probs, nil
}

// averageRolls computes mean rolls and standard deviation from a probability distribution.
func averageRolls(probs []float64) (mean, stddev float64) {
	var sx, sx2 float64
	for i := 1; i < 32; i++ {
		p := float64(i) * probs[i]
		sx += p
		sx2 += float64(i) * p
	}
	mean = sx
	variance := sx2 - sx*sx
	if variance < 0 {
		variance = 0
	}
	stddev = math.Sqrt(variance)
	return
}

// ComputeEPC computes the EPC for a one-sided checker position on 6 points.
// anBoard[0..5] = number of checkers on points 1-6.
//
// It is the six-point form of ComputeEPCPoints, kept because most callers have
// a home board and nothing else.
func ComputeEPC(anBoard [6]int) (*EPCResult, error) {
	return ComputeEPCPoints(anBoard[:])
}

// OneSidedPoints is how wide the loaded table is: the highest point a chequer
// may stand on and still have an EPC. 0 when no table is loaded.
//
// A caller asks this before deciding whether a side is answerable, which is
// what makes the answer honest: the panel says "exact, OS-08" rather than
// silently extrapolating a six-point table past its domain.
func OneSidedPoints() int {
	bearoffMu.RLock()
	defer bearoffMu.RUnlock()
	if globalBearoffDB == nil {
		return 0
	}
	return globalBearoffDB.nPoints
}

// ComputeEPCPoints computes the EPC for a one-sided position of any width the
// loaded table covers. anBoard[i] = checkers on point i+1; a board longer than
// the table's width is refused rather than truncated, and a shorter one is
// padded (a position inside a narrower domain is inside a wider table too).
func ComputeEPCPoints(anBoard []int) (*EPCResult, error) {
	bearoffMu.RLock()
	db := globalBearoffDB
	bearoffMu.RUnlock()
	if db == nil {
		return nil, fmt.Errorf("bearoff database not loaded")
	}

	// Validate total checkers
	total := 0
	for _, c := range anBoard {
		if c < 0 {
			return nil, fmt.Errorf("invalid negative checker count")
		}
		total += c
	}
	if total > db.nCheckers {
		return nil, fmt.Errorf("too many checkers: %d (max %d)", total, db.nCheckers)
	}
	if total == 0 {
		return &EPCResult{EPC: 0, MeanRolls: 0, StdDev: 0, PipCount: 0, Wastage: 0}, nil
	}
	for i := db.nPoints; i < len(anBoard); i++ {
		if anBoard[i] > 0 {
			return nil, fmt.Errorf("a chequer stands on point %d, outside the %d-point table", i+1, db.nPoints)
		}
	}

	board := make([]int, db.nPoints)
	copy(board, anBoard)
	posID := positionBearoff(board, db.nPoints, db.nCheckers)

	probs, err := db.getDistribution(posID)
	if err != nil {
		return nil, fmt.Errorf("failed to get distribution: %w", err)
	}

	meanRolls, stddev := averageRolls(probs)
	epc := meanRolls * avgPipsPerRoll

	// Compute pip count
	pipCount := 0
	for i := range board {
		pipCount += board[i] * (i + 1)
	}

	wastage := epc - float64(pipCount)

	return &EPCResult{
		EPC:       math.Round(epc*100) / 100,
		MeanRolls: math.Round(meanRolls*1000) / 1000,
		StdDev:    math.Round(stddev*1000) / 1000,
		PipCount:  pipCount,
		Wastage:   math.Round(wastage*100) / 100,
	}, nil
}
