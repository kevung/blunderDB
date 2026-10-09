<script module>
    // The shared list table of the Match, Tournament, Collection, Anki and
    // statistics panels: sticky sortable header (nextSort), click/j/k selection,
    // icon actions column, empty state. Panels describe columns and render cells
    // via the `cells` snippet. Row helpers are exported from the module script
    // for panels whose table is not always mounted.

    import { isBareLetter } from '../../utils/keys.js';

    /**
     * +1 for "next row" (j, ArrowDown), −1 for "previous row" (k, ArrowUp), 0 otherwise.
     * @param {KeyboardEvent} event
     */
    export function navigationDelta(event) {
        if (isBareLetter(event, 'j') || event.key === 'ArrowDown') return 1;
        if (isBareLetter(event, 'k') || event.key === 'ArrowUp') return -1;
        return 0;
    }

    /**
     * The row `delta` steps away from the selected one, or `null` at the ends;
     * with no selection, forward lands on the first row, back on nothing.
     *
     * @template T
     * @param {T[]} rows
     * @param {(row: T) => any} rowKey
     * @param {any} selectedKey
     * @param {number} delta
     * @returns {T | null}
     */
    export function stepSelection(rows, rowKey, selectedKey, delta) {
        if (!rows.length || !delta) return null;
        const current = selectedKey === undefined || selectedKey === null ? -1 : rows.findIndex((r) => rowKey(r) === selectedKey);
        if (current < 0) return delta > 0 ? rows[0] : null;
        const next = current + delta;
        return next >= 0 && next < rows.length ? rows[next] : null;
    }
</script>

<script>
    import { tick } from 'svelte';
    import EmptyState from './EmptyState.svelte';
    import { nextSort } from '../../utils/tableSort.js';
    import { dragReorder } from '../../utils/dragReorder.js';

    /**
     * @typedef {object} Column
     * @property {string} key
     * @property {string} [label]
     * @property {boolean} [sortable]
     * @property {boolean} [narrow]   sized to its content (`.narrow-col`)
     * @property {boolean} [actions]  the icon-button column (`.actions-col`)
     * @property {boolean} [elastic]  the one column that takes the leftover width (ADR-0085, G4)
     * @property {'left'|'center'|'right'} [align]
     * @property {string} [title]
     * @property {string} [class]     extra class on the header cell
     * @property {'asc'|'desc'} [defaultDir] direction when this column is first picked
     */

    let {
        rows = [],
        rowKey = (row) => row.id,
        /** @type {Column[]} */
        columns = [],
        /** Current sort, `{ column, direction }`; bind it to read the cycle back. */
        sort = $bindable({ column: null, direction: 'asc' }),
        /** `{ tristate }`: a third click on the same column clears the sort. */
        sortOptions = {},
        /** Key of the selected row (gets the `selected` class). */
        selectedKey = undefined,
        /** Extra classes for a row: (row, index) => string. */
        rowClass = undefined,
        /** Extra attributes spread on a row: (row, index) => object. Not `class`. */
        rowAttrs = undefined,
        /** Rows show a pointer cursor. */
        pointerRows = false,
        /** A row was clicked, or reached with j/k (then `event` is undefined). */
        onSelect = undefined,
        /** A row was double-clicked. */
        onActivate = undefined,
        /** (from, to): enables pointer drag reordering of the rows. */
        onReorder = undefined,
        emptyText = '',
        /** Offer "Import…" (and the way back to the welcome screen) under the empty text. */
        emptyActions = false,
        /** With `emptyActions`: a way to lift the filter that emptied the list, in place of the import. */
        emptyClear = null,
        /** With `emptyActions`: the panel's own primary action, in place of the import. */
        emptyAction = null,
        class: className = '',
        /** Rendered above the table in a `.detail-header` strip. */
        header = undefined,
        /** Rendered between the header strip and the table, as is. */
        subheader = undefined,
        /** Fixed row height in px, the basis of the window arithmetic. */
        rowHeight = 28,
        /** Below this many rows the whole list is rendered (drag reorder, no spacers). */
        virtualizeAbove = 200,
        /** Rows rendered beyond the visible ones, on each side. */
        buffer = 10,
        /** The window reached the last loaded rows: load more, when the list is paged. */
        onNearEnd = undefined,
        /** The cells of one row: (row, index). */
        cells
    } = $props();

    // One column takes the leftover width, so short columns stay at their content and the
    // actions column sits at the right edge. A panel names it (`elastic`); a table of short
    // columns only gets its last non-action column, else the browser spreads the slack over
    // every column and the actions float mid-row.
    let elasticIndex = $derived.by(() => {
        const named = columns.findIndex((c) => c.elastic);
        if (named >= 0) return named;
        if (columns.some((c) => !c.narrow && !c.actions)) return -1;
        return columns.findLastIndex((c) => !c.actions);
    });

    let tbodyEl = $state(null);
    let scrollEl = $state(null);
    let scrollTop = $state(0);
    let viewportHeight = $state(0);

    // The first rendered row sets the real pitch (cell padding, icon buttons).
    let measured = $state(0);
    const rh = $derived(measured || rowHeight);
    const virtual = $derived(rows.length > virtualizeAbove);
    // jsdom and a hidden panel report no height; assume a screenful then.
    const visibleCount = $derived(Math.ceil((viewportHeight || 600) / rh));
    const first = $derived(virtual ? Math.max(0, Math.floor(scrollTop / rh) - buffer) : 0);
    const last = $derived(virtual ? Math.min(rows.length, first + visibleCount + 2 * buffer) : rows.length);
    const windowRows = $derived(virtual ? rows.slice(first, last) : rows);

    $effect(() => {
        if (!virtual || !tbodyEl) return;
        void windowRows;
        const row = tbodyEl.querySelector('tr:not(.spacer)');
        const h = row ? Math.round(row.getBoundingClientRect().height) : 0;
        if (h > 0 && h !== measured) measured = h;
    });

    $effect(() => {
        if (onNearEnd && rows.length > 0 && last >= rows.length - buffer) onNearEnd();
    });

    function onScroll() {
        if (virtual && scrollEl) scrollTop = scrollEl.scrollTop;
    }

    $effect(() => {
        if (!scrollEl) return;
        viewportHeight = scrollEl.clientHeight;
        if (typeof ResizeObserver === 'undefined') return;
        const ro = new ResizeObserver(() => (viewportHeight = scrollEl.clientHeight));
        ro.observe(scrollEl);
        return () => ro.disconnect();
    });

    function handleSort(col) {
        if (!col.sortable) return;
        sort = nextSort(sort?.column ?? null, sort?.direction ?? 'asc', col.key, { tristate: !!sortOptions.tristate, defaultDir: col.defaultDir ?? 'asc' });
    }

    function ariaSort(col) {
        if (!col.sortable) return undefined;
        if (sort?.column !== col.key) return 'none';
        return sort.direction === 'asc' ? 'ascending' : 'descending';
    }

    function isSelected(row) {
        return selectedKey !== undefined && selectedKey !== null && rowKey(row) === selectedKey;
    }

    /** Scroll the row into view once the DOM reflects the current selection. */
    export async function scrollToRow(row, block = 'nearest') {
        await tick();
        const key = rowKey(row);
        const index = rows.findIndex((r) => rowKey(r) === key);
        if (index < 0) return;
        if (virtual && scrollEl) {
            // The row may not be in the DOM: scroll by arithmetic, minimally.
            const top = index * rh;
            const view = scrollEl.clientHeight || viewportHeight || 600;
            const head = scrollEl.querySelector('thead')?.offsetHeight ?? 0;
            let target = scrollEl.scrollTop;
            if (block === 'center') target = top - (view - head - rh) / 2;
            else if (top < target) target = top;
            else if (top + rh > target + view - head) target = top + rh - view + head;
            scrollEl.scrollTop = Math.max(0, target);
            scrollTop = scrollEl.scrollTop;
            return;
        }
        const el = tbodyEl?.children[index];
        if (el && typeof el.scrollIntoView === 'function') el.scrollIntoView({ behavior: 'smooth', block });
    }

    /** Move the keyboard focus onto the row (it must be activatable, i.e. carry a tabindex). */
    export async function focusRow(row) {
        await scrollToRow(row);
        await tick();
        const index = rows.findIndex((r) => rowKey(r) === rowKey(row));
        if (index < 0) return;
        const els = tbodyEl ? [...tbodyEl.querySelectorAll('tr:not(.spacer)')] : [];
        /** @type {HTMLElement | undefined} */ (els[index - first])?.focus();
    }

    export function scrollToSelected(block = 'nearest') {
        const row = rows.find((r) => isSelected(r));
        if (row) scrollToRow(row, block);
    }

    /**
     * Move the selection by `delta` rows (j/k), reporting it through `onSelect`
     * and scrolling it into view. Returns whether a row was reached.
     */
    export function navigate(delta) {
        const next = stepSelection(rows, rowKey, selectedKey, delta);
        if (!next) return false;
        onSelect?.(next, rows.indexOf(next));
        scrollToRow(next);
        return true;
    }
</script>

<div class="panel-table {className}">
    {#if header}
        <div class="detail-header">{@render header()}</div>
    {/if}
    {#if subheader}
        {@render subheader()}
    {/if}
    <div class="scroll" bind:this={scrollEl} onscroll={onScroll}>
        <table>
            <thead>
                <tr>
                    {#each columns as col, i (col.key)}
                        <th
                            class="no-select {col.class ?? ''}"
                            class:sortable={col.sortable}
                            class:narrow-col={col.narrow}
                            class:actions-col={col.actions}
                            class:elastic-col={i === elasticIndex}
                            class:align-center={col.align === 'center'}
                            class:align-right={col.align === 'right'}
                            title={col.title}
                            aria-sort={ariaSort(col)}
                        >
                            {#if col.sortable}
                                <!-- The button, not the <th>, is the keyboard-reachable control. -->
                                <button type="button" class="sort-btn" onclick={() => handleSort(col)}>
                                    {col.label ?? ''}{#if sort?.column === col.key}<span class="sort-arrow">{sort.direction === 'asc' ? '▲' : '▼'}</span>{/if}
                                </button>
                            {:else}
                                {col.label ?? ''}
                            {/if}
                        </th>
                    {/each}
                </tr>
            </thead>
            <tbody bind:this={tbodyEl} use:dragReorder={{ onReorder: onReorder ?? (() => {}), enabled: !!onReorder, itemSelector: 'tr:not(.spacer)', indexOffset: first }}>
                {#if virtual && first > 0}
                    <tr class="spacer" aria-hidden="true" style:height="{first * rh}px"><td colspan={columns.length}></td></tr>
                {/if}
                {#each windowRows as row, i (rowKey(row))}
                    {@const index = first + i}
                    <tr
                        style:height={virtual ? `${rh}px` : undefined}
                        class={rowClass?.(row, index) ?? ''}
                        class:selected={isSelected(row)}
                        class:pointer={pointerRows}
                        tabindex={onActivate ? 0 : undefined}
                        {...rowAttrs?.(row, index)}
                        onclick={onSelect ? (e) => onSelect(row, index, e) : undefined}
                        ondblclick={onActivate ? (e) => onActivate(row, index, e) : undefined}
                        onkeydown={onActivate
                            ? (e) => {
                                  // Only the row itself: Enter on a button or field inside it keeps its own gesture.
                                  if (e.key === 'Enter' && e.target === e.currentTarget && !e.ctrlKey && !e.metaKey && !e.altKey) {
                                      e.preventDefault();
                                      onActivate(row, index, e);
                                  }
                              }
                            : undefined}
                    >
                        {@render cells(row, index)}
                    </tr>
                {/each}
                {#if virtual && last < rows.length}
                    <tr class="spacer" aria-hidden="true" style:height="{(rows.length - last) * rh}px"><td colspan={columns.length}></td></tr>
                {/if}
            </tbody>
        </table>
        {#if rows.length === 0 && emptyText}
            <EmptyState text={emptyText} actions={emptyActions} clear={emptyClear} action={emptyAction} />
        {/if}
    </div>
</div>

<style>
    /* The root fills the flex column it sits in; only the table scrolls, so a
       header strip and whatever the panel puts after the table stay put. */
    .panel-table {
        flex: 1;
        min-height: 0;
        min-width: 0;
        display: flex;
        flex-direction: column;
    }

    .scroll {
        flex: 1;
        min-height: 0;
        overflow-y: auto;
        overflow-x: hidden;
    }

    table {
        width: 100%;
        border-collapse: collapse;
        font-size: var(--font-size-base);
    }

    thead {
        position: sticky;
        top: 0;
        background-color: var(--color-surface-alt);
        z-index: 1;
    }

    /* Cells come from the panel's snippet, hence :global for td. */
    th,
    .panel-table :global(td) {
        padding: 4px 8px;
        text-align: left;
        border-bottom: 1px solid var(--color-border);
    }

    th {
        font-weight: 600;
        color: var(--color-text);
        font-size: var(--font-size-small);
    }

    th.sortable {
        cursor: pointer;
    }

    th.sortable:hover {
        background-color: color-mix(in srgb, var(--color-text) 6%, var(--color-surface-alt));
    }

    th.align-center {
        text-align: center;
    }

    th.align-right {
        text-align: right;
    }

    .sort-btn {
        background: none;
        border: none;
        padding: 0;
        color: inherit;
        font-weight: inherit;
        font-size: inherit;
        cursor: pointer;
    }

    .sort-arrow {
        font-size: var(--font-size-small);
        margin-left: 3px;
        color: color-mix(in srgb, var(--color-primary) 80%, var(--color-text));
    }

    tbody tr {
        transition: background-color 0.1s;
    }

    tbody tr:hover {
        background-color: var(--color-surface-alt);
    }

    tbody tr.spacer,
    tbody tr.spacer:hover {
        background: none;
        pointer-events: none;
    }

    tbody tr.spacer > :global(td) {
        padding: 0;
        border: none;
    }

    tbody tr.pointer {
        cursor: pointer;
    }

    tbody tr.selected {
        background-color: color-mix(in srgb, var(--color-primary) 12%, var(--color-surface));
    }

    tbody tr.selected:hover {
        background-color: color-mix(in srgb, var(--color-primary) 25%, var(--color-surface));
    }

    tbody tr.editing-row {
        background-color: color-mix(in srgb, #ffc107 10%, var(--color-surface));
        cursor: default;
    }

    tbody tr.drag-over {
        border-top: 2px solid var(--color-primary);
    }

    tbody tr.dragging {
        opacity: 0.5;
    }

    /* --- Shared cell vocabulary, used by the panels' cells and header strips --- */

    .panel-table :global(.narrow-col) {
        width: 1px;
        white-space: nowrap;
        padding-left: 6px;
        padding-right: 6px;
    }

    /* Sized to its buttons, not fixed (a cap clipped the fourth); text columns absorb the rest. */
    .panel-table :global(.actions-col) {
        width: 1px;
        white-space: nowrap;
        text-align: center;
        padding: 0 4px;
    }

    /* A percentage beats the cells' 1px: this column takes the slack, the others their content. */
    .panel-table th.elastic-col {
        width: 100%;
    }

    .panel-table :global(.no-select) {
        user-select: none;
        -webkit-user-select: none;
    }

    .panel-table :global(.index-cell) {
        text-align: center;
        color: var(--color-text-muted);
    }

    .panel-table :global(.count-cell) {
        text-align: center;
        color: var(--color-text-muted);
    }

    .panel-table :global(.stat-col) {
        color: var(--color-text-muted);
        font-variant-numeric: tabular-nums;
    }

    .panel-table :global(.item-actions) {
        display: inline-flex;
        gap: 2px;
        vertical-align: middle;
    }

    .panel-table :global(.icon-btn) {
        background: none;
        border: none;
        cursor: pointer;
        font-size: var(--font-size-base);
        color: var(--color-text-muted);
        padding: 2px 4px;
        line-height: 1;
    }

    .panel-table :global(.icon-btn:hover:not(:disabled)) {
        color: var(--color-text);
    }

    .panel-table :global(.icon-btn:disabled) {
        opacity: 0.3;
        cursor: not-allowed;
    }

    .panel-table :global(.icon-btn.delete:hover:not(:disabled)) {
        color: var(--color-danger);
    }

    .detail-header {
        display: flex;
        align-items: center;
        gap: 8px;
        min-height: 24px;
        padding: 4px 8px;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        background: var(--color-surface-alt);
        border-bottom: 1px solid var(--color-border);
        flex-shrink: 0;
    }
</style>
