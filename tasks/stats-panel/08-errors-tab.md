# 08 — Onglet Répartition d'erreurs

**Goal:** Peupler `StatsErrorsTab.svelte` : breakdown par type d'action cube (NoDouble / DoubleTake / DoublePass / TooGood) avec PR/MWC + nb décisions + taux de blunder, comparaison PR checker vs PR cube, histogramme des magnitudes d'erreurs. Chaque barre cliquable charge **uniquement les positions avec erreur** sur la catégorie.

**Depends on:** 05

**Impact:** Vue clé pour répondre à « où je fais mes erreurs ? ». **Orienter l'utilisateur vers des patterns de travail concrets.**

## Context

- Plan §Répartition d'erreurs, §Interactivité (clic = *Open positions* filtrées à `error > 0`).
- Données : `statsResultStore.CubeActionBreakdown[]`, `ErrorHistogram[]`, `PRChecker`, `PRCube`, `MWCChecker`, `MWCCube`.
- Seuil « blunder » défini dans la fiche 01 (`BLUNDER_MP = 100` millipoints EMG).

## Tasks

### 1. Breakdown par action cube

- [ ] `BarChart` vertical avec 4 barres : NoDouble, DoubleTake, DoublePass, TooGood.
- [ ] Y : PR ou MWC selon toggle ; labels sur les barres avec valeur.
- [ ] Hover tooltip : nb décisions, nb blunders, taux de blunder (%), PR exact, MWC exact.
- [ ] Clic sur une barre :
  - `loadPositionsFromStatsSelection(filter, { Kind: 'cube_action', CubeAction: <label>, OnlyWithError: true })`.
  - Charge **uniquement les positions avec erreur** (l'utilisateur veut étudier ses fautes, pas ses coups corrects — §Interactivité).
- [ ] Couleurs : toutes les barres de la même couleur (accent primaire). Pas de couleur différenciée par sous-catégorie (anti-pattern §Principes UX 5).
- [ ] État vide (0 décision cube) : message discret.

### 2. Comparaison PR checker vs PR cube

- [ ] Deux barres côte à côte (`BarChart` 2 éléments).
- [ ] Y : PR ou MWC selon toggle.
- [ ] Tooltip hover : nb décisions par type, ratio checker/cube.
- [ ] Clic :
  - Barre checker → `loadPositionsFromStatsSelection(filter, { Kind: 'checker', OnlyWithError: true })`.
  - Barre cube → `{ Kind: 'cube', OnlyWithError: true }`.

### 3. Histogramme des magnitudes d'erreurs

- [ ] `Histogram` (BarChart avec axe X catégoriel).
- [ ] Buckets en millipoints : 0–5, 5–10, 10–25, 25–50, 50–100, 100+ (fiche 01 déjà calculé).
- [ ] Y : COUNT(*) dans chaque bucket.
- [ ] Labels X : intervalles en EMG (ex. « 0.000–0.005 »).
- [ ] Hover : nb exact de positions, % du total.
- [ ] Clic sur un bucket → `loadPositionsFromStatsSelection(filter, { Kind: 'error_bucket', BucketMinMP: min, BucketMaxMP: max })` (sans `OnlyWithError` car un bucket `[0, 5)` inclut les positions correctes — l'utilisateur qui clique sait ce qu'il veut).
- [ ] Le dernier bucket (100+) : `BucketMaxMP: -1`.

### 4. Ordre et hiérarchie visuelle

- [ ] Ordre vertical des 3 sous-vues :
  1. Breakdown cube action (le plus utile pour un joueur avancé).
  2. Checker vs Cube (vue d'ensemble rapide).
  3. Histogramme magnitudes (vue exploratoire).
- [ ] Chaque sous-vue a un titre h3 discret.
- [ ] §Principes UX 4 : densité OK (3 éléments majeurs dans cet onglet).

### 5. Cohérence avec le toggle PR/MWC

- [ ] Les 3 vues basculent ensemble.
- [ ] L'histogramme ne bascule **pas** (il compte des positions, pas des valeurs). Toujours en nombre absolu.

### 6. État vide / partiel

- [ ] Si `CubeActionBreakdown` est vide (aucune décision cube) → sous-vue breakdown masquée avec message.
- [ ] Si `ErrorHistogram` tous les buckets à 0 → vue masquée.

### 7. Tests Vitest

- [ ] Vérifier que les 3 sous-vues sont rendues avec les bonnes données depuis le store.
- [ ] Clic sur une barre cube_action appelle bien `loadPositionsFromStatsSelection` avec `OnlyWithError: true`.
- [ ] Clic sur un bucket appelle avec les bonnes bornes.
- [ ] Toggle PR/MWC met à jour les valeurs des barres cube_action et checker/cube, **pas** l'histogramme.
- [ ] Cas avec 0 décision cube : sous-vue cube_action affiche un message vide, pas d'erreur.

## Acceptance criteria

- [ ] Respect §Principes UX 1 (une info clé par vue : la sous-catégorie la plus problématique saute aux yeux), 2 (hover pour détail), 4 (densité faible), 5 (palette restreinte, pas de couleur par catégorie).
- [ ] Pas de pie chart, pas de radar, pas de double axe Y (anti-patterns §Principes UX).
- [ ] Clic charge bien les positions **avec erreur** sur les 2 premières vues.
- [ ] `npm test` vert, `npm run lint` clean.

## Rollback

Revert = `git revert`. Additif.
