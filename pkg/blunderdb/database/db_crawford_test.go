package database

import (
	"database/sql"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"

	_ "modernc.org/sqlite"
)

// crawfordFixture opens a database holding one 7-point match whose three games
// are: 0-0, then 6-2 (the Crawford game), then 6-3 (post-Crawford). It returns
// the open Database, its raw handle and the ids of the three games.
func crawfordFixture(t *testing.T, name string) (*Database, *sql.DB, [3]int64) {
	t.Helper()
	db := NewDatabase()
	if err := db.SetupDatabase(filepath.Join(t.TempDir(), name)); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	t.Cleanup(func() { db.db.Close() })
	raw := db.db

	matchRes, err := raw.Exec(
		`INSERT INTO match (player1_name, player2_name, match_length, match_date, import_date, game_count, match_hash)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"Alice", "Bob", 7, time.Now(), time.Now(), 3, "hash-"+name)
	if err != nil {
		t.Fatalf("insert match: %v", err)
	}
	matchID, _ := matchRes.LastInsertId()

	var games [3]int64
	for i, initial := range [3][2]int{{0, 0}, {6, 2}, {6, 3}} {
		res, err := raw.Exec(
			`INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2, winner, points_won, move_count)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			matchID, i+1, initial[0], initial[1], 1, 1, 1)
		if err != nil {
			t.Fatalf("insert game %d: %v", i+1, err)
		}
		games[i], _ = res.LastInsertId()
	}
	return db, raw, games
}

// attach records one checker move of gameID played on positionID, which is what
// makes a stored position belong to a game.
func attach(t *testing.T, raw *sql.DB, gameID, positionID int64) {
	t.Helper()
	if _, err := raw.Exec(
		`INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move)
		 VALUES (?, 1, 'checker', ?, 0, 3, 1, '8/5 6/5')`, gameID, positionID); err != nil {
		t.Fatalf("insert move: %v", err)
	}
}

// awayAndHashOf reads back the stored away score and Zobrist hash of a row.
func awayAndHashOf(t *testing.T, raw *sql.DB, id int64) ([2]int, int64) {
	t.Helper()
	var s1, s2, hash int64
	if err := raw.QueryRow(`SELECT score_1, score_2, zobrist_hash FROM position WHERE id = ?`, id).
		Scan(&s1, &s2, &hash); err != nil {
		t.Fatalf("reading position %d: %v", id, err)
	}
	return [2]int{int(s1), int(s2)}, hash
}

// TestRepairCrawfordSentinelRehashesPostCrawfordPositions is issue #338's
// repair: the importers wrote away `1` on every 1-away position, Crawford game
// or not, so a post-Crawford position already in a database reads as cube-dead
// (CONTEXT.md, « Away score »). Correcting it changes the Zobrist hash — the
// away score is part of the identity, unlike has_jacoby/has_beaver (ADR-0028) —
// so the row is REHASHED, and the analysis and comments hanging off it must
// still be there afterwards.
func TestRepairCrawfordSentinelRehashesPostCrawfordPositions(t *testing.T) {
	t.Parallel()
	db, raw, games := crawfordFixture(t, "rehash.db")

	// The position as the importer wrote it: 6-3 in a 7-pointer, so away
	// [1, 4] — with the `1` claiming a Crawford game that is two games behind.
	stale := InitializePosition()
	stale.Score = [2]int{domain.Crawford, 4}
	stale.Dice = [2]int{6, 5}
	staleID, err := db.SavePosition(&stale)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	attach(t, raw, games[2], staleID)
	if err := db.SaveAnalysis(staleID, PositionAnalysis{
		PositionID:      int(staleID),
		XGID:            "post-crawford",
		AnalysisType:    "XG Roller++",
		CheckerAnalysis: &CheckerAnalysis{Moves: []CheckerMove{{Index: 1, Move: "24/13", Equity: 0.05}}},
	}); err != nil {
		t.Fatalf("SaveAnalysis: %v", err)
	}
	if _, err := raw.Exec(
		`INSERT INTO comment (position_id, text, origin) VALUES (?, 'the trailer doubles here', 'user')`,
		staleID); err != nil {
		t.Fatalf("insert comment: %v", err)
	}

	// A position of the CRAWFORD game itself, at 6-2: away [1, 5] is right,
	// and the repair must not touch it.
	crawford := InitializePosition()
	crawford.Score = [2]int{domain.Crawford, 5}
	crawford.Dice = [2]int{4, 2}
	crawfordID, err := db.SavePosition(&crawford)
	if err != nil {
		t.Fatalf("SavePosition (Crawford game): %v", err)
	}
	attach(t, raw, games[1], crawfordID)

	// A position belonging to no game at all — typed on the board, pasted as
	// an XGID. Away `1` is then the user's own word; nothing contradicts it.
	loose := InitializePosition()
	loose.Score = [2]int{domain.Crawford, 3}
	loose.Dice = [2]int{5, 3}
	looseID, err := db.SavePosition(&loose)
	if err != nil {
		t.Fatalf("SavePosition (loose): %v", err)
	}

	_, staleHashBefore := awayAndHashOf(t, raw, staleID)

	repaired, err := db.RepairCrawfordSentinel()
	if err != nil {
		t.Fatalf("RepairCrawfordSentinel: %v", err)
	}
	if repaired != 1 {
		t.Fatalf("repaired %d positions, want 1 (only the post-Crawford one)", repaired)
	}

	away, hashAfter := awayAndHashOf(t, raw, staleID)
	if away != [2]int{domain.PostCrawford, 4} {
		t.Errorf("away score after repair = %v, want [0 4]", away)
	}
	if hashAfter == staleHashBefore {
		t.Error("the Zobrist hash did not change: the away score is part of the identity, so the row must be rehashed")
	}
	if got, _ := awayAndHashOf(t, raw, crawfordID); got != [2]int{domain.Crawford, 5} {
		t.Errorf("the Crawford game's own position was rewritten to %v", got)
	}
	if got, _ := awayAndHashOf(t, raw, looseID); got != [2]int{domain.Crawford, 3} {
		t.Errorf("a position belonging to no game was rewritten to %v", got)
	}

	// Nothing hanging off the repaired position was lost on the way.
	assertTableCount(t, raw, "position", 3)
	assertTableCount(t, raw, "analysis", 1)
	assertTableCount(t, raw, "comment", 1)
	if analysis, err := db.LoadAnalysis(staleID); err != nil || analysis == nil {
		t.Errorf("the analysis did not survive the rehash: %v", err)
	}

	// Idempotent: everything now says what it means, so a second pass is a
	// pure scan.
	again, err := db.RepairCrawfordSentinel()
	if err != nil {
		t.Fatalf("second RepairCrawfordSentinel: %v", err)
	}
	if again != 0 {
		t.Errorf("second pass repaired %d positions, want 0", again)
	}
}

// TestRepairCrawfordSentinelMergesWithTheCorrectTwin is the reason the repair
// rehashes through the dedup decision instead of an UPDATE and be done with
// it: once the away score is corrected the row IS the position already stored
// under the right hash — a transcription's, or another match's imported after
// the fix. The two must become one row, with the moves, the comment and the
// analysis of both on the survivor.
func TestRepairCrawfordSentinelMergesWithTheCorrectTwin(t *testing.T) {
	t.Parallel()
	db, raw, games := crawfordFixture(t, "merge.db")

	// The correct twin first: same board, same dice, away [0, 4].
	correct := InitializePosition()
	correct.Score = [2]int{domain.PostCrawford, 4}
	correct.Dice = [2]int{6, 5}
	correctID, err := db.SavePosition(&correct)
	if err != nil {
		t.Fatalf("SavePosition (correct twin): %v", err)
	}
	if err := db.SaveAnalysis(correctID, PositionAnalysis{
		PositionID:      int(correctID),
		XGID:            "correct",
		AnalysisType:    "XG Roller++",
		CheckerAnalysis: &CheckerAnalysis{Moves: []CheckerMove{{Index: 1, Move: "24/13", Equity: 0.05}}},
	}); err != nil {
		t.Fatalf("SaveAnalysis (correct twin): %v", err)
	}

	// And the same position as the buggy importer stored it, on the same
	// post-Crawford game, with a comment of its own.
	stale := InitializePosition()
	stale.Score = [2]int{domain.Crawford, 4}
	stale.Dice = [2]int{6, 5}
	staleID, err := db.SavePosition(&stale)
	if err != nil {
		t.Fatalf("SavePosition (stale): %v", err)
	}
	if staleID == correctID {
		t.Fatal("the two away scores hashed to the same row; the fixture proves nothing")
	}
	attach(t, raw, games[2], staleID)
	if _, err := raw.Exec(
		`INSERT INTO comment (position_id, text, origin) VALUES (?, 'my note', 'user')`, staleID); err != nil {
		t.Fatalf("insert comment: %v", err)
	}

	repaired, err := db.RepairCrawfordSentinel()
	if err != nil {
		t.Fatalf("RepairCrawfordSentinel: %v", err)
	}
	if repaired != 1 {
		t.Fatalf("repaired %d positions, want 1", repaired)
	}

	// One row left, and it is the one that was already right.
	assertTableCount(t, raw, "position", 1)
	if away, _ := awayAndHashOf(t, raw, correctID); away != [2]int{domain.PostCrawford, 4} {
		t.Errorf("the surviving position carries %v, want [0 4]", away)
	}

	// The move now names the survivor, and neither comment nor analysis was
	// dropped on the floor.
	var movePositionID int64
	if err := raw.QueryRow(`SELECT position_id FROM move`).Scan(&movePositionID); err != nil {
		t.Fatalf("reading the move: %v", err)
	}
	if movePositionID != correctID {
		t.Errorf("the move still names position %d, want the survivor %d", movePositionID, correctID)
	}
	var commentPositionID int64
	if err := raw.QueryRow(`SELECT position_id FROM comment`).Scan(&commentPositionID); err != nil {
		t.Fatalf("reading the comment: %v", err)
	}
	if commentPositionID != correctID {
		t.Errorf("the comment still names position %d, want the survivor %d", commentPositionID, correctID)
	}
	assertTableCount(t, raw, "analysis", 1)
	analysis, err := db.LoadAnalysis(correctID)
	if err != nil || analysis == nil {
		t.Fatalf("the survivor lost its analysis: %v", err)
	}
	if analysis.XGID != "correct" {
		t.Errorf("the survivor's analysis is %q, want the one it already had", analysis.XGID)
	}
}

// TestRepairCrawfordSentinelMergeKeepsTheDatabaseConsistent is the merge seen
// from the file the GUI and the CLI open: the sticky marks of the merged row
// climb onto the survivor, and no row is left pointing at the deleted one —
// PRAGMA foreign_key_check is the judge, including on the Anki tables that
// name a position twice (position_id and key, ADR-0042).
func TestRepairCrawfordSentinelMergeKeepsTheDatabaseConsistent(t *testing.T) {
	t.Parallel()
	db, raw, games := crawfordFixture(t, "consistent.db")

	correct := InitializePosition()
	correct.Score = [2]int{domain.PostCrawford, 4}
	correct.Dice = [2]int{6, 5}
	correctID, err := db.SavePosition(&correct)
	if err != nil {
		t.Fatalf("SavePosition (correct twin): %v", err)
	}
	stale := InitializePosition()
	stale.Score = [2]int{domain.Crawford, 4}
	stale.Dice = [2]int{6, 5}
	saved, err := db.SaveIndividualPosition(&stale)
	if err != nil {
		t.Fatalf("SaveIndividualPosition (stale): %v", err)
	}
	staleID := saved.ID
	if staleID == correctID {
		t.Fatal("the two away scores hashed to the same row; the fixture proves nothing")
	}
	attach(t, raw, games[2], staleID)
	if _, err := raw.Exec(`UPDATE position SET flagged = 1 WHERE id = ?`, staleID); err != nil {
		t.Fatalf("flag: %v", err)
	}

	// Both rows in one deck, the stale one reviewed there.
	deckRes, err := raw.Exec(`INSERT INTO anki_deck (name) VALUES ('deck')`)
	if err != nil {
		t.Fatalf("insert deck: %v", err)
	}
	deckID, _ := deckRes.LastInsertId()
	cardOf := map[int64]int64{}
	for _, pid := range []int64{correctID, staleID} {
		res, err := raw.Exec(
			`INSERT INTO anki_card (deck_id, kind, key, position_id) VALUES (?, ?, CAST(? AS TEXT), ?)`,
			deckID, domain.AnkiKindPosition, pid, pid)
		if err != nil {
			t.Fatalf("insert card: %v", err)
		}
		cardOf[pid], _ = res.LastInsertId()
	}
	if _, err := raw.Exec(
		`INSERT INTO anki_review_log (card_id, deck_id, kind, key, position_id, rating)
		 VALUES (?, ?, ?, CAST(? AS TEXT), ?, 3)`,
		cardOf[staleID], deckID, domain.AnkiKindPosition, staleID, staleID); err != nil {
		t.Fatalf("insert review: %v", err)
	}

	if _, err := db.RepairCrawfordSentinel(); err != nil {
		t.Fatalf("RepairCrawfordSentinel: %v", err)
	}

	assertTableCount(t, raw, "position", 1)
	var individual, flagged bool
	if err := raw.QueryRow(`SELECT individually_imported, flagged FROM position WHERE id = ?`, correctID).
		Scan(&individual, &flagged); err != nil {
		t.Fatalf("reading marks: %v", err)
	}
	if !individual || !flagged {
		t.Errorf("survivor marks individually_imported=%v flagged=%v, want both raised from the merged row", individual, flagged)
	}

	assertTableCount(t, raw, "anki_card", 1)
	var logCard, logPosition int64
	var logKey string
	if err := raw.QueryRow(`SELECT card_id, position_id, key FROM anki_review_log`).
		Scan(&logCard, &logPosition, &logKey); err != nil {
		t.Fatalf("the review of the dropped card did not survive: %v", err)
	}
	if logCard != cardOf[correctID] || logPosition != correctID || logKey != strconv.FormatInt(correctID, 10) {
		t.Errorf("review names card %d position %d key %q, want card %d position %d",
			logCard, logPosition, logKey, cardOf[correctID], correctID)
	}

	rows, err := raw.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatalf("foreign_key_check: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var table string
		var rowid sql.NullInt64
		var parent string
		var fkid int64
		if err := rows.Scan(&table, &rowid, &parent, &fkid); err != nil {
			t.Fatalf("scan foreign_key_check: %v", err)
		}
		t.Errorf("%s row %v points at a missing %s", table, rowid.Int64, parent)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("foreign_key_check: %v", err)
	}
}
