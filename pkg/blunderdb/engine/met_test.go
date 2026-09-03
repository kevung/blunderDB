package engine

import (
	"math"
	"testing"
)

// TestConvertEMGLossToMWCLoss_MoneyGame verifies that a money-game position
// (matchLength ≤ 0) returns NaN.
func TestConvertEMGLossToMWCLoss_MoneyGame(t *testing.T) {
	result := ConvertEMGLossToMWCLoss(100, 0, 0, 0, 1, 0)
	if !math.IsNaN(result) {
		t.Errorf("expected NaN for money game, got %f", result)
	}
}

// TestConvertEMGLossToMWCLoss_NegativeMatchLength also returns NaN.
func TestConvertEMGLossToMWCLoss_NegativeMatchLength(t *testing.T) {
	result := ConvertEMGLossToMWCLoss(100, 0, 0, 0, 1, -1)
	if !math.IsNaN(result) {
		t.Errorf("expected NaN for negative matchLength, got %f", result)
	}
}

// TestConvertEMGLossToMWCLoss_DMP tests DMP (1-away / 1-away, cube=1).
// In DMP: mwcWin=1.0, mwcLose=0.0, denom=1.0
// ΔMWC = emgMillipoints/1000 × 1.0/2.0 = emgMillipoints/2000
func TestConvertEMGLossToMWCLoss_DMP(t *testing.T) {
	emgMP := 200 // 0.2 EMG
	result := ConvertEMGLossToMWCLoss(emgMP, 0, 0, 0, 1, 1)
	if math.IsNaN(result) {
		t.Fatal("unexpected NaN for DMP")
	}
	want := float64(emgMP) / 2000.0
	if math.Abs(result-want) > 1e-5 {
		t.Errorf("DMP: got %.6f, want %.6f", result, want)
	}
}

// TestConvertEMGLossToMWCLoss_RoundTrip verifies the round-trip property:
// converting a ΔMWC to ΔEMG with the forward formula and back with
// ConvertEMGLossToMWCLoss recovers the original ΔMWC to within 1e-5
// (float32 precision).
//
// Test cases span DMP (1pt), Crawford, 3-away/5-away, 7pt, 11pt, 21pt matches
// with cube values 1 and 2.
func TestConvertEMGLossToMWCLoss_RoundTrip(t *testing.T) {
	type testCase struct {
		name        string
		score0      int
		score1      int
		matchLength int
		fMove       int
		cubeValue   int
	}
	cases := []testCase{
		// DMP: both players need 1 more point
		{"DMP", 0, 0, 1, 0, 1},
		// Crawford: 1-away (leader) vs 3-away (trailer), cube dead (cube=1)
		{"Crawford 1v3", 0, 2, 3, 1, 1},
		// 3-away 5-away, cube=1
		{"3away5away cube1", 2, 4, 7, 0, 1},
		// 3-away 5-away, cube=2
		{"3away5away cube2", 2, 4, 7, 0, 2},
		// 7-point match, even score, cube=1
		{"7pt even", 3, 3, 7, 0, 1},
		// 11-point match, 4away 7away, cube=2
		{"11pt 4a7a cube2", 7, 4, 11, 0, 2},
		// 21-point match, cube=4
		{"21pt cube4", 10, 8, 21, 0, 4},
	}

	// An arbitrary ΔMWC loss to test with (1.5% = 0.015).
	const deltaMWCIn = 0.015

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mwcWin := float32(GnuBGGetME(tc.score0, tc.score1, tc.matchLength, tc.fMove, tc.cubeValue, tc.fMove, false))
			mwcLose := float32(GnuBGGetME(tc.score0, tc.score1, tc.matchLength, tc.fMove, tc.cubeValue, 1-tc.fMove, false))
			denom := mwcWin - mwcLose
			if denom < 1e-7 && denom > -1e-7 {
				t.Skipf("degenerate denom (%.6f) for %s — skipping", denom, tc.name)
			}

			// Forward: ΔMWC → ΔEMG millipoints
			// ΔEMG = ΔMWC × 2 / (mwcWin − mwcLose)
			deltaEMG := deltaMWCIn * 2.0 / float64(denom)
			deltaEMGMP := int(deltaEMG * 1000)

			// Inverse
			got := ConvertEMGLossToMWCLoss(deltaEMGMP, tc.score0, tc.score1, tc.fMove, tc.cubeValue, tc.matchLength)
			if math.IsNaN(got) {
				t.Fatalf("unexpected NaN for %s", tc.name)
			}

			// Reconstruct the expected value from the same integer truncation.
			wantExact := float64(deltaEMGMP) / 1000.0 * float64(denom) / 2.0
			if math.Abs(got-wantExact) > 1e-5 {
				t.Errorf("round-trip failed: got %.8f, want %.8f (diff %.2e)", got, wantExact, math.Abs(got-wantExact))
			}

			// Sign must be positive (a loss).
			if got < 0 {
				t.Errorf("MWC loss should be non-negative, got %.8f", got)
			}
		})
	}
}

// TestConvertEMGLossToMWCLoss_ZeroError verifies that zero error yields zero MWC loss.
func TestConvertEMGLossToMWCLoss_ZeroError(t *testing.T) {
	result := ConvertEMGLossToMWCLoss(0, 3, 3, 0, 1, 7)
	if math.IsNaN(result) {
		t.Fatal("unexpected NaN")
	}
	if result != 0.0 {
		t.Errorf("expected 0.0, got %f", result)
	}
}

// TestConvertEMGLossToMWCLoss_Crawford tests a Crawford score
// (score 4-away / Crawford 1-away, cube=1). The conversion must return a
// plausible non-NaN value with the correct sign.
func TestConvertEMGLossToMWCLoss_Crawford(t *testing.T) {
	// 5-point match: leader needs 1, trailer needs 4 (trailer is on roll, fMove=1)
	result := ConvertEMGLossToMWCLoss(100, 0, 3, 1, 1, 5)
	if math.IsNaN(result) {
		t.Fatal("unexpected NaN for Crawford position")
	}
	// A positive error should yield a positive MWC loss.
	if result <= 0 {
		t.Errorf("expected positive MWC loss, got %.6f", result)
	}
}

// blunderDB writes 99999 as the match length of a money game. That sentinel used to flow
// straight into the MET lookup, which indexes a 64-entry table by away score: one such row
// among a user's matches panicked inside GetAllMatches, and because a panic in a bound
// method makes Wails answer with an empty callback, the GUI call never settled and the
// export screen simply never appeared.
func TestMoneyGameSentinelIsNotAMatch(t *testing.T) {
	for _, matchLength := range []int{0, -1, 99999, gnuBGMaxScore + 1} {
		got := ConvertEMGLossToMWCLoss(120, 0, 0, 0, 1, matchLength)
		if !math.IsNaN(got) {
			t.Fatalf("match length %d has no match equity: got %v, want NaN", matchLength, got)
		}
	}
}

func TestScoresOutsideTheMatchHaveNoEquity(t *testing.T) {
	cases := [][2]int{{7, 0}, {0, 7}, {-1, 0}, {0, -1}, {99999, 99999}}
	for _, c := range cases {
		got := ConvertEMGLossToMWCLoss(120, c[0], c[1], 0, 1, 7)
		if !math.IsNaN(got) {
			t.Fatalf("scores %v in a 7-point match: got %v, want NaN", c, got)
		}
	}
}

// Whatever it is handed, the lookup itself must never panic: it is called from bound methods
// where a panic costs far more than a wrong number.
func TestGnuBGGetMENeverPanics(t *testing.T) {
	for _, matchTo := range []int{0, 1, 7, gnuBGMaxScore, gnuBGMaxScore + 1, 99999} {
		for _, score := range []int{0, 1, 6, 99999} {
			for _, crawford := range []bool{false, true} {
				got := GnuBGGetME(score, score, matchTo, 0, 1, 0, crawford)
				if math.IsNaN(got) || got < 0 || got > 1 {
					t.Fatalf("GnuBGGetME(%d,%d,%d,crawford=%v) = %v, want a probability", score, score, matchTo, crawford, got)
				}
			}
		}
	}
}

// The fix must not move any number a real match produces.
func TestNormalMatchEquityIsUnchanged(t *testing.T) {
	got := ConvertEMGLossToMWCLoss(120, 3, 4, 0, 1, 7)
	if math.IsNaN(got) {
		t.Fatal("a 7-point match at 3-4 must have a match equity")
	}
	if got <= 0 || got > 0.2 {
		t.Fatalf("unexpected MWC loss for a 120 mp error: %v", got)
	}
}

// ── The table itself (#188) ─────────────────────────────────────────────────
//
// A table of 625 numbers is not proof-read; its PROPERTIES are. Until these
// tests the file only exercised the EMG→MWC converter, and a transcription
// error — one digit, one swapped pair of indices — would have moved a cube
// verdict one time in a thousand without anything ever looking broken.

// A handful of the published Kazaross-XG2 values, as Tom Keith's article
// (https://bkgm.com/articles/Keith/KazarossXG2MET/index.html) and GNU
// Backgammon's met/Kazaross-XG2.xml print them: the textbook cells everyone
// quotes, and the four corners of the explicit table. Checked against
// gammonNet's canonical export (data/met_kazaross_xg2.json, embedded here as
// met_kazaross_xg2.json — #24), which is now this file's actual source: the
// float64 checks below compare EXACTLY (the export IS these numbers), the
// float32 "live table" checks tolerate the overlay's rounding.
func TestKazarossXG2MatchesThePublication(t *testing.T) {
	pre := []struct {
		awayA, awayB int
		mwc          float64
	}{
		{1, 2, 0.67736}, // Crawford, leader on roll: the most quoted cell of any MET
		{2, 1, 0.32264},
		{2, 2, 0.50000},
		{2, 3, 0.59947},
		{3, 2, 0.40053},
		{2, 4, 0.66870},
		{3, 4, 0.57150},
		{4, 5, 0.57732},
		{5, 5, 0.50000},
		{3, 7, 0.76209},
		{7, 10, 0.656283},
		{7, 11, 0.700209},
		{1, 25, 0.99905},
		{25, 1, 0.00095},
		{25, 24, 0.46914},
		{24, 25, 0.53086},
		{25, 25, 0.50000},
	}
	for _, c := range pre {
		got := kazarossXG2PreCrawford[c.awayA-1][c.awayB-1]
		if math.Abs(got-c.mwc) > 1e-6 {
			t.Errorf("pre-Crawford %d-away/%d-away: table %.6f, published %.6f", c.awayA, c.awayB, got, c.mwc)
		}
		// And the live float32 table reads the same cell, to float32 rounding:
		// the overlay covers it (overlayKazarossXG2), GnuBGGetME's own lookups
		// go through metPre at the full float64 precision instead (see #24).
		if live := float64(preCrawfordMET[c.awayA-1][c.awayB-1]); math.Abs(live-got) > 1e-6 {
			t.Errorf("pre-Crawford %d-away/%d-away: live table %.6f differs from the explicit %.6f", c.awayA, c.awayB, live, got)
		}
	}

	post := []struct {
		away int
		mwc  float64
	}{
		{1, 0.50000},
		{2, 0.48803}, // the free drop: 2-away post-Crawford is barely worse than 1-away
		{3, 0.32264},
		{4, 0.31002},
		{5, 0.19012},
		{24, 0.00182},
		// The 25th post-Crawford entry: this file used to stop at 24 explicit
		// entries where gammonNet's table (and now its export) carries 25,
		// so this cell answered the Zadeh fallback instead of Kazaross's own
		// published 0.00123. Closed by #24 — the export supplies it directly.
		{25, 0.00123},
	}
	for _, c := range post {
		got := kazarossXG2PostCrawford[c.away-1]
		if math.Abs(got-c.mwc) > 1e-6 {
			t.Errorf("post-Crawford %d-away: table %.6f, published %.6f", c.away, got, c.mwc)
		}
		if live := float64(postCrawfordMET[c.away-1]); math.Abs(live-got) > 1e-6 {
			t.Errorf("post-Crawford %d-away: live table %.6f differs from the explicit %.6f", c.away, live, got)
		}
	}
}

// GnuBGGetME's own lookup (metPost/metPre) must answer the 25-away
// post-Crawford trailer at the export's full precision — not the float32
// overlay, and not the Zadeh fallback this file used to answer there. This
// is the exact case the gold README documented as a "second, narrower,
// already-known boundary disagreement" with gammonNet; #24 closes it.
func TestPostCrawford25AwayIsTheExplicitValue(t *testing.T) {
	got := metPost(24) // index 24 = trailer 25-away
	if math.Abs(got-0.00123) > 1e-9 {
		t.Errorf("metPost(24) = %.9f, want the published 0.00123 (exactly, from the export)", got)
	}

	// GnuBGGetME's raw post-Crawford lookup, nPoints=0 so it reads the current
	// score rather than a hypothetical outcome: a 26-point match at leader
	// 25 / trailer 1 puts the leader at 1-away (n0=0) and the trailer at
	// 25-away (n1=24), which the "n0 == 0" branch answers via metPost(n1).
	trailer := GnuBGGetME(25, 1, 26, 1, 0, 0, false)
	if math.Abs(trailer-0.00123) > 1e-9 {
		t.Errorf("GnuBGGetME trailer at 25-away post-Crawford = %.9f, want 0.00123", trailer)
	}
	leader := GnuBGGetME(25, 1, 26, 0, 0, 0, false)
	if math.Abs(leader-(1-0.00123)) > 1e-9 {
		t.Errorf("GnuBGGetME leader at 1-away post-Crawford (trailer 25-away) = %.9f, want %.9f", leader, 1-0.00123)
	}
}

// What one side wins the other loses: MET[i][j] + MET[j][i] = 1 over the whole
// 64×64 live table — explicit cells and Zadeh extension alike — with the
// diagonal at one half. Float32 storage rounds each cell on its own, so the
// identity holds to float32 (measured worst 2.4e-7), not to the bit; the
// explicit diagonal is exactly 0.5 as transcribed.
func TestMETIsAntisymmetric(t *testing.T) {
	worst := 0.0
	for i := 0; i < gnuBGMaxScore; i++ {
		for j := 0; j < gnuBGMaxScore; j++ {
			d := math.Abs(float64(preCrawfordMET[i][j]) + float64(preCrawfordMET[j][i]) - 1)
			if d > worst {
				worst = d
			}
			if d > 1e-6 {
				t.Errorf("MET[%d][%d] + MET[%d][%d] = %.8f, want 1", i, j, j, i, 1+d)
			}
		}
		if i < 25 && preCrawfordMET[i][i] != 0.5 {
			t.Errorf("explicit diagonal MET[%d][%d] = %v, want exactly 0.5", i, i, preCrawfordMET[i][i])
		}
		if d := math.Abs(float64(preCrawfordMET[i][i]) - 0.5); d > 1e-6 {
			t.Errorf("diagonal MET[%d][%d] = %v, want 0.5", i, i, preCrawfordMET[i][i])
		}
	}
	t.Logf("antisymmetry: worst |MET[i][j] + MET[j][i] − 1| = %.3e over %d×%d", worst, gnuBGMaxScore, gnuBGMaxScore)
}

// The seam between Neil Kazaross's explicit 25×25 table and the Zadeh
// extension: index 24 (25-away) is the last explicit row, index 25 the first
// computed one. Across it the table must stay what a MET is — needing more
// points is worse, against every opponent — and it must not jump: the first
// Zadeh row sits within a few hundredths of the last explicit one (measured
// 0.031 at 25-away/25-away → 26-away/25-away, the largest step anywhere at
// the seam), which is the same order as the steps inside the table itself.
// A seam that broke monotonicity, or stepped by a tenth, would mean the
// overlay landed one row off — plausible numbers, wrong table.
func TestZadehExtensionJoinsKazarossAtTheSeam(t *testing.T) {
	const last, first = 24, 25
	for j := 0; j < 25; j++ {
		// The float32 overlay rounds the float64 export cell; compared to
		// float32 precision, not bit-exact (see overlayKazarossXG2).
		if preCrawfordMET[last][j] != float32(kazarossXG2PreCrawford[last][j]) {
			t.Errorf("row %d, column %d is not the explicit value", last, j)
		}
		if preCrawfordMET[j][last] != float32(kazarossXG2PreCrawford[j][last]) {
			t.Errorf("row %d, column %d is not the explicit value", j, last)
		}
	}
	maxStep := 0.0
	for j := 0; j < gnuBGMaxScore; j++ {
		rowLast, rowFirst := float64(preCrawfordMET[last][j]), float64(preCrawfordMET[first][j])
		colLast, colFirst := float64(preCrawfordMET[j][last]), float64(preCrawfordMET[j][first])
		if !(rowFirst < rowLast) {
			t.Errorf("against %d-away: 26-away (%.6f) is not worse than 25-away (%.6f)", j+1, rowFirst, rowLast)
		}
		if !(colFirst > colLast) {
			t.Errorf("%d-away: a 26-away opponent (%.6f) is not better than a 25-away one (%.6f)", j+1, colFirst, colLast)
		}
		if d := rowLast - rowFirst; d > maxStep {
			maxStep = d
		}
	}
	if maxStep > 0.05 {
		t.Errorf("largest step across the seam is %.4f, want under 0.05", maxStep)
	}
	t.Logf("largest step across the Kazaross/Zadeh seam: %.4f", maxStep)

	// The whole live table is monotone in each argument — non-strictly, since
	// float32 saturates at 0.99999994 in the far corner (1-away vs 63-away
	// and 2-away vs 63-away share it).
	for i := 0; i+1 < gnuBGMaxScore; i++ {
		for j := 0; j < gnuBGMaxScore; j++ {
			if preCrawfordMET[i+1][j] > preCrawfordMET[i][j] {
				t.Errorf("MET[%d][%d] = %v > MET[%d][%d] = %v: needing more points is better?", i+1, j, preCrawfordMET[i+1][j], i, j, preCrawfordMET[i][j])
			}
		}
	}
}

// The lookup is antisymmetric in fPlayer at every score it distinguishes —
// pre-Crawford, the two post-Crawford branches, and a match already won by
// either side: what player 1 is told is one minus what player 0 is told, to
// float32, for the same game outcome.
func TestGnuBGGetMEIsAntisymmetricInPlayer(t *testing.T) {
	cases := []struct {
		name                       string
		score0, score1, matchTo, n int
		crawford                   bool
	}{
		{"pre-Crawford 3-4 of 7", 3, 4, 7, 1, false},
		{"pre-Crawford, cube 4", 2, 5, 11, 4, false},
		{"Crawford, leader is player 0", 6, 3, 7, 1, true},
		{"Crawford, leader is player 1", 3, 6, 7, 1, true},
		{"post-Crawford, player 0 at match point", 6, 4, 7, 2, false},
		{"post-Crawford, player 1 at match point", 4, 6, 7, 2, false},
		{"player 0 wins the match", 6, 0, 7, 1, false},
		{"player 1 wins the match", 0, 6, 7, 1, false},
		{"player 1 wins the match by a gammon", 0, 5, 7, 2, false},
	}
	for _, c := range cases {
		for whoWins := 0; whoWins <= 1; whoWins++ {
			p0 := GnuBGGetME(c.score0, c.score1, c.matchTo, 0, c.n, whoWins, c.crawford)
			p1 := GnuBGGetME(c.score0, c.score1, c.matchTo, 1, c.n, whoWins, c.crawford)
			if math.Abs(p0+p1-1) > 1e-6 {
				t.Errorf("%s, player %d wins %d: player 0 sees %.6f, player 1 sees %.6f — they do not sum to 1", c.name, whoWins, c.n, p0, p1)
			}
			if p0 < 0 || p0 > 1 || p1 < 0 || p1 > 1 {
				t.Errorf("%s, player %d wins %d: %.6f / %.6f are not probabilities", c.name, whoWins, c.n, p0, p1)
			}
		}
	}
	// A match already won is certain, from both sides.
	if got := GnuBGGetME(6, 0, 7, 0, 1, 0, false); got != 1 {
		t.Errorf("player 0 wins the match: player 0's MWC = %v, want 1", got)
	}
	if got := GnuBGGetME(6, 0, 7, 1, 1, 0, false); got != 0 {
		t.Errorf("player 0 wins the match: player 1's MWC = %v, want 0", got)
	}
	if got := GnuBGGetME(0, 6, 7, 1, 1, 1, false); got != 1 {
		t.Errorf("player 1 wins the match: player 1's MWC = %v, want 1", got)
	}
}

// The horizon. GnuBGGetME CLAMPS an away score past MaxScore (64) to the
// table's last row rather than refusing it — the decision this file
// documents at the clamp itself: the lookup is called from bound methods
// where a panic costs far more than a wrong number, and callers are expected
// to filter (ConvertEMGLossToMWCLoss returns NaN, gammonnet.MatchState.IsValid
// refuses). The clamp is therefore a documented degradation and this test
// pins what it degrades TO, so a caller reading 0.5 for a 70-point match
// can find out here that the number is the corner of the table and not an
// equity: both scores saturate at index 63, and MET[63][63] is one half.
func TestGnuBGGetMEClampsBeyondTheHorizon(t *testing.T) {
	at64 := GnuBGGetME(0, 0, gnuBGMaxScore, 0, 1, 0, false)
	if at64 <= 0.5 || at64 >= 0.6 {
		t.Fatalf("64-away/64-away, winning one point: %v, want a little over one half", at64)
	}
	corner := float64(preCrawfordMET[gnuBGMaxScore-1][gnuBGMaxScore-1])
	for _, matchTo := range []int{gnuBGMaxScore + 2, 70, 99, 99999} {
		got := GnuBGGetME(0, 0, matchTo, 0, 1, 0, false)
		if got != corner {
			t.Errorf("matchTo=%d: %v, want the clamped corner MET[63][63] = %v", matchTo, got, corner)
		}
		if got == at64 {
			t.Errorf("matchTo=%d answers exactly what 64 does (%v) — the clamp is not the corner", matchTo, got)
		}
	}
	// One past the horizon on one side only — a 66-point match at 0-3, the
	// leader winning a point: 65-away (clamped to 64) against 63-away. n0
	// clamps, n1 does not, and the answer is the last row's second-to-last
	// cell — not the corner.
	got := GnuBGGetME(0, 3, gnuBGMaxScore+2, 0, 1, 0, false)
	if want := float64(preCrawfordMET[gnuBGMaxScore-1][gnuBGMaxScore-2]); got != want {
		t.Errorf("65-away (clamped) vs 63-away: %v, want MET[63][62] = %v", got, want)
	}
	if got == corner {
		t.Errorf("65-away vs 63-away answers the corner: the clamp hit both sides")
	}
	// The refusal lives one layer up, and must stay there: MaxScore is what
	// gammonnet.matchMaxAway reads.
	if MaxScore != gnuBGMaxScore {
		t.Errorf("MaxScore = %d, want %d", MaxScore, gnuBGMaxScore)
	}
}
