# winget manifests: `KevinUnger.blunderDB`

[winget](https://learn.microsoft.com/windows/package-manager/) installs from
manifests hosted in the public
[microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs) repository.
This directory holds the three manifest templates (`*.yaml.in`, schema 1.6.0)
for the package `KevinUnger.blunderDB`:

| File | Manifest |
|---|---|
| `KevinUnger.blunderDB.yaml.in` | version |
| `KevinUnger.blunderDB.installer.yaml.in` | installer — type `portable`, the release `.exe`, its SHA-256, command alias `blunderdb` |
| `KevinUnger.blunderDB.locale.en-US.yaml.in` | defaultLocale — publisher, description, tags, links |

The release `.exe` is a single self-contained binary, hence `portable`: winget
copies it under its portable root, adds it to `PATH` through the `blunderdb`
alias, and uninstalls by deleting it. `winget upgrade` works version to version.

Nothing here is submitted automatically. Publishing to winget-pkgs is a pull
request against a Microsoft repository, reviewed by humans and by automated
validation, and stays a decision taken by hand for each release.

---

## 1. Rendering the manifests

Every release tag renders the manifests in CI (`package-manifests` job in
`.github/workflows/build.yml`) and attaches
`blunderDB-winget-manifests-<version>.zip` to the GitHub release. The zip holds
the `manifests/k/KevinUnger/blunderDB/<version>/` tree winget-pkgs expects.

To render locally instead (needs `curl`; the release must already exist so the
`.sha256` can be downloaded):

```bash
scripts/winget-render.sh <version>              # → dist/winget/manifests/…
scripts/winget-render.sh <version> --assets DIR # read the .sha256 from DIR
```

The checksum comes from the published
`blunderDB-windows-<version>.exe.sha256`, never recomputed from a local build:
the manifest must describe the exact bytes users download.

## 2. Validating on a Windows machine

```powershell
winget validate --manifest .\manifests\k\KevinUnger\blunderDB\<version>
winget install  --manifest .\manifests\k\KevinUnger\blunderDB\<version>   # local test
blunderdb version
winget uninstall KevinUnger.blunderDB
```

`winget install --manifest` requires the *local manifest files* setting
(`winget settings --enable LocalManifestFiles`, as administrator).

## 3. Submitting to microsoft/winget-pkgs

Two equivalent routes; both need a GitHub account and open a pull request in
your name.

**`wingetcreate`** (Microsoft's tool, `winget install wingetcreate`):

```powershell
# First submission: interactive, creates the fork + PR from the rendered files
wingetcreate submit --token <github-pat> .\manifests\k\KevinUnger\blunderDB\<version>

# Later versions, once the package exists upstream: no rendering needed,
# wingetcreate rewrites the previous manifests with the new URL and hash
wingetcreate update KevinUnger.blunderDB --version <version> `
  --urls https://github.com/kevung/blunderDB/releases/download/<version>/blunderDB-windows-<version>.exe `
  --submit --token <github-pat>
```

**Manual PR**: fork microsoft/winget-pkgs, copy the rendered
`manifests/k/KevinUnger/blunderDB/<version>/` directory into the fork at the
same path, open a PR titled `New package: KevinUnger.blunderDB version <version>`
(`Update: …` afterwards). The repository's bot runs the schema validation, the
installer download + hash check and a Defender/SmartScreen scan.

What to expect from review:

- **Unsigned, UPX-compressed binary.** The `.exe` carries no Authenticode
  signature (see the Windows security appendix in the user docs) and is packed
  with UPX. Both are accepted by winget-pkgs, but either can trip the automated
  malware scan into a false positive; the reviewers then ask for a
  Defender submission ("false positive" report at
  <https://www.microsoft.com/wdsi/filesubmission>) before merging.
- **Identifier stability.** `KevinUnger.blunderDB` is `<Publisher>.<Package>`;
  once merged it cannot be renamed, only superseded.
- **One PR per version.** Do not batch several versions.

## 4. After the first merge

Users install with:

```powershell
winget install KevinUnger.blunderDB
```

Then, for each release, only step 3 (an `Update:` PR) remains. The CI zip on
the release is the input; nothing in this repository needs editing unless the
asset name, the publisher block or the description changes — in which case edit
the `.in` templates here.
