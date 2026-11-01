-- Forward migration: the 2.51.0 wave — a stable UUID version 7 per object.
--
--   * A uuid column on position, comment, collection, match,
--     transcription, lesson and anki_card, unique per tenant. The integer ids
--     stay the keys; nothing reads the column yet.
--   * New rows draw their uuid from blunderdb_uuid7() by column default. The
--     column is added bare and the default set afterwards: ADD COLUMN with a
--     volatile default would rewrite every table under an exclusive lock.
--   * The rows held before are named by Go (backfillObjectUUIDs, run by
--     Migrate): batched, with FORCE lifted for one batch at a time, from
--     blunderdb_derive_uuid, the SQL twin of sqlshared.DeriveObjectUUID.
-- Idempotent: IF NOT EXISTS and CREATE OR REPLACE throughout.
-- Schema-visible: bumps domain.DatabaseVersion to 2.51.0.

CREATE OR REPLACE FUNCTION blunderdb_uuid7() RETURNS uuid
LANGUAGE sql VOLATILE AS $$
    SELECT encode(set_bit(set_bit(
        overlay(uuid_send(gen_random_uuid())
                placing substring(int8send(floor(extract(epoch FROM clock_timestamp()) * 1000)::bigint) FROM 3)
                FROM 1 FOR 6),
        52, 1), 53, 1), 'hex')::uuid
$$;

CREATE OR REPLACE FUNCTION blunderdb_derive_uuid(family text, key text, rank bigint) RETURNS uuid
LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT ('00000000-0000-7' || substr(h, 1, 3) || '-' || substr('89ab', 1 + (get_byte(d, 16) & 3), 1)
            || substr(h, 4, 3) || '-' || substr(h, 7, 12))::uuid
      FROM (SELECT sha256(convert_to('blunderdb/uuid/1|' || family || '|' || key || '|' || rank::text, 'UTF8')) AS d) x,
           LATERAL (SELECT encode(x.d, 'hex') AS h) y
$$;

ALTER TABLE position      ADD COLUMN IF NOT EXISTS uuid uuid;
ALTER TABLE comment       ADD COLUMN IF NOT EXISTS uuid uuid;
ALTER TABLE collection    ADD COLUMN IF NOT EXISTS uuid uuid;
ALTER TABLE match         ADD COLUMN IF NOT EXISTS uuid uuid;
ALTER TABLE transcription ADD COLUMN IF NOT EXISTS uuid uuid;
ALTER TABLE lesson        ADD COLUMN IF NOT EXISTS uuid uuid;
ALTER TABLE anki_card     ADD COLUMN IF NOT EXISTS uuid uuid;

ALTER TABLE position      ALTER COLUMN uuid SET DEFAULT blunderdb_uuid7();
ALTER TABLE comment       ALTER COLUMN uuid SET DEFAULT blunderdb_uuid7();
ALTER TABLE collection    ALTER COLUMN uuid SET DEFAULT blunderdb_uuid7();
ALTER TABLE match         ALTER COLUMN uuid SET DEFAULT blunderdb_uuid7();
ALTER TABLE transcription ALTER COLUMN uuid SET DEFAULT blunderdb_uuid7();
ALTER TABLE lesson        ALTER COLUMN uuid SET DEFAULT blunderdb_uuid7();
ALTER TABLE anki_card     ALTER COLUMN uuid SET DEFAULT blunderdb_uuid7();

CREATE UNIQUE INDEX IF NOT EXISTS idx_position_uuid      ON position (tenant_id, uuid);
CREATE UNIQUE INDEX IF NOT EXISTS idx_comment_uuid       ON comment (tenant_id, uuid);
CREATE UNIQUE INDEX IF NOT EXISTS idx_collection_uuid    ON collection (tenant_id, uuid);
CREATE UNIQUE INDEX IF NOT EXISTS idx_match_uuid         ON match (tenant_id, uuid);
CREATE UNIQUE INDEX IF NOT EXISTS idx_transcription_uuid ON transcription (tenant_id, uuid);
CREATE UNIQUE INDEX IF NOT EXISTS idx_lesson_uuid        ON lesson (tenant_id, uuid);
CREATE UNIQUE INDEX IF NOT EXISTS idx_anki_card_uuid     ON anki_card (tenant_id, uuid);
