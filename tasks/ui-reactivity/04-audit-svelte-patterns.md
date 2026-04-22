# 04 — Audit patterns Svelte 4/5 (scope panneaux + statusbar)

**Goal :** Inventorier **toutes** les incohérences `.subscribe()` vs `$effect` et les closures stales dans la portée. L'inventaire sert de todo pour les Fiches 05.*.

**Depends on :** 00.

**Impact :** Source of truth des corrections à apporter. Chaque ligne de l'audit doit être reliée à une Fiche 05.x (une ou plusieurs).

## Context

Premières trouvailles de l'exploration initiale :

| # | Fichier:lignes | Pattern | Sévérité |
|---|---|---|---|
| 1 | `App.svelte:128-130` | `positionStore.subscribe` pour EPC, dépend implicitement de `$statusBarModeStore` non tracké | 🟡 moyen |
| 2 | `App.svelte:189-196` | `currentPositionIndexStore.subscribe` async → race conditions | 🟢 bas |
| 3 | `App.svelte:198-213` | `activeTabStore.subscribe` ne gère pas `stats`/`tournaments`/`collections`/`anki` | 🔴 critique |
| 4 | `StatsPanel.svelte:11-25` | `.subscribe()` dans `onMount` au lieu de `$effect` | 🔴 critique |
| 5 | `MatchPanel.svelte:57-94` | Closure stale sur `visible` dans `openPanels.subscribe` | 🟠 grave |
| 6 | `StatusBar.svelte:26-35` | `.subscribe()` root-level, pas d'`onDestroy` | 🟡 moyen |
| 7 | `TabbedPanel.svelte:34-45` | Keep-alive laisse les enfants avec `$effect` actifs | 🟡 moyen |
| 8 | `positionService.js:881-915` | Ordre `positionStore.set` vs `statusBarModeStore.set` — race | 🟢 bas |

L'audit de cette fiche doit **confirmer ou affiner** cette liste sur le scope complet : App.svelte, TabbedPanel, MatchPanel, StatsPanel, EPCPanel, StatusBar, et les stores du répertoire `frontend/src/stores/` qui alimentent ces composants.

## Files touched

- **New:** `doc/archive/ui-reactivity-audit.md` (≤ 300 lignes, splitter si plus long).

## Tasks

### 1. Grep systématique

- [x] Grep `\.subscribe\(` dans `frontend/src/` restreint au scope. Lister toutes les occurrences avec fichier:ligne et un extrait.
- [x] Pour chaque occurrence, classifier :
  - **(a)** à convertir en `$effect` lisant le store via `$storeName`.
  - **(b)** à garder (cas rare : setup d'un side-effect global, subscription avec `onDestroy` explicite).
  - **(c)** à supprimer (doublon, subscription qui n'apporte rien).
- [x] Grep `onMount\s*\(` dans le même scope, vérifier les corps qui contiennent `.subscribe`. Lister ces cas (StatsPanel au minimum).

### 2. Grep des closures stales

- [x] Grep `let\s+\w+\s*=` dans les composants, repérer les `let` lus **à l'intérieur** d'un callback `.subscribe()` et modifiés ailleurs : candidats à la promotion `$state`.
- [x] Grep `subscribe.*async` : les handlers async dans `.subscribe()` sont candidats à un `$effect` avec `debounce` ou à un `AbortController`.

### 3. Grep des $effect à dépendances non trackées

- [x] Grep `\$effect\(` dans le scope. Pour chaque, vérifier :
  - Le corps lit-il des stores via `$storeName` (tracké) ou via `get(store)` (NON tracké) ?
  - Appelle-t-il des fonctions qui lisent des stores en interne (non tracké → doit être relu dans l'effet) ?
- [x] Repérer les effets qui appellent des fonctions externes lisant des stores ; documenter le faux-positif potentiel.

### 4. Rédaction du document

- [x] `doc/archive/ui-reactivity-audit.md` structuré :
  ```markdown
  # UI reactivity audit

  ## Méthodologie
  (brève)

  ## Synthèse
  | Fichier | Ligne | Pattern | Sévérité | Correction | Fiche |

  ## Détails par fichier
  ### App.svelte
  (une sous-section par occurrence : extrait, diagnostic, correction proposée)

  ### TabbedPanel.svelte
  ...

  ### MatchPanel.svelte
  ...

  etc.
  ```
- [x] Chaque ligne du tableau de synthèse doit avoir une colonne « Fiche » pointant vers une Fiche 05.x.

### 5. Ré-ouverture des Fiches 05.* si besoin

- [x] Si l'audit révèle un problème hors de la liste pré-identifiée, créer une nouvelle Fiche `05.g-...` plutôt que d'étendre une fiche existante au-delà de son périmètre.
- [x] Si une Fiche 05.x s'avère inutile (faux-positif), la marquer « abandonnée » dans le README du chantier et archiver la fiche avec une note.

## Acceptance

- [x] `ui-reactivity-audit.md` ≤ 300 lignes.
- [x] Toutes les occurrences du scope sont listées avec diagnostic.
- [x] Chaque occurrence « à corriger » est mappée à une Fiche 05.x.
- [x] Le README du chantier est mis à jour si de nouvelles fiches apparaissent.

## Status

- [x] Grep `.subscribe()`
- [x] Grep `onMount` + subscribe
- [x] Grep closures stales
- [x] Grep `$effect` non trackés
- [x] Doc `ui-reactivity-audit.md` rédigée
- [x] Mapping audit ↔ fiches 05.x validé
