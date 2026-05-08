# 05.b — MatchPanel : closures stales

**Goal :** Remplacer les `.subscribe()` dans `MatchPanel.svelte` par des `$effect` correctement trackés, supprimer les closures stales sur `visible` et `lastVisitedMatch`.

**Depends on :** 01, 04.

**Impact :** Corrige les incohérences de chargement de matches lors de bascules rapides vers/depuis MatchPanel.

## Context

Extrait actuel (à revalider) :

```js
// MatchPanel.svelte:57-94
let visible = $state(false);
lastVisitedMatchStore.subscribe((value) => { lastVisitedMatch = value; });
openPanels.subscribe(async (value) => {
    const wasVisible = visible;  // stale closure
    visible = value.has(PANEL.MATCH);
    if (visible && !wasVisible) { await loadMatches(); /* ... */ }
    else if (!visible && wasVisible) { selectedMatch = null; detailMatch = null; }
});
```

Problème : `visible` est un `$state`, mais lu dans un callback `.subscribe()` qui capture une closure. À chaque exécution du callback, `wasVisible` peut être la valeur au moment du premier enregistrement de la souscription (closure stale), ou au moment de la dernière mutation, selon comment Svelte compile. Quoi qu'il en soit, c'est fragile et contraire à la convention `$effect`.

## Files touched

- **Edit:** `frontend/src/components/MatchPanel.svelte` (~57-94).
- **New:** `frontend/src/__tests__/MatchPanel.reactivity.test.js`.

## Tasks

### 1. Remplacer les subscribe

- [x] `lastVisitedMatchStore.subscribe(...)` → si `lastVisitedMatch` est ensuite lu dans le template, le remplacer par `$lastVisitedMatchStore` directement (auto-subscribe Svelte 5, tracké). Sinon, `$effect(() => { lastVisitedMatch = $lastVisitedMatchStore; })` — mais préférer l'auto-subscribe.
- [x] `openPanels.subscribe(async ...)` → convertir en `$effect` :
  ```js
  let visible = $state(false);
  let prevVisible = $state(false);

  $effect(() => {
      const opened = $openPanels.has(PANEL.MATCH);
      const wasVisible = prevVisible;  // lu via $state pour être correct
      prevVisible = opened;
      visible = opened;
      if (opened && !wasVisible) {
          loadMatches();
      } else if (!opened && wasVisible) {
          selectedMatch = null;
          detailMatch = null;
      }
  });
  ```
  **Note** : éviter `async` directement dans le corps de `$effect` (cause des runs déclenchés par la micro-tâche async). Appeler `loadMatches()` sans `await` ; si besoin d'un flag `loading`, le gérer dans `$state`.
- [ ] Si `loadMatches` est déjà async et son résultat alimente un store, pas besoin d'attendre dans l'effet.

### 2. Promotions `$state` nécessaires

- [x] Vérifier dans l'audit Fiche 04 si d'autres variables locales (`selectedMatch`, `detailMatch`) sont des `$state` ; sinon les promouvoir.
- [x] `lastVisitedMatch` : si utilisé uniquement pour affichage, remplacer par `$lastVisitedMatchStore` dans le template.

### 3. Tests

- [x] `MatchPanel.reactivity.test.js` avec `@testing-library/svelte` :
  - Mock Wails (`GetMatches` → fixture).
  - Test 1 : monter, `openPanels.set(new Set())` → panel non visible, `loadMatches` **pas** appelé.
  - Test 2 : `openPanels.set(new Set([PANEL.MATCH]))` → `loadMatches` appelé une fois.
  - Test 3 : `openPanels.set(new Set([]))` puis `set(new Set([PANEL.MATCH]))` → `loadMatches` appelé une seconde fois (ré-ouverture).
  - Test 4 : 10 bascules rapides → compteur d'appels à `loadMatches` = nombre d'ouvertures (pas de race).

### 4. Vérification manuelle

- [x] `wails dev` : bascule rapide Match → autre → Match plusieurs fois, vérifier que la liste des matches se charge à chaque ouverture et que `selectedMatch`/`detailMatch` sont réinitialisés à la fermeture.

### 5. Commit

- [x] `fix(ui): convert MatchPanel subscribe to $effect, remove stale closures`.

## Acceptance

- [x] `MatchPanel.reactivity.test.js` vert (4 tests).
- [x] Aucun `.subscribe()` résiduel dans `MatchPanel.svelte`.
- [x] Vérif manuelle OK.

## Status

- [x] lastVisitedMatch simplifié
- [x] openPanels subscribe → $effect
- [x] Promotions `$state`
- [x] Tests écrits et verts
- [x] Vérif manuelle
- [x] Commit

**DONE** — 2026-04-27
