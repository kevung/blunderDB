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
modules, the frontend's npm packages and the GitHub Actions in use. Every
`uses:` in `.github/workflows/` is pinned to a commit SHA (the `# vX.Y.Z`
comment names the release it was taken from), and the repository refuses a
workflow that is not; the workflow token is read-only by default and jobs
that publish (release assets, the container image, GitHub Pages) elevate
their own. Secret scanning with push protection and Dependabot security
updates are enabled on the repository, and the `main` branch cannot be
force-pushed or deleted.

<!--
Repository settings, as applied on 2026-09-03 (issue #158), so the next
person knows what is deliberate:

- actions/permissions/workflow: default_workflow_permissions = read.
- actions/permissions: allowed_actions = selected — GitHub-owned and
  verified-creator actions plus the third-party actions the workflows already
  pin (see patterns_allowed); sha_pinning_required = true.
- security_and_analysis: secret_scanning, secret_scanning_push_protection and
  dependabot_security_updates enabled.
- Ruleset "main: no force-push, no deletion" on the default branch:
  `non_fast_forward` and `deletion` only. Deliberately NO required status
  checks and NO required pull request: this repository's workflow merges
  feature branches locally (git worktree, `git merge`) and pushes `main`
  directly — a rule requiring a pull request or a green check on `main`
  would refuse every one of those pushes. The status checks run on the push
  itself and a red `main` is fixed by the next push, not prevented by a gate.
-->
