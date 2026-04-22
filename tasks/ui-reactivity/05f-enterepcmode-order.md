# 05.f — `positionService.enterEPCMode` : ordre des `set`

**Goal :** Garantir que `statusBarModeStore` passe à `'EPC'` **avant** que `positionStore` soit mis à jour avec `epcPosition`, afin que l'effet EPC (Fiche 05.a) voie le bon mode dès le premier run.

**Depends on :** 05.a (l'effet EPC doit exister et dépendre à la fois de position et de mode).

**Impact :** Élimine une race résiduelle où, selon l'ordre d'exécution des effets Svelte, l'EPC pourrait être « ignoré » au premier set de position car le mode n'est pas encore EPC.

## Context

Extrait actuel (positionService.js:881-915 approx.) :

```js
export function enterEPCMode() {
    if (get(statusBarModeStore) === 'EPC') return;
    // ... création de epcPosition ...
    positionStore.set(epcPosition);
    statusBarModeStore.set('EPC');
}
```

Ordre : position **puis** mode. En Svelte 5, les effets sont exécutés de façon batchée, mais les `get(store)` synchrones dans un effet lisent la valeur courante. Avec l'effet EPC de 05.a :

```js
$effect(() => {
    const value = $positionStore;
    const mode = $statusBarModeStore;
    if (mode === 'EPC' && value) updateEPC(value);
});
```

Si l'effet s'exécute entre les deux `set`, il verra `mode !== 'EPC'` et l'`updateEPC` sera skippé. Svelte batch en général, donc l'ordre dans le batch compte moins ; mais garantir l'ordre éliminé le risque et rend le code auto-documentant.

## Files touched

- **Edit:** `frontend/src/services/positionService.js` (~881-935).
- **New:** `frontend/src/__tests__/positionService.enterEPCMode.test.js`.

## Tasks

### 1. Inverser l'ordre

- [ ] Dans `enterEPCMode` :
  ```js
  statusBarModeStore.set('EPC');
  positionStore.set(epcPosition);
  positionsStore.set([epcPosition]);  // si présent, vérifier la cohérence
  ```
- [ ] Dans `exitEPCMode` : symétriquement, s'assurer que `positionStore.set(savedPosition)` arrive **avant** `statusBarModeStore.set('NORMAL')` si l'effet EPC ne doit pas se re-déclencher à ce moment (sinon pas d'importance).
- [ ] Vérifier dans `updateEPC` qu'il ne lit pas `statusBarModeStore` en interne avec attente implicite (ça ne devrait pas, c'est l'effet du composant qui gate l'appel).

### 2. Test unitaire

- [ ] `positionService.enterEPCMode.test.js` :
  - Mocker les stores pour capturer l'ordre des `set`.
  - Test : appeler `enterEPCMode`, vérifier que le premier `set` est sur `statusBarModeStore`, le second sur `positionStore`.
  - Test : `exitEPCMode` symétrique.

### 3. Vérification manuelle

- [ ] `wails dev` : scénario S1 étendu (position A → EPC → Stats → position B → EPC). L'EPC affiché après retour doit refléter la position B. Déjà couvert par spec Playwright Fiche 02.

### 4. Commit

- [ ] `fix(ui): enterEPCMode sets mode before position to avoid effect race`.

## Acceptance

- [ ] Test unitaire vert.
- [ ] Spec `epc-bar-refreshes-on-return.spec.js` verte (combinée avec 05.a).

## Status

- [ ] Ordre inversé dans enterEPCMode
- [ ] Ordre cohérent dans exitEPCMode
- [ ] Test unitaire
- [ ] Vérif manuelle
- [ ] Commit
