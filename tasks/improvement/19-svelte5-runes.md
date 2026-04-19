# 19 — Svelte 5 runes migration

**Goal:** Migrate all 38 Svelte components from Svelte 4 idioms (running on Svelte 5 compatibility layer) to native Svelte 5 runes syntax.

**Depends on:** 13 (App.svelte services), 14 (modal stores), 18 (table modals) — complete frontend refactoring first so you migrate the cleaned-up code, not the old code.

**Impact:** High — enables full Svelte 5 reactivity, smaller bundle, better DevTools, removes deprecation warnings.

## Context

### Current state
- **Svelte 5.55.0** installed, but running in **Svelte 4 compatibility mode**
- **Zero** `$state`, `$derived`, or `$effect` usage anywhere
- All 38 components use `export let` for props (Svelte 4 syntax)
- All stores use `writable()`/`derived()` from `svelte/store` (Svelte 4 API)
- No `svelte.config.js` exists; `vite.config.js` has bare `svelte()` plugin call

### Migration map (Svelte 4 → Svelte 5)
| Svelte 4 | Svelte 5 |
|-----------|----------|
| `export let prop` | `let { prop } = $props()` |
| `$: derived = expr` | `let derived = $derived(expr)` |
| `$: { sideEffect() }` | `$effect(() => { sideEffect() })` |
| `import { writable } from 'svelte/store'` | `$state()` (or keep stores for cross-component) |
| `onMount(() => {...})` | `$effect(() => {...})` (runs on mount) |
| `onDestroy(() => {...})` | Return cleanup from `$effect` |
| `createEventDispatcher()` | Callback props or `$emit` |

### Strategy: incremental, bottom-up
Migrate leaf components first (no children), then work up to App.svelte last. Keep stores for cross-component state (stores still work in Svelte 5). Focus on component-level migration.

## Files touched

- **Edit:** All 38 `.svelte` files under `frontend/src/`
- **Optionally edit:** `frontend/vite.config.js` (add `compilerOptions: { runes: true }` to enforce)
- **Edit:** `frontend/src/stores/*.js` — optionally migrate to `$state` where beneficial

## Tasks

### 1. Enable runes mode globally (optional)

- [ ] Optionally add to `vite.config.js`:
  ```js
  svelte({ compilerOptions: { runes: true } })
  ```
  Or migrate component-by-component using `<svelte:options runes={true} />` per file

### 2. Migrate leaf components (no children)

Start with the simplest components — modals with few props:

- [ ] `HelpModal.svelte` — 3 props → `$props()`
- [ ] `GoToPositionModal.svelte`
- [ ] `WarningModal.svelte`
- [ ] `ImportProgressModal.svelte`
- [ ] `FileImportProgressModal.svelte`
- [ ] `CommandLine.svelte`
- [ ] `StatusBar.svelte`
- [ ] `EpcDisplay.svelte`
- [ ] All `DataTableModal.svelte` (if task 18 is done) or remaining table modals

For each:
- [ ] Replace `export let prop` → `let { prop, ... } = $props()`
- [ ] Replace `$: x = expr` → `let x = $derived(expr)`
- [ ] Replace `$: { sideEffect }` → `$effect(() => { sideEffect })`
- [ ] Replace `onMount`/`onDestroy` → `$effect` with cleanup
- [ ] Replace `createEventDispatcher` → callback props
- [ ] `npm run build` after each file to catch errors early

### 3. Migrate medium components

- [ ] `SearchModal.svelte`
- [ ] `MetModal.svelte`
- [ ] `MetadataModal.svelte`
- [ ] `ExportDatabaseModal.svelte`
- [ ] `FilterLibraryPanel.svelte`
- [ ] `SearchHistoryPanel.svelte`
- [ ] `MatchPanel.svelte`
- [ ] `CollectionPanel.svelte`
- [ ] `TournamentPanel.svelte`
- [ ] `AnkiPanel.svelte`
- [ ] `AnalysisPanel.svelte`
- [ ] `CommentPanel.svelte`

### 4. Migrate complex components

- [ ] `Board.svelte` — careful with Two.js lifecycle
- [ ] `Toolbar.svelte` — 30 props → `$props()` with destructuring
- [ ] `PositionNavigator.svelte`

### 5. Migrate App.svelte

- [ ] This is the hardest — 103 functions, 33 store subscriptions
- [ ] If task 13 (services extraction) is done, App.svelte should be much smaller
- [ ] Replace remaining `$: reactive` statements with `$derived` / `$effect`
- [ ] Replace `onMount` with `$effect`
- [ ] Replace store subscriptions with direct store reads

### 6. Optionally migrate stores to $state

- [ ] Stores that are only used within a single component tree could use `$state` instead
- [ ] Cross-component stores (`uiStore`, `positionStore`, etc.) can stay as `writable()` — Svelte 5 still supports them
- [ ] Evaluate case-by-case; don't migrate stores unless it simplifies code

### 7. Verify

- [ ] `npm run build` succeeds with zero warnings
- [ ] Remove compatibility mode flag if set globally
- [ ] Manual smoke test: all features work
- [ ] Check bundle size — should be equal or smaller

## Acceptance criteria

- [ ] Zero `export let` in any `.svelte` file (all use `$props()`)
- [ ] Zero `$: reactive` statements (all use `$derived` / `$effect`)
- [ ] Zero `onMount`/`onDestroy` (all use `$effect` with cleanup)
- [ ] `npm run build` succeeds with no deprecation warnings
- [ ] All features work identically

## Rollback

`git revert` — large changeset, but purely mechanical. Consider doing this as one commit per batch (leaf → medium → complex → App.svelte) for easier partial rollback.
