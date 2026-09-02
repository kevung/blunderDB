# Security Policy

## Reporting a vulnerability

Please do not open a public issue for a security problem. Use one of:

- GitHub private vulnerability reporting on this repository
  (**Security** tab, *Report a vulnerability*);
- email to **blunderdb@proton.me**.

Include what you found, how to reproduce it, and the version or commit
involved. You will get an acknowledgement, and a fix or an explanation, as
soon as the maintainer can manage — this is a one-person free-software
project, so allow a few days rather than hours.

Only the latest release is supported; fixes ship as a new release, not as
patches to older ones.

## What is, and is not, a vulnerability here

**The `serve` daemon performs no authentication, by design.** It trusts the
`X-Tenant-ID` header and must run behind an authenticating reverse proxy on a
private network. Reaching another tenant's data through an *unproxied*
daemon is not a vulnerability in blunderDB; a way to do so *through* a
correctly configured proxy is. See
[ADR-0005](docs/adr/0005-serve-daemon-delegates-authentication.md) and the
warnings in the server-mode documentation — they are not to be weakened.

**Encrypted exports (`.dbx`)** are protected with AES-256-GCM under a key
derived from the passphrase with Argon2id (fixed parameters: 3 passes,
64 MiB, 4 lanes, recorded in the file and refused when they differ). Since
container version 2 the cleartext header — watermark, salt, nonce — is bound
to the payload as the AEAD's additional data, so it cannot be rewritten
without detection either; version-1 files, whose header was not bound, still
open with a logged warning and should be exported again. Reads are bounded
before allocation (1 MiB header, 2 GiB payload). A weak passphrase is the
user's choice; a way to open a container without the passphrase, or to alter
one — header included — without detection, is a vulnerability.

**Watermarks and issuer identity** mark where a database *came from*; nothing
is ever recorded on the recipient's side. Reports that this could be used to
track readers are welcome but should read
[ADR-0007](docs/adr/0007-watermarks-mark-origin-and-nothing-else.md) first.

The desktop application opens local files the user chooses and talks to no
network service of its own. Match files (XG, GNU Backgammon, BGF, `.mat`) and
position text are untrusted input: a crash or a hang on a crafted file is a
bug worth reporting, and the decoders are fuzzed continuously for that reason.

## Automated checks

CI runs `govulncheck` on every push and pull request and fails on a known
vulnerability in a Go dependency; Dependabot proposes weekly updates for Go
modules, the frontend's npm packages and the GitHub Actions in use, which are
pinned by commit SHA.
