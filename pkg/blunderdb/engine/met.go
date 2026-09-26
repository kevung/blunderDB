package engine

import (
	"crypto/sha256"
	_ "embed" // required for the //go:embed directive below, even with no embed.FS use
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
)

// ============================================================================
// Match Equity Table (MET) — Kazaross-XG2, with a Zadeh fallback beyond 25 points
//
// ATTRIBUTION. The explicit table below is the **Kazaross-XG2** MET, the work of
// **Neil Kazaross**. It was produced by XG rollouts to 9 points, extended with
// Supremo full rollouts to 15 points, then to 25 points by projecting take
// points; its author describes the genesis himself: "I did not involve anyone
// else in creating Kazaross-XG2 since I did it prior to public release of XG so
// it could be included as default with XG2."
//
// The table is published IN FULL by its author — 25 away, every digit of
// precision, post-Crawford equities included — in Tom Keith's article "The
// Kazaross-XG2 Match Equity Table":
//
//	https://bkgm.com/articles/Keith/KazarossXG2MET/index.html
//
// GNU Backgammon ships that table as met/Kazaross-XG2.xml and loads it as its
// default, but GNUbg is neither its source nor its first publisher: this table
// is not a GNUbg derivative, and the gnuBG* identifiers below name only the
// Zadeh machinery. The in-app help credits it the same way. Unlike a bearoff
// database (an exact computation, a fact), rollouts are stochastic and carry
// their author's trace — hence the attribution.
//
// SOURCE OF THE NUMBERS. The 625 pre-Crawford and 25 post-Crawford entries are
// read at init from met_kazaross_xg2.json, a byte-identical copy of gammonNet's
// canonical export (data/met_kazaross_xg2.json, from Kazaross-XG2.xml, float64).
// Per gammonNet's ADR-0003 a table shared by blunderDB, gammonNet and gammonGo
// is exported once upstream and never retranscribed; metKazarossXG2SHA256 pins
// the bytes and a mismatch panics at init.
//
// WHAT DOES COME FROM GNUBG: the fallback beyond 25 points, the **Zadeh** model
// (N. Zadeh, Management Science 23, 986, 1977). gnuBGInitPreCrawfordMET /
// gnuBGInitPostCrawfordMET translate initMETZadeh() from matchequity.c, full
// 64×64, float32 to match its C `float`; the Kazaross-XG2 values are then
// overlaid for indices 0-24.
//
// preCrawfordMET[i][j] = player 0's MWC when player 0 needs i+1 pts, player 1 needs j+1 pts.
// postCrawfordMET[n] = trailer's MWC when trailer needs n+1 pts and leader needs 1 pt.
//
// Antisymmetry: preCrawfordMET[i][j] + preCrawfordMET[j][i] = 1.0
// This means MET[myAway-1][theirAway-1] gives "my" MWC for either player.
//
// ============================================================================

const (
	gnuBGMaxScore     = 64
	gnuBGMaxCubeLevel = 7
)

// MaxScore is the away-score horizon of this MET: GnuBGGetME clamps any away
// score past this many points to the table's last row/column rather than
// refusing it. Exported so gammonnet's redouble recursion shares this boundary
// instead of hardcoding a second copy.
const MaxScore = gnuBGMaxScore

// d3Array is a 3D array type used during Zadeh MET computation.
// Heap-allocated to avoid ~900KB stack pressure.
// Uses float32 to match GNUbg's native precision exactly.
type d3Array [gnuBGMaxScore][gnuBGMaxScore][gnuBGMaxCubeLevel]float32

// The live MET, as consulted by every lookup below: Neil Kazaross's Kazaross-XG2
// values for indices 0-24, the Zadeh fallback beyond. They deliberately carry no
// author in their name, because they hold both.
//
// float32 throughout, so the Zadeh iteration matches GNUbg's C `float` bit for
// bit. The Kazaross-XG2 cells are overlaid here at float32 for direct readers
// of these arrays; GnuBGGetME reads in-range indices from metPre/metPost.
var (
	preCrawfordMET  [gnuBGMaxScore][gnuBGMaxScore]float32
	postCrawfordMET [gnuBGMaxScore]float32
)

//go:embed met_kazaross_xg2.json
var metKazarossXG2Export []byte

// metKazarossXG2SHA256 pins the exact bytes of met_kazaross_xg2.json expected
// here, as recorded in gammonNet's data/met_kazaross_xg2.sha256. A hand-edit
// of the embedded file fails loudly at init.
const metKazarossXG2SHA256 = "753bdd7f901e713ba30ed6afa11d41b1ac2860d3fbf628a959c0ccabb9861d2b"

// metKazarossXG2Entry mirrors one row of the "pre" array in
// met_kazaross_xg2.json: away_a's MWC against away_b.
type metKazarossXG2Entry struct {
	AwayA int     `json:"away_a"`
	AwayB int     `json:"away_b"`
	MWC   float64 `json:"mwc"`
}

// metKazarossXG2PostEntry mirrors one entry of the "post" array: the
// trailer's MWC at that many points away, leader at 1-away.
type metKazarossXG2PostEntry struct {
	Away int     `json:"away"`
	MWC  float64 `json:"mwc"`
}

// metKazarossXG2Doc is the JSON shape of gammonNet's data/met_kazaross_xg2.json
// (see tools/extract_met.py upstream).
type metKazarossXG2Doc struct {
	Pre  []metKazarossXG2Entry     `json:"pre"`
	Post []metKazarossXG2PostEntry `json:"post"`
}

// kazarossXG2PreCrawford is the Kazaross-XG2 pre-Crawford Match Equity Table
// (25×25), the work of Neil Kazaross, in float64 — the export's precision,
// which gammonNet's gold harness requires.
// Index [i][j] = player 0's MWC when player 0 needs i+1 pts, player 1 needs j+1 pts.
var kazarossXG2PreCrawford [25][25]float64

// kazarossXG2PostCrawford is the Kazaross-XG2 post-Crawford MET, 25 entries.
// Index i = trailer's MWC when trailer needs i+1 pts and leader needs 1 pt.
// Entry for 1-away (index 0) = 0.5 (the leader wins with probability 1 minus this).
var kazarossXG2PostCrawford [25]float64

func init() {
	loadKazarossXG2FromExport()
	gnuBGInitPostCrawfordMET()
	gnuBGInitPreCrawfordMET()
	overlayKazarossXG2()
}

// loadKazarossXG2FromExport parses the embedded gammonNet export into
// kazarossXG2PreCrawford/kazarossXG2PostCrawford. It panics on a checksum
// mismatch, malformed JSON or a wrong entry count: the file ships inside the
// binary, so any failure is a vendoring defect, never user input.
func loadKazarossXG2FromExport() {
	sum := sha256.Sum256(metKazarossXG2Export)
	if got := hex.EncodeToString(sum[:]); got != metKazarossXG2SHA256 {
		panic(fmt.Sprintf(
			"engine: met_kazaross_xg2.json checksum mismatch: got %s, want %s "+
				"(re-vendor from gammonNet's data/met_kazaross_xg2.json and "+
				"data/met_kazaross_xg2.sha256)", got, metKazarossXG2SHA256))
	}

	var doc metKazarossXG2Doc
	if err := json.Unmarshal(metKazarossXG2Export, &doc); err != nil {
		panic(fmt.Sprintf("engine: cannot parse embedded met_kazaross_xg2.json: %v", err))
	}
	if len(doc.Pre) != 625 {
		panic(fmt.Sprintf("engine: met_kazaross_xg2.json: want 625 pre-Crawford entries, got %d", len(doc.Pre)))
	}
	if len(doc.Post) != 25 {
		panic(fmt.Sprintf("engine: met_kazaross_xg2.json: want 25 post-Crawford entries, got %d", len(doc.Post)))
	}
	for _, e := range doc.Pre {
		if e.AwayA < 1 || e.AwayA > 25 || e.AwayB < 1 || e.AwayB > 25 {
			panic(fmt.Sprintf("engine: met_kazaross_xg2.json: pre-Crawford entry out of range: %+v", e))
		}
		kazarossXG2PreCrawford[e.AwayA-1][e.AwayB-1] = e.MWC
	}
	for _, e := range doc.Post {
		if e.Away < 1 || e.Away > 25 {
			panic(fmt.Sprintf("engine: met_kazaross_xg2.json: post-Crawford entry out of range: %+v", e))
		}
		kazarossXG2PostCrawford[e.Away-1] = e.MWC
	}
}

// overlayKazarossXG2 overlays the 25-point Kazaross-XG2 values, rounded to
// float32, onto the Zadeh arrays for their direct readers; GnuBGGetME reads
// full precision through metPre/metPost.
func overlayKazarossXG2() {
	for i := 0; i < 25; i++ {
		for j := 0; j < 25; j++ {
			preCrawfordMET[i][j] = float32(kazarossXG2PreCrawford[i][j])
		}
	}

	for i := 0; i < 25; i++ {
		postCrawfordMET[i] = float32(kazarossXG2PostCrawford[i])
	}
}

// metPre returns preCrawfordMET[i][j] at the highest precision available:
// the float64 Kazaross-XG2 export inside 25×25, the float32 Zadeh fallback
// beyond. Negative indices are the caller's responsibility.
func metPre(i, j int) float64 {
	if i >= 0 && i < 25 && j >= 0 && j < 25 {
		return kazarossXG2PreCrawford[i][j]
	}
	return float64(preCrawfordMET[i][j])
}

// metPost returns postCrawfordMET[n] at the highest precision available:
// the float64 export for n < 25, the float32 Zadeh fallback beyond.

func metPost(n int) float64 {
	if n >= 0 && n < 25 {
		return kazarossXG2PostCrawford[n]
	}
	return float64(postCrawfordMET[n])
}

// gnuBGGetMETEntry returns preCrawfordMET[i][j] with boundary handling.
// Mirrors the GET_MET macro: i<0 → 1.0, j<0 → 0.0.
func gnuBGGetMETEntry(i, j int) float32 {
	if i < 0 {
		return 1.0
	}
	if j < 0 {
		return 0.0
	}
	return preCrawfordMET[i][j]
}

// gnuBGGetCubePrimeValue mirrors GetCubePrimeValue from matchequity.c.
// Returns 2*nCubeValue if automatic double applies, otherwise nCubeValue.
func gnuBGGetCubePrimeValue(i, j, nCubeValue int) int {
	if i < 2*nCubeValue && j >= 2*nCubeValue {
		return 2 * nCubeValue
	}
	return nCubeValue
}

// gnuBGInitPostCrawfordMET computes the post-Crawford MET using Zadeh's formula.
// Default parameters: gammon-rate-trailer=0.25, free-drop-2-away=0.015, free-drop-4-away=0.004.
func gnuBGInitPostCrawfordMET() {
	rG := float32(0.25)
	rFD2 := float32(0.015)
	rFD4 := float32(0.004)

	for i := 0; i < gnuBGMaxScore; i++ {
		pc4 := float32(1.0)
		if i-4 >= 0 {
			pc4 = postCrawfordMET[i-4]
		}
		pc2 := float32(1.0)
		if i-2 >= 0 {
			pc2 = postCrawfordMET[i-2]
		}
		postCrawfordMET[i] = rG*0.5*pc4 + (1.0-rG)*0.5*pc2

		if i == 1 {
			postCrawfordMET[i] -= rFD2
		}
		if i == 3 {
			postCrawfordMET[i] -= rFD4
		}
	}
}

// gnuBGInitPreCrawfordMET computes the pre-Crawford MET using Zadeh's formula.
// Default parameters: gammon-rate-leader=0.25, gammon-rate-trailer=0.15, delta=0.08, deltabar=0.06.
// This is a faithful translation of initMETZadeh() from GNUbg's matchequity.c.
func gnuBGInitPreCrawfordMET() {
	rG1 := float32(0.25)
	rG2 := float32(0.15)
	rDelta := float32(0.08)
	rDeltaBar := float32(0.06)

	pc := &postCrawfordMET
	met := &preCrawfordMET
	getMET := gnuBGGetMETEntry
	getCPV := gnuBGGetCubePrimeValue

	// Heap-allocate cube efficiency arrays
	d1 := new(d3Array)
	d2 := new(d3Array)
	d1bar := new(d3Array)
	d2bar := new(d3Array)

	// 1-away, n-away match equities (Crawford game row/column)
	for i := 0; i < gnuBGMaxScore; i++ {
		pcI2 := float32(1.0)
		if i-2 >= 0 {
			pcI2 = pc[i-2]
		}
		pcI1 := float32(1.0)
		if i-1 >= 0 {
			pcI1 = pc[i-1]
		}
		met[i][0] = rG1*0.5*pcI2 + (1.0-rG1)*0.5*pcI1
		met[0][i] = 1.0 - met[i][0]
	}

	// Fill the rest of the MET using Zadeh's iterative cube-adjusted formula
	for i := 0; i < gnuBGMaxScore; i++ {
		for j := 0; j <= i; j++ {
			for nCube := gnuBGMaxCubeLevel - 1; nCube >= 0; nCube-- {
				nCubeValue := 1 << nCube

				// --- D1bar ---
				nCPV := getCPV(i, j, nCubeValue)
				num := getMET(i-nCubeValue, j) -
					rG2*getMET(i, j-4*nCPV) -
					(1.0-rG2)*getMET(i, j-2*nCPV)
				den := rG1*getMET(i-4*nCPV, j) +
					(1.0-rG1)*getMET(i-2*nCPV, j) -
					rG2*getMET(i, j-4*nCPV) -
					(1.0-rG2)*getMET(i, j-2*nCPV)
				d1bar[i][j][nCube] = num / den

				if i != j {
					nCPV2 := getCPV(j, i, nCubeValue)
					numJI := getMET(j-nCubeValue, i) -
						rG2*getMET(j, i-4*nCPV2) -
						(1.0-rG2)*getMET(j, i-2*nCPV2)
					denJI := rG1*getMET(j-4*nCPV2, i) +
						(1.0-rG1)*getMET(j-2*nCPV2, i) -
						rG2*getMET(j, i-4*nCPV2) -
						(1.0-rG2)*getMET(j, i-2*nCPV2)
					d1bar[j][i][nCube] = numJI / denJI
				}

				// --- D2bar ---
				nCPV = getCPV(j, i, nCubeValue)
				num = getMET(j-nCubeValue, i) -
					rG2*getMET(j, i-4*nCPV) -
					(1.0-rG2)*getMET(j, i-2*nCPV)
				den = rG1*getMET(j-4*nCPV, i) +
					(1.0-rG1)*getMET(j-2*nCPV, i) -
					rG2*getMET(j, i-4*nCPV) -
					(1.0-rG2)*getMET(j, i-2*nCPV)
				d2bar[i][j][nCube] = num / den

				if i != j {
					nCPV2 := getCPV(i, j, nCubeValue)
					numJI := getMET(i-nCubeValue, j) -
						rG2*getMET(i, j-4*nCPV2) -
						(1.0-rG2)*getMET(i, j-2*nCPV2)
					denJI := rG1*getMET(i-4*nCPV2, j) +
						(1.0-rG1)*getMET(i-2*nCPV2, j) -
						rG2*getMET(i, j-4*nCPV2) -
						(1.0-rG2)*getMET(i, j-2*nCPV2)
					d2bar[j][i][nCube] = numJI / denJI
				}

				// --- D1 (cube efficiency adjusted) ---
				if i < 2*nCubeValue || j < 2*nCubeValue {
					d1[i][j][nCube] = d1bar[i][j][nCube]
					if i != j {
						d1[j][i][nCube] = d1bar[j][i][nCube]
					}
				} else {
					d1[i][j][nCube] = 1.0 + (d2[i][j][nCube+1]+rDelta)*(d1bar[i][j][nCube]-1.0)
					if i != j {
						d1[j][i][nCube] = 1.0 + (d2[j][i][nCube+1]+rDelta)*(d1bar[j][i][nCube]-1.0)
					}
				}

				// --- D2 (cube efficiency adjusted) ---
				if i < 2*nCubeValue || j < 2*nCubeValue {
					d2[i][j][nCube] = d2bar[i][j][nCube]
					if i != j {
						d2[j][i][nCube] = d2bar[j][i][nCube]
					}
				} else {
					d2[i][j][nCube] = 1.0 + (d1[i][j][nCube+1]+rDelta)*(d2bar[i][j][nCube]-1.0)
					if i != j {
						d2[j][i][nCube] = 1.0 + (d1[j][i][nCube+1]+rDelta)*(d2bar[j][i][nCube]-1.0)
					}
				}

				// --- Compute MET entry at cube level 0 ---
				if nCube == 0 && i > 0 && j > 0 {
					met[i][j] = ((d2[i][j][0]+rDeltaBar-0.5)*getMET(i-1, j) +
						(d1[i][j][0]+rDeltaBar-0.5)*getMET(i, j-1)) /
						(d1[i][j][0] + rDeltaBar + d2[i][j][0] + rDeltaBar - 1.0)
					if i != j {
						met[j][i] = 1.0 - met[i][j]
					}
				}
			}
		}
	}
}

// GnuBGGetME mirrors GNUbg's getME() function from matchequity.c.
// Returns the match winning chance from fPlayer's perspective after fWhoWins
// wins nPoints from the current match state.
//
// Parameters mirror the C function:
//   - score0, score1: current match scores for player 0 and player 1
//   - matchTo: match length
//   - fPlayer: whose perspective (0 or 1) to return MWC for
//   - nPoints: points won (typically cube value)
//   - fWhoWins: which player wins (0 or 1)
//   - fCrawford: whether the current game is Crawford
func GnuBGGetME(score0, score1, matchTo, fPlayer, nPoints, fWhoWins int, fCrawford bool) float64 {
	// Compute post-game "away" scores (0-indexed: n=0 means 1-away)
	notWhoWins := 0
	if fWhoWins == 0 {
		notWhoWins = 1
	}
	n0 := matchTo - (score0 + notWhoWins*nPoints) - 1
	n1 := matchTo - (score1 + fWhoWins*nPoints) - 1

	// Clamp rather than panic: callers should filter away scores past the table,
	// but a money game's 99999 match length must never index out of range.
	if n0 >= gnuBGMaxScore {
		n0 = gnuBGMaxScore - 1
	}
	if n1 >= gnuBGMaxScore {
		n1 = gnuBGMaxScore - 1
	}

	if n0 < 0 {
		if fPlayer != 0 {
			return 0.0
		}
		return 1.0
	}
	if n1 < 0 {
		if fPlayer != 0 {
			return 1.0
		}
		return 0.0
	}

	if fCrawford || matchTo-score0 == 1 || matchTo-score1 == 1 {
		if n0 == 0 {
			// Player 0 at 1-away after game
			if fPlayer != 0 {
				return metPost(n1)
			}
			return 1.0 - metPost(n1)
		}
		// Player 1 must be at or near match point
		if fPlayer != 0 {
			return 1.0 - metPost(n0)
		}
		return metPost(n0)
	}

	if fPlayer != 0 {
		return 1.0 - metPre(n0, n1)
	}
	return metPre(n0, n1)
}

// ConvertEMGLossToMWCLoss converts a loss expressed in EMG millipoints (the
// unit used internally by blunderDB: 1000 millipoints = 1 EMG) into a MWC
// loss (fraction; e.g. 0.015 = 1.5 % MWC).
//
// This is the inverse of the linear NEMG transformation used in GNUbg's
// mwc2eq():
//
//	ΔMWC = ΔEMG × (mwcWin − mwcLose) / 2
//
// The conversion applies identically to checker and cube errors because the
// NEMG mapping is simply a change of unit.
//
// Returns math.NaN() for money-game positions or when the cube/score makes the
// denominator degenerate (e.g. dead cube). Money is spelt two ways on disk — a
// match length ≤ 0 and the importers' sentinel 99999 — so any length the MET
// cannot represent counts as money.

func ConvertEMGLossToMWCLoss(emgMillipoints, score0, score1, fMove, cubeValue, matchLength int) float64 {
	if matchLength <= 0 || matchLength > gnuBGMaxScore {
		return math.NaN()
	}
	// Scores outside the match are equally unrepresentable — a position carrying the money
	// sentinel in its score, or simply corrupt, must not produce a fabricated MWC.
	if score0 < 0 || score1 < 0 || score0 >= matchLength || score1 >= matchLength {
		return math.NaN()
	}
	// Use float32 to match GNUbg's internal MET arithmetic precision.
	mwcWin := float32(GnuBGGetME(score0, score1, matchLength, fMove, cubeValue, fMove, false))
	mwcLose := float32(GnuBGGetME(score0, score1, matchLength, fMove, cubeValue, 1-fMove, false))
	denom := mwcWin - mwcLose
	if denom < 1e-7 && denom > -1e-7 {
		return math.NaN()
	}
	return (float64(emgMillipoints) / 1000.0) * float64(denom) / 2.0
}
