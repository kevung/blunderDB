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

- [x] Remplacer le bloc `.subscribe` par un `$effect` qui lit `$activeTabStore`, `$statusBarModeStore`, `$databasePathStore` **directement dans le corps de l'effet**.
- [x] Gérer `previousTab` via `$state` ou via une variable `let` capturée par l'effet (avec attention : il faut tracker `tab`, pas `previousTab`). Privilégier : 
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
- [x] Ajouter les cas manquants :
  - `tab === 'stats'` → `openPanel(PANEL.STATS)` (vérifier le nom exact dans `openPanels`).
  - `tab === 'tournaments'` → `openPanel(PANEL.TOURNAMENT)`.
  - `tab === 'collections'` → `openPanel(PANEL.COLLECTION)`.
  - `tab === 'anki'` — AnkiPanel ne s'abonne pas à `openPanels`, aucun PANEL.ANKI nécessaire.
  - Symétrique via `if/else` dans `applyTabPanels(tab)` (module `services/tabHandler.js`).
- [x] Vérifier que les constantes `PANEL.*` existent ; PANEL.STATS, PANEL.TOURNAMENT, PANEL.COLLECTION existent déjà dans uiStore.js.

### 2. Effet EPC : double dépendance

- [x] Remplacer le `positionStore.subscribe(...)` par un `$effect` avec double dépendance sur `$positionStore` et `$statusBarModeStore`.

### 3. Autres `.subscribe()` dans App.svelte

- [x] `currentPositionIndexStore.subscribe(async ...)` → converti en `$effect` avec drapeau `cancelled` contre les race conditions.
- [x] `positionReloadTriggerStore.subscribe` → converti en `$effect` (seule dépendance trackée ; `$databasePathStore` lu via `untrack`).
- [x] `positionsStore.subscribe` gardé comme `.subscribe()` intentionnel (cf. commentaire en code — pas utilisé dans le template).

### 4. Instrumentation perf

- [x] `logger.perf('App:activeTabHandler', ...)` et `logger.perf('App:epcSync', ...)` conservés dans le `$effect` du tab handler.

### 5. Tests

- [ ] **Spec Playwright** `tab-switch-stats.spec.js` (Fiche 02) doit passer au vert après le fix. (nécessite `wails dev`)
- [ ] **Spec Playwright** `epc-bar-refreshes-on-return.spec.js` (Fiche 02) doit passer au vert. (nécessite `wails dev`)
- [x] **Test unitaire** `App.tabHandler.test.js` — créé et vert (15 tests). Logique extraite dans `frontend/src/services/tabHandler.js`.

### 6. Vérification manuelle

- [ ] `wails dev` : refaire S2 dans toutes ses variantes (Match↔Stats, EPC↔Stats, Stats↔Anki).
- [ ] Refaire S1 étendu (charger position A, aller EPC, aller Stats, charger position B, retour EPC → nouvelle valeur affichée).
- [ ] Activer `VITE_PERF_THRESHOLD_MS=0 wails dev` et vérifier le journal.

### 7. Commit

- [x] Commit atomique : `fix(ui): convert App.svelte subscribe to $effect and add missing tab cases`.

## Acceptance

- [ ] Spec `tab-switch-stats.spec.js` verte. (nécessite `wails dev`)
- [ ] Spec `epc-bar-refreshes-on-return.spec.js` verte. (nécessite `wails dev`)
- [x] Test unitaire `App.tabHandler.test.js` vert (15 tests, 100 % des transitions couvertes).
- [ ] Vérification manuelle S2 OK pour toutes les variantes incluant Stats.
- [ ] Pas de régression sur les transitions qui marchaient — suite vitest complète verte (309 tests).

## Status

- [x] `activeTabStore` subscribe → $effect + cas manquants (stats, tournaments, collections)
- [x] EPC effet double dépendance ($positionStore + $statusBarModeStore)
- [x] Autres subscribe d'App.svelte convertis (currentPositionIndexStore, positionReloadTriggerStore)
- [x] Instrumentation perf conservée
- [ ] Specs Playwright vertes (nécessite `wails dev`)
- [x] Test unitaire App.tabHandler.test.js (15 tests verts)
- [ ] Vérif manuelle
- [x] Commit
