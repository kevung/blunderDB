package database

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// ── Reference JSON types ──────────────────────────────────────────────────────

// refPlayer covers both XG and gnuBG JSON schemas (see SCHEMA.md).
// Pointer fields are nil when the value was not measured / not available.
type refPlayer struct {
	// XG fields
	PR                  *float64 `json:"pr"`
	SnowieErrorRate     *float64 `json:"snowie_error_rate"`
	TotalDecisions      *int     `json:"total_decisions"`
	CheckerUnforced     *int     `json:"checker_unforced"`
	DoubleDecisions     *int     `json:"double_decisions"`
	TakeDecisions       *int     `json:"take_decisions"`
	PassDecisions       *int     `json:"pass_decisions"`
	TotalEquityErrorEMG *float64 `json:"total_equity_error_emg"`
	TotalMWCLossPct     *float64 `json:"total_mwc_loss_pct"`
	CheckerMWCLossPct   *float64 `json:"checker_mwc_loss_pct"`
	CheckerEquityEMG    *float64 `json:"checker_equity_error_emg"`
	DoubleEquityEMG     *float64 `json:"double_equity_error_emg"`
	TakeEquityEMG       *float64 `json:"take_equity_error_emg"`
	// gnuBG-only fields
	CheckerTotal   *int     `json:"checker_total"`
	CheckerForced  *int     `json:"checker_forced"`
	CheckerPRXG500 *float64 `json:"checker_pr_xg_500"`
	CubeEquityEMG  *float64 `json:"cube_equity_error_emg"`
	CubeMWCLossPct *float64 `json:"cube_mwc_loss_pct"`
	TotalCube      *int     `json:"total_cube"`
	SnowieER       *float64 `json:"snowie_er"`
}

type refMatch struct {
	MatchFile string                `json:"match_file"`
	SGFFile   string                `json:"sgf_file"`
	Players   [2]string             `json:"players"`
	XG        map[string]*refPlayer `json:"xg"`
	GnuBG     map[string]*refPlayer `json:"gnubg"`
	Notes     string                `json:"notes"`
}

func loadRefMatch(t *testing.T, path string) refMatch {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("loadRefMatch %s: %v", path, err)
	}
	var m refMatch
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("loadRefMatch %s unmarshal: %v", path, err)
	}
	return m
}

// ── Tolerances ───────────────────────────────────────────────────────────────

// parityTolerances sets per-metric acceptance thresholds.
type parityTolerances struct {
	TotalDecisions     int // all decisions combined
	CheckerDecisions   int // checker move count
	DoubleDecisions    int // cube doubling decisions
	TakeDecisions      int // take/pass count
	CloseCubeDecisions int // is_close_cube=1 count vs XG/gnuBG close cube decisions
	PR                 float64
	MWCPct             float64 // percentage points
	Equity             float64 // EMG
	SnowieER           float64 // Snowie Error Rate; 0 means use hardcoded default
	// CheckerPRGnuBG bounds blunderDB's checker PR against gnuBG's: both
	// divide the checker error by the unforced plays, but blunderDB counts
	// them as XG does (engine.IsForcedChecker) while gnuBG counts every
	// position with more than one legal play, so the denominators differ by
	// the plays whose candidates all tie.
	CheckerPRGnuBG float64
}

// tolPhase04 applies to SGF→gnuBG comparisons, where structural gaps prevent
// tighter bounds. Both sides count unforced checker + close cube decisions.
//
//	PR=0.2  — same engine; an XG import vs a gnuBG reference uses 1.0 at the call site.
//	CheckerDecisions=10 — forced-move boundary classification differs by ≤6.
//	MWCPct=3.5 — SGF classifies only 2–3/7 close cubes, plus cross-engine
//	          equity; max observed 3.33 pp (test.json SGF P1).
//	Equity=0.5 — same SGF close-cube gap; max observed 0.44 EMG.
//	SnowieER=0.5 — SGF forced moves without analysis leave bDB's denominator
//	          but stay in gnuBG's anTotalMoves (≈20 moves), plus cross-engine
//	          equity; max observed 0.34 (test.json P2).
var tolPhase04 = parityTolerances{
	TotalDecisions:     5,   // aligned denominator
	CheckerDecisions:   10,  // unforced checker; ≤6 boundary diff vs XG
	DoubleDecisions:    5,   // close doubles only
	TakeDecisions:      3,   // takes/passes: always counted
	CloseCubeDecisions: 5,   // some SGF cube positions lack equity data
	PR:                 0.2, // aligned denominator: very close to XG/gnuBG
	MWCPct:             3.5, // structural SGF close-cube gap; irreducible at this stage
	Equity:             0.5, // EMG: wider for SGF incomplete close-cube equity
	SnowieER:           0.5, // structural SGF forced-without-analysis gap
	CheckerPRGnuBG:     0.4, // max observed 0.33 (test.json SGF P2)
}

// tolXG applies to XG-vs-XG comparisons: blunderDB counts decisions and sums
// errors as XG does (ADR-less rule set documented on countedExpr,
// engine.IsForcedChecker and engine.ComputeIsCloseCube), so the counts are
// exact. The float tolerances are those of the reference itself: XG prints
// PR to 0.01, equities to 0.001 and MWC to 0.01 pp, while blunderDB sums
// errors stored to the millipoint — a sum of ~150 roundings that may drift
// past the last printed digit.
var tolXG = parityTolerances{
	PR:       0.02,
	MWCPct:   0.06,
	Equity:   0.004,
	SnowieER: 0.3, // no XG reference
}

// xgResidual is a known, explained gap between blunderDB and XG on one
// player: checker is XG's unforced count minus blunderDB's, and the float
// fields widen tolXG for the metrics that gap moves. Each one names its cause,
// and each cause lies outside what blunderDB reads from the file:
//
//   - unstored: a bear-off XG counts as a decision while the stored candidates
//     cover every legal play at one equity — XG judged it on evaluations the
//     file does not keep.
//   - unanalysed: XG stores ErrMove = -1000 for a play it never scored and
//     leaves it out; xgparser's light API does not expose ErrMove, so
//     blunderDB counts and scores the play from the stored candidates.
//   - cubeMWC: the cube errors' equities agree with XG's, their conversion to
//     MWC does not (blunderDB converts at the current cube's value from the
//     decision maker's side; XG's own cube MWC is lower). Not Crawford: the
//     MET lookup already takes the post-Crawford table at a 1-away score.
type xgResidual struct {
	checker int
	pr      float64
	equity  float64
	mwc     float64
	reason  string
}

// xgResiduals keys a fixture's player ("file.json/P1") to its residual.
var xgResiduals = map[string]xgResidual{
	"aachen-double-7pt.json/P1":    {0, 0, 0, 0.9, "cubeMWC (total MWC 20.71 vs 19.85, checker MWC within tolerance)"},
	"aachen-double-7pt.json/P2":    {-1, 0.03, 0.017, 0.45, "unanalysed (1 play)"},
	"charlot1-charlot2.json/P2":    {1, 0.02, 0, 0, "unstored (1 bear-off)"},
	"marseille-round4-7pt.json/P1": {-1, 0.09, 0.005, 0, "unanalysed (1 play)"},
	"marseille-round4-7pt.json/P2": {0, 0, 0, 0.85, "cubeMWC (total MWC 20.65 vs 19.81, checker MWC within tolerance)"},
	"test.json/P1":                 {0, 0, 0.005, 0.1, "cubeMWC (total MWC 30.12 vs 30.03, checker MWC within tolerance)"},
}

// ── Diff helpers ─────────────────────────────────────────────────────────────

func intPtr(v int) *int { return &v }

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func diffFloat(t *testing.T, label string, want *float64, got, tol float64) {
	t.Helper()
	if want == nil {
		t.Logf("  %-40s  ref=n/a   got=%+.4f", label, got)
		return
	}
	diff := math.Abs(got - *want)
	mark := "  "
	if diff > tol {
		mark = "!!"
	}
	t.Logf("%s %-40s  ref=%+.4f  got=%+.4f  diff=%.4f  tol=±%.4f", mark, label, *want, got, diff, tol)
	if diff > tol {
		t.Errorf("%s: got %.4f, ref %.4f (diff %.4f > tol %.4f)", label, got, *want, diff, tol)
	}
}

func diffInt(t *testing.T, label string, want *int, got, tol int) {
	t.Helper()
	if want == nil {
		t.Logf("  %-40s  ref=n/a  got=%d", label, got)
		return
	}
	diff := got - *want
	if diff < 0 {
		diff = -diff
	}
	mark := "  "
	if diff > tol {
		mark = "!!"
	}
	t.Logf("%s %-40s  ref=%d  got=%d  diff=%d  tol=±%d", mark, label, *want, got, diff, tol)
	if diff > tol {
		t.Errorf("%s: got %d, ref %d (diff %d > tol %d)", label, got, *want, diff, tol)
	}
}

// ── Import helpers ────────────────────────────────────────────────────────────

func importAndStats(t *testing.T, file string) (player1Name, player2Name string, matchID int64, db *Database, s *MatchDetailStats) {
	t.Helper()
	db = newTestDB(t)
	var err error
	switch filepath.Ext(file) {
	case ".xg":
		matchID, err = db.ImportXGMatch(file)
	case ".sgf":
		matchID, err = db.ImportGnuBGMatch(file)
	default:
		t.Skipf("unsupported file type: %s", file)
	}
	if err != nil {
		t.Fatalf("import %s: %v", file, err)
	}
	if err := db.db.QueryRow(`SELECT COALESCE(player1_name,''), COALESCE(player2_name,'') FROM match WHERE id = ?`, matchID).
		Scan(&player1Name, &player2Name); err != nil {
		t.Fatal(err)
	}
	s, err = db.GetMatchDetailStats(matchID)
	if err != nil {
		t.Fatalf("GetMatchDetailStats: %v", err)
	}
	return
}

// countForcedChecker returns the number of is_forced=1 checker positions for
// the given player in the given match. player is 1 or -1 (blunderDB convention).
func countForcedChecker(t testing.TB, db *Database, matchID int64, player int) int {
	t.Helper()
	var n int
	if err := db.db.QueryRow(`
		SELECT COUNT(*)
		FROM analysis a
		JOIN position p ON p.id = a.position_id
		JOIN move mv ON mv.position_id = p.id
		JOIN game g ON g.id = mv.game_id
		WHERE g.match_id = ? AND mv.player = ? AND p.decision_type = 0 AND a.is_forced = 1`,
		matchID, player).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// countCloseCube returns the number of is_close_cube=1 cube positions for
// the given player in the given match.
func countCloseCube(t testing.TB, db *Database, matchID int64, player int) int {
	t.Helper()
	var n int
	if err := db.db.QueryRow(`
		SELECT COUNT(*)
		FROM analysis a
		JOIN position p ON p.id = a.position_id
		JOIN move mv ON mv.position_id = p.id
		JOIN game g ON g.id = mv.game_id
		WHERE g.match_id = ? AND mv.player = ? AND p.decision_type = 1 AND a.is_close_cube = 1`,
		matchID, player).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// ── Comparison logic ─────────────────────────────────────────────────────────

// compareXGRef compares blunderDB stats against an XG reference player.
// Both count the same decisions (unforced checker + close cube).
func compareXGRef(t *testing.T, prefix string, ref *refPlayer, bdb MatchPlayerDetailStats, tol parityTolerances) {
	t.Helper()
	t.Logf("--- %s vs XG reference ---", prefix)
	diffInt(t, prefix+" total_decisions", ref.TotalDecisions, bdb.TotalDecisions, tol.TotalDecisions)
	diffInt(t, prefix+" checker_unforced", ref.CheckerUnforced, bdb.CheckerDecisions, tol.CheckerDecisions)
	diffInt(t, prefix+" double_decisions", ref.DoubleDecisions, bdb.DoubleDecisions, tol.DoubleDecisions)
	diffInt(t, prefix+" take_decisions", ref.TakeDecisions, bdb.TakeDecisions, tol.TakeDecisions)
	diffFloat(t, prefix+" PR", ref.PR, bdb.PR, tol.PR)
	diffFloat(t, prefix+" double_equity_emg", ref.DoubleEquityEMG, bdb.DoubleEquityError, tol.Equity)
	diffFloat(t, prefix+" take_equity_emg", ref.TakeEquityEMG, bdb.TakeEquityError, tol.Equity)
	// Snowie ER: use tol.SnowieER (no XG reference data available yet for these fixtures).
	snowieTol := tol.SnowieER
	if snowieTol == 0 {
		snowieTol = 0.3 // fallback if not set
	}
	diffFloat(t, prefix+" snowie_er", ref.SnowieErrorRate, bdb.SnowieER, snowieTol)
	diffFloat(t, prefix+" total_equity_emg", ref.TotalEquityErrorEMG, bdb.TotalEquityError, tol.Equity)
	diffFloat(t, prefix+" checker_equity_emg", ref.CheckerEquityEMG, bdb.CheckerEquityError, tol.Equity)
	if ref.TotalMWCLossPct != nil {
		mwcRef := *ref.TotalMWCLossPct
		diffFloat(t, prefix+" total_mwc_pct", &mwcRef, bdb.MWCLoss*100, tol.MWCPct)
	}
	if ref.CheckerMWCLossPct != nil {
		mwcRef := *ref.CheckerMWCLossPct
		diffFloat(t, prefix+" checker_mwc_pct", &mwcRef, bdb.CheckerMWCLoss*100, tol.MWCPct)
	}
}

// compareGnuBGRef compares blunderDB stats against a gnuBG reference player.
// bdb.CheckerDecisions counts unforced moves only, so it is compared with
// checker_unforced; the forced count is cross-checked at the call site.
func compareGnuBGRef(t *testing.T, prefix string, ref *refPlayer, bdb MatchPlayerDetailStats, tol parityTolerances) {
	t.Helper()
	t.Logf("--- %s vs gnuBG reference ---", prefix)
	diffInt(t, prefix+" checker_unforced", ref.CheckerUnforced, bdb.CheckerDecisions, tol.CheckerDecisions)
	diffFloat(t, prefix+" total_equity_emg", ref.TotalEquityErrorEMG, bdb.TotalEquityError, tol.Equity)
	if ref.TotalMWCLossPct != nil {
		mwcRef := *ref.TotalMWCLossPct
		diffFloat(t, prefix+" total_mwc_pct", &mwcRef, bdb.MWCLoss*100, tol.MWCPct)
	}
	// Checker-only equity (gnuBG ref: checker_equity_error_emg)
	diffFloat(t, prefix+" checker_equity_emg", ref.CheckerEquityEMG, bdb.CheckerEquityError, tol.Equity)
	if ref.CheckerMWCLossPct != nil {
		mwcRef := *ref.CheckerMWCLossPct
		diffFloat(t, prefix+" checker_mwc_pct", &mwcRef, bdb.CheckerMWCLoss*100, tol.MWCPct)
	}
	// checker_pr_xg_500: compare against bDB PRChecker (same 500-factor formula).
	diffFloat(t, prefix+" checker_pr_xg_500", ref.CheckerPRXG500, bdb.PRChecker, max(tol.PR, tol.CheckerPRGnuBG))
	// Snowie ER: structural tolerance. Two irreducible sources of divergence:
	//   (a) SGF forced moves without analysis are excluded from blunderDB's denominator
	//       but counted in gnuBG's anTotalMoves → denominator gap up to ~20 moves.
	//   (b) Cross-engine equity differences (XG vs gnuBG) add up to ~0.3 per player.
	//   Max observed: 0.34 (test.json P2).
	snowieGnuTol := tol.SnowieER
	if snowieGnuTol == 0 {
		snowieGnuTol = 0.5 // fallback if not set
	}
	diffFloat(t, prefix+" snowie_er", ref.SnowieER, bdb.SnowieER, snowieGnuTol)
}

// ── Main test ─────────────────────────────────────────────────────────────────

func TestStatsParity(t *testing.T) {
	t.Parallel()
	fixtures := []string{
		"testdata/stats_reference/aachen-double-7pt.json",
		"testdata/stats_reference/test.json",
		"testdata/stats_reference/charlot1-charlot2.json",
		"testdata/stats_reference/marseille-round4-7pt.json",
		"testdata/stats_reference/issue595-kev-gammonnet-7pt.json",
	}
	// tolPhase04 covers SGF→gnuBG comparisons (structural gaps prevent tightening).
	// tolXG covers XG→XG comparisons: exact counts, print-precision floats.
	tol := tolPhase04

	for _, jsonPath := range fixtures {
		t.Run(filepath.Base(jsonPath), func(t *testing.T) {
			// One database per fixture, nothing shared: this test was the
			// package's critical path.
			t.Parallel()
			ref := loadRefMatch(t, jsonPath)

			// ── XG import ─────────────────────────────────────────────────
			if ref.MatchFile != "" {
				if _, err := os.Stat(ref.MatchFile); err == nil {
					p1n, _, matchID, db, bdbStats := importAndStats(t, ref.MatchFile)
					t.Logf("XG import: %s  P1=%q", filepath.Base(ref.MatchFile), p1n)

					if ref.XG != nil {
						for i, key := range []string{"player1", "player2"} {
							rp := ref.XG[key]
							if rp == nil {
								continue
							}
							label := fmt.Sprintf("XG/P%d", i+1)
							player, bdb := 1, bdbStats.Player1
							if i == 1 {
								player, bdb = -1, bdbStats.Player2
							}
							want, xgTol := *rp, tolXG
							if r, ok := xgResiduals[fmt.Sprintf("%s/P%d", filepath.Base(jsonPath), i+1)]; ok {
								t.Logf("%s known residual: %+d checker decision(s), %s", label, r.checker, r.reason)
								want.CheckerUnforced = intPtr(*rp.CheckerUnforced - r.checker)
								want.TotalDecisions = intPtr(*rp.TotalDecisions - r.checker)
								xgTol.PR = max(xgTol.PR, r.pr)
								xgTol.Equity = max(xgTol.Equity, r.equity)
								xgTol.MWCPct = max(xgTol.MWCPct, r.mwc)
							}
							compareXGRef(t, label, &want, bdb, xgTol)
							if rp.DoubleDecisions != nil || rp.TakeDecisions != nil || rp.PassDecisions != nil {
								wantClose := derefInt(rp.DoubleDecisions) + derefInt(rp.TakeDecisions) + derefInt(rp.PassDecisions)
								gotClose := countCloseCube(t, db, matchID, player)
								diffInt(t, label+" cube_close_count", &wantClose, gotClose, 0)
							}
						}
					}
					if ref.GnuBG != nil {
						// XG-imported data vs gnuBG reference: different analysis
						// engines produce different equity values, so the PR tolerance
						// must be wider than the XG→XG or SGF→gnuBG paths.
						xgVsGnuTol := tol
						xgVsGnuTol.PR = 1.0
						xgVsGnuTol.CheckerPRGnuBG = 1.3 // max observed 1.22 (test.json P2)
						if rp := ref.GnuBG["player1"]; rp != nil {
							compareGnuBGRef(t, "XG→gnuBGref/P1", rp, bdbStats.Player1, xgVsGnuTol)
							if rp.CheckerForced != nil {
								gotForced := countForcedChecker(t, db, matchID, 1)
								// Forced count is a "best effort" diagnostic; XG moves
								// include all alternatives, so detection is accurate here.
								diffInt(t, "XG→gnuBGref/P1 checker_forced_count", rp.CheckerForced, gotForced, 45)
							}
						}
						if rp := ref.GnuBG["player2"]; rp != nil {
							compareGnuBGRef(t, "XG→gnuBGref/P2", rp, bdbStats.Player2, xgVsGnuTol)
							if rp.CheckerForced != nil {
								gotForced := countForcedChecker(t, db, matchID, -1)
								diffInt(t, "XG→gnuBGref/P2 checker_forced_count", rp.CheckerForced, gotForced, 45)
							}
						}
					}
				} else {
					t.Logf("SKIP XG import: %s not found", ref.MatchFile)
				}
			}

			// ── SGF import ────────────────────────────────────────────────
			if ref.SGFFile != "" {
				if _, err := os.Stat(ref.SGFFile); err == nil {
					p1n, _, matchID, db, bdbStats := importAndStats(t, ref.SGFFile)
					t.Logf("SGF import: %s  P1=%q", filepath.Base(ref.SGFFile), p1n)

					if ref.GnuBG != nil {
						if rp := ref.GnuBG["player1"]; rp != nil {
							compareGnuBGRef(t, "SGF/P1", rp, bdbStats.Player1, tol)
							if rp.CheckerForced != nil {
								gotForced := countForcedChecker(t, db, matchID, 1)
								// SGF forced count is a "best effort" diagnostic: gnuBG
								// omits alternatives for forced moves, so some forced
								// positions have no analysis row and are not detected.
								diffInt(t, "SGF/P1 checker_forced_count", rp.CheckerForced, gotForced, 45)
							}
							if rp.DoubleDecisions != nil || rp.TakeDecisions != nil || rp.PassDecisions != nil {
								wantClose := 0
								if rp.DoubleDecisions != nil {
									wantClose += *rp.DoubleDecisions
								}
								if rp.TakeDecisions != nil {
									wantClose += *rp.TakeDecisions
								}
								if rp.PassDecisions != nil {
									wantClose += *rp.PassDecisions
								}
								gotClose := countCloseCube(t, db, matchID, 1)
								diffInt(t, "SGF/P1 cube_close_count", &wantClose, gotClose, tol.CloseCubeDecisions)
							}
						}
						if rp := ref.GnuBG["player2"]; rp != nil {
							compareGnuBGRef(t, "SGF/P2", rp, bdbStats.Player2, tol)
							if rp.CheckerForced != nil {
								gotForced := countForcedChecker(t, db, matchID, -1)
								diffInt(t, "SGF/P2 checker_forced_count", rp.CheckerForced, gotForced, 45)
							}
							if rp.DoubleDecisions != nil || rp.TakeDecisions != nil || rp.PassDecisions != nil {
								wantClose := 0
								if rp.DoubleDecisions != nil {
									wantClose += *rp.DoubleDecisions
								}
								if rp.TakeDecisions != nil {
									wantClose += *rp.TakeDecisions
								}
								if rp.PassDecisions != nil {
									wantClose += *rp.PassDecisions
								}
								gotClose := countCloseCube(t, db, matchID, -1)
								diffInt(t, "SGF/P2 cube_close_count", &wantClose, gotClose, tol.CloseCubeDecisions)
							}
						}
					}
				} else {
					t.Logf("SKIP SGF import: %s not found", ref.SGFFile)
				}
			}
		})
	}
}
