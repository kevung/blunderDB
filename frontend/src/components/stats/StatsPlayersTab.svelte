<script>
    import { statsFilterStore } from '../../stores/statsStore.js';
    import { t } from '../../i18n/index.js';

    /**
     * @type {{
     *   rows: Array<object>|null,
     *   onSelectPlayer?: (name: string) => void
     * }}
     */
    let { rows = null, onSelectPlayer = undefined } = $props();

    /**
     * Columns of the table, in display order.
     *
     * `rate` marks the columns holding a measured rate rather than a count.
     * Sorting those keeps players with nothing measured at the bottom in both
     * directions: their zero means "not measured", and letting it win an
     * ascending sort would put the least-known players at the top of a table
     * whose whole point is to compare the known ones.
     */
    const COLUMNS = [
        { key: 'name', labelKey: 'stats.playersColPlayer', align: 'left' },
        { key: 'matches', labelKey: 'stats.playersColMatches' },
        { key: 'record', labelKey: 'stats.playersColRecord' },
        { key: 'decisions', labelKey: 'stats.playersColDecisions' },
        { key: 'pr', labelKey: 'stats.playersColPR', rate: true },
        { key: 'pr_checker', labelKey: 'stats.playersColPRChecker', rate: true },
        { key: 'pr_cube', labelKey: 'stats.playersColPRCube', rate: true },
        { key: 'snowie_er', labelKey: 'stats.playersColSnowie', rate: true },
        { key: 'blunders', labelKey: 'stats.playersColBlunders' },
        { key: 'luck', labelKey: 'stats.playersColLuck', rate: true }
    ];

    let sortKey = $state('pr');
    let sortAsc = $state(true);

    /** Value a row sorts by for a given column; null means "nothing measured". */
    function sortValue(row, key) {
        switch (key) {
            case 'name':
                return row.name?.toLowerCase() ?? '';
            case 'record':
                return row.wins - row.losses;
            case 'luck':
                return row.luck_known ? row.luck_rate_mp : null;
            case 'pr':
            case 'pr_checker':
            case 'pr_cube':
            case 'snowie_er':
                return row.decisions > 0 ? row[key] : null;
            default:
                return row[key] ?? 0;
        }
    }

    const sorted = $derived.by(() => {
        if (!rows) return [];
        const col = COLUMNS.find((c) => c.key === sortKey);
        const copy = [...rows];
        copy.sort((a, b) => {
            const va = sortValue(a, sortKey);
            const vb = sortValue(b, sortKey);
            // Unmeasured rows sink to the bottom whichever way the column is
            // sorted, so a missing figure never reads as a leading score.
            if (col?.rate) {
                if (va === null && vb === null) return a.name.localeCompare(b.name);
                if (va === null) return 1;
                if (vb === null) return -1;
            }
            let cmp;
            if (typeof va === 'string' || typeof vb === 'string') {
                cmp = String(va).localeCompare(String(vb));
            } else {
                cmp = va - vb;
            }
            if (cmp === 0) return a.name.localeCompare(b.name);
            return sortAsc ? cmp : -cmp;
        });
        return copy;
    });

    function toggleSort(key) {
        if (sortKey === key) {
            sortAsc = !sortAsc;
        } else {
            sortKey = key;
            // Lower is better for a rate, higher is more for a count.
            sortAsc = COLUMNS.find((c) => c.key === key)?.rate || key === 'name';
        }
    }

    /** A rate with nothing behind it is shown as unknown, never as a zero. */
    function fmtRate(value, known) {
        if (!known || value == null || isNaN(value)) return '—';
        return value.toFixed(2);
    }

    /** Luck, in signed millipoints per measured roll. */
    function fmtLuck(row) {
        if (!row.luck_known) return '—';
        const v = row.luck_rate_mp;
        return (v > 0 ? '+' : v < 0 ? '−' : '') + Math.abs(v).toFixed(1);
    }

    function selectPlayer(name) {
        statsFilterStore.update((f) => ({ ...f, playerName: name }));
        onSelectPlayer?.(name);
    }

    function onRowKey(event, name) {
        if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            selectPlayer(name);
        }
    }
</script>

<div class="players-tab">
    {#if !rows || rows.length === 0}
        <p class="empty-msg">{$t('stats.playersEmpty')}</p>
    {:else}
        <div class="table-scroll">
            <table class="players-table">
                <thead>
                    <tr>
                        {#each COLUMNS as col (col.key)}
                            <th class:numeric={col.align !== 'left'} aria-sort={sortKey === col.key ? (sortAsc ? 'ascending' : 'descending') : 'none'}>
                                <button class="sort-btn" onclick={() => toggleSort(col.key)}>
                                    {$t(col.labelKey)}
                                    {#if sortKey === col.key}<span class="arrow" aria-hidden="true">{sortAsc ? '▲' : '▼'}</span>{/if}
                                </button>
                            </th>
                        {/each}
                    </tr>
                </thead>
                <tbody>
                    {#each sorted as row (row.name)}
                        <tr
                            class="player-row"
                            tabindex="0"
                            role="button"
                            title={$t('stats.playersRowHint', { name: row.name })}
                            onclick={() => selectPlayer(row.name)}
                            onkeydown={(e) => onRowKey(e, row.name)}
                        >
                            <td class="name">{row.name}</td>
                            <td class="numeric">{row.matches}</td>
                            <td class="numeric">{row.wins}–{row.losses}</td>
                            <td class="numeric">{row.decisions}</td>
                            <td class="numeric strong">{fmtRate(row.pr, row.decisions > 0)}</td>
                            <td class="numeric">{fmtRate(row.pr_checker, row.checker_decisions > 0)}</td>
                            <td class="numeric">{fmtRate(row.pr_cube, row.cube_decisions > 0)}</td>
                            <td class="numeric">{fmtRate(row.snowie_er, row.decisions > 0)}</td>
                            <td class="numeric">{row.blunders}</td>
                            <td class="numeric" title={row.luck_known ? $t('stats.playersLuckRolls', { n: row.luck_rolls }) : $t('stats.playersLuckUnknown')}>
                                {fmtLuck(row)}
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        </div>
    {/if}
</div>

<style>
    .players-tab {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
        /* Bound the tab to the panel so the table can scroll inside it rather
           than pushing the panel down — which is what lets the header stay. */
        height: 100%;
        min-height: 0;
    }

    /* The table scrolls inside its own box, both ways: the panel never scrolls
       sideways, and the vertical scroll has to happen HERE for the sticky
       header to have something to stick to. An ancestor scrolling instead
       would carry the header off-screen along with the rows. */
    .table-scroll {
        flex: 1;
        min-height: 0;
        overflow: auto;
    }

    .players-table {
        width: 100%;
        border-collapse: collapse;
        font-size: var(--font-size-small);
    }

    .players-table th,
    .players-table td {
        padding: 0.3rem 0.5rem;
        border-bottom: 1px solid var(--border-color, #ddd);
        white-space: nowrap;
    }

    .players-table th {
        text-align: right;
        font-weight: 600;
        position: sticky;
        top: 0;
        z-index: 1;
        /* Opaque, or the rows show through the labels as they scroll past. */
        background: var(--panel-bg, #fff);
        /* A sticky cell leaves its own border behind when it detaches, so the
           rule under the header is drawn as an inset shadow instead. */
        border-bottom: none;
        box-shadow: inset 0 -1px 0 var(--border-color, #ddd);
    }

    .players-table th:first-child,
    .players-table td.name {
        text-align: left;
    }

    .players-table td.numeric {
        text-align: right;
        font-variant-numeric: tabular-nums;
    }

    td.strong {
        font-weight: 600;
    }

    .sort-btn {
        font: inherit;
        color: inherit;
        background: none;
        border: none;
        padding: 0;
        cursor: pointer;
    }

    .sort-btn:hover {
        text-decoration: underline;
    }

    .arrow {
        margin-left: 0.2em;
    }

    .player-row {
        cursor: pointer;
    }

    .player-row:hover,
    .player-row:focus-visible {
        background: var(--hover-bg, rgba(127, 127, 127, 0.12));
    }

    .empty-msg {
        font-size: var(--font-size-small);
        opacity: 0.75;
    }
</style>
