# Stats Panel — Task Sheets

Hierarchical breakdown of the Stats Panel architecture plan (`/home/unger/.claude/plans/je-souhaite-concevoir-et-humming-cherny.md`). Each sheet is an independent execution unit, sized ≤500 lines.

## Objectif

Ajouter un panneau *Stats* permettant à l'utilisateur de (1) situer son niveau via PR/MWC, (2) suivre sa progression dans le temps, (3) comprendre où il fait ses erreurs. Interactif et drill-down vers positions / matchs / tournois.

## Ordre d'exécution

Dépendances top-down ; une phase ne peut démarrer qu'après celles dont elle dépend.

| # | Sheet | Purpose | Depends on |
|---|---|---|---|
| 01 | [01-backend-stats.md](01-backend-stats.md) | `db_stats.go` : `StatsFilter`/`StatsResult`, `ComputeStats`, règle d'agrégation pondérée, helper `buildStatsWhereClause` | — |
| 02 | [02-mwc-conversion.md](02-mwc-conversion.md) | `ConvertEMGLossToMWCLoss` dans `db_met.go` + intégration dans `ComputeStats` | 01 |
| 03 | [03-drilldown-backend.md](03-drilldown-backend.md) | `SelectionSpec`, `GetPositionIDsByStatsSelection/Tournament/Match` | 02 |
| 04 | [04-wails-store.md](04-wails-store.md) | Binding Wails, `statsStore.js`, service `positionLoader.js` | 03 |
| 05 | [05-panel-shell-charts.md](05-panel-shell-charts.md) | `StatsPanel.svelte` squelette, onglets, Chart.js, toggle PR/MWC | 04 |
| 06 | [06-dashboard-tab.md](06-dashboard-tab.md) | Onglet Dashboard | 05 |
| 07 | [07-progression-tab.md](07-progression-tab.md) | Onglet Progression | 05 |
| 08 | [08-errors-tab.md](08-errors-tab.md) | Onglet Répartition d'erreurs | 05 |
| 09 | [09-filter-bar.md](09-filter-bar.md) | Barre de filtre commune | 05 |
| 10 | [10-cli-parity.md](10-cli-parity.md) | Étendre `cli.go list --type stats` | 02 |
| 11 | [11-docs-tests.md](11-docs-tests.md) | Doc FR/EN, test E2E, changelog | 06–10 |

**Ordre recommandé** : 01 → 02 → 03 → 04 → 05 → (06 ∥ 07 ∥ 08 ∥ 09) → 10 → 11.

## Working rules

- **Chaque fiche ≤ 500 lignes.** Si une fiche dépasse ~450 lignes, la splitter.
- **Une fiche = une PR.** Ne pas mélanger deux fiches dans un même commit.
- **Règle d'agrégation (invariant métier)** : PR et MWC agrégés par `sum/sum` (ou `sum` pour MWC cost), **jamais** par moyenne des sous-ensembles. Critique pour les PR par tournoi. Test unitaire anti-moyenne-des-moyennes obligatoire en 01.
- **Principes UX (plan §)** — re-citer en critère d'acceptation pour les fiches 06/07/08 : une info clé par vue, hover pour le détail, densité faible, palette restreinte, pas de 3D, pas de pie/radar, pas de double axe Y.
- **Re-tester `go test ./... && go test ./tests/...`** à la fin de chaque fiche — aucune fiche « done » avec une suite rouge.
- **Mettre à jour le statut** dans chaque fiche au fur et à mesure.

## Rollback strategy

- 01–04 (backend + bindings) : additif, aucune migration de schéma. Revert = `git revert`.
- 05–09 (frontend) : additif, ajoute dépendance `chart.js` à `package.json`. Revert = `git revert` + `npm install`.
- 10 (CLI) : additif, sous-commande/flag non intrusif. Revert = `git revert`.
- 11 (docs + tests) : trivialement réversible.

**Aucune migration de schéma dans ce chantier** — MWC calculé on-the-fly, pas de colonne ajoutée.

## Current status

- [x] 01 — backend stats
- [x] 02 — MWC conversion
- [x] 03 — drill-down backend
- [x] 04 — Wails + store
- [x] 05 — panel shell + charts
- [x] 06 — dashboard tab
- [x] 07 — progression tab
- [x] 08 — errors tab
- [x] 09 — filter bar
- [x] 10 — CLI parity
- [x] 11 — docs + tests

## Références plan

Plan d'architecture : `/home/unger/.claude/plans/je-souhaite-concevoir-et-humming-cherny.md`. À relire au début de chaque fiche pour le contexte.
