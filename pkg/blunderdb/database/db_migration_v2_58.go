package database

import "context"

// migrate_2_57_0_to_2_58_0 has nothing to rewrite. 2.58.0 lets an analysis
// blob carry one checker evaluation per source (domain.CheckerSourceAnalysis)
// beside the merged view; a blob written before holds a single source — its
// merged view, when one engine wrote it (engine.CheckerSources) — so no row
// is decoded or recomputed. The step raises no reader floor: the first write
// of a blob carrying sources does, so a database that never holds one stays
// readable by 2.50 to 2.57 (ADR-0113).
func (d *Database) migrate_2_57_0_to_2_58_0(context.Context) error { return nil }
