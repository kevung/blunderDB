# 05.c — StatsPanel : `onMount`-subscribe → `$effect`

**Goal :** Remplacer le pattern `onMount` + `statsFilterStore.subscribe` par un `$effect` qui lit `$statsFilterStore` et appelle `refreshStats`. Aligner StatsPanel sur les conventions Svelte 5.

**Depends on :** 01, 04.

**Impact :** Corrige les cas où `refreshStats` ne se redéclenche pas (ou à retardement) lors de changements de filtre ou de re-montage du panneau. Complète le fix principal de 05.a (qui traite la bascule d'onglets côté App.svelte).

## Context

Extrait actuel :

```js
// StatsPanel.svelte:11-25 (à revalider)
let activeTab = $state('dashboard');
let unsubscribeFilter;

onMount(() => {
    unsubscribeFilter = statsFilterStore.subscribe((filter) => {
        refreshStats(filter);
    });
});

onDestroy(() => {
    unsubscribeFilter?.();
});
```

Problème : mélange `$state` (rune Svelte 5) + subscription manuelle dans `onMount` (pattern Svelte 4). En Svelte 5, `$effect` est déjà lié au cycle de vie du composant et n'a pas besoin d'unsubscribe manuel.

## Files touched

- **Edit:** `frontend/src/components/stats/StatsPanel.svelte`.
- **New:** `frontend/src/__tests__/StatsPanel.reactivity.test.js` (ou ajouter au `StatsPanel.test.js` existant).

## Tasks

### 1. Conversion

- [x] Supprimer `onMount` / `onDestroy` / `unsubscribeFilter`.
- [x] Remplacer par :
  ```js
  $effect(() => {
      logger.perf('StatsPanel:refreshStats', () => refreshStats($statsFilterStore));
  });
  ```
- [x] `refreshStats` est fire-and-forget (sets stores uniquement) — pas besoin d'AbortController.

### 2. Dépendances implicites

- [x] `refreshStats(filter)` ne lit aucun autre store via `get(...)` — il appelle uniquement `ComputeStats(filter)` et écrit dans les stores de résultat. Aucune dépendance implicite à promouvoir.

### 3. Tests

- [x] `StatsPanel.reactivity.test.js` créé avec `@testing-library/svelte` :
  - Mock `ComputeStats` (proxy de `refreshStats` — plus fiable que mock du module statsStore).
  - T1 : montage → `ComputeStats` appelé au moins une fois avec le filtre initial.
    Note : `StatsFilterBar.onMount` appelle `statsFilterStore.set()` une fois résolue (2e appel normal).
  - T2 : `statsFilterStore.set(newFilter)` → `ComputeStats` re-appelé avec le nouveau filtre (+1 appel relatif).
  - T3 : démontage → pas de fuite (count stable après `statsFilterStore.set()`).

### 4. Vérification manuelle

- [ ] `wails dev` : modifier le filtre dans le Stats Panel, vérifier que les graphiques se mettent à jour sans délai. Bascule onglet et retour : le panel se comporte comme attendu (combinaison avec fix 05.a).
  *(Test automatisé OK — vérification manuelle à réaliser lors de la phase 06)*

### 5. Commit

- [x] `fix(ui): StatsPanel onMount+subscribe → $effect`.

## Acceptance

- [x] Test de réactivité StatsPanel vert (3 tests — T1, T2, T3).
- [x] Aucun `onMount` avec `.subscribe` dans StatsPanel.
- [ ] Vérif manuelle OK (à confirmer lors de la phase 06).

## Status

- [x] Conversion effectuée
- [x] Dépendances implicites vérifiées
- [x] Tests (318/318 passent)
- [ ] Vérif manuelle
- [x] Commit
