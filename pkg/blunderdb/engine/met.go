package engine

import "math"

// ============================================================================
// GNUbg Match Equity Table (MET) — Kazaross-XG2 (GNUbg default) with Zadeh fallback
//
// GNUbg loads the Kazaross-XG2 explicit MET by default (met/Kazaross-XG2.xml).
// This table was generated using XG rollouts to 9pts, GNUbg Supremo full rollouts
// to 15pts, and extended to 25pts by projecting take points.
//
// For entries beyond the 25×25 explicit table (matches > 25 points), GNUbg uses
// the Zadeh model (N. Zadeh, Management Science 23, 986, 1977) as a fallback.
// We replicate this by computing Zadeh first (full 64×64), then overlaying
// the Kazaross-XG2 explicit values for indices 0-24.
//
// gnuBGPreCrawfordMET[i][j] = player 0's MWC when player 0 needs i+1 pts, player 1 needs j+1 pts.
// gnuBGPostCrawfordMET[n] = trailer's MWC when trailer needs n+1 pts and leader needs 1 pt.
//
// Antisymmetry: gnuBGPreCrawfordMET[i][j] + gnuBGPreCrawfordMET[j][i] = 1.0
// This means MET[myAway-1][theirAway-1] gives "my" MWC for either player.
//
// This MET machinery lives in package engine (rather than database) so both the
// SQLite Storage backend and the Database wrapper can convert equities to MWC.
// ============================================================================

const (
	gnuBGMaxScore     = 64
	gnuBGMaxCubeLevel = 7
)

// d3Array is a 3D array type used during Zadeh MET computation.
// Heap-allocated to avoid ~900KB stack pressure.
// Uses float32 to match GNUbg's native precision exactly.
type d3Array [gnuBGMaxScore][gnuBGMaxScore][gnuBGMaxCubeLevel]float32

// MET tables use float32 internally to match GNUbg's C `float` type exactly.
// The accumulated precision of float32 arithmetic in the Zadeh iteration
// produces MET values that match GNUbg's, ensuring correct equity conversions.
var (
	gnuBGPreCrawfordMET  [gnuBGMaxScore][gnuBGMaxScore]float32
	gnuBGPostCrawfordMET [gnuBGMaxScore]float32
)

// kazarossXG2PreCrawford is the Kazaross-XG2 pre-Crawford Match Equity Table (25×25).
// This is GNUbg's DEFAULT MET (loaded from met/Kazaross-XG2.xml).
// Generated using XG rollouts to 9pts, GNUbg Supremo full rollouts to 15pts,
// extended to 25pts by projecting take points.
// Index [i][j] = player 0's MWC when player 0 needs i+1 pts, player 1 needs j+1 pts.
var kazarossXG2PreCrawford = [25][25]float32{
	{0.50000, 0.67736, 0.75076, 0.81436, 0.84179, 0.88731, 0.90724, 0.93250, 0.94402, 0.959275, 0.966442, 0.975534, 0.979845, 0.985273, 0.987893, 0.99114, 0.99273, 0.99467, 0.99563, 0.99679, 0.99737, 0.99807, 0.99842, 0.99884, 0.99905},
	{0.32264, 0.50000, 0.59947, 0.66870, 0.74359, 0.79940, 0.84225, 0.87539, 0.90197, 0.923034, 0.939311, 0.952470, 0.962495, 0.970701, 0.976887, 0.98196, 0.98580, 0.98893, 0.99129, 0.99322, 0.99466, 0.99585, 0.99675, 0.99746, 0.99802},
	{0.24924, 0.40053, 0.50000, 0.57150, 0.64795, 0.71123, 0.76209, 0.80468, 0.84017, 0.870638, 0.894417, 0.914831, 0.930702, 0.944426, 0.954931, 0.96399, 0.97093, 0.97687, 0.98139, 0.98522, 0.98814, 0.99062, 0.99248, 0.99407, 0.99527},
	{0.18564, 0.33130, 0.42850, 0.50000, 0.57732, 0.64285, 0.69924, 0.74577, 0.78799, 0.824059, 0.853955, 0.879141, 0.900233, 0.918040, 0.932657, 0.94495, 0.95499, 0.96341, 0.97021, 0.97589, 0.98044, 0.98422, 0.98726, 0.98975, 0.99174},
	{0.15821, 0.25641, 0.35205, 0.42268, 0.50000, 0.56635, 0.62638, 0.67786, 0.72540, 0.767055, 0.802732, 0.833654, 0.859934, 0.882866, 0.902013, 0.91847, 0.93223, 0.94397, 0.95367, 0.96189, 0.96864, 0.97432, 0.97896, 0.98283, 0.98600},
	{0.11269, 0.20060, 0.28877, 0.35715, 0.43365, 0.50000, 0.56261, 0.61636, 0.66787, 0.713057, 0.753427, 0.788634, 0.819569, 0.846648, 0.869999, 0.89021, 0.90756, 0.92246, 0.93508, 0.94583, 0.95488, 0.96254, 0.96894, 0.97432, 0.97879},
	{0.09276, 0.15775, 0.23791, 0.30076, 0.37362, 0.43739, 0.50000, 0.55480, 0.60854, 0.656283, 0.700209, 0.739054, 0.774121, 0.805203, 0.832566, 0.85659, 0.87761, 0.89591, 0.91171, 0.92535, 0.93702, 0.94703, 0.95553, 0.96276, 0.96887},
	{0.06750, 0.12461, 0.19532, 0.25423, 0.32214, 0.38364, 0.44520, 0.50000, 0.55442, 0.603718, 0.649899, 0.691356, 0.729447, 0.763593, 0.794397, 0.82158, 0.84578, 0.86714, 0.88589, 0.90230, 0.91658, 0.92898, 0.93968, 0.94891, 0.95682},
	{0.05598, 0.09803, 0.15983, 0.21201, 0.27460, 0.33213, 0.39146, 0.44558, 0.50000, 0.550196, 0.597926, 0.641481, 0.682119, 0.718927, 0.752814, 0.78301, 0.81037, 0.83483, 0.85662, 0.87591, 0.89294, 0.90791, 0.92098, 0.93240, 0.94230},
	{0.040725, 0.076966, 0.129362, 0.175941, 0.232945, 0.286943, 0.343717, 0.396282, 0.449804, 0.500000, 0.548547, 0.593459, 0.635880, 0.674830, 0.711113, 0.74371, 0.77375, 0.80093, 0.82543, 0.84741, 0.86703, 0.88448, 0.89991, 0.91353, 0.92550},
	{0.033558, 0.060689, 0.105583, 0.146045, 0.197268, 0.246573, 0.299791, 0.350101, 0.402074, 0.451453, 0.500000, 0.545552, 0.589242, 0.629736, 0.667927, 0.70303, 0.73530, 0.76494, 0.79198, 0.81648, 0.83862, 0.85849, 0.87629, 0.89214, 0.90622},
	{0.024466, 0.047530, 0.085169, 0.120859, 0.166346, 0.211366, 0.260946, 0.308644, 0.358519, 0.406541, 0.454448, 0.500000, 0.544068, 0.585701, 0.625259, 0.66178, 0.69610, 0.72778, 0.75703, 0.78381, 0.80826, 0.83044, 0.85051, 0.86856, 0.88476},
	{0.020155, 0.037505, 0.069298, 0.099767, 0.140066, 0.180431, 0.225879, 0.270553, 0.317881, 0.364120, 0.410758, 0.455932, 0.500000, 0.541943, 0.582545, 0.62036, 0.65619, 0.68966, 0.72081, 0.74963, 0.77619, 0.80054, 0.82276, 0.84295, 0.86123},
	{0.014727, 0.029299, 0.055574, 0.081960, 0.117134, 0.153352, 0.194797, 0.236407, 0.281073, 0.325170, 0.370264, 0.414299, 0.458057, 0.500000, 0.540750, 0.57942, 0.61634, 0.65117, 0.68391, 0.71448, 0.74290, 0.76917, 0.79339, 0.81559, 0.83586},
	{0.012107, 0.023113, 0.045069, 0.067343, 0.097987, 0.130001, 0.167434, 0.205603, 0.247186, 0.288887, 0.332073, 0.374741, 0.417455, 0.459250, 0.500000, 0.53916, 0.57679, 0.61261, 0.64659, 0.67859, 0.70862, 0.73664, 0.76265, 0.78669, 0.80883},
	{0.00886, 0.01804, 0.03601, 0.05505, 0.08153, 0.10979, 0.14341, 0.17842, 0.21699, 0.25629, 0.29697, 0.33822, 0.37964, 0.42058, 0.46084, 0.50000, 0.53796, 0.57441, 0.60929, 0.64241, 0.67376, 0.70323, 0.73084, 0.75657, 0.78046},
	{0.00727, 0.01420, 0.02907, 0.04501, 0.06777, 0.09244, 0.12239, 0.15422, 0.18963, 0.22625, 0.26470, 0.30390, 0.34381, 0.38366, 0.42321, 0.46204, 0.50000, 0.53676, 0.57222, 0.60618, 0.63856, 0.66925, 0.69822, 0.72542, 0.75087},
	{0.00533, 0.01107, 0.02313, 0.03659, 0.05603, 0.07754, 0.10409, 0.13286, 0.16517, 0.19907, 0.23506, 0.27222, 0.31034, 0.34883, 0.38739, 0.42559, 0.46324, 0.50000, 0.53574, 0.57023, 0.60336, 0.63501, 0.66510, 0.69356, 0.72038},
	{0.00437, 0.00871, 0.01861, 0.02979, 0.04633, 0.06492, 0.08829, 0.11411, 0.14338, 0.17457, 0.20802, 0.24297, 0.27919, 0.31609, 0.35341, 0.39071, 0.42778, 0.46426, 0.50000, 0.53475, 0.56838, 0.60073, 0.63171, 0.66122, 0.68921},
	{0.00321, 0.00678, 0.01478, 0.02411, 0.03811, 0.05417, 0.07465, 0.09770, 0.12409, 0.15259, 0.18352, 0.21619, 0.25037, 0.28552, 0.32141, 0.35759, 0.39382, 0.42977, 0.46525, 0.50000, 0.53387, 0.56667, 0.59830, 0.62864, 0.65760},
	{0.00263, 0.00534, 0.01186, 0.01956, 0.03136, 0.04512, 0.06298, 0.08342, 0.10706, 0.13297, 0.16138, 0.19174, 0.22381, 0.25710, 0.29138, 0.32624, 0.36144, 0.39664, 0.43162, 0.46613, 0.50000, 0.53303, 0.56508, 0.59603, 0.62576},
	{0.00193, 0.00415, 0.00938, 0.01578, 0.02568, 0.03746, 0.05297, 0.07102, 0.09209, 0.11552, 0.14151, 0.16956, 0.19946, 0.23083, 0.26336, 0.29677, 0.33075, 0.36499, 0.39927, 0.43333, 0.46697, 0.50000, 0.53226, 0.56360, 0.59391},
	{0.00158, 0.00325, 0.00752, 0.01274, 0.02104, 0.03106, 0.04447, 0.06032, 0.07902, 0.10009, 0.12371, 0.14949, 0.17724, 0.20661, 0.23735, 0.26916, 0.30178, 0.33490, 0.36829, 0.40170, 0.43492, 0.46774, 0.50000, 0.53153, 0.56221},
	{0.00116, 0.00254, 0.00593, 0.01025, 0.01717, 0.02568, 0.03724, 0.05109, 0.06760, 0.08647, 0.10786, 0.13144, 0.15705, 0.18441, 0.21331, 0.24343, 0.27458, 0.30644, 0.33878, 0.37136, 0.40397, 0.43640, 0.46847, 0.50000, 0.53086},
	{0.00095, 0.00198, 0.00473, 0.00826, 0.01400, 0.02121, 0.03113, 0.04318, 0.05770, 0.07450, 0.09378, 0.11524, 0.13877, 0.16414, 0.19117, 0.21954, 0.24913, 0.27962, 0.31079, 0.34240, 0.37424, 0.40609, 0.43779, 0.46914, 0.50000},
}

// kazarossXG2PostCrawford is the Kazaross-XG2 post-Crawford MET (24 entries, indices 0-23).
// Index i = trailer's MWC when trailer needs i+1 pts and leader needs 1 pt.
// Entry for 1-away (index 0) = 0.5 (the leader wins with probability 1 minus this).
// GNUbg copies entries 0..nLength-2 from the XML, then extends using the Zadeh formula.
var kazarossXG2PostCrawford = [24]float32{
	0.500000, 0.48803, 0.32264, 0.31002, 0.19012, 0.18072,
	0.11559, 0.10906, 0.06953, 0.065161, 0.042069, 0.039060,
	0.025371, 0.023428, 0.015304, 0.014050, 0.009240, 0.008420,
	0.005560, 0.005050, 0.003360, 0.003030, 0.002030, 0.001820,
}

func init() {
	gnuBGInitPostCrawfordMET()
	gnuBGInitPreCrawfordMET()
	// Overlay Kazaross-XG2 values (GNUbg's default MET) onto the Zadeh-computed tables.
	// For matches ≤ 25 points this gives exact GNUbg values; beyond 25 uses Zadeh as fallback.
	gnuBGOverlayKazarossXG2()
}

// gnuBGOverlayKazarossXG2 overlays the Kazaross-XG2 explicit table values onto
// the Zadeh-computed MET arrays. GNUbg's default is Kazaross-XG2 (met/Kazaross-XG2.xml),
// not Zadeh. The explicit table covers matches up to 25 points; beyond that,
// the Zadeh values remain as a reasonable fallback.
func gnuBGOverlayKazarossXG2() {
	// Pre-Crawford: copy 25×25 explicit values
	for i := 0; i < 25; i++ {
		for j := 0; j < 25; j++ {
			gnuBGPreCrawfordMET[i][j] = kazarossXG2PreCrawford[i][j]
		}
	}

	// Post-Crawford: copy 24 explicit entries (GNUbg copies 0..nLength-2 = 0..23)
	for i := 0; i < 24; i++ {
		gnuBGPostCrawfordMET[i] = kazarossXG2PostCrawford[i]
	}
}

// gnuBGGetMETEntry returns gnuBGPreCrawfordMET[i][j] with boundary handling.
// Mirrors the GET_MET macro: i<0 → 1.0, j<0 → 0.0.
func gnuBGGetMETEntry(i, j int) float32 {
	if i < 0 {
		return 1.0
	}
	if j < 0 {
		return 0.0
	}
	return gnuBGPreCrawfordMET[i][j]
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
			pc4 = gnuBGPostCrawfordMET[i-4]
		}
		pc2 := float32(1.0)
		if i-2 >= 0 {
			pc2 = gnuBGPostCrawfordMET[i-2]
		}
		gnuBGPostCrawfordMET[i] = rG*0.5*pc4 + (1.0-rG)*0.5*pc2

		if i == 1 {
			gnuBGPostCrawfordMET[i] -= rFD2
		}
		if i == 3 {
			gnuBGPostCrawfordMET[i] -= rFD4
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

	pc := &gnuBGPostCrawfordMET
	met := &gnuBGPreCrawfordMET
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

	// Check if either player has won the match
	if n0 < 0 {
		// Player 0 has won
		if fPlayer != 0 {
			return 0.0
		}
		return 1.0
	}
	if n1 < 0 {
		// Player 1 has won
		if fPlayer != 0 {
			return 1.0
		}
		return 0.0
	}

	// Crawford / post-Crawford handling
	if fCrawford || matchTo-score0 == 1 || matchTo-score1 == 1 {
		if n0 == 0 {
			// Player 0 at 1-away after game
			if fPlayer != 0 {
				return float64(gnuBGPostCrawfordMET[n1])
			}
			return float64(1.0 - gnuBGPostCrawfordMET[n1])
		}
		// Player 1 must be at or near match point
		if fPlayer != 0 {
			return float64(1.0 - gnuBGPostCrawfordMET[n0])
		}
		return float64(gnuBGPostCrawfordMET[n0])
	}

	// Normal pre-Crawford lookup
	if fPlayer != 0 {
		return float64(1.0 - gnuBGPreCrawfordMET[n0][n1])
	}
	return float64(gnuBGPreCrawfordMET[n0][n1])
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
// Returns math.NaN() for money-game positions (matchLength ≤ 0) or when the
// cube/score makes the denominator degenerate (e.g. dead cube).
func ConvertEMGLossToMWCLoss(emgMillipoints, score0, score1, fMove, cubeValue, matchLength int) float64 {
	if matchLength <= 0 {
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
