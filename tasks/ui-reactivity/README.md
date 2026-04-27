# UI reactivity — Task sheets

Ventilation du plan `/home/unger/.claude/plans/la-manipulation-des-differents-splendid-finch.md`. Chaque fiche est une unité d'exécution autonome, ≤500 lignes, avec cases à cocher reliées à des ancrages `file:line`.

## Objectif

Post-migration Svelte 5, l'UI blunderDB présente des symptômes de non-réactivité : transitions d'onglets parfois bloquées (en particulier toute transition impliquant l'onglet **Stats**), barre d'état EPC potentiellement figée. Ce chantier :

1. Met en place un **outillage de test automatisé** pour ne plus dépendre du clic manuel (`@testing-library/svelte`, Playwright, `logger.perf`).
2. **Diagnostique** précisément les incohérences de patterns Svelte 4/5 introduites sur `stat_panel` (audit).
3. **Corrige** les zones identifiées une par une, chaque fix accompagné d'un test qui verrouille le comportement.

Portée : TabbedPanel, MatchPanel, StatsPanel, EPCPanel, StatusBar, App.svelte, stores associés, `positionService.js`. Hors-scope : Board, command-line, backend Go.

## Scénarios confirmés (Fiche 00)

| Scénario | Statut sur `stat_panel` |
|---|---|
| S1 — EPC stable au retour d'onglet | ✅ (position inchangée entre visites) |
| S1 étendu — EPC reflète la nouvelle position | ⏳ à tester |
| S2 — Transitions **incluant Stats** | ❌ **cassées** (Match↔Stats, EPC↔Stats, Stats↔Anki) |
| S3 — EPC live pendant édition plateau | ✅ |

Cf. `doc/archive/ui-reactivity-scenarios.md`.

## Ordre d'exécution

Dépendances top-down ; les blocs 5.* peuvent s'exécuter en parallèle une fois 4 terminé.

| # | Fiche | But | Dépend de |
|---|---|---|---|
| 00 | [00-scenarios-and-branching.md](00-scenarios-and-branching.md) | Repro déterministe + branche `ui-reactivity` | — |
| 01 | [01-testing-library-setup.md](01-testing-library-setup.md) | `@testing-library/svelte` + Vitest + canari StatusBar | 00 |
| 02 | [02-playwright-setup.md](02-playwright-setup.md) | Playwright + harness Wails mock + 2 specs rouges | 00, 01 |
| 03 | [03-perf-instrumentation.md](03-perf-instrumentation.md) | `logger.perf`, wrappers effets, `trackRuneDeps` | 01 |
| 04 | [04-audit-svelte-patterns.md](04-audit-svelte-patterns.md) | Inventaire exhaustif des `.subscribe()` / closures stales | 00 |
| 05.a | [05a-fix-app-svelte.md](05a-fix-app-svelte.md) | App.svelte : handler tabs + propagation EPC | 02, 03, 04 |
| 05.b | [05b-fix-matchpanel.md](05b-fix-matchpanel.md) | MatchPanel : closures stales | 01, 04 |
| 05.c | [05c-fix-statspanel.md](05c-fix-statspanel.md) | StatsPanel : onMount-subscribe → `$effect` | 01, 04 |
| 05.d | [05d-fix-statusbar.md](05d-fix-statusbar.md) | StatusBar : `.subscribe()` → `$effect` | 01, 04 |
| 05.e | [05e-tabbedpanel-doc.md](05e-tabbedpanel-doc.md) | TabbedPanel : règle keep-alive + montage paresseux | 04 |
| 05.f | [05f-enterepcmode-order.md](05f-enterepcmode-order.md) | `positionService.enterEPCMode` : ordre des `set` | 05.a |
| 06 | [06-benchmark-ci.md](06-benchmark-ci.md) | Benchmarks avant/après + CI + règle dans CLAUDE.md | 05.* |

## Working rules

- **Chaque fiche ≤ 500 lignes.** Si une fiche dépasse ~450 lignes, la splitter.
- **Une fiche = un commit atomique** (ou une PR). Ne pas mélanger deux fiches.
- **Cycle test-first sur les fiches 05.*** : test rouge qui reproduit le bug → fix → test vert → commit.
- **Re-tester `npm test` (et `npm run test:e2e` dès Fiche 2)** à la fin de chaque fiche. Aucune fiche « done » avec suite rouge.
- **Mettre à jour le statut** dans chaque fiche (cases à cocher) et dans ce README (section *Current status*) au fur et à mesure.
- **Branche de travail** : `ui-reactivity`, créée depuis `stat_panel` (pas depuis `main`, car l'onglet Stats source des bugs n'existe pas sur `main`).

## Rollback strategy

- 00 (doc + branche) : trivialement réversible.
- 01–03 (outillage de test et profiling, additif) : revert = `git revert` + `npm install`.
- 04 (doc audit, additif) : revert = `git revert`.
- 05.a–05.f (fixes frontend) : chacun est un commit isolé ; revert granulaire possible. Aucun changement de schéma ou d'API backend.
- 06 (CI + doc) : revert = `git revert`.

Aucune modification du backend Go, aucune migration de schéma SQLite.

## Règle d'invariant (à rappeler dans les fixes)

> Dans ce projet (Svelte 5), tout accès à un store dans un callback ou un effet doit passer par `$effect` lisant `$storeName`, **pas** par `.subscribe()`. Les `$effect` dont le corps lit des stores **uniquement** via `get(store)` ou via une closure sont invisibles au traceur de dépendances — c'est la cause principale des bugs documentés ici.

Cf. Fiche 06 pour l'intégration de la règle dans `CLAUDE.md`.

## Current status

- [x] 00 — scénarios + branche
- [x] 01 — testing-library
- [x] 02 — Playwright
- [x] 03 — perf instrumentation
- [x] 04 — audit patterns
- [x] 05.a — fix App.svelte (subscribe → $effect, cas manquants stats/tournaments/collections, tabHandler.js, 15 tests)
- [x] 05.b — fix MatchPanel (subscribe → $effect, closures stales, 6 tests)
- [ ] 05.c — fix StatsPanel
- [ ] 05.d — fix StatusBar
- [ ] 05.e — doc TabbedPanel
- [ ] 05.f — fix enterEPCMode
- [ ] 06 — benchmark + CI

## Références

- Plan maître : `/home/unger/.claude/plans/la-manipulation-des-differents-splendid-finch.md`
- Scénarios : `doc/archive/ui-reactivity-scenarios.md`
- Audit (produit en Fiche 4) : `doc/archive/ui-reactivity-audit.md`
- Benchmark (produit en Fiche 6) : `doc/archive/ui-reactivity-benchmark.md`
