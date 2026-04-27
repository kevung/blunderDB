# 05.d — StatusBar : `.subscribe()` → `$effect`

**Goal :** Convertir les `.subscribe()` root-level de `StatusBar.svelte` en `$effect` correctement trackés. Aligner avec les conventions Svelte 5.

**Depends on :** 01, 04.

**Impact :** Corrige les éventuels cas où `commandHistory` ou `showInput` ne se mettent pas à jour correctement. Le canari de la Fiche 01 couvre déjà la réactivité d'affichage ; cette fiche concerne les effets de bord (focus, chargement history).

## Context

Extrait actuel (StatusBar.svelte:26-35) :

```js
let commandHistory = [];
commandHistoryStore.subscribe((value) => (commandHistory = value));

showCommandInputStore.subscribe(async (value) => {
    showInput = value;
    if (value) {
        await loadHistory();
        await tick();
        inputEl?.focus();
    }
});
```

Problèmes :
- `commandHistory` en `let` alors que lu dans `handleKeyDown` → closure stale possible.
- `showCommandInputStore.subscribe(async ...)` → side-effects sur focus dans un callback async, pas de cleanup.

## Files touched

- **Edit:** `frontend/src/components/StatusBar.svelte` (~21-35, potentiellement aussi lignes dans `handleKeyDown`).
- **Edit (éventuel):** `frontend/src/__tests__/StatusBar.reactivity.test.js` — ajouter tests pour les nouveaux flows.

## Tasks

### 1. `commandHistory`

- [x] Soit promouvoir en `$state` et synchroniser via `$effect(() => { commandHistory = $commandHistoryStore; })`.
- [x] Soit, plus simple, lire directement `$commandHistoryStore` là où c'est utilisé dans `handleKeyDown` — cela évite le double state. Remplacer dans `handleKeyDown` les lectures de `commandHistory` par `$commandHistoryStore` (attention : dans Svelte 5, `$storeName` hors markup doit être lu via la rune, ou via `get(store)` dans une fonction).
  - Dans un handler imperatif (event handler), utiliser `get(commandHistoryStore)` **ou** garder le `$state` local synchronisé via un `$effect`. La seconde option est plus idiomatique.
- [x] Choisir l'une des deux approches et documenter le choix dans le commit message.
  - **Choix retenu :** `let commandHistory = $derived($commandHistoryStore)` — plus idiomatique que `$state` + `$effect`, évite toute écriture dans un effet.

### 2. `showCommandInputStore`

- [x] Remplacer par `$effect` :
  ```js
  $effect(() => {
      showInput = $showCommandInputStore;
      if (showInput) {
          loadHistory().then(() => tick()).then(() => inputEl?.focus());
      }
  });
  ```
  → `showInput` peut être `$state`. Attention à ne pas déclencher de boucle : l'effet lit `showInput` indirectement, mais l'écrit ; en Svelte 5, l'écriture pendant l'exécution de l'effet est autorisée si les lectures ne dépendent que de `$showCommandInputStore` (pas de re-trigger).
  - **Choix retenu :** `let showInput = $derived($showCommandInputStore)` (plus simple, pas d'écriture dans l'effet) + `$effect` séparé pour le side-effect focus/loadHistory.

### 3. Tests (extension du canari Fiche 01)

- [x] Ajouter test : `showCommandInputStore.set(true)` → `loadHistory` est appelée (mock), l'input reçoit le focus.
- [x] Ajouter test : `showCommandInputStore.set(false)` → l'input disparaît du DOM.
- [x] Ajouter test : muter `commandHistoryStore` → navigation flèche haut récupère le nouvel historique (simuler keydown).

### 4. Vérification manuelle

- [ ] `wails dev`, appuyer sur le raccourci qui invoque `showCommandInputStore` (généralement `:` ou similaire) → input apparaît et focus OK. Enter une commande, ré-appuyer → l'historique navigable reflète la dernière.

### 5. Commit

- [x] `fix(ui): StatusBar subscribe → $effect, avoid stale history closure`.

## Acceptance

- [x] Tests étendus verts.
- [x] Aucun `.subscribe()` résiduel dans StatusBar.
- [ ] Vérif manuelle OK.

## Status

- [x] Refactor `commandHistory` → `$derived($commandHistoryStore)`
- [x] Refactor `showCommandInputStore` → `$derived` + `$effect` side-effect
- [x] Tests étendus (T6/T7/T8) — 321 tests verts
- [ ] Vérif manuelle (`wails dev`)
- [x] Commit
