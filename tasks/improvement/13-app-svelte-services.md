# 13 — Extract service modules from App.svelte

**Goal:** Reduce `App.svelte` from 4,866 lines to <500 lines by extracting business logic into service modules. Services are plain JS modules that call Wails bindings and update stores.

**Depends on:** Nothing (frontend-only).

**Impact:** Critical — the god component is the biggest frontend maintainability problem.

## Context

`App.svelte` contains **103 functions** and **4,615 lines of script**. The template is only ~250 lines. Most of the script is imperative logic that:
1. Calls Wails Go bindings (`frontend/wailsjs/go/main/Database.*`)
2. Updates Svelte stores
3. Handles errors (try/catch → `console.error` + status bar)

This logic does not need to be in a Svelte component — it can live in plain JS modules imported by the component.

## Files touched

- **New:** `frontend/src/services/databaseService.js`
- **New:** `frontend/src/services/positionService.js`
- **New:** `frontend/src/services/importService.js`
- **New:** `frontend/src/services/exportService.js`
- **New:** `frontend/src/services/clipboardService.js`
- **New:** `frontend/src/services/sessionService.js`
- **Edit:** `frontend/src/App.svelte` — remove extracted functions, import services

## Tasks

### 1. Create `services/databaseService.js`

- [ ] Extract functions related to database file management:
  - Open database (file dialog → `OpenDatabase()`)
  - Save database (file dialog → `SaveDatabase()`)
  - Create new database
  - Close database
  - Database file drag-and-drop handling
- [ ] Each function calls the Wails binding and updates the relevant store
- [ ] Export as named functions (not a class)

### 2. Create `services/positionService.js`

- [ ] Extract functions related to position management:
  - Load position by index/ID
  - Navigate positions (next, previous, first, last, goto)
  - Mirror position display
  - Delete position
  - Load analysis for current position
  - Load comments for current position
- [ ] Functions update `positionStore`, `analysisStore`, `viewStore` as needed

### 3. Create `services/importService.js`

- [ ] Extract functions related to file import:
  - Import XG/GnuBG/BGF match files
  - Import position files
  - Import from clipboard
  - Progress tracking and cancellation
  - Post-import navigation
- [ ] Functions update `databaseStore` and emit events for progress modals

### 4. Create `services/exportService.js`

- [ ] Extract functions related to export:
  - Export database with options
  - Export collections
  - Export tournaments
  - Export to clipboard (position text, analysis text)

### 5. Create `services/clipboardService.js`

- [ ] Extract clipboard operations:
  - Copy position to clipboard (various formats)
  - Copy analysis to clipboard
  - Copy board image to clipboard
  - Paste/import from clipboard

### 6. Create `services/sessionService.js`

- [ ] Extract session state functions:
  - Save session state on navigation/close
  - Restore session state on DB open
  - Auto-save triggers

### 7. Refactor App.svelte to use services

- [ ] Replace inline function bodies with service calls:
  ```svelte
  <script>
  // Before (100 lines of import logic inline):
  async function handleImportXG(filePath) {
      try { ... 50 lines of import logic ... } catch (e) { ... }
  }

  // After (delegation to service):
  import { importXGMatch } from './services/importService.js';
  async function handleImportXG(filePath) {
      await importXGMatch(filePath);
  }
  ```
- [ ] Work incrementally: extract one service at a time, test after each
- [ ] Keep event handlers and component wiring in App.svelte
- [ ] Keep reactive declarations (`$:`) that wire stores to component state

### 8. Verify nothing breaks

- [ ] `npm run build` succeeds after each service extraction
- [ ] Manual smoke test: open DB, navigate positions, import match, export, search
- [ ] All keyboard shortcuts still work
- [ ] Status bar messages still appear

## Acceptance criteria

- [ ] `App.svelte` ≤ 500 lines
- [ ] 6 service modules created in `frontend/src/services/`
- [ ] No functionality regression (all features work)
- [ ] `npm run build` succeeds
- [ ] Each service has a clear, single responsibility

## Rollback

`git revert` — pure refactoring, no behavior changes.
