# 04 — Wails binding + store frontend + positionLoader

**Goal:** Exposer `ComputeStats` et les méthodes drill-down au frontend via Wails, créer le store Svelte qui orchestre filtre ↔ fetch, et écrire le service partagé `positionLoader.js` qui route les actions *Open positions* / *Open match* / *Open tournament* vers les panneaux existants.

**Depends on:** 03

**Impact:** Plomberie sans laquelle les onglets UI ne peuvent rien afficher.

## Context

- Wails regénère `frontend/wailsjs/go/main/Database.d.ts` et `Database.js` automatiquement au `wails dev` / `wails build` dès qu'une nouvelle méthode publique apparaît sur `*Database`.
- Store pattern : voir `frontend/src/stores/tournamentStore.js`, `collectionStore.js` — writable stores minimalistes, pas de logique métier dedans.
- Toggle PR/MWC global : doit déclencher un re-render des vues sans refetch (les deux valeurs sont déjà dans `StatsResult`).
- Filtre persisté : utiliser la struct `Config` existante (`config.go` + `config.yaml` XDG path). Ajouter un sous-objet `StatsFilter`.
- Drill-down frontend : le pipeline « résultat de recherche » actuel est le chemin le plus propre pour *Open positions*. Identifier le store/service qui gère les résultats de `SearchModal` et le réutiliser (cf. `frontend/src/components/SearchModal.svelte` + `frontend/src/services/`).

## Tasks

### 1. Vérifier le binding Wails

- [ ] Lancer `wails dev` après la fiche 03.
- [ ] Vérifier que `frontend/wailsjs/go/main/Database.d.ts` contient bien :
  ```ts
  ComputeStats(arg1: main.StatsFilter): Promise<main.StatsResult>
  GetPositionIDsByStatsSelection(arg1, arg2): Promise<number[]>
  GetPositionIDsByTournament(arg1: number): Promise<number[]>
  GetPositionIDsByMatch(arg1: number): Promise<number[]>
  ```
- [ ] Si un type n'apparaît pas (ex. `StatsFilter` non exporté par `wails`), ajouter un tag struct ou un renommage pour le rendre visible.

### 2. Créer `statsStore.js`

- [ ] `frontend/src/stores/statsStore.js` :
  ```javascript
  import { writable, derived } from 'svelte/store';

  const defaultFilter = {
      playerName: '',
      tournamentIDs: [],
      dateFrom: '',
      dateTo: '',
      decisionType: -1,   // all
      matchLength: [],
  };

  export const statsFilterStore = writable(defaultFilter);
  export const statsResultStore = writable(null);
  export const statsLoadingStore = writable(false);
  export const statsErrorStore = writable(null);

  // Toggle global PR / MWC (persisté côté Config.yaml, cf. fiche 09)
  export const statsMetricStore = writable('pr'); // 'pr' | 'mwc'
  ```
- [ ] Auto-fetch : exposer une fonction `async function refreshStats()` qui appelle `ComputeStats(filter)`, met à jour `statsResultStore`. Appelée manuellement depuis `StatsPanel` au mount et sur changement de filtre (pas via `derived` pour contrôler le timing).

### 3. Créer `positionLoader.js`

- [ ] `frontend/src/services/positionLoader.js` :
  ```javascript
  import {
      GetPositionIDsByStatsSelection,
      GetPositionIDsByTournament,
      GetPositionIDsByMatch,
  } from '../../wailsjs/go/main/Database';
  import { openPanel, PANEL } from '../stores/uiStore';

  export async function loadPositionsFromSelection(ids, { focusIndex = 0 } = {}) {
      // TODO (implémentation): pousser `ids` dans le store de résultats de recherche,
      // placer l'utilisateur sur focusIndex, ouvrir PANEL.ANALYSIS
  }

  export async function loadPositionsFromStatsSelection(filter, selection) {
      const ids = await GetPositionIDsByStatsSelection(filter, selection);
      return loadPositionsFromSelection(ids);
  }

  export async function loadPositionsFromTournament(tournamentID) {
      const ids = await GetPositionIDsByTournament(tournamentID);
      return loadPositionsFromSelection(ids);
  }

  export async function loadPositionsFromMatch(matchID) {
      const ids = await GetPositionIDsByMatch(matchID);
      return loadPositionsFromSelection(ids);
  }

  export function openTournamentInPanel(tournamentID) {
      openPanel(PANEL.TOURNAMENT);
      // TODO: set selectedTournamentStore to tournamentID
  }

  export function openMatchInPanel(matchID) {
      openPanel(PANEL.MATCH);
      // TODO: set selectedMatchStore + scroll into view
  }
  ```
- [ ] Identifier le store de résultats de recherche (probablement dans `frontend/src/stores/searchStore.js` ou équivalent). Documenter l'API réutilisée dans un commentaire Go-doc-style au-dessus de `loadPositionsFromSelection`.
- [ ] `openTournamentInPanel` / `openMatchInPanel` : localiser le store de sélection dans `tournamentStore.js` / matchPanel-related store ; écrire une simple setter.

### 4. Tests Vitest

- [ ] `frontend/src/services/positionLoader.test.js` :
  - Mock `wailsjs/go/main/Database`.
  - Vérifier que `loadPositionsFromStatsSelection` appelle la bonne méthode Wails avec les bons args.
  - Vérifier que `openTournamentInPanel` met à jour le store de sélection et ouvre `PANEL.TOURNAMENT`.
- [ ] `frontend/src/stores/statsStore.test.js` :
  - Mock `ComputeStats`.
  - `refreshStats()` met `loading=true` puis `loading=false` avec `result` peuplé.
  - Erreur Wails → `statsErrorStore` rempli, `loading=false`.

### 5. Commande et toggle de panneau

- [ ] Ajouter à `frontend/src/stores/uiStore.js` :
  ```javascript
  export const PANEL = {
      // … existing …
      STATS: 'stats',
  };
  ```
- [ ] Ajouter à `frontend/src/commandProcessor.js` :
  ```javascript
  else if (command === 'stats' || command === 'st') {
      callbacks.onToggleStats?.();
  }
  ```
- [ ] Dans `frontend/src/App.svelte` (fonction `initCommandProcessor`) : `onToggleStats: () => togglePanel(PANEL.STATS)`.
- [ ] Pas de mount du composant `StatsPanel` dans cette fiche — squelette en fiche 05.

## Acceptance criteria

- [ ] `wails dev` démarre sans warning. `frontend/wailsjs/go/main/Database.d.ts` contient les 4 nouvelles méthodes.
- [ ] `npm test` vert (tests Vitest ci-dessus passent).
- [ ] `npm run lint` clean sur les nouveaux fichiers.
- [ ] La commande `:stats` dans la ligne de commande bascule bien l'état de `openPanels` (vérifiable manuellement via le devtool).

## Rollback

Revert = `git revert`. Purement additif (sauf l'entrée `PANEL.STATS` dans `uiStore.js`, qui est inerte tant que le panneau n'est pas consommé).
