<script>
    import { t } from '../i18n';
    import { checkerRows } from '../utils/analysisRows.js';

    // Rendering component for a ranked list of checker-move candidates —
    // mounted by AnalysisPanel (a stored record) and by EPCPanel (a live
    // evaluation, wired in #125). It owns no data: sorting, selection and
    // highlighting stay with the caller, which knows whether the moves come
    // from a database row or a fresh computation. The cells themselves come
    // from utils/analysisRows.js — the same rows the copied image paints —
    // so this component lays them out and nothing more.
    //
    // showProvenance (ADR-0018 rule 4): depth/engine are constant across
    // every row of a live evaluation (EPCPanel), so that caller hides the
    // two columns and shows them once in its own badge strip instead.
    // AnalysisPanel keeps them — sortedMoves there is genuinely sorted by
    // engine, so the columns carry real information.
    //
    // baseline (ADR-0018 rule 2): the pre-roll vector, in the same
    // player/opponent frame as a move row (domain.CheckerMove's own field
    // names — see EPCPanel's baselineFacts), rendered as a row pinned right
    // below the header. It is not a candidate: no selection, no play
    // notation, no error figure (rule 3 — the gap to it is the luck of the
    // roll, ADR-0010, never the merit of a play).
    let {
        moves = [],
        sortColumn = 'equity',
        sortDirection = 'desc',
        selectedMove = null,
        isPlayedMove = () => false,
        onSort = () => {},
        onRowClick = () => {},
        showProvenance = true,
        baseline = null
    } = $props();

    let block = $derived(checkerRows(moves, { t: $t, isPlayedMove, showProvenance, baseline }));

    // The equity column never carried an indicator (it is the default sort,
    // and the arrow would sit on it at every opening); the others do.
    function getSortIndicator(column) {
        if (column === 'equity' || sortColumn !== column) return '';
        return sortDirection === 'asc' ? ' ▲' : ' ▼';
    }
</script>

<div class="checker-scroll">
    <table class="checker-table">
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
                <tr class:selected={selectedMove === row.move.move} class:played={row.highlight} onclick={() => onRowClick(row.move)}>
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
        color: #777;
        text-transform: uppercase;
        letter-spacing: 0.3px;
        font-weight: 600;
        height: 24px;
        box-sizing: border-box;
        vertical-align: middle;
        background: #fff;
        /* EPCPanel's own scroll region is this table's list — the header
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
        background-color: #fafafa;
    }

    .checker-table tr.selected {
        background-color: #b3d9ff !important;
        font-weight: bold;
    }

    .checker-table tr.played {
        background-color: #fff3cd !important;
    }

    .checker-table tr.played.selected {
        background-color: #a3c9ef !important;
    }

    tbody:last-of-type tr:hover {
        background-color: #eaf1fb;
    }

    /* The pre-roll vector, in the axis of this list — a reference mark, not
       a ranked candidate (ADR-0018 rules 2/3): italic, muted, no error
       figure, closed by a heavy rule so the value of the roll it precedes
       is never read as the merit of a play. Pinned right under the sticky
       header (same fixed height as the header cells, so the two stack
       exactly), so it never scrolls away either. */
    .baseline-row td {
        height: 24px;
        box-sizing: border-box;
        vertical-align: middle;
        background: #fafafa;
        color: #999;
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
        color: #1a56c4;
    }

    .active-sort {
        color: #1a56c4;
    }

    @container (max-width: 600px) {
        .checker-scroll {
            overflow-x: auto;
        }

        .checker-table {
            min-width: 560px;
        }
    }
</style>
