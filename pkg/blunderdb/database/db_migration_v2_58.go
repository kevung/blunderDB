package database

import "context"

// migrate_2_57_0_to_2_58_0 has nothing to rewrite. 2.58.0 lets an analysis
// blob carry one checker evaluation per source (domain.CheckerSourceAnalysis)
// beside the merged view; a blob written before holds a single source, which
// is its merged view (engine.CheckerSources), so no row is decoded or
// recomputed. The bump raises domain.MinReaderVersion: a 2.57 decoder rejects
// the new payload flag.
func (d *Database) migrate_2_57_0_to_2_58_0(context.Context) error { return nil }
