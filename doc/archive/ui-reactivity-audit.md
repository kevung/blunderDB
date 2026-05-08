# UI reactivity audit

Produit par la Fiche 04. Date : 2026-04-22. Branche : `ui-reactivity`.

## Méthodologie

Grep systématique de `.subscribe(`, `onMount`, closures stales (`let` modifié dans subscribe), `$effect(` et `get(store)` sur les fichiers du scope :

- `frontend/src/App.svelte`
- `frontend/src/components/TabbedPanel.svelte`
- `frontend/src/components/MatchPanel.svelte`
- `frontend/src/components/stats/StatsPanel.svelte`
- `frontend/src/components/EPCPanel.svelte`
- `frontend/src/components/StatusBar.svelte`
- `frontend/src/services/positionService.js` (fonction `enterEPCMode`)

Chaque occurrence est classée : **(a)** à convertir en `$effect`, **(b)** à garder, **(c)** à supprimer.

---

## Synthèse

| # | Fichier | Ligne | Pattern | Sévérité | Correction | Fiche |
|---|---|---|---|---|---|---|
| 1 | App.svelte | 128 | `positionStore.subscribe` lisant `$statusBarModeStore` non tracké | 🟡 moyen | (a) `$effect` sur `$positionStore` + `$statusBarModeStore` | 05.a |
| 2 | App.svelte | 132 | `positionReloadTriggerStore.subscribe(async …)` lit `$databasePathStore` | 🟢 bas | (a) `$effect` ou garder avec note | 05.a |
| 3 | App.svelte | 136 | `positionsStore.subscribe` → `positions` (plain `let`, non `$state`) | 🟢 bas | (b) fonctionnel, pas de template direct ; documenter | 05.a |
| 4 | App.svelte | 189 | `currentPositionIndexStore.subscribe(async …)` — race avec `positions` | 🟢 bas | (b) garder ; debounce déjà en place | 05.a |
| 5 | App.svelte | 198 | `activeTabStore.subscribe` : gestion `matches` seulement, pas `stats`/`tournaments`/`collections`/`anki` | 🔴 critique | (a) compléter ou refactoriser | 05.a |
| 6 | MatchPanel | 57 | `lastVisitedMatchStore.subscribe` sans `onDestroy` | 🟡 moyen | (a) `$effect` ou ajouter cleanup | 05.b |
| 7 | MatchPanel | 61 | `tournamentsStore.subscribe` sans `onDestroy` | 🟡 moyen | (a) `$effect` ou ajouter cleanup | 05.b |
| 8 | MatchPanel | 66 | `matchPanelRefreshTriggerStore.subscribe(async …)` sans `onDestroy` | 🟡 moyen | (a) `$effect` + cleanup | 05.b |
| 9 | MatchPanel | 77 | `openPanels.subscribe(async …)` sans `onDestroy` — async race | 🟠 grave | (a) `$effect` + cleanup | 05.b |
| 10 | MatchPanel | 526 | `matchContextStore.subscribe` one-shot (`unsub()` immédiat) | 🟢 bas | (c) remplacer par `get(matchContextStore)` | 05.b |
| 11 | StatsPanel | 16–25 | `onMount` + `.subscribe()` → **double appel** `refreshStats` à l'init | 🔴 critique | (a) `$effect` — cause principale des transitions lentes | 05.c |
| 12 | StatusBar | 26 | `commandHistoryStore.subscribe` sans `onDestroy` | 🟡 moyen | (a) `$effect` ou ajouter cleanup | 05.d |
| 13 | StatusBar | 28 | `showCommandInputStore.subscribe(async …)` sans `onDestroy` | 🟡 moyen | (a) `$effect` ou ajouter cleanup | 05.d |
| 14 | TabbedPanel | — | `{:else if}` détruit/recrée `StatsPanel` à chaque visite | 🟡 moyen | documenter la règle keep-alive | 05.e |
| 15 | positionService | 881–913 | `enterEPCMode` : ordre `statusBarModeStore.set` avant `positionStore.set` | 🟢 bas | (b) ordre correct ; documenter | 05.f |

---

## Détails par fichier

### App.svelte

#### #1 — L128 : `positionStore.subscribe` + `$statusBarModeStore` non tracké

```js
positionStore.subscribe((value) => {
    if ($statusBarModeStore === 'EPC' && value) updateEPC(value);
});
```

**Diagnostic.** Le callback ne se déclenche que lorsque `positionStore` change. Si `statusBarModeStore` passe à `'EPC'` sans que `positionStore` ait changé (cas typique : l'utilisateur clique sur l'onglet EPC depuis la vue Analysis avec la même position), `updateEPC` n'est jamais appelé. L'EPC de la barre d'état reste à la valeur précédente.

**Correction proposée.**
```js
$effect(() => {
    if ($statusBarModeStore === 'EPC' && $positionStore) updateEPC($positionStore);
});
```
Les deux stores sont maintenant des dépendances trackées ; l'effet se relance dès que l'un ou l'autre change.

---

#### #2 — L132 : `positionReloadTriggerStore.subscribe(async …)`

```js
positionReloadTriggerStore.subscribe(async () => {
    if ($databasePathStore) await loadAllPositions();
});
```

**Diagnostic.** `$databasePathStore` est lu dans un callback async — si le store change après le déclenchement mais avant la résolution du `await`, la valeur est obsolète. Sévérité basse car `loadAllPositions` utilise `get(databasePathStore)` en interne.

**Correction proposée.** Convertir en `$effect` pour profiter du tracking automatique, ou documenter le risque et garder tel quel.

---

#### #3 — L136 : `positionsStore.subscribe` → `positions` (plain `let`)

```js
positionsStore.subscribe((value) => {
    positions = Array.isArray(value) ? value : [];
    ...
});
```

**Diagnostic.** `positions` est déclaré `let positions = []` (non `$state`). Le changement de `positions` n'est donc pas réactif pour le template. Cependant, `positions` n'est **pas** utilisé directement dans le template de App.svelte — il n'est lu que dans d'autres callbacks (L189). Ce n'est pas un bug de rendu, seulement un pattern fragile.

**Correction proposée.** Documenter ; aucune modification urgente.

---

#### #4 — L189 : `currentPositionIndexStore.subscribe(async …)`

```js
currentPositionIndexStore.subscribe(async (value) => {
    currentPositionIndex = value;
    if (positions.length > 0 && currentPositionIndex >= 0 && currentPositionIndex < positions.length) {
        await showPosition(positions[currentPositionIndex]);
        ...
    }
});
```

**Diagnostic.** Race condition théorique : si `positionsStore` change entre le déclenchement du callback et la résolution de `await showPosition`, `positions` sera mise à jour synchroniquement par son propre subscribe, mais `currentPositionIndex` pointera sur l'ancien tableau. Cas rare en pratique car les deux stores sont toujours mis à jour ensemble dans `loadAllPositions`.

**Correction proposée.** Garder ; debounce/save session déjà en place. Documenter.

---

#### #5 — L198 : `activeTabStore.subscribe` — onglets non gérés (**critique**)

```js
activeTabStore.subscribe((tab) => {
    ...
    if (tab === 'matches') openPanel(PANEL.MATCH);
    else closePanel(PANEL.MATCH);
});
```

**Diagnostic.** La branche `else` appelle `closePanel(PANEL.MATCH)` pour **tous** les onglets autres que `matches`, ce qui est correct pour fermer le panneau match. Mais il n'existe aucune action spécifique pour `stats`, `tournaments`, `collections`, `anki`. Conséquences :

- La condition `if/else if` précédente (`prevTab === 'epc' → exitEPCMode`) fonctionne, mais la chaîne de logique suppose qu'il n'y a que quelques onglets spéciaux. Si un onglet futur nécessite une action, il faudra modifier ce subscribe.
- La vraie sévérité critique concerne le fait que ce subscribe **utilise `.subscribe()` au lieu de `$effect`**, ce qui mélange Svelte 4 et Svelte 5. En Svelte 5, `$statusBarModeStore`, `$databasePathStore` et les autres runes lus dans le callback ne sont pas trackés — si ces stores changent, le callback ne se re-déclenche pas (ce n'est pas le but ici, mais c'est un anti-pattern à risque).

**Correction proposée.** Convertir en `$effect` lisant `$activeTabStore` et les autres stores nécessaires, ou documenter clairement pourquoi `.subscribe()` est acceptable ici.

---

### MatchPanel.svelte

#### #6–9 — Subscribes sans `onDestroy` (L57, L61, L66, L77)

MatchPanel est monté/démonté par `TabbedPanel` via `{:else if $activeTabStore === 'matches'}`. À chaque démontage, les quatre subscribes root-level restent actifs (fuite mémoire). La prochaine fois que MatchPanel est monté, de nouveaux subscribes s'ajoutent aux anciens — à terme, des callbacks multiples s'exécutent.

**Détail #9 (L77) :** `openPanels.subscribe(async …)` — async race :

```js
openPanels.subscribe(async (value) => {
    const wasVisible = visible;
    visible = value.has(PANEL.MATCH);
    if (visible && !wasVisible) {
        await loadMatches();
        if (lastVisitedMatch && lastVisitedMatch.matchID) { ... }
    }
});
```

Si `openPanels` change une seconde fois pendant `await loadMatches()`, `wasVisible` est obsolète et `visible` a été mis à jour entre-temps. Ce pattern async dans un subscribe est à risque.

**Correction proposée.** Convertir tous en `$effect` (qui s'arrête automatiquement au démontage) ou stocker les retours de subscribe et les appeler dans `onDestroy`.

#### #10 — L526 : one-shot subscribe (`swapMatchPlayers`)

```js
const unsub = matchContextStore.subscribe((v) => (currentContext = v));
unsub();
```

**Diagnostic.** Pattern équivalent à `get(matchContextStore)`. Code smell, pas un bug.

**Correction proposée.** Remplacer par `import { get } from 'svelte/store'; const currentContext = get(matchContextStore);`.

---

### StatsPanel.svelte

#### #11 — L16–25 : `onMount` + `.subscribe()` → double appel `refreshStats` (**critique**)

```js
onMount(() => {
    logger.perf('StatsPanel:refreshStats', () => refreshStats($statsFilterStore)); // appel 1
    unsubscribeFilter = statsFilterStore.subscribe((filter) => {
        logger.perf('StatsPanel:refreshStats', () => refreshStats(filter));        // appel 2 (immédiat)
    });
});
```

**Diagnostic.** `.subscribe()` en Svelte appelle **toujours** le callback immédiatement avec la valeur courante. L'appel explicite à `refreshStats($statsFilterStore)` juste avant le `.subscribe()` est donc redondant : `refreshStats` est exécuté **deux fois** à chaque montage de StatsPanel. Si `refreshStats` est coûteuse (requête DB), cela double le temps de chargement à chaque changement d'onglet vers Stats — c'est la **cause principale** du symptôme S2 (transitions bloquées incluant Stats).

En outre, `onMount` retarde l'abonnement au premier rendu (`onMount` s'exécute après le premier rendu DOM), alors qu'un `$effect` démarrerait dès l'initialisation du composant.

**Correction proposée.**

```js
$effect(() => {
    logger.perf('StatsPanel:refreshStats', () => refreshStats($statsFilterStore));
});
```

Un seul appel, tracké, déclenché immédiatement à chaque changement de `$statsFilterStore`. Pas besoin d'`onDestroy`.

---

### StatusBar.svelte

#### #12 — L26 : `commandHistoryStore.subscribe` sans `onDestroy`

```js
commandHistoryStore.subscribe((value) => (commandHistory = value));
```

**Diagnostic.** `commandHistory` est `let commandHistory = []` (plain `let`). Il est utilisé dans `handleKeyDown` (non dans le template), donc l'absence de `$state` n'est pas un bug de rendu. Mais l'absence d'`onDestroy` est une fuite si StatusBar est jamais démonté.

**Correction proposée.** Ajouter `onDestroy` ou utiliser `$derived($commandHistoryStore)`.

---

#### #13 — L28 : `showCommandInputStore.subscribe(async …)` sans `onDestroy`

```js
showCommandInputStore.subscribe(async (value) => {
    showInput = value;
    if (value) { await loadHistory(); await tick(); inputEl?.focus(); }
});
```

**Diagnostic.** `showInput = $state(false)` — l'assignment fonctionne. Mais le callback est async : si `showCommandInputStore` change à nouveau pendant `await loadHistory()`, `showInput` aura déjà été mis à `false`, mais le `tick()` et `focus()` du premier callback s'exécuteront quand même. Pas d'`onDestroy`.

**Correction proposée.** Ajouter cleanup ; ou convertir en `$effect` avec AbortController pour annuler le callback précédent.

---

### TabbedPanel.svelte

#### #14 — Pattern `{:else if}` : détruit/recrée `StatsPanel` à chaque visite

```html
{:else if $activeTabStore === 'stats'}
    <StatsPanel />
```

**Diagnostic.** Chaque switch vers l'onglet Stats **démonte** le composant précédent et **monte** StatsPanel. `onMount` se déclenche, `refreshStats` est appelé (deux fois, cf. #11). La visite précédente ne peut pas être mise en cache.

**Correction proposée.** Documenter la règle keep-alive dans la Fiche 05.e. Option principale : utiliser `{#if}` avec `hidden` (CSS `display:none`) pour les onglets qui gagnent à être gardés actifs (Stats, Analysis, EPC). Alternative : composant `<KeepAlive>`.

---

### positionService.js — `enterEPCMode`

#### #15 — L881–913 : ordre des `set`

```js
statusBarModeStore.set('EPC');           // 1
closePanel(PANEL.COMMENT);               // 2
closePanel(PANEL.ANALYSIS);             // 3
positionsStore.set([epcPosition]);       // 4
positionStore.set(epcPosition);          // 5
currentPositionIndexStore.set(0);        // 6
```

**Diagnostic.** Les stores Svelte sont synchrones : à l'étape 1, les subscribers de `statusBarModeStore` s'exécutent. `positionStore.subscribe` dans App.svelte (L128) teste `$statusBarModeStore === 'EPC'` — mais ce subscribe écoute `positionStore`, pas `statusBarModeStore`, donc il ne se déclenche pas ici. Le vrai déclenchement d'`updateEPC` intervient à l'étape 5, quand `$statusBarModeStore` est déjà `'EPC'`. L'ordre est donc correct.

**Correction proposée.** Documenter l'ordre intentionnel. Pas de modification nécessaire.

---

## EPCPanel.svelte

Utilise exclusivement des runes Svelte 5 :
```js
let isActive = $derived($statusBarModeStore === 'EPC');
let data = $derived($epcDataStore);
```
Aucun `.subscribe()`, aucun `onMount`, aucun `$effect`. **Aucune correction nécessaire.**

---

## Nouvelles fiches créées

Aucun nouveau problème hors périmètre identifié. Toutes les occurrences mappent aux fiches 05.a–05.f existantes.

## Mapping audit ↔ fiches 05.*

| Fiche | Occurrences |
|---|---|
| 05.a | #1, #2, #3, #4, #5 |
| 05.b | #6, #7, #8, #9, #10 |
| 05.c | #11 |
| 05.d | #12, #13 |
| 05.e | #14 |
| 05.f | #15 |
