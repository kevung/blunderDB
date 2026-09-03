# Contributing to blunderDB

Thank you for considering a contribution. blunderDB is a one-person free-software project; a bug report with a match file that reproduces it, a translation fix, or a pull request are all welcome. This page says how to get a change in without friction. The rules that tests do not catch live in [CLAUDE.md](CLAUDE.md): it is written for the coding agent, but its *Invariants* section is the list every contributor is held to.

## Where to talk

- Questions, feedback, a position to discuss: the [Discord server](https://discord.gg/DA5PpzM9En).
- Longer threads and release announcements: [GitHub Discussions](https://github.com/kevung/blunderDB/discussions).
- Bugs and feature requests: [issues](https://github.com/kevung/blunderDB/issues). For a bug, attach the match or position file and the version (`blunderdb version`).
- A security problem: **not** an issue — see [SECURITY.md](SECURITY.md).

Everyone taking part is bound by the [code of conduct](CODE_OF_CONDUCT.md).

## Prerequisites

Go 1.25, Node.js 23, the [Wails v2](https://wails.io/) CLI 2.10 (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2`) and, on Linux, `webkit2gtk-4.1`. `make check` also needs `golangci-lint` (v2) and `govulncheck`; `make check-all` additionally needs Docker (the PostgreSQL contract suite) and a Playwright browser (`npx playwright install`). The documentation build needs Python with `doc/requirements.txt` in a virtualenv, plus GNU gettext. A [devcontainer](.devcontainer/) is available if you would rather not install any of this locally.

A plain clone also pulls the `gh-pages` branch's own history (built LaTeX
artefacts from past documentation deploys, over 700 Mio as of 2026-09 —
`.dvi`/`.doctrees` files nobody ever checks out); `--single-branch` fetches
only `main` and avoids it:

```bash
git clone --single-branch https://github.com/kevung/blunderDB.git && cd blunderDB
git config core.hooksPath .githooks   # versioned pre-commit hook, see Workflow below
cd frontend && npm install && cd ..
make dev         # hot-reload desktop app
make build       # build/bin/blunderDB
make check       # everything CI's push-time jobs enforce: vet, tests, golangci-lint,
                 # govulncheck, frontend lint/format/tests
make check-all   # the above PLUS Docker-backed PostgreSQL tests, Playwright e2e,
                 # and the release version-string check — full CI parity
```

`main.go` embeds `frontend/dist`, so a checkout that never built the frontend fails on the embed pattern; `make check` creates the stub it needs.

## Workflow

- One change per branch, in its own git worktree: `git worktree add ../blunderDB-<feature> -b feat/<feature>` (absolute path, see CLAUDE.md). Work and commit there, then merge or open the pull request.
- Commits: conventional prefix (`feat(...)`, `fix(...)`, `docs(...)`, `refactor(...)`), imperative subject, a body that says **why**. The maintainer writes them in French; English is welcome.
- Run `make check` before pushing (`make check-all` if you touched anything Docker or e2e tests would catch). CI runs `check-all`'s full scope. The versioned pre-commit hook (`.githooks/pre-commit`, enabled by the `git config core.hooksPath .githooks` above) runs `make check-fast` on every commit: gofmt, `go vet`, and the frontend's eslint/`prettier --check` — seconds, not minutes. It is not a substitute for `make check` before pushing.
- Fill in the [pull request template](.github/PULL_REQUEST_TEMPLATE.md): its checklist is the short form of the rules below.

## Documentation is part of the change

- **A user-visible change ships with its documentation, in the same branch.** A new command, shortcut, panel or filter lands with its entry in `doc/source/raccourcis.rst`, `manuel.rst` or `cmd_mode.rst`. Undocumented features have gone undiscovered for whole release cycles.
- **A modified `.rst` ships with its eight `.po` in the same commit** (en, de, el, es, fi, it, ja, ru). French is the source; the online docs deploy from `main` and fall back to French for every gap. Regenerate the catalogues with `scripts/doc-po-update.sh` (it keeps the gettext output path relative, so the `#:` references stay relative, and repairs the `python-format` flag msgmerge re-adds at every update), translate the new `msgstr`, and check with `scripts/doc-i18n-check.sh`: your files must report `0 untranslated, 0 fuzzy`. In Japanese, inline markup touching a CJK character needs an escaped space (`\ `) at that boundary.
- The in-app help (`frontend/src/i18n/help/*.js`) is translated by hand in the nine languages too; `helpVocabulary.sync.test.js` keeps its structure in step across them.

## Invariants and decisions

Some rules are violated silently by code that passes every test: positions are identified by their Zobrist hash, the retention predicate is written in three places that must stay identical, a schema change bumps `DatabaseVersion` and comes with a migration in three backends, the `serve` daemon performs no authentication by design, one equity scale leaves the engine, the network kernel never fuses a multiply-add. They are listed, with their reasons, in the *Invariants* section of [CLAUDE.md](CLAUDE.md) — read it before touching a subsystem, and read the [architecture decision records](docs/adr/README.md) that govern it.

A change that takes a new decision of that kind records it as a new ADR in `docs/adr/` (one file, numbered, never rewritten once accepted), and adds the invariant it creates to CLAUDE.md. Discuss it first, on Discord or in an issue: it is cheaper to argue about a page than about a pull request.

## Licence

blunderDB is released under the [MIT licence](LICENSE); by contributing, you agree that your contribution is too. Code, data or fonts taken from elsewhere must be listed in [THIRD_PARTY.md](THIRD_PARTY.md) with their licence — every copy of the binary carries that file.
