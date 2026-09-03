# Signed release checksums (minisign)

Every release already carries a `SHA256SUMS` manifest (one line per asset,
consolidated from the per-file `.sha256` the `build` job produces) and, since
E.7 (#223), a [SLSA build-provenance
attestation](https://github.com/kevung/blunderDB/blob/main/.github/workflows/build.yml)
on every asset — no key to manage, verified with:

```bash
gh attestation verify <asset> --repo kevung/blunderDB
```

A detached [minisign](https://jedisct1.github.io/minisign/) signature over
`SHA256SUMS` is a second, independent way to verify a release, and is
**opt-in**: the `release` job in `.github/workflows/build.yml` signs it only
when the `MINISIGN_SECRET_KEY` repository secret is present, and ships the
manifest unsigned otherwise — the same guard-by-secret pattern as
`homebrew-tap.yml` (see `packaging/homebrew/README.md`). No release is ever
blocked by this being unset.

## Enabling it (once)

1. Generate a keypair (`minisign` from your package manager, or
   [`rsign2`](https://crates.io/crates/rsign2) — both produce
   minisign-compatible files):

   ```bash
   minisign -G -p packaging/minisign/minisign.pub -s /tmp/minisign.key -c "blunderDB release signing key (https://github.com/kevung/blunderDB)"
   ```

   Choose a password when prompted, or `-W` for an unencrypted key (then
   `MINISIGN_PASSWORD` below is not needed).

2. Commit the **public** key only:

   ```bash
   git add packaging/minisign/minisign.pub
   git commit -m "chore(release): enable minisign signing of SHA256SUMS"
   ```

3. Store the **secret** key (never committed) as a repository secret, and the
   password too if the key is encrypted:

   ```bash
   gh secret set MINISIGN_SECRET_KEY < /tmp/minisign.key
   gh secret set MINISIGN_PASSWORD   # only if the key is password-protected
   rm /tmp/minisign.key
   ```

The next tag push signs `SHA256SUMS` and attaches `SHA256SUMS.minisig` to the
release. Verify with:

```bash
minisign -V -p packaging/minisign/minisign.pub -m SHA256SUMS -x SHA256SUMS.minisig
sha256sum -c SHA256SUMS   # then check the individual asset
```

## Rotating or revoking

Repeat step 1 with a new keypair, replace `packaging/minisign/minisign.pub`,
and overwrite both secrets the same way. There is nothing else to clean up:
past releases keep the signature they were signed with, and old signatures
are never re-verified against a new key automatically.
