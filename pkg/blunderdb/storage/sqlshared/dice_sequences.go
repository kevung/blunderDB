package sqlshared

// DiceSequencesSQL is the query both backends run for DiceSequences (read
// into storage.DiceRow), with
// their own tenant filter in place of %s: one row per checker move with
// dice, plus one row per match without games and per game without such a
// move (the LEFT JOINs), ordered so a match's rows arrive together.
var DiceSequencesSQL = `SELECT m.id, COALESCE(m.player1_name,''), COALESCE(m.player2_name,''),
	COALESCE(m.match_length,0), COALESCE(m.dice_hash,''),
	COALESCE(g.id,0), COALESCE(g.initial_score_1,0), COALESCE(g.initial_score_2,0),
	COALESCE(mv.dice_1,0), COALESCE(mv.dice_2,0)
FROM match m
LEFT JOIN game g ON g.match_id = m.id
LEFT JOIN move mv ON mv.game_id = g.id AND ` + ActionIsSQL("mv.move_type", "checker") + ` AND mv.dice_1 > 0
WHERE %s
ORDER BY m.id, g.game_number, g.id, mv.move_number, mv.id`
