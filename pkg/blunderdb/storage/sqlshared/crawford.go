package sqlshared

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// La sentinelle Crawford, réparée après coup (T0.7, ADR-0045 §7).
//
// L'away score porte la règle de Crawford DANS le nombre (CONTEXT.md,
// « Away score ») : `1` veut dire « un point à faire, et cette partie EST la
// Crawford », `0` « un point à faire, la Crawford est derrière ». Les trois
// importeurs ont longtemps écrit `1` dans les deux cas — ils ne calculaient que
// matchLength − score — si bien que toute partie post-Crawford déjà importée
// est lue videau mort, là où le poursuivant double en réalité à la première
// occasion.
//
// Corriger la valeur change le hash Zobrist, puisque l'away score en fait
// partie (contrairement à has_jacoby/has_beaver, ADR-0028). La ligne est donc
// REHACHÉE, et la question que pose le rehachage — « une position correcte
// existe-t-elle déjà pour ce plateau ? » — est la décision de dédup que prend
// Save. Elle est ici posée explicitement parce que sa réponse change ce qu'on
// fait de l'ancienne ligne : fusionner dans la jumelle correcte, ou rehacher la
// ligne sur place, ce qui garde l'id et donc l'analyse, les commentaires, les
// cartes et les coups déjà accrochés, sans rien recopier.

// batchSize bounds the IN list of the move lookup: a database can hold tens of
// thousands of 1-away positions and SQLite refuses a statement with more than
// 999 parameters by default.
const crawfordBatchSize = 500

// gameFacts is what deciding Crawford needs about one game: which match it
// belongs to, its rank in that match, and the score it started on.
type gameFacts struct {
	matchID     int64
	gameNumber  int64
	initial     [2]int64
	matchLength int64
}

// RepairCrawfordSentinel rewrites the away score of every stored position that
// carries the ambiguous `1` while the game it was imported from is a
// post-Crawford one, and returns how many rows changed.
//
// What it repairs, and only that:
//
//   - a position whose EVERY move belongs to a post-Crawford game is corrected
//     to the `0` sentinel and rehashed;
//   - a position that belongs to no game at all — typed on the board, pasted as
//     an XGID, imported on its own — is left alone: away `1` is then the user's
//     own statement that this is the Crawford game, and there is no match to
//     contradict it;
//   - a position that belongs to a Crawford game AND to a post-Crawford one is
//     left alone too. Today they are one row (the away score being the same),
//     and the row is legitimately the Crawford one for one of its games;
//     splitting it would hand the copy an analysis that was computed for the
//     other score.
//
// Nothing runs it automatically: like ReclassifyDerived it is a repair of data,
// explicitly asked for (`blunderdb repair`, /v1/positions.repairCrawford).
// Running it on a database that is already correct rewrites nothing.
func RepairCrawfordSentinel(ctx context.Context, db Execer, scope string, positions storage.PositionStore) (int, error) {
	fail := func(err error) (int, error) { return 0, errf(db, "repair the Crawford sentinel", err) }

	stale, err := staleSentinelIDs(ctx, db, scope)
	if err != nil {
		return fail(err)
	}
	if len(stale) == 0 {
		return 0, nil
	}
	crawfordGames, err := crawfordGameIDs(ctx, db, scope)
	if err != nil {
		return fail(err)
	}
	gamesOf, err := gamesOfPositions(ctx, db, scope, stale)
	if err != nil {
		return fail(err)
	}

	repaired := 0
	for _, id := range stale {
		games := gamesOf[id]
		if len(games) == 0 {
			continue // no match behind it: the sentinel is the user's word
		}
		postCrawfordOnly := true
		for _, gameID := range games {
			isCrawford, known := crawfordGames[gameID]
			if !known || isCrawford {
				postCrawfordOnly = false
				break
			}
		}
		if !postCrawfordOnly {
			continue
		}
		changed, err := rehashOutOfCrawford(ctx, db, scope, positions, id)
		if err != nil {
			return fail(err)
		}
		if changed {
			repaired++
		}
	}
	return repaired, nil
}

// staleSentinelIDs lists the positions carrying a raw away score of 1 — the
// candidates, before the match behind them has its say.
func staleSentinelIDs(ctx context.Context, db Execer, scope string) ([]int64, error) {
	tenant, targs := db.TenantFilter("p", scope)
	rows, err := db.Query(ctx,
		`SELECT p.id FROM position p WHERE `+tenant+` AND (p.score_1 = 1 OR p.score_2 = 1) ORDER BY p.id`, targs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// crawfordGameIDs maps every stored game to whether it is its match's Crawford
// game: the FIRST game a player starts at match point. Derived from the
// sequence of initial scores rather than stored, exactly as the importers
// derive it for an XG file — a match has at most one Crawford game, so every
// later game starting at match point is a post-Crawford one.
func crawfordGameIDs(ctx context.Context, db Execer, scope string) (map[int64]bool, error) {
	gtenant, gargs := db.TenantFilter("g", scope)
	mtenant, margs := db.TenantFilter("m", scope)
	rows, err := db.Query(ctx,
		`SELECT g.id, g.match_id, g.game_number, g.initial_score_1, g.initial_score_2, m.match_length
		   FROM game g INNER JOIN match m ON m.id = g.match_id
		  WHERE `+gtenant+` AND `+mtenant, append(append([]any{}, gargs...), margs...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	facts := make(map[int64]gameFacts)
	for rows.Next() {
		var id int64
		var f gameFacts
		if err := rows.Scan(&id, &f.matchID, &f.gameNumber, &f.initial[0], &f.initial[1], &f.matchLength); err != nil {
			return nil, err
		}
		facts[id] = f
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// The Crawford game of a match is the lowest-numbered game at match point.
	// Ties on game_number (a re-imported match, a corrupt file) are broken by
	// the game id, so the answer does not depend on the row order.
	best := make(map[int64]int64) // match id → the game id that is its Crawford
	ids := make([]int64, 0, len(facts))
	for id := range facts {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		f := facts[id]
		if f.matchLength <= 0 {
			continue // money play has no Crawford game
		}
		if f.initial[0] != f.matchLength-1 && f.initial[1] != f.matchLength-1 {
			continue
		}
		if cur, ok := best[f.matchID]; !ok || f.gameNumber < facts[cur].gameNumber {
			best[f.matchID] = id
		}
	}
	crawford := make(map[int64]bool, len(facts))
	for id, f := range facts {
		crawford[id] = best[f.matchID] == id
	}
	return crawford, nil
}

// gamesOfPositions lists, per position, the games its moves belong to. A
// position stored once can be reached from several games (that is what the
// Zobrist dedup is for), which is why the answer is a set and not a game.
func gamesOfPositions(ctx context.Context, db Execer, scope string, positionIDs []int64) (map[int64][]int64, error) {
	out := make(map[int64][]int64, len(positionIDs))
	for start := 0; start < len(positionIDs); start += crawfordBatchSize {
		end := min(start+crawfordBatchSize, len(positionIDs))
		chunk := positionIDs[start:end]

		tenant, targs := db.TenantFilter("mv", scope)
		args := append([]any{}, targs...)
		for _, id := range chunk {
			args = append(args, id)
		}
		rows, err := db.Query(ctx,
			`SELECT DISTINCT mv.position_id, mv.game_id FROM move mv
			  WHERE `+tenant+` AND mv.position_id IN (`+Placeholders(len(chunk))+`)`, args...)
		if err != nil {
			return nil, err
		}
		err = func() error {
			defer rows.Close()
			for rows.Next() {
				var positionID, gameID int64
				if err := rows.Scan(&positionID, &gameID); err != nil {
					return err
				}
				out[positionID] = append(out[positionID], gameID)
			}
			return rows.Err()
		}()
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// rehashOutOfCrawford turns the position's away `1` into the post-Crawford `0`
// and rehashes it, reporting whether anything changed.
//
// The rehash asks the dedup question first: a position already stored with the
// corrected score is the row to keep, and this one is merged into it — that is
// the whole reason this is not an UPDATE and be done with it. When no such row
// exists the id is kept and the row rewritten in place, which is the same final
// state a re-save would reach without moving a single analysis, comment, card
// or move to a new id.
func rehashOutOfCrawford(ctx context.Context, db Execer, scope string, positions storage.PositionStore, id int64) (bool, error) {
	pos, err := positions.Load(ctx, scope, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return false, nil // deleted under us; nothing to repair
		}
		return false, err
	}
	fixed := *pos
	corrected := false
	for i, away := range fixed.Score {
		if away == 1 {
			fixed.Score[i] = 0
			corrected = true
		}
	}
	if !corrected {
		return false, nil
	}

	norm := fixed.NormalizeForStorage()
	norm.ID = id
	keepID, held, err := positions.Exists(ctx, scope, engine.ZobristHash(&norm))
	if err != nil {
		return false, err
	}
	if held && keepID != id {
		return true, db.Transact(ctx, func(tx Execer) error {
			return MergePositionInto(ctx, tx, scope, keepID, id)
		})
	}
	if err := positions.Update(ctx, scope, &norm); err != nil {
		return false, err
	}
	return true, nil
}

// MergePositionInto moves everything attached to the duplicate position dupID
// onto keepID — the row the Zobrist index already holds — and deletes dupID.
//
// What hangs off a position, and what each does in the merge:
//
//   - match moves and comments follow the position;
//   - collection memberships follow it, except in a collection keepID is
//     already in, where the duplicate membership goes with dupID;
//   - Anki cards follow it, and so does the key that names the position inside
//     its deck (ADR-0042: a position card's key is its position id as text).
//     In a deck keepID already has a card in, dupID's card goes, but its
//     review journal is handed to keepID's card first — the reviews happened,
//     and the journal is what the retention figures are read from;
//   - the review journal follows it, key included;
//   - an analysis follows only when keepID has none (there is one per
//     position, and the held row's own wins);
//   - the sticky marks (individually_imported, flagged — ADR-0001, ADR-0006)
//     are raised on keepID when dupID carried them, and never lowered;
//   - the trash entries that name dupID by id — a deleted comment, a deleted
//     collection's member list — are rewritten to name keepID, so restoring
//     them later lands on the survivor instead of on a row that no longer
//     exists.
//
// Written with an explicit DELETE of the conflicting rows rather than SQLite's
// `UPDATE OR IGNORE`, which PostgreSQL has no equivalent of: the final state is
// the same, and the statement is one both backends run.
func MergePositionInto(ctx context.Context, tx Execer, scope string, keepID, dupID int64) error {
	fail := func(err error) error {
		return fmt.Errorf("merging position %d into %d: %w", dupID, keepID, err)
	}

	// A card of dupID's in a deck where keepID already has one is about to
	// go; its journal goes to keepID's card of the same deck first, or the
	// ON DELETE CASCADE on card_id would take the reviews with it.
	{
		ktenant, kargs := tx.TenantFilter("k", scope)
		ltenant, largs := tx.TenantFilter("", scope)
		ctenant, cargs := tx.TenantFilter("c", scope)
		k2tenant, k2args := tx.TenantFilter("k2", scope)
		args := append([]any{}, kargs...)
		args = append(args, keepID)
		args = append(args, largs...)
		args = append(args, dupID)
		args = append(args, cargs...)
		args = append(args, dupID)
		args = append(args, k2args...)
		args = append(args, keepID)
		if _, err := tx.Exec(ctx,
			`UPDATE anki_review_log
			    SET card_id = (SELECT k.id FROM anki_card k
			                    WHERE `+ktenant+` AND k.position_id = ? AND k.deck_id = anki_review_log.deck_id)
			  WHERE `+ltenant+` AND position_id = ?
			    AND card_id IN (SELECT c.id FROM anki_card c
			                     WHERE `+ctenant+` AND c.position_id = ?
			                       AND c.deck_id IN (SELECT k2.deck_id FROM anki_card k2
			                                          WHERE `+k2tenant+` AND k2.position_id = ?))`,
			args...); err != nil {
			return fail(err)
		}
	}

	// The uniquely-constrained memberships: a duplicate the kept row already
	// holds is dropped, so the re-pointing below cannot collide.
	for _, dedup := range []struct{ table, by string }{
		{"collection_position", "collection_id"},
		{"anki_card", "deck_id"},
	} {
		dtenant, dargs := tx.TenantFilter("", scope)
		stenant, sargs := tx.TenantFilter("", scope)
		args := append([]any{}, dargs...)
		args = append(args, dupID)
		args = append(args, sargs...)
		args = append(args, keepID)
		if _, err := tx.Exec(ctx,
			`DELETE FROM `+dedup.table+` WHERE `+dtenant+` AND position_id = ?
			   AND `+dedup.by+` IN (SELECT `+dedup.by+` FROM `+dedup.table+`
			                        WHERE `+stenant+` AND position_id = ?)`, args...); err != nil {
			return fail(err)
		}
	}
	for _, table := range []string{"move", "collection_position", "comment"} {
		tenant, targs := tx.TenantFilter("", scope)
		args := append([]any{keepID}, targs...)
		args = append(args, dupID)
		if _, err := tx.Exec(ctx,
			`UPDATE `+table+` SET position_id = ? WHERE `+tenant+` AND position_id = ?`, args...); err != nil {
			return fail(err)
		}
	}
	// A position card and its journal name the position twice: in position_id
	// and in the key (ADR-0042). Both move, or the card is found by one and
	// not by the other — and the deck's unique (deck_id, kind, key) would let a
	// later sync add a second card for the survivor.
	keepKey := strconv.FormatInt(keepID, 10)
	for _, table := range []string{"anki_card", "anki_review_log"} {
		tenant, targs := tx.TenantFilter("", scope)
		args := append([]any{keepID, keepKey}, targs...)
		args = append(args, dupID)
		if _, err := tx.Exec(ctx,
			`UPDATE `+table+` SET position_id = ?, key = ? WHERE `+tenant+` AND position_id = ?`, args...); err != nil {
			return fail(err)
		}
	}

	if err := mergeAnalysisInto(ctx, tx, scope, keepID, dupID); err != nil {
		return fail(err)
	}
	if err := raiseStickyMarks(ctx, tx, scope, keepID, dupID); err != nil {
		return fail(err)
	}
	if err := repointTrash(ctx, tx, scope, keepID, dupID); err != nil {
		return fail(err)
	}

	tenant, targs := tx.TenantFilter("", scope)
	if _, err := tx.Exec(ctx,
		`DELETE FROM position WHERE `+tenant+` AND id = ?`, append(targs, dupID)...); err != nil {
		return fail(err)
	}
	return nil
}

// repointTrash rewrites the trash entries that name dupID by id so they name
// keepID. Two kinds do: a deleted comment (restored onto its position id) and a
// deleted collection (restored with its member ids). A deleted POSITION names
// no id worth rewriting — it is restored by re-Saving its board, which the
// Zobrist dedup already sends to the survivor. The trash holds thirty days at
// most, so reading its two kinds whole is cheap.
func repointTrash(ctx context.Context, tx Execer, scope string, keepID, dupID int64) error {
	tenant, targs := tx.TenantFilter("", scope)
	rows, err := tx.Query(ctx,
		`SELECT id, kind, payload FROM trash WHERE `+tenant+` AND kind IN (?, ?)`,
		append(append([]any{}, targs...), string(domain.TrashComment), string(domain.TrashCollection))...)
	if err != nil {
		return err
	}
	type rewrite struct {
		id      int64
		payload string
	}
	var rewrites []rewrite
	err = func() error {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var kind, payload string
			if err := rows.Scan(&id, &kind, &payload); err != nil {
				return err
			}
			updated, changed, err := repointTrashPayload(domain.TrashKind(kind), payload, keepID, dupID)
			if err != nil {
				return fmt.Errorf("trash entry %d: %w", id, err)
			}
			if changed {
				rewrites = append(rewrites, rewrite{id, updated})
			}
		}
		return rows.Err()
	}()
	if err != nil {
		return err
	}
	for _, r := range rewrites {
		utenant, uargs := tx.TenantFilter("", scope)
		args := append([]any{r.payload}, uargs...)
		args = append(args, r.id)
		if _, err := tx.Exec(ctx,
			`UPDATE trash SET payload = ? WHERE `+utenant+` AND id = ?`, args...); err != nil {
			return err
		}
	}
	return nil
}

// repointTrashPayload is repointTrash's decision for one payload: the rewritten
// JSON, and whether anything named dupID. A collection that held both rows
// keeps the survivor once, at the earlier of the two places.
func repointTrashPayload(kind domain.TrashKind, payload string, keepID, dupID int64) (string, bool, error) {
	var out any
	switch kind {
	case domain.TrashComment:
		var p domain.TrashCommentPayload
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return "", false, err
		}
		if p.Comment.PositionID != dupID {
			return payload, false, nil
		}
		p.Comment.PositionID = keepID
		out = p
	case domain.TrashCollection:
		var p domain.TrashCollectionPayload
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return "", false, err
		}
		if !slices.Contains(p.PositionIDs, dupID) {
			return payload, false, nil
		}
		ids := make([]int64, 0, len(p.PositionIDs))
		for _, id := range p.PositionIDs {
			if id == dupID {
				id = keepID
			}
			if !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
		p.PositionIDs = ids
		out = p
	default:
		return payload, false, nil
	}
	blob, err := json.Marshal(out)
	if err != nil {
		return "", false, err
	}
	return string(blob), true, nil
}

// mergeAnalysisInto hands the duplicate's analysis to the kept position when
// that one has none. The blob names its position INSIDE the JSON as well as in
// the row, so it is re-encoded rather than merely re-pointed.
func mergeAnalysisInto(ctx context.Context, tx Execer, scope string, keepID, dupID int64) error {
	tenant, targs := tx.TenantFilter("", scope)
	var keepHas int
	if err := tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM analysis WHERE `+tenant+` AND position_id = ?`,
		append(append([]any{}, targs...), keepID)...).Scan(&keepHas); err != nil {
		return err
	}
	if keepHas > 0 {
		return nil // the kept row's own analysis wins; the duplicate's cascades away
	}

	dtenant, dargs := tx.TenantFilter("", scope)
	var data []byte
	err := tx.QueryRow(ctx,
		`SELECT data FROM analysis WHERE `+dtenant+` AND position_id = ?`,
		append(append([]any{}, dargs...), dupID)...).Scan(&data)
	if errors.Is(err, ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	analysis, err := engine.DecodeAnalysisFromStorage(data)
	if err != nil {
		return fmt.Errorf("decode analysis: %w", err)
	}
	analysis.PositionID = int(keepID)
	encoded, err := engine.EncodeAnalysisForStorage(&analysis)
	if err != nil {
		return fmt.Errorf("encode analysis: %w", err)
	}
	utenant, uargs := tx.TenantFilter("", scope)
	args := append([]any{keepID, encoded}, uargs...)
	args = append(args, dupID)
	_, err = tx.Exec(ctx,
		`UPDATE analysis SET position_id = ?, data = ? WHERE `+utenant+` AND position_id = ?`, args...)
	return err
}

// raiseStickyMarks copies the duplicate's provenance and study marks onto the
// kept row. They only ever go up — that is what makes them sticky.
func raiseStickyMarks(ctx context.Context, tx Execer, scope string, keepID, dupID int64) error {
	for _, mark := range []string{"individually_imported", "flagged"} {
		ktenant, kargs := tx.TenantFilter("", scope)
		dtenant, dargs := tx.TenantFilter("d", scope)
		args := append([]any{tx.BoolArg(true)}, kargs...)
		args = append(args, keepID)
		args = append(args, dargs...)
		args = append(args, dupID)
		if _, err := tx.Exec(ctx,
			`UPDATE position SET `+mark+` = ? WHERE `+ktenant+` AND id = ? AND `+tx.Bool(mark, false)+`
			   AND EXISTS (SELECT 1 FROM position d WHERE `+dtenant+` AND d.id = ? AND `+tx.Bool("d."+mark, true)+`)`,
			args...); err != nil {
			return err
		}
	}
	return nil
}
