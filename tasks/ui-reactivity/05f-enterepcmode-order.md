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

**Note 2026-04-27 :** le code de `enterEPCMode` avait déjà le bon ordonnancement
(`statusBarModeStore.set('EPC')` avant `positionStore.set(epcPosition)`) lors de
l'implémentation de cette fiche — la correction avait été appliquée en Fiche 05.a.
De même, `exitEPCMode` positionne `statusBarModeStore.set('NORMAL')` avant de
restaurer la position (comportement correct : évite que l'effet EPC se
re-déclenche). Aucun changement de code requis.

- [x] Dans `enterEPCMode` : `statusBarModeStore.set('EPC')` avant `positionStore.set(epcPosition)` — **déjà en place**.
- [x] Dans `exitEPCMode` : `statusBarModeStore.set('NORMAL')` avant `positionStore.set(savedPosition)` — **déjà en place**.
- [x] Vérifier dans `updateEPC` qu'il ne lit pas `statusBarModeStore` en interne — confirmé, l'appel est gaté par l'effet du composant.

### 2. Test unitaire

- [x] `positionService.enterEPCMode.test.js` créé avec 5 tests :
  - T1 — `statusBarModeStore(EPC)` appelé avant `positionStore` dans `enterEPCMode`.
  - T2 — `statusBarModeStore(EPC)` appelé avant `positionsStore` dans `enterEPCMode`.
  - T3 — idempotent : second appel ignoré si déjà en mode EPC.
  - T4 — `statusBarModeStore(NORMAL)` appelé avant `positionStore` dans `exitEPCMode`.
  - T5 — `exitEPCMode` ignoré si mode ≠ EPC.
  - Stratégie : vrais stores Svelte + `vi.spyOn` call-through pour capturer l'ordre.

### 3. Vérification manuelle

- [x] Code vérifié (ordonnancement correct depuis 05.a). Scénario S1 étendu couvert par spec Playwright Fiche 02.

### 4. Commit

- [x] `test(ui): positionService.enterEPCMode order of set calls (T1-T5, 326 tests verts)`.

## Acceptance

- [x] Tests unitaires verts (5/5).
- [x] Spec `epc-bar-refreshes-on-return.spec.js` verte (combinée avec 05.a).

## Status

- [x] Ordre confirmé dans enterEPCMode (déjà correct depuis 05.a)
- [x] Ordre confirmé dans exitEPCMode (déjà correct depuis 05.a)
- [x] Test unitaire (5 tests)
- [x] Vérif manuelle (code review)
- [x] Commit
