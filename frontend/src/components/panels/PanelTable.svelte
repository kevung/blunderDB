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
    import { nextSort } from '../../utils/tableSort.js';
    import { dragReorder } from '../../utils/dragReorder.js';

    /**
     * @typedef {object} Column
     * @property {string} key
     * @property {string} [label]
     * @property {boolean} [sortable]
     * @property {boolean} [narrow]   sized to its content (`.narrow-col`)
     * @property {boolean} [actions]  the icon-button column (`.actions-col`)
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
        class: className = '',
        /** Rendered above the table in a `.detail-header` strip. */
        header = undefined,
        /** Rendered between the header strip and the table, as is. */
        subheader = undefined,
        /** The cells of one row: (row, index). */
        cells
    } = $props();

    let tbodyEl = $state(null);

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
        const el = index >= 0 ? tbodyEl?.children[index] : null;
        if (el && typeof el.scrollIntoView === 'function') el.scrollIntoView({ behavior: 'smooth', block });
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
    <div class="scroll">
        <table>
            <thead>
                <tr>
                    {#each columns as col (col.key)}
                        <th
                            class="no-select {col.class ?? ''}"
                            class:sortable={col.sortable}
                            class:narrow-col={col.narrow}
                            class:actions-col={col.actions}
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
            <tbody bind:this={tbodyEl} use:dragReorder={{ onReorder: onReorder ?? (() => {}), enabled: !!onReorder }}>
                {#each rows as row, index (rowKey(row))}
                    <tr
                        class={rowClass?.(row, index) ?? ''}
                        class:selected={isSelected(row)}
                        class:pointer={pointerRows}
                        {...rowAttrs?.(row, index)}
                        onclick={onSelect ? (e) => onSelect(row, index, e) : undefined}
                        ondblclick={onActivate ? (e) => onActivate(row, index, e) : undefined}
                    >
                        {@render cells(row, index)}
                    </tr>
                {/each}
            </tbody>
        </table>
        {#if rows.length === 0 && emptyText}
            <div class="empty-state">{emptyText}</div>
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

    .empty-state {
        text-align: center;
        color: var(--color-text-muted);
        padding: 24px;
        font-size: var(--font-size-base);
    }
</style>
