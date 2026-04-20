# 17 — Strip/gate console.log calls

**Goal:** Remove or gate the 156+ `console.*` calls in `App.svelte` and other frontend files behind a debug flag, so production builds are clean.

**Depends on:** 16 (ESLint configured — can enforce `no-console` rule after cleanup).

**Impact:** Medium — reduces noise in browser DevTools, small perf win.

## Context

- `App.svelte` alone has 156 `console.log`, `console.warn`, `console.error` calls
- Other components and stores likely add more
- These were useful during development but are noise in production
- Some `console.error` calls in catch blocks may be worth keeping

## Files touched

- **New:** `frontend/src/utils/logger.js` (simple debug logger)
- **Edit:** `frontend/src/App.svelte` — replace `console.*` calls
- **Edit:** `frontend/src/components/*.svelte` — replace `console.*` calls
- **Edit:** `frontend/src/stores/*.js` — replace `console.*` calls
- **Edit:** `frontend/eslint.config.js` — add `no-console` rule

## Tasks

### 1. Create a lightweight logger

- [x] Create `frontend/src/utils/logger.js`:
  ```js
  const isDev = import.meta.env.DEV;

  export const logger = {
      log: (...args) => isDev && console.log(...args),
      warn: (...args) => isDev && console.warn(...args),
      error: (...args) => console.error(...args), // always log errors
      debug: (...args) => isDev && console.debug(...args),
  };
  ```
- [x] `import.meta.env.DEV` is `true` during `vite dev`, `false` after `vite build`

### 2. Categorize console calls

- [x] Audit all `console.*` calls in `App.svelte`:
  - **Remove:** Pure debug noise (`console.log('position loaded', ...)`)
  - **Keep as error:** Genuine error logging in catch blocks → `logger.error(...)`
  - **Gate behind dev:** Diagnostic info useful during development → `logger.log(...)`
- [x] Same audit for components and stores

### 3. Replace in App.svelte

- [x] Add `import { logger } from './utils/logger.js';` at top
- [x] Replace `console.log(...)` → `logger.log(...)` (or remove if pure noise)
- [x] Replace `console.warn(...)` → `logger.warn(...)`
- [x] Keep `console.error(...)` in catch blocks as `logger.error(...)` (always logs)
- [x] Work in batches to keep diffs reviewable

### 4. Replace in other files

- [x] Process `frontend/src/components/*.svelte`
- [x] Process `frontend/src/stores/*.js`
- [x] Process `frontend/src/commandProcessor.js`
- [x] Process `frontend/src/services/*.js` (if they exist from task 13)

### 5. Add ESLint rule

- [x] Add `no-console` rule to `eslint.config.js`:
  ```js
  {
      rules: {
          'no-console': ['warn', { allow: ['error'] }],
      },
  }
  ```
- [x] Or if using the logger everywhere, make it an error:
  ```js
  'no-console': 'error',
  ```
  and add `// eslint-disable-next-line no-console` only in `logger.js`

### 6. Verify

- [x] `npm run build` succeeds
- [x] `npm run lint` passes (no `console.*` lint errors)
- [ ] Open app in dev mode → logger messages appear (manual)
- [ ] Production build → no console messages in browser DevTools (manual)

## Acceptance criteria

- [x] Zero direct `console.log/warn` calls in source files (outside `logger.js`)
- [x] `logger.error()` still works in production for genuine errors
- [x] `no-console` ESLint rule prevents regressions
- [x] `npm run build` succeeds

## Rollback

`git revert` — mechanical replacement, no behavior changes.
