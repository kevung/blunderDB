package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// linkPositionToMatch writes the move -> game -> match chain that is the
// ONLY thing tying a Position to a Match (a position row knows nothing of
// where it came from). Written in raw SQL rather than through the importers
// because the chain is exactly what the scoped query walks, and going
// through an import would test the importer instead.
func linkPositionToMatch(t *testing.T, d *Database, matchName string, positionIDs ...int64) int64 {
	t.Helper()

	res, err := d.db.Exec(`INSERT INTO match (player1_name, player2_name, match_length) VALUES (?, 'Adversaire', 7)`, matchName)
	if err != nil {
		t.Fatalf("INSERT match: %v", err)
	}
	matchID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("match id: %v", err)
	}

	res, err = d.db.Exec(`INSERT INTO game (match_id, game_number) VALUES (?, 1)`, matchID)
	if err != nil {
		t.Fatalf("INSERT game: %v", err)
	}
	gameID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("game id: %v", err)
	}

	for i, id := range positionIDs {
		if _, err := d.db.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, player) VALUES (?, ?, 'CheckerMove', ?, 1)`,
			gameID, i+1, id); err != nil {
			t.Fatalf("INSERT move: %v", err)
		}
	}
	return matchID
}

// TestAnalyzeMatchWithGammonNetStaysInsideItsMatch is ADR-0045 §8's own
// promise: saving a transcription analyses THAT match and sweeps nothing
// else. Two matches, both unanalysed; after the scoped batch the second is
// exactly as it was, and the library-wide count of positions without an
// analysis has fallen by precisely the first match's share.
func TestAnalyzeMatchWithGammonNetStaysInsideItsMatch(t *testing.T) {
	t.Parallel()
	d := newBatchTestDB(t)

	save := func(white, black int) int64 {
		t.Helper()
		p := racePosition(white, black, domain.White)
		id, err := d.SavePosition(&p)
		if err != nil {
			t.Fatalf("SavePosition: %v", err)
		}
		return id
	}

	targetA, targetB := save(6, 19), save(8, 17)
	other := save(4, 21)

	matchID := linkPositionToMatch(t, d, "Ciblé", targetA, targetB)
	linkPositionToMatch(t, d, "Épargné", other)

	before, err := d.CountPositionsWithoutAnalysis()
	if err != nil {
		t.Fatalf("CountPositionsWithoutAnalysis: %v", err)
	}
	if before != 3 {
		t.Fatalf("CountPositionsWithoutAnalysis = %d, want 3", before)
	}
	scoped, err := d.CountMatchPositionsToAnalyze(matchID)
	if err != nil {
		t.Fatalf("CountMatchPositionsToAnalyze: %v", err)
	}
	if scoped != 2 {
		t.Fatalf("CountMatchPositionsToAnalyze = %d, want 2", scoped)
	}

	// 0-ply, one job: the test's own choice, for speed. The batch's
	// behaviour does not depend on either (CLI_USAGE.md: identical writes
	// whatever --jobs says).
	summary, err := d.AnalyzeMatchWithGammonNet(context.Background(), matchID, 0, 0, 0, 1, nil, nil)
	if err != nil {
		t.Fatalf("AnalyzeMatchWithGammonNet: %v", err)
	}
	if summary.Evaluated != 2 || summary.Failed != 0 {
		t.Fatalf("summary = %+v, want two evaluated and no failure", summary)
	}

	for _, id := range []int64{targetA, targetB} {
		a, err := d.LoadAnalysis(id)
		if err != nil {
			t.Fatalf("LoadAnalysis(%d): %v", id, err)
		}
		if a == nil {
			t.Fatalf("position %d of the targeted match was left without an analysis", id)
		}
	}

	// The other match is untouched — the whole point of the scope.
	// LoadAnalysis reports "no analysis" as sql.ErrNoRows, which is the
	// answer this assertion wants.
	if a, err := d.LoadAnalysis(other); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("a position of another match was analysed (analysis %+v, err %v)", a, err)
	}

	after, err := d.CountPositionsWithoutAnalysis()
	if err != nil {
		t.Fatalf("CountPositionsWithoutAnalysis (after): %v", err)
	}
	if after != before-scoped {
		t.Errorf("CountPositionsWithoutAnalysis after the scoped run = %d, want %d (before %d minus the match's %d)", after, before-scoped, before, scoped)
	}
}

// TestAnalyzeMatchWithGammonNetFillsOnlyTheGap is ADR-0013 inside the scope:
// a position of the targeted match that already carries an analysis — an XG
// one here, but the rule ignores the engine — is not recomputed, which is
// also what makes re-saving a corrected transcription cheap: only the
// positions the correction created are left to analyse.
func TestAnalyzeMatchWithGammonNetFillsOnlyTheGap(t *testing.T) {
	t.Parallel()
	d := newBatchTestDB(t)

	analysed := racePosition(6, 19, domain.White)
	analysedID, err := d.SavePosition(&analysed)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	if err := d.SaveAnalysis(analysedID, PositionAnalysis{
		PositionID:            int(analysedID),
		AnalysisType:          "DoublingCube",
		AnalysisEngineVersion: "XG",
		DoublingCubeAnalysis:  &DoublingCubeAnalysis{AnalysisDepth: "4-ply", AnalysisEngine: "XG", BestCubeAction: "No Double"},
	}); err != nil {
		t.Fatalf("SaveAnalysis: %v", err)
	}

	fresh := racePosition(8, 17, domain.White)
	freshID, err := d.SavePosition(&fresh)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}

	// The same position twice in the match, as a repeated board or a
	// double and its take would be: the DISTINCT in the query is what stops
	// it being analysed twice.
	matchID := linkPositionToMatch(t, d, "Retouché", analysedID, freshID, freshID)

	summary, err := d.AnalyzeMatchWithGammonNet(context.Background(), matchID, 0, 0, 0, 1, nil, nil)
	if err != nil {
		t.Fatalf("AnalyzeMatchWithGammonNet: %v", err)
	}
	if summary.Processed() != 1 {
		t.Fatalf("summary = %+v, want exactly one position processed (the gap), not the already-analysed one nor a duplicate", summary)
	}

	got, err := d.LoadAnalysis(analysedID)
	if err != nil {
		t.Fatalf("LoadAnalysis(analysed): %v", err)
	}
	if got.DoublingCubeAnalysis == nil || got.DoublingCubeAnalysis.AnalysisEngine != "XG" {
		t.Errorf("the pre-existing XG analysis was overwritten: %+v", got.DoublingCubeAnalysis)
	}

	filled, err := d.LoadAnalysis(freshID)
	if err != nil {
		t.Fatalf("LoadAnalysis(fresh): %v", err)
	}
	if filled == nil || filled.DoublingCubeAnalysis == nil {
		t.Fatal("the gap of the targeted match was not filled")
	}
	if filled.DoublingCubeAnalysis.AnalysisEngine != gammonnet.EngineVersion {
		t.Errorf("AnalysisEngine = %q, want %q", filled.DoublingCubeAnalysis.AnalysisEngine, gammonnet.EngineVersion)
	}

	// Nothing left, so a second save of the same match is a no-op.
	left, err := d.CountMatchPositionsToAnalyze(matchID)
	if err != nil {
		t.Fatalf("CountMatchPositionsToAnalyze: %v", err)
	}
	if left != 0 {
		t.Errorf("CountMatchPositionsToAnalyze after the run = %d, want 0", left)
	}
}

// TestAnalyzeMatchWithGammonNetUnknownMatch: a match id with no game behind
// it names no position, so the batch is a quiet no-op rather than a
// library-wide sweep — the failure mode a "0 means everything" sentinel
// would have.
func TestAnalyzeMatchWithGammonNetUnknownMatch(t *testing.T) {
	t.Parallel()
	d := newBatchTestDB(t)

	p := racePosition(6, 19, domain.White)
	if _, err := d.SavePosition(&p); err != nil {
		t.Fatalf("SavePosition: %v", err)
	}

	summary, err := d.AnalyzeMatchWithGammonNet(context.Background(), 4242, 0, 0, 0, 1, nil, nil)
	if err != nil {
		t.Fatalf("AnalyzeMatchWithGammonNet: %v", err)
	}
	if summary.Processed() != 0 {
		t.Fatalf("summary = %+v, want nothing processed for a match that names no position", summary)
	}
	n, err := d.CountPositionsWithoutAnalysis()
	if err != nil {
		t.Fatalf("CountPositionsWithoutAnalysis: %v", err)
	}
	if n != 1 {
		t.Errorf("CountPositionsWithoutAnalysis = %d, want the library's single position still unanalysed", n)
	}
}
