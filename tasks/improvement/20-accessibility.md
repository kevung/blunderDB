# 20 — Accessibility improvements

**Goal:** Add baseline WCAG 2.1 AA accessibility to all modals, panels, and the board component.

**Depends on:** 14 (modal consolidation), 18 (table modals) — fewer modals to patch.

**Impact:** Medium — makes the app usable for keyboard-only users and improves screen reader support.

## Context

### Current accessibility audit

| Category | Status |
|----------|--------|
| `role="dialog"` on modals | 4 of 18 modals have it (78% missing) |
| `aria-label` on modals | 0 of 18 |
| `aria-labelledby` / `aria-describedby` | Zero usage |
| Focus trapping in modals | None |
| Board accessible description | None (plain `<div>`, no alt text) |
| Keyboard navigation in panels | Partial (Escape closes, no Tab trapping) |
| Toolbar buttons | Good — all 22 have `aria-label` |
| Screen reader live regions | None |

### What to fix (in priority order)
1. **Modal semantics** — add `role="dialog"`, `aria-modal="true"`, `aria-label`
2. **Focus trapping** — trap Tab/Shift+Tab within open modals
3. **Board description** — add `aria-label` describing current position
4. **Live regions** — announce status bar changes to screen readers

## Files touched

- **New:** `frontend/src/utils/focusTrap.js`
- **Edit:** All modal `.svelte` files (or just `DataTableModal.svelte` if task 18 is done)
- **Edit:** `frontend/src/components/Board.svelte`
- **Edit:** `frontend/src/components/StatusBar.svelte`
- **Edit:** Panel components (`AnalysisPanel`, `MatchPanel`, etc.)

## Tasks

### 1. Create focus trap utility

- [x] Create `frontend/src/utils/focusTrap.js`:
  ```js
  export function trapFocus(node) {
      const focusableSelector = 'a[href], button, textarea, input, select, [tabindex]:not([tabindex="-1"])';

      function handleKeydown(e) {
          if (e.key !== 'Tab') return;
          const focusable = [...node.querySelectorAll(focusableSelector)];
          if (focusable.length === 0) return;
          const first = focusable[0];
          const last = focusable[focusable.length - 1];
          if (e.shiftKey && document.activeElement === first) {
              e.preventDefault();
              last.focus();
          } else if (!e.shiftKey && document.activeElement === last) {
              e.preventDefault();
              first.focus();
          }
      }

      node.addEventListener('keydown', handleKeydown);
      // Focus first focusable element on mount
      const first = node.querySelector(focusableSelector);
      if (first) first.focus();

      return {
          destroy() { node.removeEventListener('keydown', handleKeydown); }
      };
  }
  ```
- [x] This can be used as a Svelte `use:` action: `<div use:trapFocus>`

### 2. Add `role="dialog"` and `aria-label` to all modals

- [x] For each modal, add to the modal overlay/container:
  ```html
  <div class="modal-overlay" role="dialog" aria-modal="true" aria-label="Search positions">
  ```
- [x] List of modals to update (if task 18 is NOT done — 14 modals missing the attributes):
  - HelpModal, SearchModal, GoToPositionModal, WarningModal, MetModal,
    MetadataModal, ExportDatabaseModal, ImportProgressModal, FileImportProgressModal,
    TakePoint2Modal, TakePoint2LastModal, TakePoint2LiveModal, TakePoint4Modal,
    TakePoint4LastModal, TakePoint4LiveModal, GammonValue1Modal, GammonValue2Modal,
    GammonValue4Modal
- [x] If task 18 IS done: just update `DataTableModal.svelte` + remaining modals

### 3. Add focus trapping to modals

- [x] Apply `use:trapFocus` to each modal container:
  ```svelte
  {#if visible}
      <div class="modal-overlay" role="dialog" aria-modal="true" use:trapFocus>
  ```
- [x] Ensure focus returns to the trigger element when modal closes

### 4. Add board accessibility

- [x] Add `aria-label` to the board container describing the current position:
  ```html
  <div id="backgammon-board" role="img" aria-label={boardDescription}>
  ```
- [x] Generate `boardDescription` from current position state, e.g.:
  ```
  "Backgammon board. Player 1 pip count 120, Player 2 pip count 95. Player 1 to move."
  ```
- [x] Update `boardDescription` when position changes

### 5. Add live region for status messages

- [x] Make the status bar a live region:
  ```html
  <div class="status-bar" role="status" aria-live="polite">
  ```
- [x] Screen readers will announce status changes (import progress, errors, etc.)

### 6. Add keyboard navigation in panels

- [x] Ensure panel close buttons are focusable
- [x] Ensure lists in panels (match list, collection list, search history) are navigable with arrow keys
- [x] Add `tabindex="0"` to interactive elements that aren't natively focusable

### 7. Verify

- [x] Test with keyboard only: Tab through all modals, Escape closes, focus returns
- [x] Test with a screen reader (or browser accessibility tools):
  - Modals announced as dialogs
  - Board position described
  - Status messages announced
- [x] `npm run build` succeeds
- [x] No visual regressions

## Acceptance criteria

- [x] All modals have `role="dialog"`, `aria-modal="true"`, `aria-label`
- [x] Focus is trapped within open modals
- [x] Board has a descriptive `aria-label`
- [x] Status bar has `role="status"` and `aria-live="polite"`
- [x] `npm run build` succeeds

## Rollback

`git revert` — additive attributes only, no behavior changes.
