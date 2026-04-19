# 16 — Add ESLint/Prettier to frontend

**Goal:** Configure ESLint and Prettier for the frontend codebase to catch bugs and enforce consistent style.

**Depends on:** Nothing.

**Impact:** Medium — prevents common JS errors, makes diffs cleaner.

## Context

- Zero linting configured in `frontend/`
- 156 `console.log/warn/error` calls in `App.svelte` alone
- Mix of single/double quotes, inconsistent semicolons
- Svelte 5 (running in Svelte 4 compat mode) — need Svelte ESLint plugin

## Files touched

- **Edit:** `frontend/package.json` — add devDependencies and lint scripts
- **New:** `frontend/eslint.config.js` (flat config format)
- **New:** `frontend/.prettierrc` (or `.prettierrc.json`)
- **New:** `frontend/.prettierignore`
- **Edit:** `frontend/src/**/*.{js,svelte}` — auto-fix formatting (bulk commit)

## Tasks

### 1. Install ESLint + Prettier

- [ ] Install packages:
  ```bash
  cd frontend
  npm install -D eslint prettier eslint-config-prettier \
      eslint-plugin-svelte svelte-eslint-parser \
      @eslint/js globals
  ```

### 2. Configure ESLint (flat config)

- [ ] Create `frontend/eslint.config.js`:
  ```js
  import js from '@eslint/js';
  import svelte from 'eslint-plugin-svelte';
  import prettier from 'eslint-config-prettier';
  import globals from 'globals';

  export default [
      js.configs.recommended,
      ...svelte.configs['flat/recommended'],
      prettier,
      ...svelte.configs['flat/prettier'],
      {
          languageOptions: {
              globals: {
                  ...globals.browser,
                  ...globals.node,
              },
          },
      },
      {
          ignores: ['wailsjs/**', 'dist/**', 'node_modules/**'],
      },
  ];
  ```

### 3. Configure Prettier

- [ ] Create `frontend/.prettierrc`:
  ```json
  {
      "singleQuote": true,
      "tabWidth": 4,
      "semi": true,
      "trailingComma": "all",
      "printWidth": 100,
      "plugins": ["prettier-plugin-svelte"],
      "overrides": [
          { "files": "*.svelte", "options": { "parser": "svelte" } }
      ]
  }
  ```
  (Match existing code style — inspect a few files first to determine current conventions)
- [ ] Install Svelte Prettier plugin:
  ```bash
  npm install -D prettier-plugin-svelte
  ```
- [ ] Create `frontend/.prettierignore`:
  ```
  wailsjs/
  dist/
  node_modules/
  ```

### 4. Add npm scripts

- [ ] Add to `frontend/package.json` scripts:
  ```json
  "lint": "eslint src/",
  "lint:fix": "eslint src/ --fix",
  "format": "prettier --write src/",
  "format:check": "prettier --check src/"
  ```

### 5. Run initial lint, triage errors

- [ ] Run `npx eslint src/` to see all errors
- [ ] Fix critical errors that indicate real bugs (unused vars used in template are OK in Svelte)
- [ ] For non-critical issues, add targeted `eslint-disable` comments only where necessary
- [ ] Do NOT auto-fix everything blindly — review the diff

### 6. Run initial format

- [ ] Run `npx prettier --write src/` to normalize formatting
- [ ] Review the diff, ensure no logic changes
- [ ] Commit formatting as a **separate commit** from the config setup (cleaner history)

### 7. Optional: Add to CI

- [ ] Add lint check to CI:
  ```yaml
  - name: Frontend lint
    working-directory: frontend
    run: |
        npm ci
        npm run lint
        npm run format:check
  ```

## Acceptance criteria

- [ ] `npm run lint` runs ESLint across `src/` with zero errors (warnings OK)
- [ ] `npm run format:check` passes (all files formatted)
- [ ] Config ignores `wailsjs/` directory (auto-generated)
- [ ] `.svelte` files are linted and formatted
- [ ] No logic changes introduced by formatting — verified by reviewing diff

## Rollback

`git revert` — two commits: (1) config+dependencies, (2) formatting changes. Revert both.
