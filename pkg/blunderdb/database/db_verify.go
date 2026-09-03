package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// OrphanCounts is what blunderdb verify reports about referential integrity:
// child rows whose parent row is gone. The schema declares ON DELETE CASCADE
// on every one of these foreign keys, so a healthy database holds none; they
// appear when a deletion ran on a connection where foreign_keys was OFF —
// which every pooled connection but one did before issue #157 was fixed —
// and the rows outlive that bug in databases written back then.
type OrphanCounts struct {
	GamesWithoutMatch       int64 `json:"games_without_match"`
	MovesWithoutGame        int64 `json:"moves_without_game"`
	MoveAnalysesWithoutMove int64 `json:"move_analyses_without_move"`
	AnalysesWithoutPosition int64 `json:"analyses_without_position"`
	ReviewsWithoutDeck      int64 `json:"reviews_without_deck"`
	ReviewsWithoutPosition  int64 `json:"reviews_without_position"`
}

// Total is the number of orphaned rows across all six relations.
func (o OrphanCounts) Total() int64 {
	return o.GamesWithoutMatch + o.MovesWithoutGame + o.MoveAnalysesWithoutMove +
		o.AnalysesWithoutPosition + o.ReviewsWithoutDeck + o.ReviewsWithoutPosition
}

// orphanQueries counts, per child table, the rows whose parent no longer
// exists. A NULL foreign key counts too: none of these columns is ever NULL
// for a row the application wrote (move.position_id may be, but move is
// judged against game here, not position).
var orphanQueries = []struct {
	name  string
	query string
	dest  func(*OrphanCounts) *int64
}{
	{"game without match",
		`SELECT COUNT(*) FROM game g LEFT JOIN match m ON m.id = g.match_id WHERE m.id IS NULL`,
		func(o *OrphanCounts) *int64 { return &o.GamesWithoutMatch }},
	{"move without game",
		`SELECT COUNT(*) FROM move mv LEFT JOIN game g ON g.id = mv.game_id WHERE g.id IS NULL`,
		func(o *OrphanCounts) *int64 { return &o.MovesWithoutGame }},
	{"move_analysis without move",
		`SELECT COUNT(*) FROM move_analysis ma LEFT JOIN move mv ON mv.id = ma.move_id WHERE mv.id IS NULL`,
		func(o *OrphanCounts) *int64 { return &o.MoveAnalysesWithoutMove }},
	{"analysis without position",
		`SELECT COUNT(*) FROM analysis a LEFT JOIN position p ON p.id = a.position_id WHERE p.id IS NULL`,
		func(o *OrphanCounts) *int64 { return &o.AnalysesWithoutPosition }},
	// The review journal's deck_id and position_id became foreign keys in
	// 2.18.0 (issue #185); databases created before that had nothing saying
	// the deck and the position had to exist, and SQLite adds no foreign key
	// to a table that already exists, so an upgraded file still has none.
	{"anki_review_log without deck",
		`SELECT COUNT(*) FROM anki_review_log l LEFT JOIN anki_deck d ON d.id = l.deck_id WHERE d.id IS NULL`,
		func(o *OrphanCounts) *int64 { return &o.ReviewsWithoutDeck }},
	{"anki_review_log without position",
		`SELECT COUNT(*) FROM anki_review_log l LEFT JOIN position p ON p.id = l.position_id WHERE p.id IS NULL`,
		func(o *OrphanCounts) *int64 { return &o.ReviewsWithoutPosition }},
}

// CountOrphans counts the child rows whose parent row is missing (see
// OrphanCounts). It only reads; nothing is purged.
func (d *Database) CountOrphans() (OrphanCounts, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var counts OrphanCounts
	if d.db == nil {
		return counts, fmt.Errorf("no database is currently open")
	}
	for _, q := range orphanQueries {
		if err := d.db.QueryRow(q.query).Scan(q.dest(&counts)); err != nil {
			return counts, fmt.Errorf("counting %s orphans: %w", q.name, err)
		}
	}
	return counts, nil
}

// CheckSchema reports what the open database lacks against the reference
// schema — the tables, columns and indexes sqlite.EnsureSchema tried to add
// on open and could not (it logs the failure and lets the database open; a
// UNIQUE index over rows that violate it is the usual case). blunderdb verify
// prints the result, so the gap is seen where a log line is not. It only
// reads; nothing is repaired.
func (d *Database) CheckSchema() (sqlite.SchemaDrift, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return sqlite.SchemaDrift{}, fmt.Errorf("no database is currently open")
	}
	return sqlite.CheckSchema(context.Background(), d.db)
}

// ConstraintViolation is one rule of the current DDL and the number of rows
// that break it.
type ConstraintViolation struct {
	// Name is the rule, as the DDL states it.
	Name string `json:"name"`
	// Count is how many rows break it. Zero for a healthy database.
	Count int64 `json:"count"`
}

// constraintQueries counts, per rule the fresh schema declares, the rows an
// existing database holds that would not be accepted today.
//
// SQLite adds neither a CHECK nor a NOT NULL through ALTER TABLE: the only way
// to put them on an existing table is to rebuild it, which on a table holding
// hundreds of thousands of positions is a long, disk-hungry operation to
// enforce what the writing code already guarantees. So the constraints are
// stated by the fresh DDL (storage/sqlite schemaStatements) and an existing
// database is judged against them here, where `blunderdb verify` can say so.
//
// A NULL never counts: a CHECK on a NULL is unknown, not violated, and the
// scalar columns are nullable by design. The one exception is the hash itself,
// whose absence is the finding.
var constraintQueries = []struct {
	name  string
	query string
}{
	{"position.zobrist_hash NOT NULL",
		`SELECT COUNT(*) FROM position WHERE zobrist_hash IS NULL`},
	{"position.dice_1 BETWEEN 0 AND 6",
		`SELECT COUNT(*) FROM position WHERE dice_1 IS NOT NULL AND dice_1 NOT BETWEEN 0 AND 6`},
	{"position.dice_2 BETWEEN 0 AND 6",
		`SELECT COUNT(*) FROM position WHERE dice_2 IS NOT NULL AND dice_2 NOT BETWEEN 0 AND 6`},
	{"position.cube_value >= 0",
		`SELECT COUNT(*) FROM position WHERE cube_value IS NOT NULL AND cube_value < 0`},
	{"position.pip_1 >= 0",
		`SELECT COUNT(*) FROM position WHERE pip_1 IS NOT NULL AND pip_1 < 0`},
	{"position.pip_2 >= 0",
		`SELECT COUNT(*) FROM position WHERE pip_2 IS NOT NULL AND pip_2 < 0`},
	{"position.off_1 BETWEEN 0 AND 15",
		`SELECT COUNT(*) FROM position WHERE off_1 IS NOT NULL AND off_1 NOT BETWEEN 0 AND 15`},
	{"position.off_2 BETWEEN 0 AND 15",
		`SELECT COUNT(*) FROM position WHERE off_2 IS NOT NULL AND off_2 NOT BETWEEN 0 AND 15`},
	{"anki_review_log.rating BETWEEN 1 AND 4",
		`SELECT COUNT(*) FROM anki_review_log WHERE rating NOT BETWEEN 1 AND 4`},
	{"analysis(position_id) UNIQUE",
		`SELECT COALESCE(SUM(n - 1), 0) FROM (
			SELECT COUNT(*) AS n FROM analysis GROUP BY position_id HAVING COUNT(*) > 1)`},
}

// CheckConstraints reports, rule by rule, how many rows of the open database
// would not be accepted by the schema a fresh database is created with. It
// only reads; nothing is repaired. Every rule is returned, breached or not, so
// a caller reading the JSON sees what was checked and not only what failed.
func (d *Database) CheckConstraints() ([]ConstraintViolation, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	out := make([]ConstraintViolation, 0, len(constraintQueries))
	for _, q := range constraintQueries {
		var n int64
		if err := d.db.QueryRow(q.query).Scan(&n); err != nil {
			return nil, fmt.Errorf("checking %s: %w", q.name, err)
		}
		out = append(out, ConstraintViolation{Name: q.name, Count: n})
	}
	return out, nil
}

// TotalConstraintViolations is the number of offending rows across every rule.
func TotalConstraintViolations(v []ConstraintViolation) int64 {
	var total int64
	for _, c := range v {
		total += c.Count
	}
	return total
}

// CounterDrift is what blunderdb verify reports about the two denormalised
// counters, match.game_count and game.move_count.
//
// Both are written once, at import, from what the SOURCE FILE held —
// `len(match.Games)`, `len(movesData)` in ingest — and are never recomputed.
// They are what the match list and the game view display. A difference from
// the rows actually stored is therefore not automatically corruption: an
// importer that skipped a game it could not map, or a move that produced no
// row, leaves a legitimate gap. It is still worth seeing, because nothing else
// in the database says the displayed figure and the stored rows disagree.
type CounterDrift struct {
	// MatchesWithWrongGameCount is the number of matches whose game_count is
	// not the number of game rows they hold.
	MatchesWithWrongGameCount int64 `json:"matches_with_wrong_game_count"`
	// GamesWithWrongMoveCount is the number of games whose move_count is not
	// the number of move rows they hold.
	GamesWithWrongMoveCount int64 `json:"games_with_wrong_move_count"`
	// WorstGameCountGap and WorstMoveCountGap are the largest absolute
	// differences found, which is what tells a handful of skipped games from a
	// counter that means nothing at all.
	WorstGameCountGap int64 `json:"worst_game_count_gap"`
	WorstMoveCountGap int64 `json:"worst_move_count_gap"`
}

// Total is the number of rows whose counter disagrees with what they hold.
func (c CounterDrift) Total() int64 {
	return c.MatchesWithWrongGameCount + c.GamesWithWrongMoveCount
}

// CheckCounters recomputes match.game_count and game.move_count from the rows
// and reports how many disagree, and by how much at worst. It only reads;
// nothing is rewritten — the counter records what the source file said, and
// overwriting it with what was stored would erase the very discrepancy that is
// worth looking at.
func (d *Database) CheckCounters() (CounterDrift, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var drift CounterDrift
	if d.db == nil {
		return drift, fmt.Errorf("no database is currently open")
	}
	for _, q := range []struct {
		name      string
		query     string
		rows, gap *int64
	}{
		{"match.game_count",
			`SELECT COUNT(*), COALESCE(MAX(gap), 0) FROM (
				SELECT ABS(COALESCE(m.game_count, 0) -
					(SELECT COUNT(*) FROM game g WHERE g.match_id = m.id)) AS gap
				FROM match m) WHERE gap > 0`,
			&drift.MatchesWithWrongGameCount, &drift.WorstGameCountGap},
		{"game.move_count",
			`SELECT COUNT(*), COALESCE(MAX(gap), 0) FROM (
				SELECT ABS(COALESCE(g.move_count, 0) -
					(SELECT COUNT(*) FROM move mv WHERE mv.game_id = g.id)) AS gap
				FROM game g) WHERE gap > 0`,
			&drift.GamesWithWrongMoveCount, &drift.WorstMoveCountGap},
	} {
		if err := d.db.QueryRow(q.query).Scan(q.rows, q.gap); err != nil {
			return drift, fmt.Errorf("recomputing %s: %w", q.name, err)
		}
	}
	return drift, nil
}
