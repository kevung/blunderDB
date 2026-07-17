---
name: release-blunderdb
description: >-
  Drive a full blunderDB release: audit the Sphinx docs for staleness across all
  9 languages (French source + 8 gettext translations), update the version/history
  changelog table (FR source + every translation) with the new user-facing
  features only, cut the version with scripts/release.sh, and publish a
  changelog-style GitHub release with gh. Use when the user asks to "release",
  "faire une release", "publier une version", "cut a release", or "release
  blunderDB <version>".
---

# Release blunderDB

This skill performs an end-to-end release of blunderDB. A release is **outward
facing and hard to reverse** (it pushes a tag that triggers a CI matrix build and
publishes binaries + docs). So: gather everything, show the user the plan and the
diffs, and get explicit confirmation **before** pushing the tag or creating the
GitHub release. Never push or publish silently.

Work from the repo root (`/home/unger/src/blunderDB`). All shell snippets assume
that cwd.

## What a release consists of

1. **Doc pass** — make sure the Sphinx docs are up to date in **all 9 languages**:
   the French source plus the 8 gettext translations (en, de, el, es, fi, it, ja,
   ru). New/changed features documented in the French source and translated in
   every catalog.
2. **Version history table** — add a new row mentioning **only the new
   user-facing features** (not refactors, CI fixes, internal plumbing), updated in
   the **French source and every translation**.
3. **Linux packaging metadata** — add the new version to the AppStream
   `<releases>` list (shown in software centers for the `.deb`/`.rpm`/Flatpak).
   Applies to **every** release, patches included.
4. **Cut the version** via `scripts/release.sh`.
5. **GitHub release notes** — write a changelog-style description (**in English**)
   and publish it with `gh`.

Then, **after** the release is published, the AUR package republishes
automatically (Phase 5) and the Flatpak manifest can be bumped (follow-up).

Do them in this order. Steps 1–3 are committed *before* the release commit so the
tag captures up-to-date docs and packaging metadata.

---

## Phase 0 — Orient and decide the version

Run these first and read the output:

```bash
./scripts/release.sh --check                 # current conf.py / metaStore.js / latest tag
git tag --sort=-creatordate | head -5        # recent tags
git log "$(git describe --tags --abbrev=0)"..HEAD --oneline   # commits since last release
git status                                    # working tree must be clean before releasing
```

From `git log <lastTag>..HEAD`, build the list of **user-facing changes**. Use the
conventional-commit prefixes as a guide:

- `feat(...)` → a new feature → belongs in the changelog and probably needs docs.
- `fix(...)` → a user-visible bug fix → mention if it matters to users.
- `docs`, `refactor`, `chore`, `ci`, `test`, `build` → **exclude** from the
  feature changelog (they are not new features). The user was explicit: the
  history table lists **new functionalities only**.

Decide the new version number with semver against the *current* version:

- New features, backward compatible → bump **minor** (X.**Y+1**.0).
- Only bug fixes → bump **patch** (X.Y.**Z+1**).
- Breaking change → bump **major**.

**Patch releases skip the changelog table.** If the bump is a patch (X.Y.Z, Z>0),
**skip Phase 2 entirely** — the history table in the docs lists only X.Y.0
(minor/major) releases, never patches. Do Phase 1 (if any doc prose needs fixing),
then go straight to Phase 3 with `scripts/release.sh <ver>` (no `--changelog`), and
write the English GitHub release notes in Phase 4.

If the change set touches the SQLite schema, remember `DatabaseVersion` in
`pkg/blunderdb/domain/` is **independent** of the release version — it is bumped
in its own commit alongside a migration, not by this skill. Do not conflate them.

**Confirm the version number with the user** (propose one, let them override)
before doing anything that writes files. Use AskUserQuestion if unsure.

---

## Phase 1 — Documentation audit (all 9 languages)

**i18n model (important — verify it still holds, but this is how the repo works):**

- **French is the source language.** All human-written content lives in the
  `doc/source/*.rst` files **in French** (`conf.py`: `language = 'fr'`,
  `gettext_compact = False`).
- **The other 8 languages are gettext translations**, not parallel `.rst` trees.
  The full set is defined once in `conf.py` as `LANGUAGES` and mirrored in
  `doc/build.py` as `LANG`: **fr** (source) + **en, de, el, es, fi, it, ja, ru**
  (translations). Each `source/<name>.rst` has a catalog at
  `doc/source/locale/<code>/LC_MESSAGES/<name>.po` whose `msgstr` entries hold the
  translated text. (`locale/fr/LC_MESSAGES/*.po` also exists but is mostly empty
  because French is the source.)
- `doc/build.py` loops `LANG`, building HTML (`sphinx-build -b html -D
  language=<code>`) and the LaTeX/PDF for each. It exports `BLUNDERDB_DOC_LANG`
  so `conf.py` can pick per-language PDF fonts/engine (**ja** uses upLaTeX +
  dvipdfmx; **el/ru** pin GNU FreeFont under XeLaTeX; the Latin languages use the
  default XeLaTeX setup). GitHub Pages publishes from `gh-pages` on tag push, so
  docs must be correct *before* the tag. The in-page language switcher
  (`_templates/versions.html`) is driven by `conf.py`'s `html_context['languages']`
  — every code in `LANGUAGES` gets a link, and each `../<code>/index.html` target
  must exist (i.e. that language must be built) or the link 404s.

So **"update the docs in every language" concretely means:** edit the French
`.rst`, then update the matching `msgstr` in **all 8** translation catalogs
(`locale/<code>/LC_MESSAGES/<name>.po`) for any new/changed strings. Do **not**
look for non-French `.rst` files — there aren't any. The PDF download link on the
download page is per-language: each catalog's `msgstr` for that sentence should
reference its own `|latest_<code>_pdf|` substitution (all 9 are defined in
`conf.py`'s `rst_prolog`).

> **Translation quality caveat.** The non-French translations were seeded by LLM
> and the maintainer cannot proofread ja/fi/ru/el. Treat empty/`fuzzy` `msgstr`
> as the source of truth for what still needs human attention, and surface the
> per-language staleness report (step 3 below) so the user can decide whether to
> ship or defer a given language.

Steps:

1. Confirm the layout hasn't changed (the language set lives in two places that
   must agree):

   ```bash
   grep -nE "LANGUAGES|language|locale|gettext" doc/source/conf.py
   grep -n "LANG =" doc/build.py
   ls doc/source/locale/                 # one dir per translated language
   ```

2. For each user-facing change since the last tag (the `feat`/`fix` set from
   Phase 0), make sure the feature is documented in the relevant French `.rst`
   (`manuel.rst`, `guide_utilisateur.rst`, `cmd_mode.rst`, `cli.rst`,
   `raccourcis.rst`, `stats.rst`, `faq.rst`, the `annexe_*.rst`, etc.). Add or fix
   the French prose first. Keep each file ≤500 lines (project rule); split if
   needed.

   **Also audit the developer docs** — they have no other forcing function and
   have drifted for whole release cycles before. Re-verify each factual claim
   in `CLAUDE.md` against the code, minimally:

   ```bash
   grep -n "DatabaseVersion =" pkg/blunderdb/domain/domain.go   # vs CLAUDE.md
   grep -nE "go-version|node-version|version: v2" .github/workflows/build.yml
   sed -n '20,45p' main.go                                      # mode dispatch list
   ```

   Check that `CLAUDE.md`'s toolchain versions, `DatabaseVersion`, mode list,
   and any named files/symbols still exist (`grep` a few). Same for
   `CLI_USAGE.md` vs `internal/cli/` (every subcommand and flag), and the
   package doc of `pkg/blunderdb/storage/storage.go` (it is the architecture
   reference). Fix drift *now*, in this release's doc commit.

3. Regenerate **all** translation catalogs so new/changed French strings get
   fresh `msgid` entries, then translate them. The repo's `doc/README.txt`
   documents the workflow:

   ```bash
   cd doc
   make gettext                                              # extract msgids → build/gettext/*.pot
   sphinx-intl update -l fr -l en -l de -l el -l es -l fi -l it -l ja -l ru
   ```

   Then **check translation freshness per language** — this is the "verify the
   translations are up to date" step. Empty or `#, fuzzy` `msgstr` are the strings
   that still need work after the French source changed:

   ```bash
   # NB: msgattrib takes ONE file at a time — loop per .po, don't glob.
   for lang in en de el es fi it ja ru; do
     empty=0
     for f in doc/source/locale/$lang/LC_MESSAGES/*.po; do
       c=$(msgattrib --untranslated "$f" 2>/dev/null | grep -c '^msgid "')
       empty=$((empty + c - 1))   # -1 drops the header msgid "" msgattrib always emits
     done
     fuzzy=$(grep -rl '#, fuzzy' doc/source/locale/$lang/LC_MESSAGES/ 2>/dev/null | wc -l)
     echo "$lang: ~$empty untranslated entries, $fuzzy file(s) with fuzzy strings"
   done
   ```

   (`msgattrib` ships with gettext; if absent, fall back to
   `grep -c '^msgstr ""$' …` per file.) Fill in the `msgstr` for every new/changed/`fuzzy`
   entry in each affected `doc/source/locale/<code>/LC_MESSAGES/<name>.po`,
   translating accurately in the house style of the surrounding entries; clear the
   `#, fuzzy` flag once a `msgstr` is done. **Report the per-language counts to the
   user** — if some language can't be fully translated this cycle, that's a
   conscious call (it falls back to French for the untranslated strings), not a
   silent gap.

4. Build to confirm nothing is broken — this builds **all 9 languages** (HTML +
   PDF):

   ```bash
   cd doc && python build.py    # needs doc/requirements.txt (use the doc/.venv); LaTeX only for the PDF
   ```

   The Japanese PDF needs upLaTeX + `texlive-lang-cjk`/`texlive-lang-japanese`;
   el/ru PDFs need `fonts-freefont-otf` (CI installs all of these). If only the
   LaTeX/PDF step fails locally for missing TeX deps, that's acceptable — note it
   to the user rather than blocking; CI on the tag does the authoritative build.
   Known minor issue: the `✗` (U+2717) glyph in the stats-parity table is absent
   from the Japanese CMap and renders as a warning — cosmetic, not fatal. The
   `doc/build/` output is git-ignored — don't commit it.

---

## Phase 2 — Version history / changelog table (FR source + all translations)

> **MAJOR/MINOR RELEASES ONLY — skip this entire phase for patch releases.**
> The history table lists only **X.Y.0** (minor/major) releases. A patch/fix
> release (X.Y.Z with Z>0, e.g. `0.26.1`) gets **no row** — no FR row, no `.po`
> retranslation, no `make gettext`. The user considers patches too minor for the
> table; it is a high-level feature history, not an exhaustive log. For a patch
> release, jump straight from Phase 1 to Phase 3, run `scripts/release.sh <ver>`
> **without** `--changelog`, and still write the English GitHub release notes in
> Phase 4. Only do the steps below when cutting an X.Y.0.

The version-history table is the **"Historique des versions"** `csv-table` in
`doc/source/index.rst` (French source). Its rows look exactly like:

```rst
.. csv-table::
   :header: "Version", "Date", "Cause et/ou nature des évolutions"
   :widths: 5, 7, 20
   :align: center
   :class: align-center-table

   0.1.0, 31/12/2024, "Création version beta."
   ...
   0.19.0, 07/05/2026, "Ajout du panneau Stats : ... Voir :ref:`stats`."
```

Each row is `   X.Y.Z, dd/mm/yyyy, "<French description>"` — 3-space indented, the
description double-quoted and allowed to span multiple lines (blank lines inside
the quotes separate sub-items). There is **only one table, in French**; each
language's rendering comes from its gettext catalog
`doc/source/locale/<code>/LC_MESSAGES/index.po` (Sphinx extracts each table cell
as a translatable string). So the changelog work = **French row in `index.rst`**
plus the `msgstr` for that cell in **every** `locale/<code>/LC_MESSAGES/index.po`
(en, de, el, es, fi, it, ja, ru).

Steps:

1. **Write the new row's description** — a short, user-facing summary listing
   **only the new features** from Phase 0 (exclude refactors/CI/test/chore),
   phrased for end users, in the same tone as recent rows. Reference doc sections
   with `:ref:` where the existing rows do. Write it in **French** (it's the
   source cell).

2. **Add the French row.** Either:
   - let `scripts/release.sh <version> --changelog "<French text>"` insert it
     automatically in Phase 3 (it appends `   <version>, <dd/mm/yyyy>, "<text>"`
     before the `Sommaire`/toctree section — note this produces a **single-line**
     cell), **or**
   - add it by hand now if you want a multi-line cell matching the richer recent
     rows. Match indentation, date format `dd/mm/yyyy`, and quoting exactly.

   Show the user the resulting row before committing.

3. **Add every translation.** After the French row exists, regenerate and
   translate all catalogs so the table's other-language columns update:

   ```bash
   cd doc
   make gettext
   sphinx-intl update -l fr -l en -l de -l el -l es -l fi -l it -l ja -l ru
   ```

   Then set the `msgstr` for the new changelog cell (and any other new/`fuzzy`
   index entries) in **each** `doc/source/locale/<code>/LC_MESSAGES/index.po`,
   matching the wording style of previous entries in that language. This is the
   step that keeps every version table in sync — don't skip it. Run the
   per-language freshness scan from Phase 1 step 3 against `index.po` to confirm no
   catalog was left with an untranslated changelog cell.

4. Commit the doc + changelog work **before** cutting the release, e.g.
   `docs(release): update docs and changelog (9 languages) for <version>`.
   (release.sh warns on a dirty tree; committing first keeps the `Release
   <version>` commit clean.) If you let release.sh insert the French row in Phase
   3, do the `.po` translations + commit either right after, or fold it into the
   release commit — just make sure the pushed tag contains every language.
   Releases go out from `main`; only branch first if the user isn't on the
   intended branch.

---

## Phase 2b — Linux packaging metadata (every release, incl. patches)

The Linux packages (`.deb`/`.rpm` built by `nfpm` in CI, the AUR package, and the
Flatpak) ship an **AppStream metainfo** file,
`build/linux/io.github.kevung.blunderDB.metainfo.xml`. Its `<releases>` list is
what GNOME Software / KDE Discover show as the version history. Add the new
version at the **top** of the list (newest first), dated with the release day:

```xml
  <releases>
    <release version="<version>" date="<YYYY-MM-DD>" />
    ...existing entries...
  </releases>
```

Unlike the Phase 2 changelog table, this is **not** feature-gated — add an entry
for **every** release, patches included (it's a version list, not a feature log).
A bare `version`/`date` entry is enough; validate if `appstreamcli` is available:

```bash
appstreamcli validate --no-net build/linux/io.github.kevung.blunderDB.metainfo.xml
```

Commit this with the doc/changelog work (Phase 1–2), before cutting the release,
so the tag captures it. This is the only per-release edit the packaging needs —
the `.deb`/`.rpm` names, the AUR PKGBUILD and the tarballs all derive their
version from the tag automatically in CI.

---

## Phase 3 — Cut the version with release.sh

`scripts/release.sh` updates the version string in three places
(`doc/source/conf.py`, `frontend/src/stores/metaStore.js`, and — if
`--changelog` is given — `doc/source/index.rst`), then commits `Release <version>`
and creates the `<version>` tag.

Recommended invocation (do **not** auto-push yet so the user can review):

```bash
# If you already inserted both changelog rows manually in Phase 2:
./scripts/release.sh <version>

# Or let it insert the French changelog row for you:
./scripts/release.sh <version> --changelog "<concise FR feature summary>"
```

The script is interactive (it prints the diff and asks to commit/tag, and prompts
on a dirty tree). Run it in the foreground and let those prompts surface to the
user. **Do not pass `--push` in this step.**

After it creates the commit + tag, show the user:

```bash
git show --stat HEAD          # the Release <version> commit
git tag -l <version>          # confirm the tag exists
```

**Get explicit confirmation, then push** (this is the irreversible, CI-triggering
step):

```bash
git push origin main && git push origin <version>
```

Pushing the tag triggers the CI matrix build (binaries + PDFs) and the gh-pages
docs deploy.

---

## Phase 4 — GitHub release notes (changelog style, via gh)

The tag push triggers the CI matrix build, which **publishes the binaries + PDFs
as a GitHub release** (per CLAUDE.md). So the release usually already exists —
your job is normally to write its **notes**, not create it. Repo:
`github.com/kevung/blunderDB`.

```bash
gh run list --limit 5                # find the build run kicked off by the tag
gh release view <version>            # exists (CI made it)? -> edit notes; missing -> create
```

Write the notes **in English** (project convention — always English, regardless
of what older releases used), in **changelog style** — grouped, terse,
user-facing. A good shape:

```markdown
## blunderDB <version>

### ✨ New features
- <feature 1, user-facing wording>
- <feature 2>

### 🐛 Fixes
- <user-visible fix, if any>

**Full changelog**: https://github.com/<owner>/<repo>/compare/<prevTag>...<version>
```

Derive the bullets from the same `feat(...)`/`fix(...)` set as Phase 0 — keep
internal commits out. Note that the Phase 2 changelog has a French source row;
the GitHub release notes are the **English** rendering of the same feature list —
reuse the English wording you wrote into `locale/en/LC_MESSAGES/index.po`. Check
`gh release view <prevTag>` for tone/formatting, but the **language is always
English** even if older notes were French.

Publish:

```bash
# If CI already created the release, update its notes without clobbering assets:
gh release edit <version> --notes-file /tmp/relnotes.md

# If no release exists yet (e.g. you're not relying on CI to create it):
gh release create <version> --title "blunderDB <version>" --notes-file /tmp/relnotes.md
```

Prefer `--notes-file` over `--notes` for multi-line markdown. If CI is still
building when you check, wait for it (`gh run watch` / `gh run list`) before
editing the release so you don't race the asset upload, or just edit notes (which
doesn't touch assets).

---

## Phase 5 — Downstream packages (after the release is published)

Once the GitHub release exists with its assets (CI uploads the raw binaries,
`.tar.gz`, `.deb`, `.rpm` and `.sha256`), two downstream packages follow.

**AUR (`blunderdb-bin`) — automatic.** `.github/workflows/aur.yml` triggers on
`release: published`, waits for the 4.1 tarball, and pushes an updated PKGBUILD to
the AUR. It only runs if the `AUR_SSH_PRIVATE_KEY` secret is set (otherwise it
no-ops — see `packaging/aur/README.md` for the one-time account/SSH setup). Verify
after the release:

```bash
gh run list --workflow=aur.yml --limit 3      # did it run / succeed?
# then check https://aur.archlinux.org/packages/blunderdb-bin shows the new version
```

If the secret isn't configured (or the run failed), publish manually from an Arch
box: `scripts/aur-publish.sh <version> --push`.

**Flatpak — manual follow-up.** `packaging/flatpak/io.github.kevung.blunderDB.yml`
pins the tarball `url` + `sha256`. It is **not** wired into CI (Flathub builds on
its own infra). If/when maintaining a Flatpak, bump those two fields to the new
release (the `flathub-external-data-checker` bot can automate this once on
Flathub). Skip unless the user is actively shipping the Flatpak.

---

## Final report to the user

Summarize: version released, the doc files touched (French source + which
translation catalogs), the per-language translation-freshness counts, the
changelog rows added, the AppStream `<releases>` entry, the tag pushed, the
GitHub release URL, the CI run status/URL, and the AUR publish status (workflow
run + package page, or "skipped — secret not set"). Flag anything skipped (e.g. a
PDF build that needed LaTeX, or a language left partially untranslated) so the
user can follow up.

## Guardrails

- Confirm the version number, and confirm again before the tag **push** and the
  GitHub release publish — both are outward-facing.
- New-features-only in the history table and release notes; exclude refactors,
  CI/build/test/chore commits.
- The history table is **major/minor (X.Y.0) only** — patch releases (X.Y.Z, Z>0)
  get no changelog row and skip Phase 2 entirely. But the AppStream `<releases>`
  entry (Phase 2b) is added for **every** release, patches included.
- GitHub release notes are always written in **English**.
- Keep all 9 languages in lockstep — every feature documented in the French
  source and translated in every catalog (en, de, el, es, fi, it, ja, ru), or the
  per-language gap explicitly surfaced to the user.
- Don't bump `DatabaseVersion` here (it's schema-coupled and independent).
- Don't commit changes to the sample `*.db` files in the repo root.
- Respect the ≤500-line doc rule.
