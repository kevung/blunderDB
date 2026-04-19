# 14 — Consolidate modal visibility management

**Goal:** Replace 20+ individual boolean stores for modal/panel visibility with a unified system that's easy to extend.

**Depends on:** 13 (App.svelte services — reduces the surface area of store subscriptions to update).

**Impact:** High — eliminates fragile derived stores and 33 manual subscriptions in App.svelte.

## Context

### Current state (`uiStore.js`, 252 lines)

**12 modal stores** (exclusive — only one modal at a time):
```js
showSearchModalStore, showMetModalStore, showTakePoint2LastModalStore,
showTakePoint2LiveModalStore, showTakePoint4LastModalStore, showTakePoint4LiveModalStore,
showGammonValue1ModalStore, showGammonValue2ModalStore, showGammonValue4ModalStore,
showWarningModalStore, showMetadataModalStore, showGoToPositionModalStore,
showExportDatabaseModalStore, showTakePoint2ModalStore, showTakePoint4ModalStore,
showHelpStore, showCommandStore
```

**7 panel stores** (can be open simultaneously):
```js
showAnalysisStore, showCommentStore, showFilterLibraryPanelStore,
showSearchHistoryPanelStore, showMatchPanelStore, showCollectionPanelStore,
showTournamentPanelStore
```

**3 fragile derived stores** that manually enumerate every flag:
```js
isAnyModalOrPanelOpenStore  // subscribes to 23 stores
isAnyModalOpenStore          // subscribes to 17 stores
isAnyPanelOpenStore          // subscribes to 7 stores
```

Adding a new modal requires: (1) new `writable(false)`, (2) update 3 derived stores, (3) add subscription in App.svelte. Easy to forget a step.

### Target state

```js
// Single modal store (only one modal at a time)
export const activeModal = writable(null);
// 'search' | 'met' | 'takePoint2Last' | ... | null

// Panel set (multiple panels can be open)
export const openPanels = writable(new Set());
// Set<'analysis' | 'comment' | 'match' | 'collection' | ...>

// Derived (automatic, no manual enumeration)
export const isAnyModalOpen = derived(activeModal, $m => $m !== null);
export const isAnyPanelOpen = derived(openPanels, $p => $p.size > 0);
export const isAnyModalOrPanelOpen = derived(
    [activeModal, openPanels],
    ([$m, $p]) => $m !== null || $p.size > 0
);
```

Adding a new modal = use a new string constant. No store changes, no derived store changes.

## Files touched

- **Edit:** `frontend/src/stores/uiStore.js`
- **Edit:** `frontend/src/App.svelte`
- **Edit:** All components that read/write modal stores (SearchModal, HelpModal, ExportDatabaseModal, MetModal, etc.)
- **Edit:** `frontend/src/components/Toolbar.svelte` (buttons that toggle modals)

## Tasks

### 1. Define modal/panel constants

- [ ] Add constants to `uiStore.js`:
  ```js
  // Modal identifiers
  export const MODAL = {
      SEARCH: 'search',
      MET: 'met',
      TAKE_POINT_2_LAST: 'takePoint2Last',
      TAKE_POINT_2_LIVE: 'takePoint2Live',
      TAKE_POINT_4_LAST: 'takePoint4Last',
      TAKE_POINT_4_LIVE: 'takePoint4Live',
      GAMMON_VALUE_1: 'gammonValue1',
      GAMMON_VALUE_2: 'gammonValue2',
      GAMMON_VALUE_4: 'gammonValue4',
      WARNING: 'warning',
      METADATA: 'metadata',
      GO_TO_POSITION: 'goToPosition',
      EXPORT_DATABASE: 'exportDatabase',
      TAKE_POINT_2: 'takePoint2',
      TAKE_POINT_4: 'takePoint4',
      HELP: 'help',
      COMMAND: 'command',
  };

  // Panel identifiers
  export const PANEL = {
      ANALYSIS: 'analysis',
      COMMENT: 'comment',
      FILTER_LIBRARY: 'filterLibrary',
      SEARCH_HISTORY: 'searchHistory',
      MATCH: 'match',
      COLLECTION: 'collection',
      TOURNAMENT: 'tournament',
  };
  ```

### 2. Create new store primitives

- [ ] Replace 12+ modal writable stores with:
  ```js
  export const activeModal = writable(null);
  ```
- [ ] Replace 7 panel writable stores with:
  ```js
  export const openPanels = writable(new Set());
  ```
- [ ] Add helper functions:
  ```js
  export function openModal(name) { activeModal.set(name); }
  export function closeModal() { activeModal.set(null); }
  export function togglePanel(name) {
      openPanels.update(s => {
          const next = new Set(s);
          if (next.has(name)) next.delete(name); else next.add(name);
          return next;
      });
  }
  export function closePanel(name) {
      openPanels.update(s => { const next = new Set(s); next.delete(name); return next; });
  }
  export function isPanelOpen(name) {
      return derived(openPanels, $p => $p.has(name));
  }
  ```

### 3. Replace derived stores

- [ ] Replace manual enumeration with automatic:
  ```js
  export const isAnyModalOpen = derived(activeModal, $m => $m !== null);
  export const isAnyPanelOpen = derived(openPanels, $p => $p.size > 0);
  export const isAnyModalOrPanelOpen = derived(
      [activeModal, openPanels],
      ([$m, $p]) => $m !== null || $p.size > 0
  );
  ```

### 4. Update components

- [ ] **Each modal component**: Replace `$showXxxModalStore` with check against `$activeModal === MODAL.XXX`
- [ ] **Each panel component**: Replace `$showXxxStore` with `$openPanels.has(PANEL.XXX)` or use `isPanelOpen(PANEL.XXX)`
- [ ] **Toolbar.svelte**: Replace `showXxxModalStore.set(true)` with `openModal(MODAL.XXX)`, and `showXxxStore.update(v => !v)` with `togglePanel(PANEL.XXX)`
- [ ] **App.svelte**: Remove 33 manual subscriptions — use reactive `$activeModal` and `$openPanels` directly

### 5. Remove old stores

- [ ] Delete all individual `showXxxStore` / `showXxxModalStore` from `uiStore.js`
- [ ] Remove the 3 old derived stores with manual enumeration
- [ ] Verify `uiStore.js` is significantly shorter (~100 lines vs. 252)

### 6. Test

- [ ] `npm run build` succeeds
- [ ] Manual test: open/close every modal, toggle every panel
- [ ] Verify only one modal can be open at a time (opening a modal closes the previous)
- [ ] Verify panels can be open simultaneously
- [ ] Verify keyboard shortcuts (Escape closes modal, etc.)

## Acceptance criteria

- [ ] Single `activeModal` store replaces 12+ individual modal stores
- [ ] Single `openPanels` store replaces 7 individual panel stores
- [ ] Derived stores are automatic (no manual enumeration)
- [ ] `uiStore.js` ≤ 120 lines
- [ ] Adding a new modal = one string constant, zero store changes
- [ ] All modals and panels function correctly

## Rollback

`git revert` — store structure change, but all reactive bindings are mechanical.
