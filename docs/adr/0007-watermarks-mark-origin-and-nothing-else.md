# Watermarks mark origin and nothing else; the recipient's side records nothing

Status: accepted.

## Context

Teachers hand out position databases and would like them not to travel further. blunderDB is
MIT-licensed and public (any restriction can be recompiled away), the artefact is a plain
SQLite file (`sqlite3 .dump` reads it without blunderDB), and `modernc.org/sqlite` cannot
encrypt. Prevention is unreachable; an author does not litigate over a position database, so
tracking holders buys a remedy nobody uses at the price of surveillance.

## Decision

1. **A Watermark states the origin** of a database: producer, description, free note. It is
   signed with the producer's Issuer identity (Ed25519, created silently on the first
   watermarked export, exportable as one `.bdbid` file). The promise, in these words in the
   manual and the UI: **tamper-evident and unforgeable, never unremovable.**
2. **A password wraps the export in a `.dbx` container**: AES-256-GCM, key by Argon2id
   (64 MiB, 3 passes, 4 lanes, per-file salt). The password is checked on every open; the
   header is cleartext so the origin stays readable. It protects the file in transit, not the
   database. The recipient enters it once and gets an ordinary database.
3. **The recipient's side records nothing**: no registry, log, counter or lineage. Every
   mechanism is a write by the producer, on a file they are making, before anyone else has it.
4. The Watermark is the exact signed canonical JSON in one `metadata` row — no table, no
   `DatabaseVersion` bump, nothing for the serve daemon.
5. Export copies metadata by allow-list (`issuance.CarriedMetadataKeys` / `issuance.Carried`),
   never by exclusion.

## Consequences

- A redistributed file carries its origin unless deliberately stripped — the whole claim.
- Nobody is watched and nobody can be accused wrongly.
- Importing a watermarked database into one's own carries nothing over.
- The signature proves "marked by the holder of that key"; publishing the fingerprint ties it
  to a person. Losing the identity changes later fingerprints; rotation/revocation out of scope.
- `blunderdb info` reads the origin of any file, protected or not, and never writes.
- `.dbx` is a distributed file format: once published it cannot be withdrawn.
- Rejected: per-recipient watermarks, holder registry, import lineage — each writes on someone
  else's disk; together they make a position database a tracking system.
- Rejected: encryption at rest — a driver change through the whole storage layer for a
  barrier the recipient holds the key to.
- Rejected: phoning home — network dependency, makes teachers data controllers, forkable away.
- Rejected: redundant watermark copies, per-copy content variations — obfuscation, or
  corruption of the material and of Zobrist dedup.
- Rejected: an unsigned watermark — worth little, and the identity costs the user nothing.
- Rejected: normalised tables — re-serialising signed bytes breeds signature bugs.

## Guard

`pkg/blunderdb/issuance/issuance_test.go`, `pkg/blunderdb/issuance/container_test.go`,
`pkg/blunderdb/database/export_test.go`.
