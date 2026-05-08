# 07 — Onglet Progression

**Goal:** Peupler `StatsProgressionTab.svelte` : courbe PR/MWC par tournoi dans l'ordre chronologique avec bandes de grade backgammon, et scatter PR/MWC par match. Chaque point est cliquable avec menu contextuel 2 choix (*Open tournament/match* vs *Open positions*).

**Depends on:** 05

**Impact:** Vue clé pour répondre à « comment je progresse ? ». **Lisibilité et interprétation rapide sont primordiales.**

## Context

- Plan §Progression, §Règle d'agrégation (PR tournoi = `sum/sum`, PAS moyenne des PR matchs), §Interactivité.
- Données : `statsResultStore.PerTournament[]` (ordre chrono via `date`), `PerMatch[]`.
- Bandes de grade backgammon (norme XG/gnuBG, seuils en PR) :
  - World Class : < 2
  - Expert : 2–4
  - World : 4–6 (certaines refs)
  - Advanced : 6–9
  - Intermediate : 9–12
  - Casual : 12–16
  - Beginner : ≥ 16
- Charts : `LineChart.svelte`, `ScatterChart.svelte` (fiche 05).

## Tasks

### 1. Constantes grades

- [x] Créer `frontend/src/components/stats/gradeBands.js` :
  ```javascript
  export const GRADE_BANDS = [
      { label: 'World Class', min: 0, max: 2,  color: '#...' },
      { label: 'Expert',      min: 2, max: 4,  color: '#...' },
      { label: 'Advanced',    min: 4, max: 6,  color: '#...' },
      { label: 'Intermediate',min: 6, max: 9,  color: '#...' },
      { label: 'Casual',      min: 9, max: 12, color: '#...' },
      { label: 'Beginner',    min: 12, max: Infinity, color: '#...' },
  ];
  ```
- [x] Couleurs : dégradé très doux, du vert pâle (World Class) au rouge pâle (Beginner). **Bandes horizontales en fond**, pas de barres saturées. Respect §Principes UX 5.

### 2. Courbe PR par tournoi

- [x] `LineChart` avec :
  - X : nom du tournoi tronqué (labels courts) ; tooltip avec nom complet + date.
  - Y : PR ou MWC selon `$statsMetricStore`.
  - Datasets : un seul dataset, ligne principale (accent primaire).
  - `options.plugins.annotation` ou plugin custom : bandes horizontales `GRADE_BANDS`.
- [x] Tooltip hover : nom tournoi complet, date, PR ou MWC, nb décisions, nb matchs du tournoi.
- [x] Clic sur un point : ouvrir un menu contextuel (petit popover) avec 2 boutons :
  - *Open tournament* → `openTournamentInPanel(tournamentID)` (affiche la liste des matchs du tournoi dans le panneau Tournoi).
  - *Open positions* → `loadPositionsFromTournament(tournamentID)` (charge toutes les positions du tournoi dans la navigation Analysis).
- [x] Légende masquée (un seul dataset, §Principes UX 7).
- [x] Fallback : si 1 seul tournoi, afficher juste une carte avec le chiffre (pas de courbe à 1 point).

### 3. Scatter PR par match

- [x] `ScatterChart` avec :
  - X : `match_date` (axe temporel).
  - Y : PR ou MWC du match.
  - Taille du point : proportionnelle à `NumDecisions` (clampée pour éviter les outliers visuels).
  - Datasets : un seul, points accent primaire.
- [x] Tooltip hover : date + heure du match, noms des deux joueurs (via `MatchStats.PlayerName` — celui du perspective filter), PR, nb décisions.
- [x] Clic sur un point → menu contextuel 2 choix :
  - *Open match* → `openMatchInPanel(matchID)`.
  - *Open positions* → `loadPositionsFromMatch(matchID)` (toutes positions du match avec erreur, cf. fiche 03 `OnlyWithError`).
- [x] Bandes de grade horizontales en fond (même palette que la courbe).

### 4. Menu contextuel réutilisable

- [x] Composant `frontend/src/components/stats/ContextMenu.svelte` (petit popover ancré au point cliqué).
- [x] Props : `{ x, y, items: [{ label, onClick }] }`.
- [x] Fermeture au clic extérieur ou touche Escape.
- [x] A11y : focusable, keyboard navigation.

### 5. État vide

- [x] Aucun tournoi sur le filtre → message « Aucun tournoi dans la période. Importez des matchs taggués avec un tournoi pour voir votre progression. »
- [x] Aucun match (exceptionnel) → même message.

### 6. Styles

- [x] Les deux graphes en deux lignes empilées (pas de side-by-side forcé). Chaque graphe prend toute la largeur.
- [x] Hauteur ~250-300 px par graphe.
- [x] Axes : gridline X discret, Y avec labels unité « PR » ou « MWC % » une seule fois (§Principes UX 7).

### 7. Tests Vitest

- [x] Mocker Chart.js pour vérifier que les datasets sont bien construits à partir du store.
- [x] Simuler un clic sur un point : vérifier que le menu contextuel s'ouvre avec les 2 items.
- [x] Vérifier l'appel correct à `openTournamentInPanel` / `loadPositionsFromTournament` selon le choix.
- [x] Tester le comportement avec 0 tournoi (état vide affiché).
- [x] Tester que le toggle PR/MWC change le dataset Y.

## Acceptance criteria

- [x] Courbe PR par tournoi affiche bien la valeur pondérée (pas la moyenne des matchs).
- [x] Bandes de grade visibles en fond mais non invasives (transparence).
- [x] Menu contextuel offre bien les 2 choix pour tournoi et match.
- [x] Respect des §Principes UX 1, 2, 4, 5, 7.
- [x] `npm test` vert, `npm run lint` clean.

## Rollback

Revert = `git revert`. Additif.
