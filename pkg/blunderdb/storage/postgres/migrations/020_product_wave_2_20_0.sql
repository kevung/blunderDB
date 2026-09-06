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
