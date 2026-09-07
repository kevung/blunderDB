-- Forward migration: an Anki card can be a score (issue #324, ADR-0042).
--
-- A card carries a KIND and a KEY. `position` with the position's id written
-- as text, which is what every card that already exists becomes — nothing old
-- changes meaning — and `score` with the unordered score ("3:5"), the deck the
-- application fills with the 36 scores of 2 to 9 away.
--
-- position_id therefore stops being mandatory. A score card has no position,
-- and writing 0 there would be a foreign key pointing at no row: the column
-- becomes NULL-able, and `kind` is what says whether reading it means
-- anything. The composite (tenant_id, position_id) foreign key keeps working
-- unchanged — a MATCH SIMPLE key with a NULL column is not enforced, which is
-- exactly the semantics wanted here.

ALTER TABLE anki_card ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'position';
ALTER TABLE anki_card ADD COLUMN IF NOT EXISTS key  TEXT NOT NULL DEFAULT '';
ALTER TABLE anki_review_log ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'position';
ALTER TABLE anki_review_log ADD COLUMN IF NOT EXISTS key  TEXT NOT NULL DEFAULT '';

-- Every card and every logged review that predates the columns is a position
-- card. Set-based and restricted to the rows the default left empty, so a
-- second application changes nothing.
UPDATE anki_card       SET key = position_id::text WHERE key = '' AND position_id IS NOT NULL;
UPDATE anki_review_log SET key = position_id::text WHERE key = '' AND position_id IS NOT NULL;

ALTER TABLE anki_card       ALTER COLUMN position_id DROP NOT NULL;
ALTER TABLE anki_review_log ALTER COLUMN position_id DROP NOT NULL;

-- A deck holds one card per question, whatever the question is about. The
-- pair (deck_id, position_id) said that about the only kind of card that
-- existed before today; (deck_id, kind, key) says it about all of them, and it
-- is the index the sync's "ON CONFLICT (deck_id, kind, key)" infers. No
-- tenant_id in either: deck ids are global (BIGSERIAL), so a deck belongs to
-- one tenant by construction — the same reason the constraint it replaces
-- carried none.
ALTER TABLE anki_card DROP CONSTRAINT IF EXISTS anki_card_deck_id_position_id_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_anki_card_identity ON anki_card (deck_id, kind, key);
