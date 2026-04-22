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
├── helpers/
│   └── wailsMock.js          # Helper pour mocker window.go.main.Database
├── StatusBar.reactivity.test.js  # Canari de réactivité (composant monté)
├── StatsPanel.test.js            # Tests store/logique
├── commandProcessor.test.js      # Tests pure fonctions
└── ...
```

## Pattern : mount + mutate store + assert DOM

```js
// 1. Mocker les bindings Wails AVANT l'import du composant
vi.mock('../../../wailsjs/go/main/Database.js', () => ({
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
}));

// 2. Importer stores et composant
import { statusBarTextStore } from '../stores/uiStore.js';
import MyComponent from '../components/MyComponent.svelte';
import { render, screen, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';

// 3. Réinitialiser les stores dans beforeEach
beforeEach(() => statusBarTextStore.set(''));
afterEach(cleanup);  // libérer le DOM après chaque test

// 4. Tester
test('réactivité', async () => {
    render(MyComponent);
    statusBarTextStore.set('Hello');
    await tick();                          // laisser Svelte propager
    expect(screen.getByText('Hello')).toBeInTheDocument();
});
```

## Mocks Wails

Deux approches selon le besoin :

### `vi.mock` (recommandé, par fichier de test)

```js
vi.mock('../../../wailsjs/go/main/Database.js', () => ({
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(),
}));
```

Vitest hisse automatiquement `vi.mock` en tête de fichier — déclarer avant ou après les imports ne change rien.

### `installWailsMock` (overrides dynamiques par test)

```js
import { installWailsMock, uninstallWailsMock } from './helpers/wailsMock.js';

beforeEach(() => installWailsMock({ LoadCommandHistory: () => Promise.resolve(['cmd1']) }));
afterEach(uninstallWailsMock);
```

Utile lorsque chaque test a besoin d'une réponse différente.

## Règle

> Tout nouveau composant ajouté au périmètre de réactivité doit avoir au moins
> un test vérifiant qu'une mutation de store se reflète dans le DOM
> (`store.set(x)` → `await tick()` → `expect(dom).toContain(x)`).
