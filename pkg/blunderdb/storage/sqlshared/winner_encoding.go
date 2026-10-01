package sqlshared

// NormalizeGameWinnerSQL rewrites every game.winner into the one encoding the
// library keeps (domain.WinnerPlayer1 = 1, WinnerPlayer2 = -1,
// WinnerUnfinished = 0). It is the data step of the 2.26.0 migration, run as
// is by both backends: SQLite's migrate_2_25_0_to_2_26_0 and PostgreSQL's
// 028_game_winner_encoding.sql, which must carry this text verbatim
// (TestWinnerMigrationSQLIsShared).
//
// Before it, each importer stored its source's encoding: XG's 1 / -1 / 0 for
// .xg, which is already the target; gnubg's 0 = player 1, 1 = player 2,
// -1 = unfinished for .sgf, .mat, pasted text and transcriptions; and 0 for
// every BGF game, whose winner was never read. Value 1 means opposite players
// in the first two, so the source decides, and the source is not always
// recorded. Hence, in order:
//
//  1. A game that won no points is unfinished.
//  2. The scores decide where they can: when the next game of the match starts
//     with exactly one player's score higher, that player won. This is the
//     truth of the record, whatever was stored, and it also repairs the
//     matches of the second encoding whose players were swapped (the swap
//     negated the winner as if it were XG's).
//  3. Every other game — the last of a match, a money session whose scores
//     never move — is read in its match's encoding. That encoding is first
//     read off the games step 2 settled: stored values that agree only with
//     XG's, or only with gnubg's, name it. When they name neither, or both,
//     the source does: a batch format or a file named .xg is XG's, .bgf is
//     BGF's, anything else (.sgf, .mat, a paste, a transcription) gnubg's.
//     A BGF match is BGF's whatever its games say, since its stored 0 tells
//     nothing.
//  4. A game of a BGF match that step 2 cannot settle stays unfinished: the
//     final score that names the last game's winner is in the file, not in
//     the library. Importing the file again restores it.
const NormalizeGameWinnerSQL = `WITH g AS (
    SELECT g.id, g.match_id,
           COALESCE(g.winner, 0) AS w,
           COALESCE(g.points_won, 0) AS p,
           COALESCE(g.initial_score_1, 0) AS s1,
           COALESCE(g.initial_score_2, 0) AS s2,
           LEAD(COALESCE(g.initial_score_1, 0)) OVER (PARTITION BY g.match_id ORDER BY g.game_number, g.id) AS n1,
           LEAD(COALESCE(g.initial_score_2, 0)) OVER (PARTITION BY g.match_id ORDER BY g.game_number, g.id) AS n2,
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
           SUM(CASE WHEN truth IN (1, -1) AND w = truth THEN 1 ELSE 0 END) AS xg_votes,
           SUM(CASE WHEN (truth = 1 AND w = 0) OR (truth = -1 AND w = 1) THEN 1 ELSE 0 END) AS gnubg_votes
      FROM t
     GROUP BY match_id
), e AS (
    SELECT t.id, t.w, t.truth,
           CASE WHEN t.hint = 'bgf' THEN 'bgf'
                WHEN v.xg_votes > 0 AND v.gnubg_votes = 0 THEN 'xg'
                WHEN v.gnubg_votes > 0 AND v.xg_votes = 0 THEN 'gnubg'
                ELSE t.hint END AS enc
      FROM t
      JOIN v ON v.match_id = t.match_id
), c AS (
    SELECT id,
           CASE WHEN truth IS NOT NULL THEN truth
                WHEN enc = 'xg' AND w IN (1, -1) THEN w
                WHEN enc = 'gnubg' AND w = 0 THEN 1
                WHEN enc = 'gnubg' AND w = 1 THEN -1
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
