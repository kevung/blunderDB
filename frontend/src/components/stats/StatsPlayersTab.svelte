<script>
    import { statsFilterStore } from '../../stores/statsStore.js';
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

    /** A rate is measured only with decisions behind it; unmeasured sorts last and fails a filter. */
    const rated = (/** @type {string} */ key, /** @type {string} */ count) => (/** @type {any} */ row) => (row[count] > 0 ? row[key] : null);

    /**
     * Columns in display order, each with the value it sorts and filters by. First click
     * sorts in the natural direction (rate ascending, count descending).
     */
    const COLUMNS = [
        { key: 'compare', labelKey: 'stats.compareColumn', hintKey: 'stats.compareColumnHint', align: 'left', plain: true },
        { key: 'name', labelKey: 'stats.playersColPlayer', align: 'left', filter: 'text', value: (/** @type {any} */ r) => r.name ?? '' },
        { key: 'matches', labelKey: 'stats.playersColMatches', filter: 'number' },
        { key: 'record', labelKey: 'stats.playersColRecord', value: (/** @type {any} */ r) => r.wins - r.losses },
        { key: 'decisions', labelKey: 'stats.playersColDecisions', filter: 'number' },
        { key: 'pr', labelKey: 'stats.playersColPR', hintKey: 'stats.playersColPRHint', rate: true, filter: 'number', value: rated('pr', 'decisions') },
        { key: 'pr_checker', labelKey: 'stats.playersColPRChecker', hintKey: 'stats.playersColPRHint', rate: true, filter: 'number', value: rated('pr_checker', 'checker_decisions') },
        { key: 'pr_cube', labelKey: 'stats.playersColPRCube', hintKey: 'stats.playersColPRHint', rate: true, filter: 'number', value: rated('pr_cube', 'cube_decisions') },
        { key: 'snowie_er', labelKey: 'stats.playersColSnowie', rate: true, filter: 'number', value: rated('snowie_er', 'decisions') },
        { key: 'mwc7', labelKey: 'mwc7.short', hintKey: 'mwc7.name', rate: true, filter: 'number', value: (/** @type {any} */ r) => mwc7Percent(r.mwc7) },
        { key: 'blunders', labelKey: 'stats.playersColBlunders', filter: 'number' },
        { key: 'luck', labelKey: 'stats.playersColLuck', hintKey: 'stats.playersColLuckHint', rate: true, filter: 'number', value: (/** @type {any} */ r) => (r.luck_known ? r.luck_rate_mp : null) }
    ];

    const columns = $derived(
        COLUMNS.map((col) => ({
            key: col.key,
            label: $t(col.labelKey),
            title: col.hintKey ? $t(col.hintKey) : undefined,
            sortable: !col.plain,
            align: col.align === 'left' ? 'left' : 'right',
            defaultDir: col.rate || col.key === 'name' ? 'asc' : 'desc',
            filter: /** @type {'number' | 'text' | undefined} */ (col.filter),
            value: col.value ?? ((/** @type {any} */ r) => r[col.key] ?? 0)
        }))
    );

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
        statsFilterStore.update((f) => ({ ...f, playerName: name }));
        onSelectPlayer?.(name);
    }

    function onRowKey(/** @type {KeyboardEvent} */ event, /** @type {string} */ name) {
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
            {rows}
            viewId="stats.players"
            rowKey={(row) => row.name}
            {columns}
            sort={{ column: 'pr', direction: 'asc' }}
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
                <td class="numeric">{row.matches}</td>
                <td class="numeric">{row.wins}–{row.losses}</td>
                <td class="numeric">{row.decisions}</td>
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
