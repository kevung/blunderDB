# 15 — Install vitest + test `commandProcessor.js`

**Goal:** Establish frontend test infrastructure and write tests for the command processor — the highest-value frontend test target (pure function, complex parsing logic, 322 lines).

**Depends on:** Nothing.

**Impact:** High — zero frontend tests exist today.

## Context

- `frontend/package.json` has zero test dependencies
- `commandProcessor.js` (322 lines) parses the in-app command line — pure function with complex logic, no DOM needed
- Stores in `frontend/src/stores/` contain derived computations that could be tested
- No vitest, jest, or testing-library configured

## Files touched

- **Edit:** `frontend/package.json` — add dev dependencies and test script
- **New:** `frontend/vitest.config.js`
- **New:** `frontend/src/__tests__/commandProcessor.test.js`
- **New:** `frontend/src/__tests__/stores/` (optional, for store tests)

## Tasks

### 1. Install test infrastructure

- [x] Install vitest and related packages:
  ```bash
  cd frontend
  npm install -D vitest @testing-library/svelte jsdom
  ```
- [x] Add test script to `package.json`:
  ```json
  "scripts": {
      "dev": "vite",
      "build": "vite build",
      "preview": "vite preview",
      "test": "vitest run",
      "test:watch": "vitest"
  }
  ```

### 2. Configure vitest

- [x] Create `frontend/vitest.config.js`:
  ```js
  import { defineConfig } from 'vitest/config';
  import { svelte } from '@sveltejs/vite-plugin-svelte';

  export default defineConfig({
      plugins: [svelte({ hot: !process.env.VITEST })],
      test: {
          environment: 'jsdom',
          include: ['src/**/*.{test,spec}.{js,ts}'],
          globals: true,
      },
  });
  ```

### 3. Test `commandProcessor.js` — core parsing

- [x] Read `commandProcessor.js` to understand its API:
  - What does it export? (likely a `processCommand(input)` function)
  - What are the command formats? (filter expressions, search queries, collection commands, etc.)
- [x] Create `frontend/src/__tests__/commandProcessor.test.js`
- [x] Test basic command recognition:
  ```js
  describe('commandProcessor', () => {
      test('parses empty input', () => { ... });
      test('parses help command', () => { ... });
      test('parses search command', () => { ... });
  });
  ```

### 4. Test filter expression parsing

- [x] Test pip count filters: `pip>30`, `pip<10`, `pip5,20`
- [x] Test win rate filters: `win>50`, `win<30`
- [x] Test gammon rate filters: `gam>10`
- [x] Test equity filters: `eq>0.5`, `eq-0.3,0.5`
- [x] Test dice filters: `dice 31`, `dice 66`
- [x] Test move error filters: `E>0.05`
- [x] Test combined filters: multiple filters in one command

### 5. Test collection/deck commands

- [ ] Test collection commands: create, add, remove, list
- [ ] Test deck/Anki commands: create, sync, review
- [ ] Test export commands

### 6. Test edge cases

- [x] Empty input
- [x] Unknown command
- [x] Malformed filter expressions (e.g. `pip>abc`)
- [x] Extra whitespace, case sensitivity
- [x] Special characters in search text

### 7. Mock Wails bindings

- [x] `commandProcessor.js` likely calls Wails bindings via imported functions
- [x] Mock these with vitest mocking:
  ```js
  vi.mock('../../wailsjs/go/main/Database', () => ({
      LoadPositionsByFilters: vi.fn().mockResolvedValue([]),
      // ...
  }));
  ```
- [x] Or if the processor returns a parsed command object (pure function), no mocking needed

### 8. Optional: test store logic

- [ ] Test `viewStore.js` derived computations
- [ ] Test `uiStore.js` derived stores (before task 14 refactors them)
- [ ] Test `positionStore.js` helper functions

### 9. Add to CI (if task 01 adds frontend check)

- [ ] Optionally add `npm test` to CI workflow:
  ```yaml
  - name: Frontend tests
    working-directory: frontend
    run: npm ci && npm test
  ```

## Acceptance criteria

- [x] `npm test` command exists and runs vitest
- [x] ≥20 tests for `commandProcessor.js` covering all major command types
- [x] Tests pass: `cd frontend && npm test`
- [x] Tests run without a browser or Wails runtime (jsdom environment + mocks)
- [x] Foundation ready for adding more frontend tests

## Rollback

`git revert` — additive only (new files + `package.json` dev dependency changes).
