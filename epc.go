package main

import (
	"embed"
	"encoding/binary"
	"fmt"
	"math"
)

//go:embed gnubg_os6.bd
var bearoffDB embed.FS

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

var globalBearoffDB *BearoffDatabase

func init() {
	var err error
	globalBearoffDB, err = loadBearoffDatabase()
	if err != nil {
		fmt.Printf("Warning: failed to load bearoff database: %v\n", err)
	}
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

// loadBearoffDatabase loads the embedded gnubg_os6.bd file.
func loadBearoffDatabase() (*BearoffDatabase, error) {
	raw, err := bearoffDB.ReadFile("gnubg_os6.bd")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded bearoff database: %w", err)
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

	db := &BearoffDatabase{
		nPoints:   6,
		nCheckers: 15,
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

// positionBearoff converts a checker arrangement on 6 points to a combinatorial index.
// anBoard[0..5] = number of checkers on points 1-6 (point 1 = bearing off next).
// This implements the GNUbg PositionBearoff function using combinatorial number system.
func positionBearoff(anBoard [6]int, nPoints, nCheckers int) int {
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
func ComputeEPC(anBoard [6]int) (*EPCResult, error) {
	if globalBearoffDB == nil {
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
	if total > 15 {
		return nil, fmt.Errorf("too many checkers: %d (max 15)", total)
	}
	if total == 0 {
		return &EPCResult{EPC: 0, MeanRolls: 0, StdDev: 0, PipCount: 0, Wastage: 0}, nil
	}

	posID := positionBearoff(anBoard, 6, 15)

	probs, err := globalBearoffDB.getDistribution(posID)
	if err != nil {
		return nil, fmt.Errorf("failed to get distribution: %w", err)
	}

	meanRolls, stddev := averageRolls(probs)
	epc := meanRolls * avgPipsPerRoll

	// Compute pip count
	pipCount := 0
	for i := 0; i < 6; i++ {
		pipCount += anBoard[i] * (i + 1)
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

// ComputeEPCFromPosition computes the EPC for both players from a full board position.
// It extracts checkers on points 1-6 for each side (the bearing-off zone).
// Returns EPC for bottom player and top player.
func (d *Database) ComputeEPCFromPosition(position Position) (map[string]interface{}, error) {
	// In the board model: Points[1..6] = bottom player's bearing-off zone (points 1-6)
	// Points[19..24] = top player's bearing-off zone (from top player's perspective, points 24-19)
	// But we need to check from each player's perspective.

	// Bottom player (Black, index 0): points 1-6 in the board
	// Board.Points[1] to Board.Points[6]
	var bottomBoard [6]int
	bottomTotal := 0
	bottomAllInHome := true
	for i := 0; i < 6; i++ {
		pt := position.Board.Points[i+1]
		if pt.Color == Black {
			bottomBoard[i] = pt.Checkers
			bottomTotal += pt.Checkers
		}
	}
	// Check if all bottom player's checkers are in home board or borne off
	for i := 7; i <= 24; i++ {
		pt := position.Board.Points[i]
		if pt.Color == Black && pt.Checkers > 0 {
			bottomAllInHome = false
			break
		}
	}
	// Also check bar
	if position.Board.Points[BlackBar].Color == Black && position.Board.Points[BlackBar].Checkers > 0 {
		bottomAllInHome = false
	}

	// Top player (White, index 1): points 24-19 in the board (indices 24,23,22,21,20,19)
	// From White's perspective, point 24 is their 1-point, 23 is their 2-point, etc.
	var topBoard [6]int
	topTotal := 0
	topAllInHome := true
	for i := 0; i < 6; i++ {
		pt := position.Board.Points[24-i]
		if pt.Color == White {
			topBoard[i] = pt.Checkers
			topTotal += pt.Checkers
		}
	}
	// Check if all top player's checkers are in home board or borne off
	for i := 1; i <= 18; i++ {
		pt := position.Board.Points[i]
		if pt.Color == White && pt.Checkers > 0 {
			topAllInHome = false
			break
		}
	}
	if position.Board.Points[WhiteBar].Color == White && position.Board.Points[WhiteBar].Checkers > 0 {
		topAllInHome = false
	}

	result := map[string]interface{}{
		"bottomEPC":          nil,
		"topEPC":             nil,
		"bottomAllInHome":    bottomAllInHome,
		"topAllInHome":       topAllInHome,
		"bottomCheckerCount": bottomTotal,
		"topCheckerCount":    topTotal,
	}

	if bottomAllInHome && bottomTotal > 0 {
		epc, err := ComputeEPC(bottomBoard)
		if err == nil {
			result["bottomEPC"] = epc
		}
	}

	if topAllInHome && topTotal > 0 {
		epc, err := ComputeEPC(topBoard)
		if err == nil {
			result["topEPC"] = epc
		}
	}

	return result, nil
}
