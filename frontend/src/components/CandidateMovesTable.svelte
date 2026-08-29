<script>
    import { t } from '../i18n';

    // Rendering component for a ranked list of checker-move candidates —
    // mounted by AnalysisPanel (a stored record) and by EPCPanel (a live
    // evaluation, wired in #125). It owns no data: sorting, selection and
    // highlighting stay with the caller, which knows whether the moves come
    // from a database row or a fresh computation.
    let { moves = [], sortColumn = 'equity', sortDirection = 'desc', selectedMove = null, isPlayedMove = () => false, onSort = () => {}, onRowClick = () => {} } = $props();

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
            </tr>
        </thead>
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
                    <td>{move.analysisDepth}</td>
                    <td>{move.analysisEngine || ''}</td>
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
        border-spacing: 0;
    }

    th,
    td {
        border: 1px solid #ddd;
        padding: 2px;
        text-align: center;
    }

    th {
        background-color: #f2f2f2;
    }

    th:nth-child(1) {
        width: 150px;
    }

    th:nth-child(n + 2) {
        width: 60px;
    }

    .checker-table th:nth-child(1),
    .checker-table td:nth-child(1) {
        border-right: 2px solid #ccc;
    }

    .checker-table th:nth-child(3),
    .checker-table td:nth-child(3) {
        border-right: 2px solid #ccc;
    }

    .checker-table th:nth-child(6),
    .checker-table td:nth-child(6) {
        border-right: 2px solid #ccc;
    }

    .checker-table th:nth-child(9),
    .checker-table td:nth-child(9) {
        border-right: 2px solid #ccc;
    }

    .checker-table tr:nth-child(even) {
        background-color: #fdfdfd;
    }

    .checker-table tr:nth-child(odd) {
        background-color: #ffffff;
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

    .checker-table tbody tr:hover {
        background-color: #e6f2ff;
    }

    .sortable {
        cursor: pointer;
        user-select: none;
        position: relative;
    }

    .sortable:hover {
        background-color: #e0e0e0;
    }

    .active-sort {
        background-color: #dde8f0;
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
