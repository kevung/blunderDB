-- Forward migration: extend anki_card with suspend/bury state. A suspended card
-- is excluded from review indefinitely; a buried card is hidden until
-- buried_until passes (typically the next day). Idempotent, so it is safe on a
-- fresh database (whose baseline already includes the columns) and on existing
-- databases bootstrapped before they existed.

ALTER TABLE anki_card ADD COLUMN IF NOT EXISTS suspended BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE anki_card ADD COLUMN IF NOT EXISTS buried_until TIMESTAMPTZ;

UPDATE metadata SET value = '2.12.0' WHERE key = 'database_version';
