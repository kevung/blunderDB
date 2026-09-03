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
}

// Total is the number of orphaned rows across all four relations.
func (o OrphanCounts) Total() int64 {
	return o.GamesWithoutMatch + o.MovesWithoutGame + o.MoveAnalysesWithoutMove + o.AnalysesWithoutPosition
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
