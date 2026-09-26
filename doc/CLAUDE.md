# doc — rules

Sphinx, nine languages: French is the source, `source/locale/` holds the gettext catalogues
(en, de, el, es, fi, it, ja, ru). Build: `source .venv/bin/activate && cd doc && python build.py`.

- **A modified `.rst` ships with its eight `.po` in the same commit** — the online docs deploy
  from `main` and fall back to French. `scripts/doc-po-update.sh`, fill the empty `msgstr` with
  `scripts/po-fill.py <translations.json>`, then `scripts/doc-i18n-check.sh` must end on
  "all translations complete". Never run sphinx-intl by hand (absolute paths rewrite every `#:`)
  and never re-wrap a `.po` (msgcat, Babel, polib) — it buries the real diff.
- **Never read a `.po` or `manuel.rst` whole**: `grep -n` the msgid, then read the range.
  Translating is a job for a Sonnet sub-agent that returns "N entries filled".
- **Present tense, published version only**: no "coming soon", no command that does not work
  today. The future lives in GitHub milestones and Discussions, linked from *À propos* only.
- **The guide shows a task, the manual describes a screen**: a screen description in
  `guide_utilisateur.rst` becomes a `:ref:` to the manual's panel section.
- **A new page is proposed with its price**: one French line ≈ twenty catalogue lines across
  the eight languages. Prefer a sentence on an existing page or a `:ref:`.
- **The in-app help is generated** from `manuel.rst`, `raccourcis.rst`, `cmd_mode.rst` and
  `frontend/src/i18n/help/prose/`: run `make help` after changing them; `go test ./cmd/help-gen`
  fails while a bundle is stale (ADR-0034).
- Japanese: RST inline markup touching CJK needs an escaped space (`\ `) in the `msgstr`.
- Never commit a `.mo`: a stale one shadows the `.po` in CI.
- Documents under `tasks/` and design notes stay ≤ 500 lines; split into an index + topics.
