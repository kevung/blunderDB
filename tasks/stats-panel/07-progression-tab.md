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

- [ ] Créer `frontend/src/components/stats/gradeBands.js` :
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
- [ ] Couleurs : dégradé très doux, du vert pâle (World Class) au rouge pâle (Beginner). **Bandes horizontales en fond**, pas de barres saturées. Respect §Principes UX 5.

### 2. Courbe PR par tournoi

- [ ] `LineChart` avec :
  - X : nom du tournoi tronqué (labels courts) ; tooltip avec nom complet + date.
  - Y : PR ou MWC selon `$statsMetricStore`.
  - Datasets : un seul dataset, ligne principale (accent primaire).
  - `options.plugins.annotation` ou plugin custom : bandes horizontales `GRADE_BANDS`.
- [ ] Tooltip hover : nom tournoi complet, date, PR ou MWC, nb décisions, nb matchs du tournoi.
- [ ] Clic sur un point : ouvrir un menu contextuel (petit popover) avec 2 boutons :
  - *Open tournament* → `openTournamentInPanel(tournamentID)` (affiche la liste des matchs du tournoi dans le panneau Tournoi).
  - *Open positions* → `loadPositionsFromTournament(tournamentID)` (charge toutes les positions du tournoi dans la navigation Analysis).
- [ ] Légende masquée (un seul dataset, §Principes UX 7).
- [ ] Fallback : si 1 seul tournoi, afficher juste une carte avec le chiffre (pas de courbe à 1 point).

### 3. Scatter PR par match

- [ ] `ScatterChart` avec :
  - X : `match_date` (axe temporel).
  - Y : PR ou MWC du match.
  - Taille du point : proportionnelle à `NumDecisions` (clampée pour éviter les outliers visuels).
  - Datasets : un seul, points accent primaire.
- [ ] Tooltip hover : date + heure du match, noms des deux joueurs (via `MatchStats.PlayerName` — celui du perspective filter), PR, nb décisions.
- [ ] Clic sur un point → menu contextuel 2 choix :
  - *Open match* → `openMatchInPanel(matchID)`.
  - *Open positions* → `loadPositionsFromMatch(matchID)` (toutes positions du match avec erreur, cf. fiche 03 `OnlyWithError`).
- [ ] Bandes de grade horizontales en fond (même palette que la courbe).

### 4. Menu contextuel réutilisable

- [ ] Composant `frontend/src/components/stats/ContextMenu.svelte` (petit popover ancré au point cliqué).
- [ ] Props : `{ x, y, items: [{ label, onClick }] }`.
- [ ] Fermeture au clic extérieur ou touche Escape.
- [ ] A11y : focusable, keyboard navigation.

### 5. État vide

- [ ] Aucun tournoi sur le filtre → message « Aucun tournoi dans la période. Importez des matchs taggués avec un tournoi pour voir votre progression. »
- [ ] Aucun match (exceptionnel) → même message.

### 6. Styles

- [ ] Les deux graphes en deux lignes empilées (pas de side-by-side forcé). Chaque graphe prend toute la largeur.
- [ ] Hauteur ~250-300 px par graphe.
- [ ] Axes : gridline X discret, Y avec labels unité « PR » ou « MWC % » une seule fois (§Principes UX 7).

### 7. Tests Vitest

- [ ] Mocker Chart.js pour vérifier que les datasets sont bien construits à partir du store.
- [ ] Simuler un clic sur un point : vérifier que le menu contextuel s'ouvre avec les 2 items.
- [ ] Vérifier l'appel correct à `openTournamentInPanel` / `loadPositionsFromTournament` selon le choix.
- [ ] Tester le comportement avec 0 tournoi (état vide affiché).
- [ ] Tester que le toggle PR/MWC change le dataset Y.

## Acceptance criteria

- [ ] Courbe PR par tournoi affiche bien la valeur pondérée (pas la moyenne des matchs).
- [ ] Bandes de grade visibles en fond mais non invasives (transparence).
- [ ] Menu contextuel offre bien les 2 choix pour tournoi et match.
- [ ] Respect des §Principes UX 1, 2, 4, 5, 7.
- [ ] `npm test` vert, `npm run lint` clean.

## Rollback

Revert = `git revert`. Additif.
