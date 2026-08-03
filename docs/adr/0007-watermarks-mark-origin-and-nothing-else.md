# Watermarks mark origin and nothing else; the recipient's side records nothing

## Status

accepted

Supersedes the first, unreleased version of this decision, which added per-recipient
watermarks, a holder registry and an import lineage. That design was implemented, then
removed before release — see *Considered options*.

## Context

Teachers hand out databases of positions after their lessons and would like them not to
travel further than the people they taught. The request arrives as "can we put a password on
it, or a watermark, and can I see who has a copy".

Three facts settle what is possible before any design starts. blunderDB is **MIT-licensed and
public**, so any restriction it enforces can be removed by recompiling. The distributed
artefact is a **plain SQLite file**, so `sqlite3 course.db .dump` extracts everything without
running blunderDB at all. And the driver is `modernc.org/sqlite`, which cannot encrypt —
encryption at rest would mean replacing the driver (SQLCipher via CGO, which breaks the
pure-Go build of `cmd/serve`, or a wasm driver), a migration touching every backend, both
storage implementations, the pool, the PRAGMAs and the CI matrix.

The first version of this decision concluded that prevention was unreachable and that
**attribution** was the achievable goal: a per-recipient watermark, a registry of the machines
that had opened each copy, and a lineage carried forward on import, so a leaked copy would
name its origin.

That design was built and then withdrawn after the people it was for looked at it. The
objection was not technical. **In practice an author does not litigate over a position
database.** The apparatus therefore bought a remedy nobody would use, while costing:

- a mechanism that quietly recorded, on someone else's machine, that they had opened a file;
- a privacy obligation the teacher would have carried without knowing it;
- a promise ("discreet, not concealed") that had to be defended in the manual because it was
  uncomfortable;
- and a substantial amount of code — registries, chaining, salts, batches, lineage — in a
  tool whose subject is backgammon.

## Decision

Two mechanisms, both optional, both chosen at export, freely combined:

**A Watermark states the origin of a database.** Who produced it, what it is, and any note the
producer wants attached — terms of use, a contact address, a request not to redistribute. It
is signed with the producer's **Issuer identity** (Ed25519, created silently on the first
watermarked export, exportable as a single `.bdbid` file), so it cannot be altered and cannot
be fabricated in someone else's name. The promise, stated in these words in the manual and in
the UI:

> **tamper-evident and unforgeable, never unremovable.**

**A password wraps the export in an encrypted container** (`.bdbx`, Argon2id + AES-GCM). It
protects the file *in transit* — the stray copy in a downloads folder, the attachment
forwarded by mistake — not the database, since whoever the password was given to can open it.
The container's header is **cleartext**, so the origin of a file stays readable without it.
The recipient is asked for the password once; the result is an ordinary database.

**The recipient's side records nothing.** No registry, no log, no counter, no lineage —
opening a watermarked database is exactly like opening any other. This is the load-bearing
constraint, not a consequence: every mechanism above is a write performed by the producer, on
a file the producer is making, before anyone else has it.

The Watermark is stored as canonical JSON in one `metadata` row: no new table, no
`DatabaseVersion` bump, no PostgreSQL migration, nothing for the serve daemon to implement.
Export copies metadata by **allow-list** (`issuance.CarriedMetadataKeys`).

## Considered options

- **The previous design — per-recipient watermarks, holder registry, lineage.** Implemented,
  then removed. It was coherent and it worked; it answered a question its users did not have.
  Its parts are worth naming so they are not proposed again: a *holder registry* (one entry
  per machine that opened a copy) required writing to a file on someone else's disk; a
  *lineage* carried forward on import required the same; an *issue register* of recipients
  and passwords had to be kept out of every exported file by hand. Each was defensible alone;
  together they made a position database into a tracking system.
- **Encrypting the database at rest.** Rejected on cost, not principle: it forces a driver
  change through the whole storage layer for a barrier the legitimate recipient holds the key
  to anyway. A transport container buys the same practical effect for a fraction of the
  surface.
- **Phoning home on open**, the only channel that would tell a producer in real time who holds
  a copy. Rejected: it makes a local-first, offline tool depend on a network, turns a teacher
  into a data controller over their students, fails behind any firewall, and is removed by the
  first fork.
- **Spreading the Watermark redundantly across tables** so removal takes effort. Rejected as
  obfuscation that cannot work in a public MIT repository with a published schema.
- **Watermarking the content itself** — per-copy variations in the set of positions. Rejected:
  it corrupts the teaching material it is meant to protect, and collides with Zobrist
  deduplication.
- **An unsigned watermark**, saving the identity machinery. Rejected: a free-text origin
  anyone can type is worth little, and the identity is created without the user ever being
  asked, so it costs them nothing.
- **Real tables instead of a `metadata` document.** Rejected: this is file-level data with no
  join, search or index to offer, and normalising it would mean rebuilding a byte-identical
  document to check a signature — the classic source of signature bugs that surfaces months
  later on an accent or an empty field. Storing the exact signed bytes removes the class.

## Consequences

- A recipient who redistributes a file passes on its origin unless they deliberately strip
  it. That is the whole of what is claimed, and the manual says so in those words.
- **Nobody can be accused wrongly**, and nobody is watched. Both follow from the same choice.
- Importing a watermarked database into one's own carries nothing over — the mark stays with
  the file it was applied to. A recipient's database is theirs.
- The signature proves "marked by the holder of that key", not "that key is Jean Dupont".
  Publishing the fingerprint is what ties the two, and is optional: a producer verifying their
  own file matches it against their own identity.
- The desktop app gains a durable per-person identity. Losing it means later exports carry a
  different fingerprint; files already marked still verify, since they embed the public key.
  Rotation and revocation are out of scope.
- `blunderdb info` reads the origin of any file, including a protected one, and never writes.
- The `.bdbx` container is a **new distributed file format**. Once published it cannot be
  withdrawn.
