---
name: release-blunderdb
description: >-
  Drive a full blunderDB release: audit the French + English Sphinx docs for
  staleness, update the version/history changelog table (FR and EN) with the new
  user-facing features only, cut the version with scripts/release.sh, and publish
  a changelog-style GitHub release with gh. Use when the user asks to "release",
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

1. **Doc pass** — make sure the Sphinx docs are up to date in **both** French and
   English (new/changed features documented in both languages).
2. **Version history table** — add a new row mentioning **only the new
   user-facing features** (not refactors, CI fixes, internal plumbing), updated in
   **both** French and English.
3. **Cut the version** via `scripts/release.sh`.
4. **GitHub release notes** — write a changelog-style description (**in English**)
   and publish it with `gh`.

Do them in this order. Steps 1–2 are committed *before* the release commit so the
tag captures up-to-date docs.

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

If the change set touches the SQLite schema, remember `DatabaseVersion` in
`pkg/blunderdb/domain/` is **independent** of the release version — it is bumped
in its own commit alongside a migration, not by this skill. Do not conflate them.

**Confirm the version number with the user** (propose one, let them override)
before doing anything that writes files. Use AskUserQuestion if unsure.

---

## Phase 1 — Documentation audit (FR + EN)

**i18n model (important — verify it still holds, but this is how the repo works):**

- **French is the source language.** All human-written content lives in the
  `doc/source/*.rst` files **in French** (`conf.py`: `language = 'fr'`,
  `gettext_compact = False`).
- **English is a gettext translation**, not a parallel `.rst` tree. Each
  `source/<name>.rst` has a catalog at
  `doc/source/locale/en/LC_MESSAGES/<name>.po` whose `msgstr` entries hold the
  English text. (`locale/fr/LC_MESSAGES/*.po` also exists but is mostly empty
  because French is the source.)
- `doc/build.py` builds both languages — `sphinx-build -b html -D language=fr`
  then `-D language=en`, plus the LaTeX/PDF for each. GitHub Pages publishes from
  `gh-pages` on tag push, so docs must be correct *before* the tag.

So **"update the docs in French and English" concretely means:** edit the French
`.rst`, then update the matching English `.po` catalog's `msgstr` for any
new/changed strings. Do **not** look for English `.rst` files — there aren't any.

Steps:

1. Confirm the layout hasn't changed:

   ```bash
   grep -nE "language|locale|gettext" doc/source/conf.py
   ls doc/source/locale/en/LC_MESSAGES/
   ```

2. For each user-facing change since the last tag (the `feat`/`fix` set from
   Phase 0), make sure the feature is documented in the relevant French `.rst`
   (`manuel.rst`, `guide_utilisateur.rst`, `cmd_mode.rst`, `cli.rst`,
   `raccourcis.rst`, `stats.rst`, `faq.rst`, the `annexe_*.rst`, etc.). Add or fix
   the French prose first. Keep each file ≤500 lines (project rule); split if
   needed.

3. Regenerate the translation catalogs so new/changed French strings get fresh
   `msgid` entries, then translate them. The repo's `doc/README.txt` documents the
   workflow:

   ```bash
   cd doc
   make gettext                          # extract msgids → build/gettext/*.pot
   sphinx-intl update -l fr -l en        # update locale/{fr,en}/LC_MESSAGES/*.po
   ```

   Then fill in the **English `msgstr`** for every new/changed/`fuzzy` entry in the
   affected `doc/source/locale/en/LC_MESSAGES/<name>.po` files (the README points
   to lokalize, but editing the `.po` directly is fine). Translate accurately and
   in the house style of the surrounding entries; clear the `#, fuzzy` flag once a
   `msgstr` is done.

4. Build to confirm nothing is broken:

   ```bash
   cd doc && python build.py    # needs doc/requirements.txt (use the doc/.venv); LaTeX only for the PDF
   ```

   If only the LaTeX/PDF step fails for missing TeX deps, that's acceptable — note
   it to the user rather than blocking; CI on the tag does the authoritative build.
   The `doc/build/` output is git-ignored — don't commit it.

---

## Phase 2 — Version history / changelog table (FR + EN)

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
the quotes separate sub-items). There is **only one table, in French**; its
English rendering comes from the gettext catalog
`doc/source/locale/en/LC_MESSAGES/index.po` (Sphinx extracts each table cell as a
translatable string). So "FR + EN changelog" = **French row in `index.rst`** plus
**English `msgstr` for that cell in `index.po`**.

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

3. **Add the English translation.** After the French row exists, regenerate and
   translate the catalog so the table's English column updates:

   ```bash
   cd doc
   make gettext
   sphinx-intl update -l fr -l en
   ```

   Then set the English `msgstr` for the new changelog cell (and any other
   new/`fuzzy` index entries) in `doc/source/locale/en/LC_MESSAGES/index.po`,
   matching the English wording style of previous entries. This is the step that
   keeps the **English** version table in sync — don't skip it.

4. Commit the doc + changelog work **before** cutting the release, e.g.
   `docs(release): update FR/EN docs and changelog for <version>`. (release.sh
   warns on a dirty tree; committing first keeps the `Release <version>` commit
   clean.) If you let release.sh insert the French row in Phase 3, do the `.po`
   translation + commit either right after, or fold it into the release commit —
   just make sure the pushed tag contains both languages. Releases go out from
   `main`; only branch first if the user isn't on the intended branch.

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
internal commits out. Note that the FR/EN changelog in Phase 2 has a French
source row; the GitHub release notes are the **English** rendering of the same
feature list — reuse the English wording you wrote into `index.po`. Check
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

## Final report to the user

Summarize: version released, the FR/EN doc files touched, the changelog rows
added, the tag pushed, the GitHub release URL, and the CI run status/URL. Flag
anything skipped (e.g. PDF build that needed LaTeX) so the user can follow up.

## Guardrails

- Confirm the version number, and confirm again before the tag **push** and the
  GitHub release publish — both are outward-facing.
- New-features-only in the history table and release notes; exclude refactors,
  CI/build/test/chore commits.
- GitHub release notes are always written in **English**.
- Keep French and English in lockstep — every feature documented and listed in
  both.
- Don't bump `DatabaseVersion` here (it's schema-coupled and independent).
- Don't commit changes to the sample `*.db` files in the repo root.
- Respect the ≤500-line doc rule.
