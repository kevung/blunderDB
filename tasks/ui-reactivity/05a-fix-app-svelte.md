# 05.a — App.svelte : handler `activeTabStore` et propagation EPC

**Goal :** Convertir les `.subscribe()` d'App.svelte en `$effect` correctement trackés, étendre le handler d'onglets aux cas manquants (`stats`, `tournaments`, `collections`, `anki`), ajouter un effet EPC qui se déclenche aussi bien sur changement de position que sur passage en mode EPC.

**Depends on :** 02 (spec rouge S2 existe), 03 (instrumentation perf), 04 (audit validé).

**Impact :** Corrige le **symptôme dominant** (S2 — toutes transitions Stats cassées). Après cette fiche, la spec Playwright `tab-switch-stats.spec.js` doit passer au vert.

## Context

Extraits critiques actuels (références à revalider après `git diff` le jour du fix — les lignes peuvent avoir bougé) :

```js
// App.svelte:128-130
positionStore.subscribe((value) => {
    if ($statusBarModeStore === 'EPC' && value) updateEPC(value);
});
```
→ la lecture `$statusBarModeStore` **dans** le callback est une auto-subscription compilée en `get()`, donc non trackée : le callback ne re-court pas quand le mode change.

```js
// App.svelte:198-213
activeTabStore.subscribe((tab) => {
    const prevTab = previousTab;
    previousTab = tab;
    if (tab === 'search' && $databasePathStore && $statusBarModeStore !== 'EDIT') enterEditMode();
    else if (prevTab === 'search' && tab !== 'search' && $statusBarModeStore === 'EDIT') exitEditMode();
    if (tab === 'epc' && $statusBarModeStore !== 'EPC') enterEPCMode();
    else if (prevTab === 'epc' && tab !== 'epc' && $statusBarModeStore === 'EPC') exitEPCMode();
    if (tab === 'matches') openPanel(PANEL.MATCH);
    else if (prevTab === 'matches' && tab !== 'matches') closePanel(PANEL.MATCH);
});
```
→ aucun cas `stats`, `tournaments`, `collections`, `anki`. C'est le **bug principal** confirmé par l'utilisateur (S2).

## Files touched

- **Edit:** `frontend/src/App.svelte` (principalement ~128-218).
- **Edit (éventuel):** stores dans `frontend/src/stores/` si nommage de panels à ajouter (`openPanels.js` ou équivalent).
- **New:** `frontend/src/__tests__/App.tabHandler.test.js` — test unitaire qui monte un harnais minimal et vérifie l'ouverture/fermeture des panels pour chaque cas de tab.

## Tasks

### 1. Convertir `activeTabStore.subscribe` → `$effect`

- [ ] Remplacer le bloc `.subscribe` par un `$effect` qui lit `$activeTabStore`, `$statusBarModeStore`, `$databasePathStore` **directement dans le corps de l'effet**.
- [ ] Gérer `previousTab` via `$state` ou via une variable `let` capturée par l'effet (avec attention : il faut tracker `tab`, pas `previousTab`). Privilégier : 
  ```js
  let previousTab = $state(null);
  $effect(() => {
      const tab = $activeTabStore;
      const prev = previousTab;
      previousTab = tab;
      // ...
  });
  ```
  Note : si `$effect` ne doit pas se re-run sur mutation de `previousTab`, utiliser `untrack(() => previousTab)` depuis `svelte`.
- [ ] Ajouter les cas manquants :
  - `tab === 'stats'` → `openPanel(PANEL.STATS)` (vérifier le nom exact dans `openPanels`).
  - `tab === 'tournaments'` → `openPanel(PANEL.TOURNAMENT)`.
  - `tab === 'collections'` → `openPanel(PANEL.COLLECTION)`.
  - `tab === 'anki'` → `openPanel(PANEL.ANKI)` (ou équivalent).
  - Pour chacun, symétrique `closePanel(...)` quand on en sort (`prev === 'stats' && tab !== 'stats'`).
- [ ] Vérifier que les constantes `PANEL.*` existent ; en ajouter si manquantes (minimalement).

### 2. Effet EPC : double dépendance

- [ ] Remplacer le `positionStore.subscribe(...)` par :
  ```js
  $effect(() => {
      const value = $positionStore;
      const mode = $statusBarModeStore;
      if (mode === 'EPC' && value) updateEPC(value);
  });
  ```
  → l'effet se re-run quand **soit** la position change, **soit** le mode passe à EPC, ce qui corrige le cas « retour à EPC avec nouvelle position ».

### 3. Autres `.subscribe()` dans App.svelte

- [ ] `currentPositionIndexStore.subscribe(async ...)` → convertir en `$effect` async. Attention aux race conditions : utiliser un `AbortController` ou un drapeau local `let current = ++counter` pour ignorer les résultats obsolètes.
- [ ] Passer en revue **tous** les `.subscribe()` restants dans App.svelte (via l'audit Fiche 04) et les convertir de manière cohérente.

### 4. Instrumentation perf

- [ ] Wrapper les corps d'effets critiques dans `logger.perf('App:activeTabHandler', ...)` et `logger.perf('App:epcSync', ...)` (Fiche 03 prérequise).

### 5. Tests

- [ ] **Spec Playwright** `tab-switch-stats.spec.js` (Fiche 02) doit passer au vert après le fix. C'est le critère principal.
- [ ] **Spec Playwright** `epc-bar-refreshes-on-return.spec.js` (Fiche 02) doit passer au vert.
- [ ] **Test unitaire** `App.tabHandler.test.js` : extraire la logique du handler dans une fonction pure `handleTabChange(prev, tab, mode, dbPath) → { actionsOuvrir, actionsFermer }` (refactor interne, pas de changement UX), et tester chaque transition. Cette extraction est **recommandée** pour faciliter le test — pas obligatoire.
- [ ] Si l'extraction est faite : `frontend/src/App.svelte` appelle `handleTabChange(...)` depuis l'effet ; la fonction elle-même vit dans un module séparé (`frontend/src/services/tabHandler.js` ?) ou en tête d'App.svelte.

### 6. Vérification manuelle

- [ ] `wails dev` : refaire S2 dans toutes ses variantes (Match↔Stats, EPC↔Stats, Stats↔Anki). Chaque bascule doit montrer le contenu du bon panel immédiatement.
- [ ] Refaire S1 étendu (charger position A, aller EPC, aller Stats, charger position B, retour EPC → nouvelle valeur affichée).
- [ ] Activer `VITE_PERF_THRESHOLD_MS=0 wails dev` et vérifier le journal : `App:activeTabHandler` ≤ 16 ms, `App:epcSync` ≤ 16 ms, pas de boucle.

### 7. Commit

- [ ] Commit atomique avec message au format conventional commit : `fix(ui): convert App.svelte subscribe to $effect and add missing tab cases`.

## Acceptance

- [ ] Spec `tab-switch-stats.spec.js` verte.
- [ ] Spec `epc-bar-refreshes-on-return.spec.js` verte.
- [ ] Test unitaire `App.tabHandler.test.js` vert.
- [ ] Vérification manuelle S2 OK pour toutes les variantes incluant Stats.
- [ ] Pas de régression sur les transitions qui marchaient (Match↔EPC, etc.) — couvert par spec existante ou test manuel.

## Status

- [ ] `activeTabStore` subscribe → $effect + cas manquants
- [ ] EPC effet double dépendance
- [ ] Autres subscribe d'App.svelte convertis
- [ ] Instrumentation perf
- [ ] Specs Playwright vertes
- [ ] Test unitaire
- [ ] Vérif manuelle
- [ ] Commit
