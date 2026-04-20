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

### 1. Enable runes mode globally

- [x] Added to `vite.config.js`:
  ```js
  svelte({ compilerOptions: { runes: true } })
  ```

### 2–5. Migrate all components

All 30 `.svelte` files migrated in a single pass via automated script + manual fixes:

- [x] `export let` → `$props()` destructuring (all components with props)
- [x] `$: x = expr` → `let x = $derived(expr)` or `$derived.by(() => {...})` for multi-line
- [x] `$: { sideEffect }` and `$: if (cond)` → `$effect(() => { ... })`
- [x] `on:event` → `onevent` (onclick, onkeydown, onblur, etc.)
- [x] `on:event|modifier={handler}` → inline handlers with `stopPropagation()`/`preventDefault()`
- [x] `a11y-*` ignore comments → `a11y_*` (underscore format for Svelte 5)

**Not migrated (intentionally):**
- `onMount`/`onDestroy` kept where they manage DOM event listeners (cleanup semantics differ)
- `createEventDispatcher` not present (already using callback props)
- Svelte stores kept as `writable()`/`derived()` for cross-component state (still fully supported in Svelte 5)

### 6. Verify

- [x] `npm run build` succeeds (warnings only, no errors)
- [x] `npm test` — 125 tests pass
- [x] `npx eslint` — only 3 pre-existing unused-prop warnings (not introduced by migration)
- [x] `go test ./...` — all backend tests pass

## Acceptance criteria

- [x] Zero `export let` in any `.svelte` file (all use `$props()`)
- [x] Zero `$: reactive` statements (all use `$derived` / `$effect`)
- [ ] Zero `onMount`/`onDestroy` (kept intentionally — cleanup semantics)
- [x] `npm run build` succeeds with no errors
- [x] All features work identically (tests pass)

## Rollback

`git revert` — large changeset, but purely mechanical. Consider doing this as one commit per batch (leaf → medium → complex → App.svelte) for easier partial rollback.
