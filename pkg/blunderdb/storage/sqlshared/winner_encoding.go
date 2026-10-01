package sqlshared

// NormalizeGameWinnerSQL rewrites every game.winner into the one encoding the
// library keeps (domain.WinnerPlayer1 = 1, WinnerPlayer2 = -1,
// WinnerUnfinished = 0). It is the data step of the 2.26.0 migration, run as
// is by both backends: SQLite's migrate_2_25_0_to_2_26_0 and PostgreSQL's
// 028_game_winner_encoding.sql, which must carry this text verbatim
// (TestWinnerMigrationSQLIsShared). It is not idempotent — a normalized 1
// reads as gnubg's player 2 — so each backend records it in the transaction
// that runs it.
//
// Before it, each importer stored its source's encoding: XG's 1 / -1 / 0 for
// .xg, which is already the target; gnubg's 0 = player 1, 1 = player 2,
// -1 = unfinished for .sgf, .mat, pasted text and transcriptions; and 0 for
// every BGF game, whose winner was never read. Swapping the players of a match
// negated its winners as if they were XG's, which turns gnubg's into a third
// encoding: 0 = player 2, -1 = player 1, 1 = unfinished. Value 1 means
// different things in each, so the match decides, in order:
//
//  1. A game that won no points is unfinished.
//  2. The scores decide where they can: when the next game of the match starts
//     with exactly one player's score higher, that player won, whatever was
//     stored.
//  3. The games step 2 settled name the match's encoding when their stored
//     values agree with exactly one of XG's, gnubg's and swapped gnubg's; every
//     other game of the match is read in it. A BGF match takes no part: its
//     stored 0 tells nothing.
//  4. The last game of a match with a length, when exactly one player's score
//     plus the points reaches the length, was won by that player. It comes
//     after step 3 because a match saved between two games ends on a finished
//     game that did not end it, where the stored value, once its encoding is
//     known, is the better witness.
//  5. Otherwise the source names the encoding: a batch format or a file named
//     .xg is XG's, anything else (.sgf, .mat, a paste, a transcription)
//     gnubg's. A BGF game still unsettled stays unfinished: the final score
//     that names it is in the file, not in the library, and importing the
//     file again restores it.
const NormalizeGameWinnerSQL = `WITH g AS (
    SELECT g.id, g.match_id,
           COALESCE(g.winner, 0) AS w,
           COALESCE(g.points_won, 0) AS p,
           COALESCE(g.initial_score_1, 0) AS s1,
           COALESCE(g.initial_score_2, 0) AS s2,
           LEAD(COALESCE(g.initial_score_1, 0)) OVER (PARTITION BY g.match_id ORDER BY g.game_number, g.id) AS n1,
           LEAD(COALESCE(g.initial_score_2, 0)) OVER (PARTITION BY g.match_id ORDER BY g.game_number, g.id) AS n2,
           COALESCE(m.match_length, 0) AS len,
           CASE WHEN LOWER(COALESCE(b.format, '')) = 'bgf' OR LOWER(COALESCE(m.file_path, '')) LIKE '%.bgf' THEN 'bgf'
                WHEN LOWER(COALESCE(b.format, '')) = 'xg' OR LOWER(COALESCE(m.file_path, '')) LIKE '%.xg' THEN 'xg'
                ELSE 'gnubg' END AS hint
      FROM game g
      JOIN match m ON m.id = g.match_id
      ` + WinnerBatchJoin + `
), t AS (
    SELECT g.*,
           CASE WHEN p <= 0 THEN 0
                WHEN n1 IS NOT NULL AND n1 > s1 AND n2 = s2 THEN 1
                WHEN n1 IS NOT NULL AND n2 > s2 AND n1 = s1 THEN -1
           END AS truth
      FROM g
), v AS (
    SELECT match_id,
           SUM(CASE WHEN (truth = 1 AND w = 1) OR (truth = -1 AND w = -1) THEN 1 ELSE 0 END) AS xg,
           SUM(CASE WHEN (truth = 1 AND w = 0) OR (truth = -1 AND w = 1) THEN 1 ELSE 0 END) AS gnubg,
           SUM(CASE WHEN (truth = 1 AND w = -1) OR (truth = -1 AND w = 0) THEN 1 ELSE 0 END) AS swapped
      FROM t
     GROUP BY match_id
), e AS (
    SELECT t.*,
           CASE WHEN t.hint = 'bgf' THEN ''
                WHEN v.xg > 0 AND v.gnubg = 0 AND v.swapped = 0 THEN 'xg'
                WHEN v.gnubg > 0 AND v.xg = 0 AND v.swapped = 0 THEN 'gnubg'
                WHEN v.swapped > 0 AND v.xg = 0 AND v.gnubg = 0 THEN 'swapped'
                ELSE '' END AS voted
      FROM t
      JOIN v ON v.match_id = t.match_id
), c AS (
    SELECT id,
           CASE WHEN truth IS NOT NULL THEN truth
                WHEN voted = 'xg' AND w IN (1, -1) THEN w
                WHEN voted = 'gnubg' AND w = 0 THEN 1
                WHEN voted = 'gnubg' AND w = 1 THEN -1
                WHEN voted = 'swapped' AND w = -1 THEN 1
                WHEN voted = 'swapped' AND w = 0 THEN -1
                WHEN voted <> '' THEN 0
                WHEN n1 IS NULL AND len > 0 AND s1 + p >= len AND s2 + p < len THEN 1
                WHEN n1 IS NULL AND len > 0 AND s2 + p >= len AND s1 + p < len THEN -1
                WHEN hint = 'xg' AND w IN (1, -1) THEN w
                WHEN hint = 'gnubg' AND w = 0 THEN 1
                WHEN hint = 'gnubg' AND w = 1 THEN -1
                ELSE 0 END AS nw
      FROM e
)
UPDATE game SET winner = c.nw
  FROM c
 WHERE game.id = c.id`

// WinnerBatchJoin is the clause of NormalizeGameWinnerSQL that reads a match's
// import batch. A library older than import_batch has no batch to read, and
// SQLite's step swaps the clause for an empty one there.
const WinnerBatchJoin = `LEFT JOIN import_batch b ON b.id = m.import_batch_id`
