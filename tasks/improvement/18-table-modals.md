# 18 — Parameterise duplicated table modals

**Goal:** Replace 9 near-identical TakePoint/GammonValue modal components (985 lines total) with a single generic `DataTableModal.svelte`.

**Depends on:** 14 (modal store consolidation — modals switch to `activeModal` pattern).

**Impact:** Medium — removes ~800 lines of copy-pasted code.

## Context

### Current state: 9 copy-pasted modals

| Component | Lines | Data source | Format |
|-----------|------:|-------------|--------|
| `TakePoint2LastModal.svelte` | 98 | `takePoint2LastStore` | `.toFixed(1)` |
| `TakePoint2LiveModal.svelte` | 98 | `takePoint2LiveStore` | `.toFixed(1)` |
| `TakePoint4LastModal.svelte` | 97 | `takePoint4LastStore` | `.toFixed(0)` |
| `TakePoint4LiveModal.svelte` | 99 | `takePoint4LiveStore` | `.toFixed(0)` |
| `GammonValue1Modal.svelte` | 98 | `gammonValue1Store` | `.toFixed(2)` |
| `GammonValue2Modal.svelte` | 99 | `gammonValue2Store` | `.toFixed(2)` |
| `GammonValue4Modal.svelte` | 99 | `gammonValue4Store` | `.toFixed(2)` |
| `TakePoint2Modal.svelte` | 149 | 2 stores side-by-side | `.toFixed(1)` |
| `TakePoint4Modal.svelte` | 148 | 2 stores side-by-side | `.toFixed(0)` |
| **Total** | **985** | | |

All 9 share the same structure:
- Modal overlay + content wrapper
- `<table>` with `thead`/`tbody`
- Alternating row colors CSS
- Escape-to-close handler
- Only differences: data store, `toFixed()` precision, title, single vs. dual-table layout

### Target: one generic component

```svelte
<!-- DataTableModal.svelte -->
<script>
    export let visible = false;
    export let onClose;
    export let title = '';
    export let tables = [];    // [{title, data, precision}]
</script>
```

- Single table → `tables = [{ data: $store, precision: 1 }]`
- Dual table → `tables = [{ title: 'Live', data: $live, precision: 1 }, { title: 'Last', data: $last, precision: 1 }]`

## Files touched

- **New:** `frontend/src/components/DataTableModal.svelte`
- **Delete:** `frontend/src/components/TakePoint2LastModal.svelte`
- **Delete:** `frontend/src/components/TakePoint2LiveModal.svelte`
- **Delete:** `frontend/src/components/TakePoint4LastModal.svelte`
- **Delete:** `frontend/src/components/TakePoint4LiveModal.svelte`
- **Delete:** `frontend/src/components/GammonValue1Modal.svelte`
- **Delete:** `frontend/src/components/GammonValue2Modal.svelte`
- **Delete:** `frontend/src/components/GammonValue4Modal.svelte`
- **Delete:** `frontend/src/components/TakePoint2Modal.svelte`
- **Delete:** `frontend/src/components/TakePoint4Modal.svelte`
- **Edit:** `frontend/src/App.svelte` — replace 9 component usages with `DataTableModal`

## Tasks

### 1. Create `DataTableModal.svelte`

- [x] Implement generic modal with props:
  - `visible: boolean`
  - `onClose: () => void`
  - `title: string`
  - `tables: Array<{ title?: string, data: number[][], precision: number }>`
- [x] Support both single-table and side-by-side dual-table layout
- [x] Include the Escape-to-close handler
- [x] Include shared CSS (modal overlay, content, alternating rows)
- [x] Use `{cell.toFixed(precision)}` for cell formatting

### 2. Replace simple modals (7 files)

- [x] Replace each single-table modal with `<DataTableModal>` in `App.svelte`:
  ```svelte
  <!-- Before: 7 separate components -->
  <TakePoint2LastModal visible={showTakePoint2Last} onClose={...} />
  <GammonValue1Modal visible={showGammonValue1} onClose={...} />

  <!-- After: 7 instances of DataTableModal -->
  <DataTableModal visible={...} onClose={...} title="Take Point 2 (Last)"
      tables={[{ data: $takePoint2LastStore, precision: 1 }]} />
  <DataTableModal visible={...} onClose={...} title="Gammon Value (1-cube)"
      tables={[{ data: $gammonValue1Store, precision: 2 }]} />
  ```
- [x] Delete the 7 old component files

### 3. Replace dual-table modals (2 files)

- [x] Replace `TakePoint2Modal` and `TakePoint4Modal`:
  ```svelte
  <DataTableModal visible={...} onClose={...} title="Take Point 2"
      tables={[
          { title: 'Live', data: $takePoint2LiveStore, precision: 1 },
          { title: 'Last', data: $takePoint2LastStore, precision: 1 },
      ]} />
  ```
- [x] Delete the 2 old component files

### 4. Verify

- [x] `npm run build` succeeds
- [ ] Visual check: each table displays the same data as before (manual)
- [ ] Escape key closes modals (manual)
- [ ] Mouse scroll on table still works (manual)
- [ ] All 9 modal triggers still open the correct table (manual)

## Acceptance criteria

- [x] 9 old modal files deleted
- [x] 1 new `DataTableModal.svelte` (116 lines)
- [x] Net deletion: 981 lines (1012 deleted, 31 added)
- [ ] All tables render identically to before (manual)
- [x] `npm run build` succeeds

## Rollback

`git revert` — reverting adds back the 9 files and removes `DataTableModal.svelte`.
