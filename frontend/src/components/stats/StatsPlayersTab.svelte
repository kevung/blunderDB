<script>
    import { tick } from 'svelte';
    import { get } from 'svelte/store';
    import { statsFilterStore, patchStatsFilter, defaultStatsFilter } from '../../stores/statsStore.js';
    import { StatsMatchIDs } from '../../../wailsjs/go/database/Database.js';
    import { loadPositionsFromStatsSelection, openMatchListInPanel } from '../../services/positionLoader.js';
    import CountLink from '../panels/CountLink.svelte';
    import { fmtMwc7, mwc7Percent, mwc7Tooltip } from '../../utils/mwc7.js';
    import { t } from '../../i18n/index.js';
    import PanelTable from '../panels/PanelTable.svelte';
    import PlayerComparison from './PlayerComparison.svelte';

    /**
     * @type {{
     *   rows: Array<any>|null,
     *   onSelectPlayer?: (name: string) => void
     * }}
     */
    let { rows = null, onSelectPlayer = undefined } = $props();

    /**
     * Columns in display order. `rate` columns keep unmeasured players (zero)
     * at the bottom in both sort directions.
     */
    const COLUMNS = [
        { key: 'compare', labelKey: 'stats.compareColumn', hintKey: 'stats.compareColumnHint', align: 'left', plain: true },
        { key: 'name', labelKey: 'stats.playersColPlayer', align: 'left' },
        { key: 'matches', labelKey: 'stats.playersColMatches' },
        { key: 'record', labelKey: 'stats.playersColRecord' },
        { key: 'decisions', labelKey: 'stats.playersColDecisions' },
        { key: 'pr', labelKey: 'stats.playersColPR', hintKey: 'stats.playersColPRHint', rate: true },
        { key: 'pr_checker', labelKey: 'stats.playersColPRChecker', hintKey: 'stats.playersColPRHint', rate: true },
        { key: 'pr_cube', labelKey: 'stats.playersColPRCube', hintKey: 'stats.playersColPRHint', rate: true },
        { key: 'snowie_er', labelKey: 'stats.playersColSnowie', rate: true },
        { key: 'mwc7', labelKey: 'mwc7.short', hintKey: 'mwc7.name', rate: true },
        { key: 'blunders', labelKey: 'stats.playersColBlunders' },
        { key: 'luck', labelKey: 'stats.playersColLuck', hintKey: 'stats.playersColLuckHint', rate: true }
    ];

    // First click sorts in the natural direction (rate ascending, count descending).
    let sort = $state({ column: 'pr', direction: 'asc' });
    const columns = $derived(
        COLUMNS.map((col) => ({
            key: col.key,
            label: $t(col.labelKey),
            title: col.hintKey ? $t(col.hintKey) : undefined,
            sortable: !col.plain,
            align: col.align === 'left' ? 'left' : 'right',
            defaultDir: col.rate || col.key === 'name' ? 'asc' : 'desc'
        }))
    );

    /** Value a row sorts by for a given column; null means "nothing measured". */
    function sortValue(/** @type {any} */ row, /** @type {string} */ key) {
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
            case 'mwc7':
                return mwc7Percent(row.mwc7);
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
    function fmtRate(/** @type {number|null|undefined} */ value, /** @type {boolean} */ known) {
        if (!known || value == null || isNaN(value)) return '—';
        return value.toFixed(2);
    }

    /** Luck, in signed millipoints per measured roll. */
    function fmtLuck(/** @type {any} */ row) {
        if (!row.luck_known) return '—';
        const v = row.luck_rate_mp;
        return (v > 0 ? '+' : v < 0 ? '−' : '') + Math.abs(v).toFixed(1);
    }

    // Au plus deux joueurs comparés : au troisième, le plus ancien sort.
    let compared = $state(/** @type {string[]} */ ([]));

    /** @param {string} name */
    function toggleCompare(name) {
        compared = compared.includes(name) ? compared.filter((n) => n !== name) : [...compared, name].slice(-2);
    }

    const comparedRows = $derived(compared.map((name) => (rows ?? []).find((r) => r.name === name)).filter(Boolean));

    // La sélection ne survit pas à un changement de tableau.
    $effect(() => {
        const names = new Set((rows ?? []).map((r) => r.name));
        if (compared.some((n) => !names.has(n))) compared = compared.filter((n) => names.has(n));
    });

    function selectPlayer(/** @type {string} */ name) {
        patchStatsFilter({ playerName: name });
        onSelectPlayer?.(name);
    }

    /** @type {any} */
    let table = $state();
    const selectedName = $derived($statsFilterStore.playerName);

    // The filter's player is marked in the full table and brought into view.
    $effect(() => {
        if (!selectedName || !table || sorted.length === 0) return;
        tick().then(() => table?.scrollToSelected('center'));
    });

    /**
     * The filter the table was computed under, narrowed to one player: the table honours the
     * dates, tournaments and match lengths only, and counts every decision type.
     */
    function playerFilter(/** @type {string} */ name) {
        const f = get(statsFilterStore);
        return { ...defaultStatsFilter, tournamentIDs: f.tournamentIDs, dateFrom: f.dateFrom, dateTo: f.dateTo, matchLength: f.matchLength, playerName: name };
    }

    /** A link inside a row opens its own target, not the player's dashboard. */
    function linkTo(/** @type {() => any} */ open) {
        return (/** @type {MouseEvent} */ e) => {
            e.stopPropagation();
            open();
        };
    }

    const openMatches = (/** @type {string} */ name) => linkTo(async () => openMatchListInPanel((await StatsMatchIDs(/** @type {any} */ (playerFilter(name)))) ?? []));
    const openDecisions = (/** @type {string} */ name) => linkTo(() => loadPositionsFromStatsSelection(playerFilter(name), { Kind: 'all', OnlyWithError: false }));

    function onRowKey(/** @type {KeyboardEvent} */ event, /** @type {string} */ name) {
        // Enter on a link of the row is that link's, not the row's.
        if (event.target !== event.currentTarget) return;
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
        {#if comparedRows.length === 2}
            <PlayerComparison a={comparedRows[0]} b={comparedRows[1]} onClear={() => (compared = [])} />
        {/if}
        <PanelTable
            bind:this={table}
            rows={sorted}
            selectedKey={selectedName || undefined}
            rowKey={(row) => row.name}
            {columns}
            bind:sort
            pointerRows
            rowAttrs={(row) => ({ tabindex: 0, role: 'button', title: $t('stats.playersRowHint', { name: row.name }), onkeydown: (/** @type {KeyboardEvent} */ e) => onRowKey(e, row.name) })}
            onSelect={(row) => selectPlayer(row.name)}
        >
            {#snippet cells(row)}
                <td class="compare">
                    <input
                        type="checkbox"
                        checked={compared.includes(row.name)}
                        aria-label={$t('stats.compareToggle', { name: row.name })}
                        onclick={(e) => {
                            // Cocher n'est pas ouvrir : sans cela le clic de
                            // ligne filtrerait aussitôt sur ce joueur.
                            e.stopPropagation();
                            toggleCompare(row.name);
                        }}
                    />
                </td>
                <td class="name">{row.name}</td>
                <td class="numeric"><CountLink label={String(row.matches)} onclick={openMatches(row.name)} /></td>
                <td class="numeric"><CountLink label={`${row.wins}–${row.losses}`} onclick={openMatches(row.name)} /></td>
                <td class="numeric"><CountLink label={String(row.decisions)} onclick={openDecisions(row.name)} /></td>
                <td class="numeric strong">{fmtRate(row.pr, row.decisions > 0)}</td>
                <td class="numeric">{fmtRate(row.pr_checker, row.checker_decisions > 0)}</td>
                <td class="numeric">{fmtRate(row.pr_cube, row.cube_decisions > 0)}</td>
                <td class="numeric">{fmtRate(row.snowie_er, row.decisions > 0)}</td>
                <td class="numeric" title={mwc7Tooltip(row.mwc7, $t)}>{fmtMwc7(row.mwc7)}</td>
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

    td.compare {
        width: 1.6rem;
        text-align: center;
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
