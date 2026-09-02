# Homebrew cask: `blunderdb`

[`blunderdb.rb.in`](blunderdb.rb.in) is a
[Homebrew cask](https://docs.brew.sh/Cask-Cookbook) template for the macOS
release asset `blunderDB-macos-<version>.zip` (one universal arm64 + x86_64
`.app`). It installs the bundle as `/Applications/blunderDB.app` and links its
executable as `blunderdb` on the `PATH`, since the same binary is the
command-line interface.

Nothing here is pushed anywhere automatically. Homebrew's main cask repository
(`homebrew/cask`) has notability requirements (GitHub stars/forks thresholds)
a small project rarely meets, so the intended home is a **tap** under the
author's account, `kevung/homebrew-tap`. Creating it and pushing each release
to it is a decision taken by hand.

---

## 1. Rendering the cask

Every release tag renders the cask in CI (`package-manifests` job in
`.github/workflows/build.yml`) and attaches `blunderdb-<version>.rb` to the
GitHub release. It is the file to commit into the tap as `Casks/blunderdb.rb`.

To render locally instead (needs `curl`; the release must already exist so the
`.sha256` can be downloaded):

```bash
scripts/homebrew-render.sh <version>              # → dist/homebrew/Casks/blunderdb.rb
scripts/homebrew-render.sh <version> --assets DIR # read the .sha256 from DIR
```

The checksum comes from the published `blunderDB-macos-<version>.zip.sha256`,
never recomputed from a local build: the cask must describe the exact bytes
users download.

## 2. Creating the tap (once)

A tap is a GitHub repository named `homebrew-<name>`; Homebrew maps
`kevung/tap` to `github.com/kevung/homebrew-tap`.

```bash
gh repo create kevung/homebrew-tap --public --description "Homebrew tap for blunderDB" --clone
cd homebrew-tap
mkdir Casks
cp /path/to/blunderdb-<version>.rb Casks/blunderdb.rb
git add Casks/blunderdb.rb
git commit -m "blunderdb <version>"
git push
```

No `Formula/` directory, no `tap_migrations`, nothing else is required. Users
then run:

```bash
brew tap kevung/tap
brew install --cask blunderdb
```

## 3. Each release

Copy the `blunderdb-<version>.rb` release asset over `Casks/blunderdb.rb` in
the tap, commit, push. `brew upgrade --cask blunderdb` picks it up; the
`livecheck` block lets `brew livecheck blunderdb` compare the tap against the
latest GitHub release.

## 4. Validating on a Mac

```bash
brew style --fix Casks/blunderdb.rb       # rubocop, Homebrew's cask rules
brew audit --cask --online Casks/blunderdb.rb
brew install --cask ./Casks/blunderdb.rb  # local install test
blunderdb version
brew uninstall --cask blunderdb
```

## 5. Not notarized: what it means for `brew install --cask`

The `.app` carries no Apple Developer signature and is not notarized (see
`doc/source/annexe_mac_securite.rst`, published at
<https://kevung.github.io/blunderDB/en/annexe_mac_securite.html>). Homebrew
installs it anyway — a cask is a scripted download, not an App Store
submission — but macOS stamps the downloaded zip with the quarantine
attribute, so the first launch is refused by Gatekeeper with the usual
"cannot be opened because the developer cannot be verified" dialog. Users have
two ways through, both spelled out in the cask's `caveats` (printed after
`brew install`):

- right-click `blunderDB.app` in `/Applications` and choose *Open*, once;
- or install with `brew install --cask --no-quarantine blunderdb`, which skips
  the quarantine attribute altogether (the `--no-quarantine` flag is a
  Homebrew option, not something the cask can set on the user's behalf).

`brew upgrade` re-downloads a quarantined zip, so the right-click dance is
needed again after every upgrade unless the user installed with
`--no-quarantine`. Homebrew does not, and cannot, sign the bundle: only an
Apple Developer Program membership and a notarization step in CI would remove
the dialog, and that is out of scope for this project (see the appendix).
