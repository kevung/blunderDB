-- Forward migration: the 2.20.0 product wave (lot I/J of the 2026-09b plan).
--
--   * position.max_cube — the session's cube ceiling, as the log2 exponent the
--     XGID's tenth field carries (issue #271).
--
-- max_cube defaults to 0, which means "the source stated no ceiling" and NOT
-- "the cube may not pass 1". Every row that predates the column comes from an
-- identifier that either carried no ceiling or carried XG's own no-limit
-- value, so 0 is the truth about them rather than a placeholder.
--
-- It is deliberately NOT part of the Zobrist hash and no rehash follows: only
-- an XGID ever sets it, so hashing it would split one money position across
-- two rows depending on which identifier reached the database first — the
-- conclusion ADR-0028 reached for has_jacoby and has_beaver.

ALTER TABLE position ADD COLUMN IF NOT EXISTS max_cube INTEGER NOT NULL DEFAULT 0;

--   * position.game_type — the derived plan of play (issue #291). Like
--     game_phase before it, it is 0 ("unknown") on every existing row until a
--     repair pass reclassifies them: the label is DERIVED, so backfilling it
--     in SQL would mean writing the classifier twice, once in Go and once in
--     a dialect. positions.reclassify is the one place it is computed.

ALTER TABLE position ADD COLUMN IF NOT EXISTS game_type INTEGER NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_position_game_type ON position (tenant_id, game_type);

--   * collection.filter_query — a LIVING collection (issue #282): a search
--     query in the grammar the command bar speaks, re-evaluated every time the
--     collection is opened. Empty is the ordinary case, a hand-made list whose
--     membership lives in collection_position — so no backfill is needed and
--     no existing collection changes behaviour.

ALTER TABLE collection ADD COLUMN IF NOT EXISTS filter_query TEXT NOT NULL DEFAULT '';
