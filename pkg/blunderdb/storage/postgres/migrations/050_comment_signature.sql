-- Forward migration: the 2.47.0 wave — a comment can carry its producer's
-- signature.
--
--   * comment.signature — the sealed statement over the comment's text and
--     author (issuance.SealComment), written into an exported file and kept
--     verbatim by an import; NULL when unsigned. An edit clears it (ADR-0099).
-- Schema-visible: bumps domain.DatabaseVersion to 2.47.0.

ALTER TABLE comment ADD COLUMN IF NOT EXISTS signature TEXT;
