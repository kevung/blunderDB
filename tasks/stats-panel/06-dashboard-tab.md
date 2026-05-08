# 06 — Onglet Dashboard

**Goal:** Peupler `StatsDashboardTab.svelte` : cartes PR/MWC (global, checker, cube, rolling N), totaux, liste des 10 pires blunders. Toutes les cartes sont cliquables (drill-down vers positions correspondantes).

**Depends on:** 05

**Impact:** Première vue perçue par l'utilisateur quand il ouvre le panneau. **Doit donner la réponse à « où en suis-je ? » en un regard.**

## Context

- Plan §Dashboard.
- Plan §Principes UX — critère d'acceptation explicite (densité ≤ 6 éléments majeurs, hover pour détail, pas de KPI card colorées).
- Données : `statsResultStore.{PRGlobal, PRChecker, PRCube, PRRolling, MWCGlobal, …, Totals, TopBlunders}` (fiches 01–02).
- Drill-down : `loadPositionsFromStatsSelection(filter, {Kind: …})` (fiche 04).

## Tasks

### 1. Cartes principales (ligne du haut)

- [x] Grille CSS **3 colonnes** (PR Global / PR Checker / PR Cube) — **pas de KPI BI coloré** ; simple fond neutre, chiffre en gros, label en petit, unité explicite.
- [x] Bascule selon `$statsMetricStore` : affiche `PRGlobal` ou `MWCGlobal` (avec `— ` si `NaN`).
- [x] Hover : tooltip avec `Totals.NumDecisions`, formule, date range filtre.
- [x] Clic :
  - Carte PR Global → `loadPositionsFromStatsSelection(filter, {Kind: 'all'})`.
  - Carte PR Checker → `{Kind: 'checker', OnlyWithError: false}`.
  - Carte PR Cube → `{Kind: 'cube', OnlyWithError: false}`.
- [x] `cursor: pointer` + `aria-label="Open N positions"`.

### 2. Totaux (ligne discrète)

- [x] Sous les cartes principales, une ligne simple texte :
  `42 tournois · 186 matchs · 8 214 décisions · Période: 2024-01-12 → 2026-04-12`
- [x] Pas de card, juste un `<p class="stats-totals">` avec styles discrets. Hover sur « Période » → dates exactes.

### 3. Rolling N

- [x] Bloc avec les valeurs rolling pour N ∈ {5, 10, 50, 100, 250, 500, 1000}.
- [x] Affichage : un mini-tableau ou une barre horizontale de valeurs. **Ne pas** utiliser de sparkline (anti-pattern du plan).
- [x] Si peu de décisions (ex. N=1000 mais seulement 200 décisions totales), afficher « — » pour les N indisponibles.
- [x] Clic sur une valeur N → `loadPositionsFromStatsSelection(filter, {Kind: 'last_n', LastN: N})`.
- [x] Hover : nb exact de décisions utilisées.

### 4. Top 10 blunders

- [x] Liste `<ol>` avec chaque ligne :
  - Position #ID (raccourci)
  - Type (cube/checker) + catégorie cube si applicable
  - Erreur EMG ou MWC selon toggle
  - Nom du match + date (si disponible)
- [x] Clic sur une ligne → `loadPositionsFromStatsSelection(filter, {Kind: 'position', PositionID: id})` (ouvre directement cette unique position).
- [x] Icône ou lien secondaire sur la ligne : *Open match* qui ouvre le match dans son panneau (`openMatchInPanel(matchID)` du `positionLoader`).
- [x] Hover : `best_cube_action` si cube, indicateur « DoubleTake » / « Checker play », score au moment de la décision.

### 5. État vide

- [x] Si `Totals.NumDecisions === 0` (filtre trop restrictif), afficher un message sobre : *« Aucune décision sur la période filtrée. Élargissez les filtres. »* Pas de cards vides ni d'erreur technique.

### 6. Styles

- [x] Respecter §Principes UX 4 et 5 :
  - Limite 6 éléments majeurs : 3 cartes + totaux + rolling + top blunders = 6. OK.
  - Palette : fond panneau neutre, accent sur les chiffres, pas de gradient.
  - Chiffres principaux en typographie `tabular-nums`.
- [x] Responsive : sur largeur < 600 px, grille cartes passe à 1 colonne, top blunders liste compacte.

### 7. Tests Vitest

- [x] Composant monté avec un store mocké contenant des valeurs PR/MWC et blunders.
- [x] Toggle PR/MWC change les valeurs affichées.
- [x] Clic sur une carte appelle la bonne méthode `loadPositionsFromStatsSelection` avec le bon `SelectionSpec`.
- [x] Clic sur une ligne blunder appelle `loadPositionsFromStatsSelection` avec `Kind: "position"`.
- [x] État `Totals.NumDecisions === 0` affiche le message d'état vide.

## Acceptance criteria

- [x] Vue lisible en un regard, respect des §Principes UX.
- [x] Toutes les cartes cliquables chargent bien les positions attendues (test manuel + Vitest).
- [x] Toggle PR/MWC cohérent sur toute la vue.
- [x] `npm test` vert, `npm run lint` clean.

## Rollback

Revert = `git revert`. Additif.
