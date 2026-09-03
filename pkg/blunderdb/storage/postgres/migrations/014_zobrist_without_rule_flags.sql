-- Forward migration: the Jacoby and beaver flags leave the position identity
-- (ADR-0028, issue #171, schema 2.18.0).
--
-- Until 2.18.0 engine.ZobristHash folded has_jacoby and has_beaver into the
-- hash. They state which optional rules the session was played under, not what
-- is on the board, and no import format but an XGID carries them — every file
-- importer leaves both at 0. The same money position pasted from an XGID
-- (Jacoby set) and imported from a match file (Jacoby clear) therefore hashed
-- differently and landed on two rows inside one tenant.
--
-- A Zobrist hash is a XOR of keys, so undoing a fold is XORing the same key
-- back in. The two retired keys are drawn from the same fixed seed as every
-- other Zobrist key and are pinned here as signed 64-bit literals; SQLite
-- stores the hash in a signed INTEGER and PostgreSQL in a BIGINT, and `#` is
-- bitwise XOR over the same 64 bits in both.
--   jacoby: 8689949525750215883 (0x7898e70d666a74cb)
--   beaver: 3682622054658624179 (0x331b4d3f51f24ab3)
-- zobrist_retired_keys_test.go fails if either literal drifts from the key
-- engine.RetiredFlagDelta returns.
--
-- Rehashing can bring two rows of one tenant onto one hash: those two rows were
-- always the same position, and are merged with the OLDER row (lowest id) kept,
-- as the SQLite step does through mergePositionInto. What hangs off the
-- duplicate follows it onto the keeper; what the keeper already has wins.
--
-- One deliberate difference from the SQLite step: an analysis moved onto the
-- keeper carries a stale `positionId` INSIDE its compressed JSON payload, which
-- SQL cannot rewrite. The row names the right position, every read path uses
-- the row, and the next analysis.save on that position rewrites the payload.
--
-- Applied once, and once only: XORing a key in is its own inverse, so unlike
-- every other migration here this one is NOT idempotent by construction.
-- schema_migrations is the guard — migrateForward records the file and never
-- replays it — and a freshly bootstrapped database carries no pre-2.18.0 hash
-- to convert. The temporary tables are dropped explicitly at both ends because
-- a pooled connection outlives the statement batch.
--
-- Like the backfill in 002, this migration reads and writes every tenant's
-- rows with no app.tenant_id set: on a database where ApplyRLS has FORCEd
-- row-level security it therefore sees nothing and converts nothing. Run it
-- before ApplyRLS, or as a role the policy admits.

DROP TABLE IF EXISTS rehash_final;
DROP TABLE IF EXISTS rehash_dup;

-- The hash each row will END with: unchanged for a row carrying neither flag.
CREATE TEMPORARY TABLE rehash_final AS
SELECT id,
       tenant_id,
       zobrist_hash
         # (CASE WHEN has_jacoby THEN 8689949525750215883 ELSE 0 END)
         # (CASE WHEN has_beaver THEN 3682622054658624179 ELSE 0 END) AS new_hash,
       (COALESCE(has_jacoby, FALSE) OR COALESCE(has_beaver, FALSE))    AS changed
FROM position
WHERE zobrist_hash IS NOT NULL;

-- Duplicates revealed by the rehash: the oldest row of each (tenant, hash) is
-- the keeper, every other row of that group is folded into it.
CREATE TEMPORARY TABLE rehash_dup AS
SELECT r.id AS dup_id, k.keep_id, r.tenant_id
FROM rehash_final r
JOIN (SELECT tenant_id, new_hash, min(id) AS keep_id
      FROM rehash_final GROUP BY tenant_id, new_hash) k
  ON k.tenant_id = r.tenant_id AND k.new_hash = r.new_hash
WHERE r.id <> k.keep_id;

-- Sticky marks are raised on the keeper and never lowered (ADR-0001, ADR-0006).
UPDATE position p SET individually_imported = TRUE
FROM rehash_dup d JOIN position q ON q.id = d.dup_id
WHERE p.id = d.keep_id AND q.individually_imported AND NOT p.individually_imported;

UPDATE position p SET flagged = TRUE
FROM rehash_dup d JOIN position q ON q.id = d.dup_id
WHERE p.id = d.keep_id AND q.flagged AND NOT p.flagged;

-- Everything that hangs off a duplicate follows it onto the keeper.
UPDATE move m SET position_id = d.keep_id FROM rehash_dup d WHERE m.position_id = d.dup_id;
UPDATE comment c SET position_id = d.keep_id FROM rehash_dup d WHERE c.position_id = d.dup_id;
UPDATE anki_review_log l SET position_id = d.keep_id FROM rehash_dup d WHERE l.position_id = d.dup_id;

-- A membership or a card the keeper already holds cannot be moved (UNIQUE);
-- it goes with the duplicate through ON DELETE CASCADE.
UPDATE collection_position cp SET position_id = d.keep_id
FROM rehash_dup d
WHERE cp.position_id = d.dup_id
  AND NOT EXISTS (SELECT 1 FROM collection_position o
                  WHERE o.collection_id = cp.collection_id AND o.position_id = d.keep_id);

UPDATE anki_card ac SET position_id = d.keep_id
FROM rehash_dup d
WHERE ac.position_id = d.dup_id
  AND NOT EXISTS (SELECT 1 FROM anki_card o
                  WHERE o.deck_id = ac.deck_id AND o.position_id = d.keep_id);

-- An analysis follows only onto a keeper that has none: one analysis per
-- position, and the kept row's own analysis wins.
UPDATE analysis a SET position_id = d.keep_id
FROM rehash_dup d
WHERE a.position_id = d.dup_id
  AND NOT EXISTS (SELECT 1 FROM analysis o WHERE o.position_id = d.keep_id);

DELETE FROM position p USING rehash_dup d WHERE p.id = d.dup_id;

-- A plain UNIQUE index is checked row by row, and the rehash is a permutation:
-- row A can be taking the hash row B is about to leave. Drop it for the length
-- of the UPDATE and build it back.
DROP INDEX IF EXISTS idx_position_zobrist;

UPDATE position p SET zobrist_hash = r.new_hash
FROM rehash_final r
WHERE r.id = p.id AND r.changed AND p.zobrist_hash IS DISTINCT FROM r.new_hash;

CREATE UNIQUE INDEX IF NOT EXISTS idx_position_zobrist ON position (tenant_id, zobrist_hash);

DROP TABLE rehash_dup;
DROP TABLE rehash_final;

UPDATE metadata SET value = '2.18.0' WHERE key = 'database_version';
