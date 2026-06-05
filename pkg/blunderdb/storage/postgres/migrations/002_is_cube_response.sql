-- Forward migration: add the is_cube_response column to position and backfill
-- take/pass responses from the move table. Idempotent, so it is safe on a fresh
-- database (whose v2.7.0 baseline already includes the column) and on existing
-- databases bootstrapped before this column existed.
--
-- The response predicate mirrors engine.IsResponseCubeAction: an action is a
-- take/pass response when, after lower-casing and removing spaces, it does NOT
-- contain "double" and is "dt"/"dp" or contains take/pass/drop.

ALTER TABLE position ADD COLUMN IF NOT EXISTS is_cube_response BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_position_cube_response ON position (tenant_id, decision_type) WHERE is_cube_response;

UPDATE position p
SET is_cube_response = TRUE
FROM (
    SELECT DISTINCT position_id
    FROM move
    WHERE cube_action IS NOT NULL
      AND lower(replace(cube_action, ' ', '')) NOT LIKE '%double%'
      AND (
            lower(replace(cube_action, ' ', '')) IN ('dt', 'dp')
         OR lower(replace(cube_action, ' ', '')) LIKE '%take%'
         OR lower(replace(cube_action, ' ', '')) LIKE '%pass%'
         OR lower(replace(cube_action, ' ', '')) LIKE '%drop%'
      )
) resp
WHERE resp.position_id = p.id
  AND p.decision_type = 1;

UPDATE metadata SET value = '2.10.0' WHERE key = 'database_version';
