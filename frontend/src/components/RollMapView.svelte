<script>
    // The roll map: the 21 rolls of a position, best play and equity of each. The position is the
    // one the trail leads to; descending plays a roll's best play (or another) and shows the
    // replies. Only the view on screen computes in the foreground: the map shares one search slot
    // with the panel's own evaluation, so a newer start supersedes the last. Once the map is done,
    // the maps one roll below it are prefetched in the background (every reply at 0 then 1 ply,
    // the row selected at the map's depth), so that descending answers at once; leaving the map
    // cancels that prefetch.
    import { onMount, onDestroy, untrack } from 'svelte';
    import { EventsOn } from '../../wailsjs/runtime/runtime.js';
    import { StartRollMap, CancelRollMap, PrefetchRollMaps, CancelRollMapPrefetch, LegalMoves } from '../../wailsjs/go/gui/App.js';
    import { GetGammonNetDisplayPly, GetGammonNetPruneK } from '../../wailsjs/go/main/Config.js';
    import { positionStore } from '../stores/positionStore';
    import { rollTrailStore } from '../stores/mapStore.js';
    import { descend, climbTo } from '../services/mapService.js';
    import { ROLL_COUNT, rollStats, sortRows, deviationScale, signed, rollLabel, rollIndex } from '../utils/rollMap.js';
    import { positionAfterPlay } from '../utils/moveNotation.js';
    import { nextMapTag } from '../utils/mapTag.js';
    import { logger } from '../utils/logger.js';
    import { t } from '../i18n';

    let { active = true } = $props();

    const MAX_PLY = 2;

    let rows = $state(/** @type {Array<any>} */ (Array(ROLL_COUNT).fill(null)));
    let done = $state(/** @type {any} */ (null));
    let error = $state('');
    let targetPly = $state(MAX_PLY);
    let sortKey = $state(/** @type {'roll' | 'move' | 'equity' | 'gap'} */ ('roll'));
    let descending = $state(false);
    let marketOnly = $state(false);
    let selected = $state(-1);
    let others = $state(/** @type {Array<any> | null} */ (null));
    // ready[i] is the depth the replies to row i are prefetched to; a row is marked once they
    // stand at the map's own depth, the one a descent shows.
    let ready = $state(/** @type {Array<number>} */ (Array(ROLL_COUNT).fill(-1)));
    let pruneK = 0;
    let tag = nextMapTag();

    let trail = $derived($rollTrailStore);
    let last = $derived(trail.steps.length ? trail.steps[trail.steps.length - 1] : null);
    let shown = $derived(last ? last.after : $positionStore);
    // What the map is of: the board on screen, the trail's last one when it has a play.
    let request = $derived(JSON.stringify(shown));

    $effect(() => {
        const key = request;
        const on = active;
        if (!on || !key) return;
        untrack(start);
        return () => {
            tag = nextMapTag();
            CancelRollMap().catch(() => {});
            CancelRollMapPrefetch().catch(() => {});
        };
    });

    // The prefetch follows the row selected: its replies alone go past 1 ply.
    $effect(() => {
        const map = done;
        const focus = selected;
        if (!map || !active) return;
        untrack(() => prefetch(map, focus));
    });

    /** @param {any} map @param {number} focus */
    function prefetch(map, focus) {
        PrefetchRollMaps(/** @type {any} */ (shown), map.rows ?? [], targetPly, pruneK, focus, tag).catch((/** @type {any} */ e) => logger.error('roll map prefetch:', e));
    }

    async function start() {
        const mine = (tag = nextMapTag());
        rows = Array(ROLL_COUNT).fill(null);
        done = null;
        error = '';
        selected = -1;
        others = null;
        ready = Array(ROLL_COUNT).fill(-1);
        let ply = MAX_PLY;
        pruneK = 0;
        try {
            ply = Math.min(MAX_PLY, Math.max(0, await GetGammonNetDisplayPly()));
            pruneK = await GetGammonNetPruneK();
        } catch (e) {
            logger.error('roll map settings:', e);
        }
        if (mine !== tag) return;
        targetPly = ply;
        try {
            await StartRollMap(/** @type {any} */ (shown), ply, pruneK, mine);
        } catch (e) {
            if (mine === tag) error = String(e);
        }
    }

    /** @type {Array<() => void>} */
    let off = [];
    onMount(() => {
        off = [
            EventsOn('gammonnet-rolls:row', (e) => {
                if (e?.tag !== tag) return;
                rows[e.index] = { ...e.row, index: e.index };
            }),
            EventsOn('gammonnet-rolls:done', (e) => {
                if (e?.tag !== tag) return;
                done = e.map;
                (e.map?.rows ?? []).forEach((/** @type {any} */ row, /** @type {number} */ i) => (rows[i] = { ...row, index: i }));
            }),
            EventsOn('gammonnet-rolls:prefetched', (e) => {
                if (e?.tag !== tag) return;
                ready[e.index] = Math.max(ready[e.index], e.ply);
            }),
            EventsOn('gammonnet-rolls:cancelled', () => {}),
            EventsOn('gammonnet-rolls:error', (e) => {
                if (e?.tag !== tag) return;
                error = String(e?.message ?? e);
            })
        ];
    });
    onDestroy(() => {
        tag = nextMapTag();
        off.forEach((f) => f && f());
        CancelRollMap().catch(() => {});
        CancelRollMapPrefetch().catch(() => {});
    });

    let present = $derived(rows.filter(Boolean));
    let stats = $derived(rollStats(present, done?.average));
    let spread = $derived(Math.max(0, ...present.map((r) => Math.abs(r.equity - stats.average))));
    let cubeAccess = $derived(done ? !!done.cubeAccess : present.some((r) => r.marketLost !== undefined && r.marketLost !== null));
    let visible = $derived(sortRows(marketOnly ? present.filter((r) => r.marketLost) : present, sortKey, descending, stats.average));
    let refined = $derived(present.filter((r) => r.ply >= targetPly).length);
    let mover = $derived(trail.steps.length % 2 === 0);

    /** @param {'roll' | 'move' | 'equity' | 'gap'} key */
    function sortBy(key) {
        if (sortKey === key) descending = !descending;
        else {
            sortKey = key;
            descending = key === 'equity';
        }
    }
    /** @param {string} key */
    const ariaSort = (key) => (sortKey === key ? (descending ? 'descending' : 'ascending') : 'none');

    /** @param {number} equity */
    function level(equity) {
        return deviationScale(equity, stats.average, spread);
    }
    /** The cell's background: grey at the mean, towards orange below and blue above. */
    /** @param {number} equity */
    function shade(equity) {
        const v = level(equity);
        const colour = v < 0 ? 'var(--map-worse)' : 'var(--map-better)';
        return `background: color-mix(in srgb, ${colour} ${Math.round(Math.abs(v) * 70)}%, var(--map-neutral))`;
    }

    let selectedRow = $derived(selected >= 0 ? rows[selected] : null);

    /** @param {any} row */
    function pick(row) {
        selected = selected === row.index ? -1 : row.index;
        others = null;
    }

    function replies() {
        const row = selectedRow;
        if (row) descend(row.die1, row.die2, row.move ?? '');
    }

    /** @param {any} board */
    const boardKey = (board) => JSON.stringify([(board?.points ?? []).map((/** @type {any} */ p) => (p.checkers ? [p.color, p.checkers] : 0)), board?.bearoff ?? []]);

    /** The legal plays of the selected roll but its best one, told apart by the board they leave. */
    async function chooseOther() {
        const row = selectedRow;
        if (!row || row.dance) return;
        const asked = request;
        const rolled = { ...shown, dice: [row.die1, row.die2] };
        const best = row.move ? positionAfterPlay(rolled, row.die1, row.die2, row.move) : null;
        const bestKey = best ? boardKey(best.board) : null;
        /** @type {any[]} */
        let plays = [];
        try {
            plays = (await LegalMoves(/** @type {any} */ (rolled))) ?? [];
        } catch (e) {
            logger.error('roll map: legal plays:', e);
        }
        if (asked !== request || selectedRow !== row) return;
        others = plays.filter((/** @type {any} */ p) => boardKey(p.result?.board) !== bestKey);
    }

    /** @param {any} play a domain.LegalPlay */
    function playOther(play) {
        const row = selectedRow;
        if (row) descend(row.die1, row.die2, play.notation, play.result?.board);
    }

    /** One link of the trail: who played, the roll and its play. */
    /** @param {any} step @param {number} i */
    const crumb = (step, i) => `${i % 2 === 0 ? $t('maps.rolls.me') : $t('maps.rolls.him')} ${step.die1}-${step.die2} ${step.move || $t('maps.rolls.dance')}`;
</script>

<div class="roll-map">
    <nav class="trail" aria-label={$t('maps.rolls.trail')}>
        <button type="button" class="crumb" onclick={() => climbTo(0)} disabled={!trail.steps.length}>{$t('maps.rolls.start')}</button>
        {#each trail.steps as step, i (i)}
            <span class="sep" aria-hidden="true">›</span>
            <button type="button" class="crumb" onclick={() => climbTo(i + 1)} disabled={i === trail.steps.length - 1}>{crumb(step, i)}</button>
        {/each}
        <span class="sep" aria-hidden="true">›</span>
        <span class="crumb current" aria-current="true">{mover ? $t('maps.rolls.myRolls') : $t('maps.rolls.hisRolls')}</span>
    </nav>

    {#if error}
        <div class="error" role="alert">{$t('maps.rolls.error')} : {error}</div>
    {:else if !present.length}
        <div class="empty">{$t('maps.rolls.computing')}…</div>
    {:else}
        <div class="summary">
            <span><b>{$t('maps.rolls.average')}</b> {stats.average.toFixed(3)}</span>
            {#if cubeAccess}
                <button type="button" class="market" aria-pressed={marketOnly} title={$t('maps.rolls.marketTip')} onclick={() => (marketOnly = !marketOnly)}>
                    {$t('maps.rolls.market')}
                    {stats.marketLost}/36
                </button>
            {/if}
            <span>
                <b>{$t('maps.rolls.best')}</b>
                {stats.best ? `${rollLabel(stats.best)} ${stats.best.equity.toFixed(3)}` : ''} ·
                <b>{$t('maps.rolls.worst')}</b>
                {stats.worst ? `${rollLabel(stats.worst)} ${stats.worst.equity.toFixed(3)}` : ''}
            </span>
            <span><b>{$t('maps.rolls.spread')}</b> {stats.stdev.toFixed(3)}</span>
            <span class="progress" aria-live="polite">{$t('maps.rolls.progress', { ply: targetPly, n: refined, total: ROLL_COUNT })}</span>
        </div>

        <div class="body">
            <table class="rows">
                <thead>
                    <tr>
                        <th aria-sort={ariaSort('roll')}><button type="button" onclick={() => sortBy('roll')}>{$t('maps.rolls.roll')}</button></th>
                        <th aria-sort={ariaSort('move')}><button type="button" onclick={() => sortBy('move')}>{$t('maps.rolls.move')}</button></th>
                        <th class="num" aria-sort={ariaSort('equity')}><button type="button" onclick={() => sortBy('equity')}>{$t('maps.rolls.equity')}</button></th>
                        <th aria-sort={ariaSort('gap')}><button type="button" onclick={() => sortBy('gap')}>{$t('maps.rolls.gap')}</button></th>
                        <th>{cubeAccess ? $t('maps.rolls.market') : ''}</th>
                    </tr>
                </thead>
                <tbody>
                    {#each visible as row (row.index)}
                        {@const dev = row.equity - stats.average}
                        <tr class:selected={selected === row.index} class:pending={row.ply < targetPly} class:ready={ready[row.index] >= targetPly} onclick={() => pick(row)}>
                            <td class="roll">{rollLabel(row)}</td>
                            <td class="move">{row.dance ? $t('maps.rolls.dance') : row.move}</td>
                            <td class="num">{row.equity.toFixed(3)}</td>
                            <td class="gap">
                                <span class="bar" aria-hidden="true"><i class:neg={dev < 0} style={`width:${Math.abs(level(row.equity)) * 50}%`}></i></span>
                                <span class="figure">{signed(dev)}</span>
                            </td>
                            <td class="mark"
                                >{#if row.marketLost}<span title={$t('maps.rolls.marketTip')}>{$t('maps.verdict.short.double_pass')}</span>{/if}</td
                            >
                        </tr>
                    {/each}
                </tbody>
            </table>

            <div class="grid" role="grid" aria-label={$t('maps.view.rolls')}>
                {#each [1, 2, 3, 4, 5, 6] as a (a)}
                    {#each [1, 2, 3, 4, 5, 6] as b (b)}
                        {@const row = rows[rollIndex(a, b)]}
                        {#if row}
                            <button
                                type="button"
                                class="cell"
                                class:lost={row.marketLost}
                                class:pending={row.ply < targetPly}
                                class:ready={ready[row.index] >= targetPly}
                                class:selected={selected === row.index}
                                style={shade(row.equity)}
                                onclick={() => pick(row)}
                                title={`${rollLabel(row)} ${row.equity.toFixed(3)}`}
                            >
                                <span class="dice">{a}{b}</span>
                                <span class="dev">{signed(row.equity - stats.average, 2)}</span>
                            </button>
                        {:else}
                            <span class="cell empty-cell"></span>
                        {/if}
                    {/each}
                {/each}
            </div>
        </div>

        {#if selectedRow}
            <div class="actions">
                <b>{rollLabel(selectedRow)}</b>
                <button type="button" onclick={replies}>{$t('maps.rolls.replies')} ›</button>
                {#if !selectedRow.dance}
                    <button type="button" onclick={chooseOther}>{$t('maps.rolls.other')}</button>
                {/if}
            </div>
            {#if others}
                <ul class="others">
                    {#each others as play, i (i)}
                        <li><button type="button" onclick={() => playOther(play)}>{play.notation}</button></li>
                    {/each}
                </ul>
            {/if}
        {/if}
    {/if}
</div>

<style>
    .roll-map {
        display: flex;
        flex-direction: column;
        gap: var(--space-2);
        min-height: 0;
        overflow: auto;
        padding: var(--space-2);
        color: var(--color-text);
        font-size: var(--font-size-base);
    }
    .trail {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: var(--space-1);
    }
    .crumb {
        background: none;
        border: none;
        padding: 0 2px;
        color: var(--color-primary);
        cursor: pointer;
        text-decoration: underline;
    }
    .crumb:disabled,
    .crumb.current {
        color: var(--color-text);
        text-decoration: none;
        cursor: default;
        font-weight: 600;
    }
    .sep {
        color: var(--color-text-muted);
    }
    .summary {
        display: flex;
        flex-wrap: wrap;
        gap: var(--space-1) var(--space-3);
        align-items: center;
    }
    .progress {
        color: var(--color-text-muted);
        margin-left: auto;
    }
    .market {
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        padding: 0 var(--space-2);
        cursor: pointer;
    }
    .market[aria-pressed='true'] {
        border-color: var(--color-primary);
        font-weight: 700;
        outline: 1px solid var(--color-primary);
    }
    .body {
        display: flex;
        flex-wrap: wrap;
        gap: var(--space-3);
        align-items: flex-start;
    }
    .rows {
        border-collapse: collapse;
        font-size: var(--font-size-small);
        flex: 1 1 280px;
    }
    .rows th button {
        font-weight: 700;
        background: none;
        border: none;
        color: var(--color-text);
        cursor: pointer;
        padding: 2px 4px;
    }
    .rows th[aria-sort='ascending'] button::after {
        content: ' ▲';
    }
    .rows th[aria-sort='descending'] button::after {
        content: ' ▼';
    }
    .rows td {
        padding: 1px 6px;
        white-space: nowrap;
    }
    .rows tbody tr {
        cursor: pointer;
    }
    .rows tbody tr:hover {
        background: var(--color-row-hover);
    }
    .rows tbody tr.selected {
        outline: 2px solid var(--color-primary);
        outline-offset: -2px;
    }
    .rows tr.pending td {
        color: var(--color-text-muted);
    }
    /* The replies to this roll are prefetched: descending answers at once. */
    .rows tr.ready td.roll::after {
        content: ' ›';
        color: var(--color-text-muted);
    }
    .num {
        text-align: right;
        font-family: var(--font-family-mono);
    }
    .gap {
        display: flex;
        align-items: center;
        gap: var(--space-1);
    }
    .bar {
        position: relative;
        display: inline-block;
        width: 60px;
        height: 8px;
        background: var(--color-surface-alt);
    }
    .bar::before {
        content: '';
        position: absolute;
        left: 50%;
        top: -2px;
        bottom: -2px;
        width: 1px;
        background: var(--color-text-muted);
    }
    .bar i {
        position: absolute;
        left: 50%;
        top: 0;
        bottom: 0;
        background: var(--map-better);
    }
    .bar i.neg {
        left: auto;
        right: 50%;
        background: var(--map-worse);
    }
    .figure {
        font-family: var(--font-family-mono);
    }
    .mark span {
        font-weight: 700;
        border: 1px solid currentColor;
        border-radius: var(--radius);
        padding: 0 3px;
    }
    .grid {
        display: grid;
        grid-template-columns: repeat(6, 44px);
        gap: 2px;
    }
    .cell {
        font-size: var(--font-size-small);
        height: 40px;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        color: var(--color-text);
        border: 2px solid transparent;
        border-radius: var(--radius);
        cursor: pointer;
        padding: 0;
    }
    .cell.empty-cell {
        cursor: default;
    }
    .cell.lost {
        border: 2px solid var(--color-text);
    }
    .cell.pending {
        border-style: dashed;
        border-color: var(--color-text-muted);
    }
    .cell.ready {
        box-shadow: inset 0 -2px 0 var(--color-text-muted);
    }
    .cell.selected {
        outline: 2px solid var(--color-primary);
    }
    .dice {
        font-weight: 700;
    }
    .dev {
        font-family: var(--font-family-mono);
    }
    .actions {
        display: flex;
        gap: var(--space-2);
        align-items: center;
    }
    .actions button,
    .others button {
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        padding: 2px var(--space-2);
        cursor: pointer;
    }
    .others {
        display: flex;
        flex-wrap: wrap;
        gap: var(--space-1);
        list-style: none;
        margin: 0;
        padding: 0;
    }
    .error {
        color: var(--color-danger);
    }
    .empty {
        color: var(--color-text-muted);
    }
</style>
