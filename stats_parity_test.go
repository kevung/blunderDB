package main

import (
	"encoding/json"
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
// Phase 01 values are intentionally wide (blunderDB counts forced moves and
// all No Double positions); they tighten after fiches 02–04.
type parityTolerances struct {
	TotalDecisions     int // all decisions combined
	CheckerDecisions   int // checker move count
	DoubleDecisions    int // cube doubling decisions (very wide pre-fiche 03)
	TakeDecisions      int // take/pass count
	CloseCubeDecisions int // is_close_cube=1 count vs XG/gnuBG close cube decisions
	PR                 float64
	MWCPct             float64 // percentage points
	Equity             float64 // EMG
	SnowieER           float64 // Snowie Error Rate; 0 means use hardcoded default
}

// tolPhase01 documents the blunderDB state before fiches 02–04 (pre forced-moves + pre close-cube).
// Kept for historical reference; not used in production tests.
//
// Explanations for wide values:
//
//	CheckerDecisions=45: bDB counts all checker positions (incl. forced). gnuBG
//	  ref stores unforced only; observed deltas up to 40 (38 forced + import gaps).
//	DoubleDecisions=100: bDB counts all No Doubles; XG/gnuBG only close ones
//	  (factor 2–6× overcounting pre-fiche 03).
//	PR=3.0: inflated denominator (all decisions vs unforced+close) depresses bDB PR
//	  by up to 2+ points vs XG/gnuBG reference.
var tolPhase01 = parityTolerances{
	TotalDecisions:     100,  // bDB includes forced + all No Doubles
	CheckerDecisions:   45,   // bDB includes forced moves + SGF import gaps
	DoubleDecisions:    100,  // bDB counts all No Doubles
	TakeDecisions:      3,    // takes/passes should align closely
	CloseCubeDecisions: 5,    // is_close_cube=1 vs XG/gnuBG close cube decisions
	PR:                 3.0,  // wide until denominator fixed (fiche 04)
	MWCPct:             3.5,  // percentage points
	Equity:             0.35, // EMG
	SnowieER:           2.0,  // very wide pre-fix
}

// tolPhase04 applies after fiche 04 (PR denominator fix). Both bDB and
// XG/gnuBG now count the same decisions: unforced checker + close cube.
//
//	PR=0.2  — tight for XG→XG and SGF→gnuBG (same engine). For XG-import
//	          vs gnuBG reference (different engines), call-site uses 1.0.
//	CheckerDecisions=10 — bDB and XG may differ by ≤6 on forced-move boundary
//	          classification; 10 leaves margin.
//	MWCPct=3.5 — SGF incomplete close-cube classification (only 2–3/7 cubes
//	          classified) and cross-engine equity differences cause ≥3.2 gaps.
//	          XG-vs-XG comparisons use tolPhase04XG (tighter, 1.0 pp).
//	Equity=0.5 — SGF close-cube equity is incomplete (only 2/7 classified);
//	          removing non-close cube equity increases the gap with gnuBG ref.
// tolPhase04 applies to SGF→gnuBG comparisons where structural gaps prevent tighter bounds.
//
//	MWCPct=3.5 — SGF incomplete close-cube classification (only 2–3/7 cubes
//	          classified) and cross-engine equity differences produce gaps up to 3.33 pp.
//	          Max observed: 3.33 pp (test.json SGF P1).
//	Equity=0.5 — SGF close-cube equity is incomplete (only 2/7 classified);
//	          max observed: 0.44 EMG (test.json SGF P1).
//	SnowieER=0.5 — two structural sources: (a) SGF forced moves without analysis excluded
//	          from bDB denominator but counted in gnuBG anTotalMoves (gap ≈20 moves);
//	          (b) cross-engine equity differences. Max observed: 0.34 (test.json P2).
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
}

// tolPhaseFinal applies to XG-vs-XG comparisons (same analysis engine).
// All values tightened to reflect the aligned denominator (fiches 02–04) and
// the cube error formula fix (fiche 06). Used in production since fiche 07.
//
//	CheckerDecisions=7 — max observed: 6 (Aachen P2). Residual from forced-move
//	          boundary classification at the 1-legal-move threshold.
//	PR=0.1    — max observed: 0.086 (Aachen P1). Very tight after denominator alignment.
//	MWCPct=1.0 — max observed: 0.984 pp (Aachen P1). Limit kept at 1.0 — one more
//	          pp would require per-decision eq2mwc conversion (out of scope).
//	Equity=0.05 — max observed: 0.015 EMG (Aachen P2). Analysis engine rounding only.
//	SnowieER=0.3 — no XG Snowie ER reference available yet; tolerance kept from fiche 05.
var tolPhaseFinal = parityTolerances{
	TotalDecisions:     5,
	CheckerDecisions:   7,    // tightened from 10; max observed diff: 6
	DoubleDecisions:    5,
	TakeDecisions:      3,
	CloseCubeDecisions: 5,
	PR:                 0.1,  // tightened from 0.2; max observed diff: 0.086
	MWCPct:             1.0,  // cube error fix (fiche 06); max observed: 0.984 pp
	Equity:             0.05, // tightened from 0.5; max observed diff: 0.015
	SnowieER:           0.3,  // no XG ref yet; maintained from fiche 05
}

// ── Diff helpers ─────────────────────────────────────────────────────────────

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
	db.db.QueryRow(`SELECT COALESCE(player1_name,''), COALESCE(player2_name,'') FROM match WHERE id = ?`, matchID).
		Scan(&player1Name, &player2Name)
	s, err = db.GetMatchDetailStats(matchID)
	if err != nil {
		t.Fatalf("GetMatchDetailStats: %v", err)
	}
	return
}

// countForcedChecker returns the number of is_forced=1 checker positions for
// the given player in the given match. player is 1 or -1 (blunderDB convention).
func countForcedChecker(db *Database, matchID int64, player int) int {
	var n int
	db.db.QueryRow(`
		SELECT COUNT(*)
		FROM analysis a
		JOIN position p ON p.id = a.position_id
		JOIN move mv ON mv.position_id = p.id
		JOIN game g ON g.id = mv.game_id
		WHERE g.match_id = ? AND mv.player = ? AND p.decision_type = 0 AND a.is_forced = 1`,
		matchID, player).Scan(&n)
	return n
}

// countCloseCube returns the number of is_close_cube=1 cube positions for
// the given player in the given match.
func countCloseCube(db *Database, matchID int64, player int) int {
	var n int
	db.db.QueryRow(`
		SELECT COUNT(*)
		FROM analysis a
		JOIN position p ON p.id = a.position_id
		JOIN move mv ON mv.position_id = p.id
		JOIN game g ON g.id = mv.game_id
		WHERE g.match_id = ? AND mv.player = ? AND p.decision_type = 1 AND a.is_close_cube = 1`,
		matchID, player).Scan(&n)
	return n
}

// ── Comparison logic ─────────────────────────────────────────────────────────

// compareXGRef compares blunderDB stats against an XG reference player.
// After fiches 02–04: bDB counts the same decisions as XG (unforced checker + close cube).
func compareXGRef(t *testing.T, prefix string, ref *refPlayer, bdb MatchPlayerDetailStats, tol parityTolerances) {
	t.Helper()
	t.Logf("--- %s vs XG reference ---", prefix)
	diffInt(t, prefix+" total_decisions", ref.TotalDecisions, bdb.TotalDecisions, tol.TotalDecisions)
	diffInt(t, prefix+" checker_unforced", ref.CheckerUnforced, bdb.CheckerDecisions, tol.CheckerDecisions)
	diffInt(t, prefix+" double_decisions", ref.DoubleDecisions, bdb.DoubleDecisions, tol.DoubleDecisions)
	diffInt(t, prefix+" take_decisions", ref.TakeDecisions, bdb.TakeDecisions, tol.TakeDecisions)
	diffFloat(t, prefix+" PR", ref.PR, bdb.PR, tol.PR)
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
// After fiche 04, bdb.CheckerDecisions = unforced only, so it aligns with
// checker_unforced. checker_total (including forced) is no longer compared here.
func compareGnuBGRef(t *testing.T, prefix string, ref *refPlayer, bdb MatchPlayerDetailStats, tol parityTolerances) {
	t.Helper()
	t.Logf("--- %s vs gnuBG reference ---", prefix)
	// After fiche 04: bDB.CheckerDecisions = unforced only → compare directly.
	diffInt(t, prefix+" checker_unforced", ref.CheckerUnforced, bdb.CheckerDecisions, tol.CheckerDecisions)
	// checker_total was compared here before fiche 04. After the fix, bDB counts
	// only unforced moves, so checker_total (incl. forced) diverges by forced_count.
	// The forced count cross-check is done at call-site via countForcedChecker.
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
	diffFloat(t, prefix+" checker_pr_xg_500", ref.CheckerPRXG500, bdb.PRChecker, tol.PR)
	// Snowie ER: structural tolerance. Two irreducible sources of divergence:
	//   (a) SGF forced moves without analysis are excluded from blunderDB's denominator
	//       but counted in gnuBG's anTotalMoves → denominator gap up to ~20 moves.
	//   (b) Cross-engine equity differences (XG vs gnuBG) add up to ~0.3 per player.
	//   Max observed: 0.34 (test.json P2). Structural; documented in fiche 05 findings.
	snowieGnuTol := tol.SnowieER
	if snowieGnuTol == 0 {
		snowieGnuTol = 0.5 // fallback if not set
	}
	diffFloat(t, prefix+" snowie_er", ref.SnowieER, bdb.SnowieER, snowieGnuTol)
}

// ── Main test ─────────────────────────────────────────────────────────────────

func TestStatsParity(t *testing.T) {
	fixtures := []string{
		"testdata/stats_reference/aachen-double-7pt.json",
		"testdata/stats_reference/test.json",
		"testdata/stats_reference/charlot1-charlot2.json",
	}
	// tolPhase04 covers SGF→gnuBG comparisons (structural gaps prevent tightening).
	// tolPhaseFinal covers XG→XG comparisons (same engine, tightest achievable).
	tol := tolPhase04

	for _, jsonPath := range fixtures {
		t.Run(filepath.Base(jsonPath), func(t *testing.T) {
			ref := loadRefMatch(t, jsonPath)

			// ── XG import ─────────────────────────────────────────────────
			if ref.MatchFile != "" {
				if _, err := os.Stat(ref.MatchFile); err == nil {
					p1n, _, matchID, db, bdbStats := importAndStats(t, ref.MatchFile)
					t.Logf("XG import: %s  P1=%q", filepath.Base(ref.MatchFile), p1n)

					if ref.XG != nil {
						// XG-vs-XG: same engine, tightest tolerances (fiche 07 final).
						xgTol := tolPhaseFinal
						if rp := ref.XG["player1"]; rp != nil {
							compareXGRef(t, "XG/P1", rp, bdbStats.Player1, xgTol)
							if rp.CheckerForced != nil {
								gotForced := countForcedChecker(db, matchID, 1)
								diffInt(t, "XG/P1 checker_forced_count", rp.CheckerForced, gotForced, xgTol.CheckerDecisions)
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
								gotClose := countCloseCube(db, matchID, 1)
								diffInt(t, "XG/P1 cube_close_count", &wantClose, gotClose, xgTol.CloseCubeDecisions)
							}
						}
						if rp := ref.XG["player2"]; rp != nil {
							compareXGRef(t, "XG/P2", rp, bdbStats.Player2, xgTol)
							if rp.CheckerForced != nil {
								gotForced := countForcedChecker(db, matchID, -1)
								diffInt(t, "XG/P2 checker_forced_count", rp.CheckerForced, gotForced, xgTol.CheckerDecisions)
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
								gotClose := countCloseCube(db, matchID, -1)
								diffInt(t, "XG/P2 cube_close_count", &wantClose, gotClose, xgTol.CloseCubeDecisions)
							}
						}
					}
					if ref.GnuBG != nil {
						// XG-imported data vs gnuBG reference: different analysis
						// engines produce different equity values, so the PR tolerance
						// must be wider than the XG→XG or SGF→gnuBG paths.
						xgVsGnuTol := tol
						xgVsGnuTol.PR = 1.0
						if rp := ref.GnuBG["player1"]; rp != nil {
							compareGnuBGRef(t, "XG→gnuBGref/P1", rp, bdbStats.Player1, xgVsGnuTol)
							if rp.CheckerForced != nil {
								gotForced := countForcedChecker(db, matchID, 1)
								// Forced count is a "best effort" diagnostic; XG moves
								// include all alternatives, so detection is accurate here.
								diffInt(t, "XG→gnuBGref/P1 checker_forced_count", rp.CheckerForced, gotForced, 45)
							}
						}
						if rp := ref.GnuBG["player2"]; rp != nil {
							compareGnuBGRef(t, "XG→gnuBGref/P2", rp, bdbStats.Player2, xgVsGnuTol)
							if rp.CheckerForced != nil {
								gotForced := countForcedChecker(db, matchID, -1)
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
								gotForced := countForcedChecker(db, matchID, 1)
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
								gotClose := countCloseCube(db, matchID, 1)
								diffInt(t, "SGF/P1 cube_close_count", &wantClose, gotClose, tol.CloseCubeDecisions)
							}
						}
						if rp := ref.GnuBG["player2"]; rp != nil {
							compareGnuBGRef(t, "SGF/P2", rp, bdbStats.Player2, tol)
							if rp.CheckerForced != nil {
								gotForced := countForcedChecker(db, matchID, -1)
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
								gotClose := countCloseCube(db, matchID, -1)
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
