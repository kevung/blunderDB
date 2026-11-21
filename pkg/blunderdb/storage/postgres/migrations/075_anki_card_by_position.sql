-- Forward migration: the cards of a position are read by the position
-- (no schema change a reader sees, DatabaseVersion untouched).
--
-- A table of positions lists, per row, the decks that hold it
-- (ListFieldsStore): one query for a page of ids, which without this index
-- reads every anki_card row of the tenant — no anki_card index leads with
-- position_id since the (deck_id, position_id) key gave way to the card's
-- identity. The same index serves the cascade from a deleted position through
-- the (tenant_id, position_id) foreign key, which also had none.
--
-- Back: DROP INDEX idx_anki_card_position;
--
-- CREATE INDEX holds a SHARE lock on anki_card (reads pass, writes wait) until
-- this file's transaction ends. lock_timeout makes a busy table fail the
-- migration, retried at the next start. Idempotent. No COMMIT, no role
-- switch: it holds on blunderdb_user under FORCE RLS.

SET LOCAL lock_timeout = '15s';

CREATE INDEX IF NOT EXISTS idx_anki_card_position ON anki_card (tenant_id, position_id);
