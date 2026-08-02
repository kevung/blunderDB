# Issued copies are attributable, never unshareable

## Status

accepted

## Context

Teachers hand out databases of positions after their lessons and would like them not to
travel further than the people they taught. The request arrives naturally as "can we put a
password on the database, or a watermark, and can I see who has a copy".

Three facts settle most of what is possible before any design starts.

blunderDB is **MIT-licensed and public**, so any restriction it enforces can be removed by
recompiling. The distributed artefact is a **plain SQLite file**, so `sqlite3 course.db
.dump` extracts everything without running blunderDB at all. And the driver is
`modernc.org/sqlite`, which cannot encrypt — encryption at rest would mean replacing the
driver (SQLCipher via CGO, which breaks the pure-Go build of `cmd/serve`, or a wasm driver),
a migration touching every backend, both storage implementations, the pool, the PRAGMAs and
the CI matrix.

The desktop app also makes **no network calls whatsoever** (`net/http` appears only in the
serve daemon), and `AnalyzeImportDatabase` / `CommitImportDatabase` let anyone merge a
received database into their own from the GUI in two clicks.

So the honest question is not "how do we stop redistribution" — that is unreachable — but
"what does a copy that leaks tell us, and what do we promise".

## Decision

An **Issued copy** carries a **Watermark**: a signed statement by its Issuer of which
Distribution it belongs to and, when the Issuer named one, which recipient it was for. The
promise attached to it is stated in exactly these terms, in the manual and in the UI:

> **tamper-evident and unforgeable, never unremovable.**

A Watermark cannot be altered, and no one can fabricate one in someone else's name — so a
copy can never be pinned on the wrong person. It can be deleted by a Holder who sets out to
delete it, and blunderDB says so rather than implying a protection it does not provide.

Four choices follow from that promise.

**Signatures, always, with an identity nobody has to configure.** Watermarks are signed with
an Ed25519 **Issuer identity** created silently on first emission and stored in the config
directory — already an *essential* host capability under ADR-0004. It exports and imports as
a single file so the Issuer can carry it between machines; that transfer file may carry a
passphrase, the local one does not. There is no "do not sign" option: nobody rationally
chooses to make their own watermark forgeable, and unsigned emission would only double the
states the verification path has to explain.

**Attribution, not prevention — and therefore no phone-home.** The Issuer learns nothing
until a copy comes back to them. Two regimes cover how copies are produced: *nominative*,
one file per recipient, attribution certain from the moment of emission; and *collective*,
a single file for a whole group, where the Watermark names the Distribution and the
**Holder registry** is the only attribution vector. The registry records one pseudonymous
fingerprint per distinct machine — salted per Distribution, so copies from the same course
can be cross-referenced with each other but a machine cannot be recognised across courses.
It grows only when the copy is opened as a working database in the GUI: never from the CLI,
never from the daemon. That narrow rule exists so that examining a suspect file cannot write
the examiner's own machine into the evidence.

**Discreet, but never covert.** Nothing interrupts the Holder — no dialogue, no banner, no
prompt for a name. The Watermark and the registry sit in a collapsed section of the metadata
panel for whoever looks, and the manual documents the whole mechanism. The deterrent effect
is deliberately traded away for that discretion; what is kept is a dispositive that can be
produced in a dispute *because* it was documented. The covert variant — silently recording
a third party's identity with nothing anywhere disclosing it — was rejected: it would make
the teacher an undeclared data controller over their students, and would ship a clandestine
tracker in a codebase everyone can read.

**Nothing personal is collected.** The only personal data in an Issued copy is what the
Issuer deliberately wrote — the name of the person they handed it to, a fact about their own
relationship. Everything read from a recipient's machine is a one-way fingerprint.

The whole feature lives in `metadata` key/value rows holding canonical signed JSON
documents: no new table, no `DatabaseVersion` bump, no PostgreSQL migration, no storage
contract test, nothing for the serve daemon to implement. Export copies documents by
**allow-list**: the Watermark is replaced, the Holder registry emptied, **Lineage** carried
forward with the source's own Watermark appended, and the **Issue register** never leaves
the Issuer's database.

Password protection, when it ships, protects the *transport* and not the database: an
encrypted container with a **cleartext header**, decrypted once into an ordinary `.db` on
first opening. It is labelled as protecting against a third party opening a stray file, never
as protecting the database.

## Considered options

- **Encrypting the database at rest.** Rejected on cost, not on principle: it forces a driver
  change through the whole storage layer for a barrier that the legitimate recipient — the
  actual source of leaks — holds the key to anyway. A transport container buys the same
  practical effect for a fraction of the surface.
- **Phoning home on open**, the only channel that would tell an Issuer in real time who holds
  a copy. Rejected: it makes a local-first, offline tool depend on a network, turns a teacher
  into a data controller over their students, fails behind any firewall, and is removed by the
  first fork. The renunciation is explicit — an Issuer will never have a live list of holders,
  only the list of copies they issued plus the ability to identify one that comes back.
- **Spreading the Watermark redundantly across tables** so removal takes effort. Rejected as
  obfuscation that cannot work in a public MIT repository with a published schema: the code
  that hides it documents exactly how to find it, and it dirties the schema for nothing.
- **Watermarking the content itself** — per-copy variations in the set of positions, which
  would survive re-import. Rejected: it corrupts the teaching material it is meant to protect,
  and collides with Zobrist deduplication.
- **Refusing to import an Issued copy** into another database, to close the laundering path.
  Rejected: merging a course into one's own database is exactly what a student paid for.
  Lineage keeps the trace instead.
- **Per-position provenance**, echoing the sticky `individually_imported` flag of ADR-0001.
  Rejected: a position can come from several sources, deduplication merges the rows, and it
  would burden the hottest table in the schema with a file-level fact.
- **Real tables instead of `metadata` documents.** Rejected: this is file-level data with no
  join, search or index to offer, and normalising it would mean rebuilding a byte-identical
  document to check a signature — the classic source of signature bugs that surfaces months
  later on an accent or an empty field. Storing the exact signed bytes removes the class.
- **A blocking "who are you?" dialogue on first opening**, which would have made the seal
  certain and the deterrent real. Rejected once discretion was chosen: the two cannot coexist.
- **A read-only distributed database.** Rejected as a protection — it stops writing, not
  copying — and as a product: annotating, collecting and turning positions into Anki cards is
  what the tool is for. It also cannot coexist with a Holder registry, which needs to write.

## Consequences

- A careless re-sharer is identified; a determined one is not. That is the accepted ceiling,
  and the manual states it in those words so no teacher believes they hold a lock.
- **Nobody can be accused wrongly.** This is the property the signature actually buys, and it
  is worth more than removability would have been.
- The signature proves "made by the holder of key K", not "K is Jean Dupont". In practice the
  person verifying is the person who signed, so the loop closes locally; publishing the
  fingerprint only serves a recipient wanting to check where a file came from.
- `blunderdb info` becomes the forensic entry point, non-mutating by construction. No GUI
  inspection action ships initially — the case is rare and the feature is meant to stay quiet.
- The desktop app gains a durable per-person identity, a concept it did not have before.
  Losing it means later emissions carry a different fingerprint; copies already issued still
  verify, since they embed the public key. Rotation and revocation are out of scope.
- Nominative emission is unusable beyond a handful of recipients until batch emission ships.
  Shipping the core without it risks the regime never being used in practice.
- The encrypted container introduces a **new distributed file format**. Once published it
  cannot be withdrawn, which is why it is sequenced last and separately.
