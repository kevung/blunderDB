-- Forward migration: the Anki review journal's deck_id and position_id become
-- real foreign keys (issue #185, schema 2.18.0).
--
-- anki_review_log.card_id has always cascaded from anki_card, and a card
-- cascades from both its deck and its position — so in practice a log row
-- cannot outlive either. "In practice" is what the orphans of issue #157 were
-- made of, though: a deletion made on a connection that did not enforce foreign
-- keys left rows behind that the schema said could not exist. The two columns
-- now say what they mean.
--
-- Composite (tenant_id, …) from the start, not the single-column form issue
-- #185 first shipped this migration with: 017 (issue #235) promotes every
-- other tenant-scoped foreign key the same way, and a freshly bootstrapped
-- database's 001 baseline already declares these three composite (card_id
-- included) — writing single-column FKs here only to have 017 immediately
-- replace them would leave a database that ran 016 before 017 existed
-- briefly, and pointlessly, unprotected against a mismatched tenant_id.
--
-- Added NOT VALID: the constraint governs every row written from here on, and
-- the rows already there are not scanned. A database that carries a dangling
-- journal row must keep opening — `blunderdb verify` counts those rows, and
-- purging a user's review history is not a migration's decision to take. An
-- operator who has checked can promote the constraint with
-- `ALTER TABLE anki_review_log VALIDATE CONSTRAINT anki_review_log_deck_tenant_fkey`.
--
-- Idempotent: each constraint is added only if no constraint of that name
-- exists on the table, which is what a freshly bootstrapped database (whose
-- 001 baseline declares both) already has.

DO $$
DECLARE
    c record;
BEGIN
    FOR c IN SELECT * FROM (VALUES
        ('anki_review_log_deck_tenant_fkey',     'deck_id',     'anki_deck'),
        ('anki_review_log_position_tenant_fkey', 'position_id', 'position')
    ) AS t(name, col, ref) LOOP
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint pc
            JOIN pg_class rel ON rel.oid = pc.conrelid
            JOIN pg_namespace n ON n.oid = rel.relnamespace
            WHERE n.nspname = current_schema()
              AND rel.relname = 'anki_review_log'
              AND pc.conname = c.name
        ) THEN
            EXECUTE format(
                'ALTER TABLE anki_review_log ADD CONSTRAINT %I FOREIGN KEY (tenant_id, %I) REFERENCES %I (tenant_id, id) ON DELETE CASCADE NOT VALID',
                c.name, c.col, c.ref);
        END IF;
    END LOOP;
END $$;
