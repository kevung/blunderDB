# Tests frontend — Guide

## Lancer les tests

```bash
cd frontend
npm test           # vitest run (single pass)
npm run test:watch # vitest (mode watch)
```

## Structure

```
src/__tests__/
├── StatusBar.reactivity.test.js  # Canari de réactivité (composant monté)
├── StatsPanel.test.js            # Tests store/logique
├── commandProcessor.test.js      # Tests pure fonctions
└── ...

src/__mocks__/
└── wails.js   # create{Database,App,Config,Runtime}Mock() — mock complet, réfléchi sur wailsjs/go/**
```

## Pattern : mount + mutate store + assert DOM

```js
// 1. Mocker les bindings Wails AVANT l'import du composant
vi.mock('../../../wailsjs/go/database/Database.js', () => ({
    LoadCommandHistory: vi.fn(() => Promise.resolve([]))
}));

// 2. Importer stores et composant
import { statusBarTextStore } from '../stores/uiStore.js';
import MyComponent from '../components/MyComponent.svelte';
import { render, screen, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';

// 3. Réinitialiser les stores dans beforeEach
beforeEach(() => statusBarTextStore.set(''));
afterEach(cleanup); // libérer le DOM après chaque test

// 4. Tester
test('réactivité', async () => {
    render(MyComponent);
    statusBarTextStore.set('Hello');
    await tick(); // laisser Svelte propager
    expect(screen.getByText('Hello')).toBeInTheDocument();
});
```

## Mocks Wails

Deux approches selon le besoin :

### `vi.mock` à la main (un test qui n'exerce que quelques fonctions)

```js
vi.mock('../../../wailsjs/go/database/Database.js', () => ({
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn()
}));
```

Vitest hisse automatiquement `vi.mock` en tête de fichier — déclarer avant ou après les imports ne change rien.

### `src/__mocks__/wails.js` (mock complet, un vi.fn() par fonction exportée)

Pour un composant qui touche beaucoup de bindings sans que le test se soucie
de chacune d'elles (`CollectionPanel.test.js`, `databaseService.test.js`) :

```js
vi.mock('../../wailsjs/go/database/Database.js', async () => {
    const { createDatabaseMock } = await import('../__mocks__/wails.js');
    return createDatabaseMock({ GetAllCollections: vi.fn().mockResolvedValue(SAMPLE) });
});
```

`createDatabaseMock`/`createAppMock`/`createConfigMock`/`createRuntimeMock` lisent
la liste des fonctions exportées par le fichier généré réel — un renommage côté
Go ne peut donc pas laisser un mock partiel silencieusement en retard ; voir la
docstring de `src/__mocks__/wails.js` et `wailsMock.sync.test.js`.

## Règle

> Tout nouveau composant ajouté au périmètre de réactivité doit avoir au moins
> un test vérifiant qu'une mutation de store se reflète dans le DOM
> (`store.set(x)` → `await tick()` → `expect(dom).toContain(x)`).
