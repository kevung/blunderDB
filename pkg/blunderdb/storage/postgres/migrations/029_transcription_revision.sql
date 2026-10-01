-- Forward migration: the 2.27.0 wave — transcription.revision (ADR-0057 rule 4).
--
-- A gesture names the revision of the draft it was typed against; a write
-- that finds another revision is refused with a conflict. The version is a
-- column, not a field of the JSON document, so that it is read and compared
-- without decoding the draft. Existing rows start at 1, as a fresh insert
-- does. Schema-visible: bumps domain.DatabaseVersion to 2.27.0.

ALTER TABLE transcription ADD COLUMN IF NOT EXISTS revision BIGINT NOT NULL DEFAULT 1;
