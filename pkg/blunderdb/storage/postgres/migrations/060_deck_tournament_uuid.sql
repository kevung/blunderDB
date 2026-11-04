-- Forward migration: the 2.57.0 wave — the Anki deck and the tournament join
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
--   * One journal trigger per table, the function being 055's; the update
--     trigger of a deck and of a tournament, and 055's of a match, leave out
--     the columns that are local (see below).
-- No COMMIT, no role switch: it holds on blunderdb_user under FORCE RLS.
-- ALTER TABLE and CREATE TRIGGER lock the two (small) tables, and the trigger of
-- match for the instant of its swap: lock_timeout
-- makes a busy one fail the migration, retried at the next start, rather
-- than queue writers behind a long transaction. Idempotent.
-- Schema-visible: bumps domain.DatabaseVersion to 2.57.0.

SET LOCAL lock_timeout = '15s';

ALTER TABLE anki_deck  ADD COLUMN IF NOT EXISTS uuid uuid;
ALTER TABLE tournament ADD COLUMN IF NOT EXISTS uuid uuid;

ALTER TABLE anki_deck  ALTER COLUMN uuid SET DEFAULT blunderdb_uuid7();
ALTER TABLE tournament ALTER COLUMN uuid SET DEFAULT blunderdb_uuid7();

CREATE UNIQUE INDEX IF NOT EXISTS idx_anki_deck_uuid  ON anki_deck (tenant_id, uuid);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tournament_uuid ON tournament (tenant_id, uuid);

-- What a local gesture rewrites without the object having changed stays out of
-- what dates it: a synchronisation of a deck touches updated_at, a tournament's
-- order and rencontre are the library's own, and nothing of that is in the
-- changeset. sqlshared.LocalColumns spells the same lists.
DROP TRIGGER IF EXISTS change_log_anki_deck ON anki_deck;
CREATE TRIGGER change_log_anki_deck AFTER INSERT OR DELETE ON anki_deck
    FOR EACH ROW EXECUTE FUNCTION blunderdb_journal('anki_deck');
DROP TRIGGER IF EXISTS change_log_anki_deck_update ON anki_deck;
CREATE TRIGGER change_log_anki_deck_update AFTER UPDATE ON anki_deck
    FOR EACH ROW WHEN (
        (to_jsonb(OLD) - ARRAY['updated_at']) IS DISTINCT FROM (to_jsonb(NEW) - ARRAY['updated_at'])
    ) EXECUTE FUNCTION blunderdb_journal('anki_deck');
DROP TRIGGER IF EXISTS change_log_tournament ON tournament;
CREATE TRIGGER change_log_tournament AFTER INSERT OR DELETE ON tournament
    FOR EACH ROW EXECUTE FUNCTION blunderdb_journal('tournament');
DROP TRIGGER IF EXISTS change_log_tournament_update ON tournament;
CREATE TRIGGER change_log_tournament_update AFTER UPDATE ON tournament
    FOR EACH ROW WHEN (
        (to_jsonb(OLD) - ARRAY['updated_at', 'sort_order', 'rencontre_id', 'rencontre_rooms'])
        IS DISTINCT FROM
        (to_jsonb(NEW) - ARRAY['updated_at', 'sort_order', 'rencontre_id', 'rencontre_rooms'])
    ) EXECUTE FUNCTION blunderdb_journal('tournament');

-- A match's attachment stays in each library: a tournament travels with its
-- header and note, and no match names one. tournament_id was local since 055;
-- the order the match holds there is too, so reordering a tournament dates no
-- match and cannot pass a stale copy for the later write.
DROP TRIGGER IF EXISTS change_log_match_update ON match;
CREATE TRIGGER change_log_match_update AFTER UPDATE ON match
    FOR EACH ROW WHEN (
        (to_jsonb(OLD) - ARRAY['import_date', 'tournament_id', 'tournament_sort_order', 'import_batch_id', 'last_visited_position', 'last_visited_at'])
        IS DISTINCT FROM
        (to_jsonb(NEW) - ARRAY['import_date', 'tournament_id', 'tournament_sort_order', 'import_batch_id', 'last_visited_position', 'last_visited_at'])
    ) EXECUTE FUNCTION blunderdb_journal('match');
