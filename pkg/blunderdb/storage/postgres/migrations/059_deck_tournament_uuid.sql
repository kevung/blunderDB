-- Forward migration: the 2.56.0 wave — the Anki deck and the tournament join
-- the objects that carry a stable uuid and a change-journal entry, so the
-- changeset carries them (with the cards, the transcriptions and the other
-- families that already had a uuid since 054).
--
--   * A uuid column on anki_deck and tournament, unique per tenant, named by
--     column default for the rows written from now on. The column is added
--     bare and the default set afterwards: ADD COLUMN with a volatile default
--     would rewrite the table under an exclusive lock.
--   * The rows held before are named by Go (backfillObjectUUIDs, run by
--     Migrate), once per family: a library that passed the pass of 054 still
--     gets this one (its own metadata row, sqlshared.MarkFor).
--   * One journal trigger per table, the function being 055's.
-- No COMMIT, no role switch: it holds on blunderdb_user under FORCE RLS.
-- ALTER TABLE and CREATE TRIGGER lock the two (small) tables: lock_timeout
-- makes a busy one fail the migration, retried at the next start, rather
-- than queue writers behind a long transaction. Idempotent.
-- Schema-visible: bumps domain.DatabaseVersion to 2.56.0.

SET LOCAL lock_timeout = '15s';

ALTER TABLE anki_deck  ADD COLUMN IF NOT EXISTS uuid uuid;
ALTER TABLE tournament ADD COLUMN IF NOT EXISTS uuid uuid;

ALTER TABLE anki_deck  ALTER COLUMN uuid SET DEFAULT blunderdb_uuid7();
ALTER TABLE tournament ALTER COLUMN uuid SET DEFAULT blunderdb_uuid7();

CREATE UNIQUE INDEX IF NOT EXISTS idx_anki_deck_uuid  ON anki_deck (tenant_id, uuid);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tournament_uuid ON tournament (tenant_id, uuid);

DROP TRIGGER IF EXISTS change_log_anki_deck ON anki_deck;
CREATE TRIGGER change_log_anki_deck AFTER INSERT OR UPDATE OR DELETE ON anki_deck
    FOR EACH ROW EXECUTE FUNCTION blunderdb_journal('anki_deck');
DROP TRIGGER IF EXISTS change_log_tournament ON tournament;
CREATE TRIGGER change_log_tournament AFTER INSERT OR UPDATE OR DELETE ON tournament
    FOR EACH ROW EXECUTE FUNCTION blunderdb_journal('tournament');
