<script>
    import { statsFilterStore } from '../../stores/statsStore.js';
    import { t } from '../../i18n/index.js';
    import PanelTable from '../panels/PanelTable.svelte';

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

    // The header cycles through nextSort (in PanelTable): a column is first
    // picked in its natural direction — lower is better for a rate, higher is
    // more for a count — and a second click flips it.
    let sort = $state({ column: 'pr', direction: 'asc' });
    const columns = $derived(
        COLUMNS.map((col) => ({
            key: col.key,
            label: $t(col.labelKey),
            sortable: true,
            align: col.align === 'left' ? 'left' : 'right',
            defaultDir: col.rate || col.key === 'name' ? 'asc' : 'desc'
        }))
    );

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
        const sortKey = sort.column;
        const sortAsc = sort.direction === 'asc';
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
        <PanelTable
            rows={sorted}
            rowKey={(row) => row.name}
            {columns}
            bind:sort
            pointerRows
            rowAttrs={(row) => ({ tabindex: 0, role: 'button', title: $t('stats.playersRowHint', { name: row.name }), onkeydown: (e) => onRowKey(e, row.name) })}
            onSelect={(row) => selectPlayer(row.name)}
        >
            {#snippet cells(row)}
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
            {/snippet}
        </PanelTable>
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

    /* Every figure is a number set flush right in tabular digits; the name
       column keeps the left alignment the header gives it. */
    .players-tab :global(th),
    .players-tab :global(td) {
        white-space: nowrap;
    }

    td.numeric {
        text-align: right;
        font-variant-numeric: tabular-nums;
    }

    td.strong {
        font-weight: 600;
    }

    .empty-msg {
        font-size: var(--font-size-small);
        opacity: 0.75;
    }
</style>
