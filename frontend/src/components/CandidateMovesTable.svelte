<script>
    import { t } from '../i18n';
    import { checkerRows } from '../utils/analysisRows.js';

    // Lays out ranked checker-move candidates for AnalysisPanel and EvalPanel;
    // sorting, selection and highlighting stay with the caller, cells come from
    // utils/analysisRows.js.
    // showProvenance (ADR-0018 rule 4): EvalPanel hides depth/engine columns.
    // baseline (ADR-0018 rules 2, 3): the pre-roll vector as a pinned row in the
    // move rows' frame — not a candidate, no error figure (ADR-0010).
    // isMoney (ADR-0016 point 6): the equity header's referential; undefined keeps it plain.
    // projection (ADR-0048 decision 4): `judge` (nine columns) or `identify`
    // (move, equity, error), defined in utils/analysisRows.js.
    let {
        moves = [],
        sortColumn = 'equity',
        sortDirection = 'desc',
        selectedMove = null,
        isPlayedMove = () => false,
        onSort = () => {},
        onRowClick = () => {},
        // Double-clic : valider (ADR-0048 décision 11) ; le simple clic sélectionne.
        onRowDblClick = undefined,
        showProvenance = true,
        baseline = null,
        isMoney = undefined,
        projection = 'judge'
    } = $props();

    let block = $derived(checkerRows(moves, { t: $t, isPlayedMove, showProvenance, baseline, isMoney, projection }));

    // The equity column never carried an indicator (it is the default sort,
    // and the arrow would sit on it at every opening); the others do.
    function getSortIndicator(column) {
        if (column === 'equity' || sortColumn !== column) return '';
        return sortDirection === 'asc' ? ' ▲' : ' ▼';
    }
</script>

<div class="checker-scroll">
    <table class="checker-table" class:identify={projection === 'identify'}>
        <thead>
            <tr>
                {#each block.columns as column, i (column)}
                    <th
                        class="sortable"
                        class:active-sort={sortColumn === column}
                        onclick={(e) => {
                            e.stopPropagation();
                            onSort(column);
                        }}>{block.header[i]}{getSortIndicator(column)}</th
                    >
                {/each}
            </tr>
        </thead>
        {#if block.baseline}
            <tbody>
                <tr class="baseline-row" title={$t('eval.baselineTooltip')}>
                    <td class="baseline-label">{block.baseline.label}</td>
                    {#each block.baseline.cells as cell, i (i)}
                        <td>{cell}</td>
                    {/each}
                </tr>
            </tbody>
        {/if}
        <tbody>
            {#each block.rows as row (row.key)}
                <tr class:selected={selectedMove === row.move.move} class:played={row.highlight} onclick={() => onRowClick(row.move)} ondblclick={() => onRowDblClick?.(row.move)}>
                    <td>{row.label}</td>
                    {#each row.cells as cell, i (i)}
                        <td>{cell}</td>
                    {/each}
                </tr>
            {/each}
        </tbody>
    </table>
</div>

<style>
    .checker-scroll {
        width: 100%;
    }

    .checker-table {
        margin: 0 auto;
        width: 100%;
        font-size: var(--font-size-base);
        border-collapse: collapse;
    }

    th,
    td {
        padding: 2px 10px;
        text-align: center;
        white-space: nowrap;
        font-variant-numeric: tabular-nums;
    }

    th {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        text-transform: uppercase;
        letter-spacing: 0.3px;
        font-weight: 600;
        height: 24px;
        box-sizing: border-box;
        vertical-align: middle;
        background: var(--color-surface);
        /* EvalPanel's own scroll region is this table's list — the header
           stays readable while the rows scroll under it (ADR-0017). */
        position: sticky;
        top: 0;
        z-index: 2;
    }

    th:nth-child(1) {
        width: 150px;
    }

    th:nth-child(n + 2) {
        width: 60px;
    }

    /* `identify` : la notation prend la place libre (un double se tronquait à 150 px). */
    .checker-table.identify th:nth-child(1) {
        width: auto;
        text-align: left;
    }

    .checker-table.identify td:nth-child(1) {
        text-align: left;
    }

    .checker-table th:nth-child(3),
    .checker-table td:nth-child(3),
    .checker-table th:nth-child(6),
    .checker-table td:nth-child(6),
    .checker-table th:nth-child(9),
    .checker-table td:nth-child(9) {
        border-right: 2px solid #e0e0e0;
    }

    tbody:last-of-type tr:not(:first-child) td {
        border-top: 1px solid #eee;
    }

    tbody:last-of-type tr:nth-child(even) {
        background-color: var(--color-surface-alt);
    }

    .checker-table tr.selected {
        background-color: color-mix(in srgb, var(--color-primary) 30%, var(--color-surface)) !important;
        font-weight: bold;
    }

    .checker-table tr.played {
        background-color: color-mix(in srgb, #ffc107 20%, var(--color-surface)) !important;
    }

    .checker-table tr.played.selected {
        background-color: color-mix(in srgb, var(--color-primary) 38%, var(--color-surface)) !important;
    }

    tbody:last-of-type tr:hover {
        background-color: color-mix(in srgb, var(--color-primary) 8%, var(--color-surface));
    }

    /* Baseline row (ADR-0018 rules 2/3): a muted reference, closed by a heavy
       rule, pinned under the sticky header at the header's height. */
    .baseline-row td {
        height: 24px;
        box-sizing: border-box;
        vertical-align: middle;
        background: var(--color-surface-alt);
        color: var(--color-text-muted);
        font-style: italic;
        border-bottom: 2px solid #ddd;
        position: sticky;
        top: 24px;
        z-index: 1;
    }

    .baseline-label {
        font-size: var(--font-size-small);
        font-weight: 600;
    }

    .sortable {
        cursor: pointer;
        user-select: none;
    }

    .sortable:hover {
        color: var(--color-primary);
    }

    .active-sort {
        color: var(--color-primary);
    }

    @container (max-width: 600px) {
        .checker-scroll {
            overflow-x: auto;
        }

        .checker-table:not(.identify) {
            min-width: 560px;
        }
    }
</style>
