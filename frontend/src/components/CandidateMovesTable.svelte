<script>
    import { t } from '../i18n';

    // Rendering component for a ranked list of checker-move candidates —
    // mounted by AnalysisPanel (a stored record) and by EPCPanel (a live
    // evaluation, wired in #125). It owns no data: sorting, selection and
    // highlighting stay with the caller, which knows whether the moves come
    // from a database row or a fresh computation.
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

    function getSortIndicator(column) {
        if (sortColumn !== column) return '';
        return sortDirection === 'asc' ? ' ▲' : ' ▼';
    }

    function formatEquity(value) {
        return value >= 0 ? `+${value.toFixed(3)}` : value.toFixed(3);
    }
</script>

<div class="checker-scroll">
    <table class="checker-table">
        <thead>
            <tr>
                <th
                    class="sortable"
                    class:active-sort={sortColumn === 'move'}
                    onclick={(e) => {
                        e.stopPropagation();
                        onSort('move');
                    }}>{$t('analysis.move')}{getSortIndicator('move')}</th
                >
                <th
                    class="sortable"
                    class:active-sort={sortColumn === 'equity'}
                    onclick={(e) => {
                        e.stopPropagation();
                        onSort('equity');
                    }}>{$t('analysis.equity')}</th
                >
                <th
                    class="sortable"
                    class:active-sort={sortColumn === 'error'}
                    onclick={(e) => {
                        e.stopPropagation();
                        onSort('error');
                    }}>{$t('analysis.error')}{getSortIndicator('error')}</th
                >
                <th
                    class="sortable"
                    class:active-sort={sortColumn === 'pw'}
                    onclick={(e) => {
                        e.stopPropagation();
                        onSort('pw');
                    }}>{$t('analysis.playerWin')}{getSortIndicator('pw')}</th
                >
                <th
                    class="sortable"
                    class:active-sort={sortColumn === 'pg'}
                    onclick={(e) => {
                        e.stopPropagation();
                        onSort('pg');
                    }}>{$t('analysis.playerGammon')}{getSortIndicator('pg')}</th
                >
                <th
                    class="sortable"
                    class:active-sort={sortColumn === 'pb'}
                    onclick={(e) => {
                        e.stopPropagation();
                        onSort('pb');
                    }}>{$t('analysis.playerBackgammon')}{getSortIndicator('pb')}</th
                >
                <th
                    class="sortable"
                    class:active-sort={sortColumn === 'ow'}
                    onclick={(e) => {
                        e.stopPropagation();
                        onSort('ow');
                    }}>{$t('analysis.opponentWin')}{getSortIndicator('ow')}</th
                >
                <th
                    class="sortable"
                    class:active-sort={sortColumn === 'og'}
                    onclick={(e) => {
                        e.stopPropagation();
                        onSort('og');
                    }}>{$t('analysis.opponentGammon')}{getSortIndicator('og')}</th
                >
                <th
                    class="sortable"
                    class:active-sort={sortColumn === 'ob'}
                    onclick={(e) => {
                        e.stopPropagation();
                        onSort('ob');
                    }}>{$t('analysis.opponentBackgammon')}{getSortIndicator('ob')}</th
                >
                {#if showProvenance}
                    <th
                        class="sortable"
                        class:active-sort={sortColumn === 'depth'}
                        onclick={(e) => {
                            e.stopPropagation();
                            onSort('depth');
                        }}>{$t('analysis.depth')}{getSortIndicator('depth')}</th
                    >
                    <th
                        class="sortable"
                        class:active-sort={sortColumn === 'engine'}
                        onclick={(e) => {
                            e.stopPropagation();
                            onSort('engine');
                        }}>{$t('analysis.engine')}{getSortIndicator('engine')}</th
                    >
                {/if}
            </tr>
        </thead>
        {#if baseline}
            <tbody>
                <tr class="baseline-row" title={$t('eval.baselineTooltip')}>
                    <td class="baseline-label">{$t('eval.baseline')}</td>
                    <td>{formatEquity(baseline.cubelessEquity ?? 0)}</td>
                    <td></td>
                    <td>{(baseline.playerWinChance ?? 0).toFixed(2)}</td>
                    <td>{(baseline.playerGammonChance ?? 0).toFixed(2)}</td>
                    <td>{(baseline.playerBackgammonChance ?? 0).toFixed(2)}</td>
                    <td>{(baseline.opponentWinChance ?? 0).toFixed(2)}</td>
                    <td>{(baseline.opponentGammonChance ?? 0).toFixed(2)}</td>
                    <td>{(baseline.opponentBackgammonChance ?? 0).toFixed(2)}</td>
                    {#if showProvenance}
                        <td></td>
                        <td></td>
                    {/if}
                </tr>
            </tbody>
        {/if}
        <tbody>
            {#each moves as move (move.index ?? move.move)}
                <tr class:selected={selectedMove === move.move} class:played={isPlayedMove(move)} onclick={() => onRowClick(move)}>
                    <td>{move.move}</td>
                    <td>{formatEquity(move.equity || 0)}</td>
                    <td>{formatEquity(move.equityError || 0)}</td>
                    <td>{(move.playerWinChance || 0).toFixed(2)}</td>
                    <td>{(move.playerGammonChance || 0).toFixed(2)}</td>
                    <td>{(move.playerBackgammonChance || 0).toFixed(2)}</td>
                    <td>{(move.opponentWinChance || 0).toFixed(2)}</td>
                    <td>{(move.opponentGammonChance || 0).toFixed(2)}</td>
                    <td>{(move.opponentBackgammonChance || 0).toFixed(2)}</td>
                    {#if showProvenance}
                        <td>{move.analysisDepth}</td>
                        <td>{move.analysisEngine || ''}</td>
                    {/if}
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
