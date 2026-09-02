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
