# The in-app help is generated from the documentation

Status: accepted.

## Context

The in-app help was hand-written HTML in nine languages, 81 % of it a second copy of the
documentation's shortcut, command and manual pages. Nothing kept the two in step, and the copy
drifted both ways: whole features and renamed panels missing from the help, sections and filter
tokens the help never had. The documentation has gettext catalogues and a completeness gate; the
help had nine files edited by hand, no completeness check, and was excluded from eslint while
being injected with `{@html}`.

## Decision

1. **The help bundles are a build artefact of the documentation.** `cmd/help-gen` (Go, standard
   library only) renders `frontend/src/i18n/help/<lang>.js` for the nine documentation
   languages; `make help` runs it. Each file opens with a `// GENERATED FILE` banner and is never
   edited by hand.
2. **Sources per tab:**

   | tab | source |
   |---|---|
   | `shortcuts` | `doc/source/raccourcis.rst` |
   | `commands` | `doc/source/cmd_mode.rst` |
   | `manual` | `doc/source/manuel.rst`, whole — no section manifest |
   | `about` | `frontend/src/i18n/help/prose/<lang>.html`, hand-written, permanently |

   A tab is generated when the documentation states it; About is application metadata with no
   `.rst` and is never generated.
3. **Sphinx and Python are not in the loop.** The generator reads the `.rst` and the `.po`
   catalogues directly (Sphinx extracts each `csv-table` cell as its own `msgid`). It knows
   sections, paragraphs, bullet lists, `csv-table`, the `note`/`tip`/`warning`/`important`/
   `caution` admonitions, indented bodies, the `math` block, and inline literal, interpreted,
   emphasis, strong, `:ref:` and backslash escapes (including the Japanese `\ `). Unknown
   directives — `figure::` included — are skipped whole.
4. **A missing translation is an error, never a French fallback.** A missing `msgstr` aborts the
   run.
5. **The bundles are committed, not built at `npm run build`**, so no Python toolchain sits on the
   path of `wails dev` or the four CI build runners.
6. **Escaping is the generator's job.** Every source string is HTML-escaped before the
   inline-markup pass, which can emit only the fixed tag vocabulary in `render.go`.

## Consequences

- Changing `manuel.rst`, `raccourcis.rst`, `cmd_mode.rst` or a prose fragment requires
  `make help`; the help inherits the documentation's translation tooling
  (`scripts/doc-po-update.sh`, `scripts/doc-i18n-check.sh`).
- The bundles weigh ~1.7 MB over nine languages; only English is loaded at start-up, the others
  when the modal opens. That is the price of the manual being present offline.
- The reader checks the sources: an unresolvable `:ref:` or misplaced label shows up in the
  generated diff.
- A construct the reader does not know disappears silently; a *changed* string does not (the
  `msgid` lookup fails). Adding a construct is a reviewable `.rst` change visible in the next
  `make help` diff.
- Rejected: running Sphinx at frontend build (Python becomes a hard dependency of every build,
  and docutils HTML needs its own reduction pass). Rejected: a table of contents with links to
  the site (useless offline, at a club or a tournament table). Rejected: hand-written help plus a
  section-title test (catches missing sections; the drift was stale sentences). Rejected: a
  section manifest for the manual tab (a second list to maintain, which is what drifted).
  Rejected: generating the `.rst` from the help (gettext works on the `.rst`).

## Guard

`TestHelpBundlesAreCurrent` in `cmd/help-gen/help_gen_test.go`;
`frontend/src/__tests__/help.safety.test.js` (no script, event handler, `javascript:`/`data:` URL,
unhardened link or stray placeholder in the shipped bundles); `helpVocabulary.sync.test.js`
(commands table against `commandVocabulary.js`, nine languages).
