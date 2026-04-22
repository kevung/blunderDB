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

- [ ] Supprimer `onMount` / `onDestroy` / `unsubscribeFilter`.
- [ ] Remplacer par :
  ```js
  $effect(() => {
      refreshStats($statsFilterStore);
  });
  ```
- [ ] Si `refreshStats` est async et doit être abortable, wrapper dans un `AbortController` ou un simple drapeau local `let token = ++counter; refreshStats($statsFilterStore).then(res => { if (token === counter) ... })`.

### 2. Dépendances implicites

- [ ] Vérifier dans `refreshStats(filter)` qu'il ne lit pas d'autres stores via `get(...)`. Si oui, promouvoir les lectures à `$effect` pour tracking correct, ou ajouter les stores explicitement dans la closure de l'effet (`const x = $otherStore;`).

### 3. Tests

- [ ] `StatsPanel.reactivity.test.js` avec `@testing-library/svelte` :
  - Mock `refreshStats` (spy).
  - Test 1 : monter → `refreshStats` appelé une fois avec la valeur initiale du filtre.
  - Test 2 : `statsFilterStore.set({ ...new filter })` → `refreshStats` appelé avec le nouveau filtre.
  - Test 3 : démontage → pas de fuite (pas d'appel à `refreshStats` après démontage même si filter change).

### 4. Vérification manuelle

- [ ] `wails dev` : modifier le filtre dans le Stats Panel, vérifier que les graphiques se mettent à jour sans délai. Bascule onglet et retour : le panel se comporte comme attendu (combinaison avec fix 05.a).

### 5. Commit

- [ ] `fix(ui): StatsPanel onMount+subscribe → $effect`.

## Acceptance

- [ ] Test de réactivité StatsPanel vert (3 tests).
- [ ] Aucun `onMount` avec `.subscribe` dans StatsPanel.
- [ ] Vérif manuelle OK.

## Status

- [ ] Conversion effectuée
- [ ] Dépendances implicites vérifiées
- [ ] Tests
- [ ] Vérif manuelle
- [ ] Commit
