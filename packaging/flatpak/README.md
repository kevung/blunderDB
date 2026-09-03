# Flatpak: `io.github.kevung.blunderDB`

[`io.github.kevung.blunderDB.yml`](io.github.kevung.blunderDB.yml) is a
Flatpak manifest that repackages the prebuilt `webkit2gtk-4.1` release
tarball (the same artifact the GitHub release publishes) — no Go/Node build
step, `org.gnome.Platform//47` supplies WebKitGTK. This is the simplest route
and is enough to produce a local `.flatpak` bundle or a self-hosted repo. It
is **not** what a Flathub submission needs (see *Flathub* below).

Nothing here is submitted to Flathub automatically. The committed manifest
is both a template **and** buildable as-is: its `url`/`sha256` always point
at the latest published release, kept current by `scripts/release.sh` (see
the comment at the top of the file). Nobody has to remember to update this
file by hand for local builds to work.

---

## 1. Building and installing locally

```bash
flatpak install -y flathub org.gnome.Platform//47 org.gnome.Sdk//47
flatpak-builder --user --install --force-clean build-dir \
  packaging/flatpak/io.github.kevung.blunderDB.yml
flatpak run io.github.kevung.blunderDB
```

`flatpak-builder` downloads the tarball named in the manifest and verifies
its `sha256`, so this always builds the latest released version — no local
release build required.

**Test before trusting the sandbox**: open and *save* a `.db` file located
outside the sandbox (e.g. under `~/Documents`). If Wails' native GTK file
dialog does not route through the XDG file-chooser portal, the
`--filesystem=home` grant in `finish-args` is what makes arbitrary `.db`
paths work; narrow it if portal access proves sufficient on your setup.

## 2. What CI does on a release tag (already automated)

The `flatpak` job in `.github/workflows/build.yml` runs on every tag push,
after the Linux `webkit2gtk-4.1` build:

1. Takes the just-built tarball from the `ubuntu-latest` build artifact
   (not from the GitHub release — no need to wait for the upload).
2. Renders a **copy** of this manifest with a local `path:` source pointing
   at that tarball and its real checksum (the committed `url:`/`sha256:`
   are never read by this job).
3. Builds the bundle inside the `flathub-infra` container image (which
   carries `flatpak-builder` and the GNOME 47 runtime) and attaches
   `blunderDB-<version>.flatpak` + its `.sha256` to the release.

Nothing to maintain here: this path does not depend on the committed
manifest's `url`/`sha256` staying current, only on `scripts/homebrew-render.sh`-style
rendering happening inside the job itself.

## 3. Keeping the committed manifest buildable (automatic)

`scripts/release.sh` rewrites the `url:`/`sha256:` fields to the version
tag it is about to supersede — the tag being cut has no release assets yet,
but the previous one is already fully published, so the file committed at
every release always points at a real, downloadable, checksum-verified
tarball. This runs as part of every `scripts/release.sh <version>` call; no
extra step needed.

If the checksum download fails (e.g. no network), `release.sh` warns and
leaves the file untouched rather than failing the release — rerun
manually once online:

```bash
scripts/release.sh --check   # shows the current version, does not modify files
```

(there is no separate render script for this file — the update is inlined
in `release.sh` since, unlike the winget/Homebrew renders, it edits the
tracked file in place rather than producing a `dist/` artifact.)

## 4. Flathub — from-source build (human, not started)

Flathub builds on its own infrastructure, **without network access**: every
dependency (the Go module graph, the npm tree) must be vendored into the
manifest as `sources`, and the metainfo must additionally carry
`<screenshots>` (H.5) to pass `appstreamcli`'s stricter Flathub-side checks.
This is a from-source manifest, not the tarball-repackaging one in this
directory — see `docs/recherche/P16-distribution-desktop.md` §1 and §4 for
the concrete recipe (`flatpak-go-mod`, `flatpak-node-generator`, runtime
`org.gnome.Platform` 46+, submission as a PR to `flathub/flathub`). Tracked
in `tasks/BACKLOG.md`; multi-week effort, out of scope for fiche H.3.
