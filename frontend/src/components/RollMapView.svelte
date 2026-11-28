<script>
    // The roll map: the 21 rolls of a position, best play and equity of each. The position is the
    // one the trail leads to; a roll selected stays on it, its dice and play previewed on the board
    // (rollPreviewStore), until a descent plays that play (or another) and shows the replies. Only the view on screen computes in the foreground: the map shares one search slot
    // with the panel's own evaluation, so a newer start supersedes the last. Once the map is done,
    // the maps one roll below it are prefetched in the background (every reply at 0 then 1 ply,
    // the row selected at the map's depth), so that descending answers at once; leaving the map
    // cancels that prefetch.
    import { onMount, onDestroy, untrack } from 'svelte';
    import { EventsOn } from '../../wailsjs/runtime/runtime.js';
    import { StartRollMap, CancelRollMap, PrefetchRollMaps, CancelRollMapPrefetch, EvaluatePositionImmediate } from '../../wailsjs/go/gui/App.js';
    import { GetGammonNetDisplayPly, GetGammonNetPruneK, GetGammonNetCandidates } from '../../wailsjs/go/main/Config.js';
    import { positionStore } from '../stores/positionStore';
    import { rollTrailStore, rollPreviewStore } from '../stores/mapStore.js';
    import { descend, climbTo } from '../services/mapService.js';
    import { evalCache, evalCacheKey, rankMoves, plyOf } from '../services/evalCache.js';
    import { ROLL_COUNT, rollStats, sortRows, deviationScale, signed, rollLabel, rollIndex } from '../utils/rollMap.js';
    import MiniDie from './MiniDie.svelte';
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
    // The other plays of the selected roll, shown under its row while expanded; hovered is the
    // one whose arrows the board draws instead of the best play's.
    let expanded = $state(false);
    let others = $state(/** @type {Array<{ move: string, gap: number, ply: number }> | null} */ (null));
    let hovered = $state(/** @type {string | null} */ (null));
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
        hovered = null;
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
        rollPreviewStore.set(null);
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

    // The board stays on the position before the roll: the roll's dice and the play are previewed.
    $effect(() => {
        const row = active ? selectedRow : null;
        const play = hovered;
        rollPreviewStore.set(row ? { dice: [row.die1, row.die2], move: play ?? (row.dance ? '' : (row.move ?? '')) } : null);
    });

    // The other plays follow the selection while the list is open.
    $effect(() => {
        const focus = selected;
        const open = expanded;
        others = null;
        if (!open || focus < 0) return;
        untrack(() => loadOthers(focus));
    });

    /** @param {any} row */
    function pick(row) {
        if (selected !== row.index) hovered = null;
        selected = row.index;
    }

    /** Descend with the selected roll's best play, or with play. @param {string} [play] */
    function replies(play) {
        const row = selectedRow;
        if (!row) return;
        selected = -1;
        hovered = null;
        descend(row.die1, row.die2, play ?? row.move ?? '');
    }

    /** @param {any} row */
    function descendRow(row) {
        pick(row);
        replies();
    }

    /** @param {MouseEvent} event */
    function toggleOthers(event) {
        event.stopPropagation();
        expanded = !expanded;
    }

    /** The legal plays of the selected roll but its best, each measured against it on one search. */
    /** @param {number} index */
    async function loadOthers(index) {
        const row = rows[index];
        if (!row || row.dance) return;
        const asked = request;
        const rolled = { ...JSON.parse(JSON.stringify(shown)), id: 0, dice: [row.die1, row.die2] };
        /** @type {any} */
        let entry = null;
        try {
            const [pruneK, candidates] = await Promise.all([GetGammonNetPruneK(), GetGammonNetCandidates()]);
            const result = await EvaluatePositionImmediate(/** @type {any} */ (rolled), pruneK, candidates);
            // A deeper ranking of this roll, computed earlier in the Eval panel, takes precedence.
            entry = result?.refused ? null : evalCache.put(evalCacheKey(rolled, { pruneK, candidates }), result);
        } catch (e) {
            logger.error('roll map: other plays:', e);
        }
        if (asked !== request || selected !== index || !expanded) return;
        const moves = rankMoves(entry?.moves ?? []);
        const best = moves.find((m) => m.move === row.move) ?? moves[0];
        others = moves.filter((m) => m !== best).map((m) => ({ move: m.move, gap: m.equity - best.equity, ply: plyOf(m.analysisDepth) }));
    }

    /** Enter descends, Escape drops the selection. @param {KeyboardEvent} event */
    function onKey(event) {
        if (event.ctrlKey || event.metaKey || event.altKey || event.shiftKey) return;
        if (event.key === 'Enter' && selectedRow) {
            event.preventDefault();
            event.stopPropagation();
            replies();
        } else if (event.key === 'Escape' && selected >= 0) {
            event.preventDefault();
            event.stopPropagation();
            selected = -1;
            hovered = null;
        }
    }

    /** Who plays the n-th link of the trail. @param {number} i */
    const side = (i) => (i % 2 === 0 ? 'me' : 'him');

    /** One link of the trail, for assistive technology: who played, the roll and its play. */
    /** @param {any} step @param {number} i */
    const crumb = (step, i) => `${i % 2 === 0 ? $t('maps.rolls.me') : $t('maps.rolls.him')} ${step.die1}-${step.die2} ${step.move || $t('maps.rolls.dance')}`;
</script>

<!-- Focusable so that a click on a row leaves Enter and Escape here. -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="roll-map" tabindex="-1" onkeydown={onKey}>
    <nav class="trail" aria-label={$t('maps.rolls.trail')}>
        <button type="button" class="crumb" onclick={() => climbTo(0)} disabled={!trail.steps.length}>{$t('maps.rolls.start')}</button>
        {#each trail.steps as step, i (i)}
            <span class="sep" aria-hidden="true">›</span>
            <button type="button" class="crumb step {side(i)}" onclick={() => climbTo(i + 1)} disabled={i === trail.steps.length - 1} title={crumb(step, i)} aria-label={crumb(step, i)}>
                <i class="side" aria-hidden="true"></i><MiniDie value={step.die1} /><MiniDie value={step.die2} />
            </button>
        {/each}
        <span class="sep" aria-hidden="true">›</span>
        <span
            class="crumb step pending {mover ? 'me' : 'him'}"
            aria-current="true"
            title={selectedRow ? `${mover ? $t('maps.rolls.myRolls') : $t('maps.rolls.hisRolls')} ${rollLabel(selectedRow)}` : mover ? $t('maps.rolls.myRolls') : $t('maps.rolls.hisRolls')}
        >
            <i class="side" aria-hidden="true"></i><MiniDie value={selectedRow?.die1 ?? 0} /><MiniDie value={selectedRow?.die2 ?? 0} />
        </span>
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
                        {@const open = selected === row.index && expanded && !row.dance}
                        <tr
                            class:selected={selected === row.index}
                            class:pending={row.ply < targetPly}
                            class:ready={ready[row.index] >= targetPly}
                            onclick={() => pick(row)}
                            ondblclick={() => descendRow(row)}
                        >
                            <td class="roll">{rollLabel(row)}</td>
                            <td class="move">
                                {#if selected === row.index && !row.dance}
                                    <button
                                        type="button"
                                        class="chevron"
                                        aria-expanded={open}
                                        title={$t('maps.rolls.otherPlays')}
                                        aria-label={$t('maps.rolls.otherPlays')}
                                        onclick={toggleOthers}
                                        ondblclick={(e) => e.stopPropagation()}>{open ? '▾' : '▸'}</button
                                    >
                                {/if}
                                {row.dance ? $t('maps.rolls.dance') : row.move}
                            </td>
                            <td class="num">{row.equity.toFixed(3)}</td>
                            <td class="gap">
                                <span class="bar" aria-hidden="true"><i class:neg={dev < 0} style={`width:${Math.abs(level(row.equity)) * 50}%`}></i></span>
                                <span class="figure">{signed(dev)}</span>
                            </td>
                            <td class="mark"
                                >{#if row.marketLost}<span title={$t('maps.rolls.marketTip')}>{$t('maps.verdict.short.double_pass')}</span>{/if}</td
                            >
                        </tr>
                        {#if open}
                            {#each others ?? [] as play (play.move)}
                                <tr
                                    class="other"
                                    class:hovered={hovered === play.move}
                                    onmouseenter={() => (hovered = play.move)}
                                    onmouseleave={() => (hovered = null)}
                                    onclick={() => replies(play.move)}
                                >
                                    <td></td>
                                    <td class="move">{play.move}</td>
                                    <td class="num ply">{play.ply >= 0 ? `${play.ply}-ply` : ''}</td>
                                    <td class="gap"><span class="figure">{signed(play.gap)}</span></td>
                                    <td></td>
                                </tr>
                            {/each}
                        {/if}
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
                                ondblclick={() => descendRow(row)}
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
                <button type="button" onclick={() => replies()}>{$t('maps.rolls.replies')} ›</button>
            </div>
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
    .roll-map:focus {
        outline: none;
    }
    .crumb.step {
        --side: var(--map-play-1);
        display: inline-flex;
        align-items: center;
        gap: 2px;
        padding: 1px 3px;
        border: 1px solid transparent;
        border-radius: var(--radius);
        color: var(--color-text);
        text-decoration: none;
    }
    .crumb.step.him {
        --side: var(--map-play-2);
    }
    .crumb.step:not(:disabled):hover {
        border-color: var(--color-border);
    }
    .crumb.step.pending {
        border: 1px dashed var(--side);
        font-weight: 400;
    }
    .side {
        width: 4px;
        align-self: stretch;
        border-radius: 2px;
        background: var(--side);
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
    .actions button {
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        padding: 2px var(--space-2);
        cursor: pointer;
    }
    .chevron {
        background: none;
        border: none;
        padding: 0 2px;
        color: var(--color-text);
        cursor: pointer;
    }
    .rows tr.other td {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }
    .rows tr.other td.move {
        padding-left: calc(6px + 1.2em);
    }
    .rows tr.other.hovered td,
    .rows tr.other:hover td {
        color: var(--color-text);
    }
    .ply {
        font-size: var(--font-size-small);
    }
    .error {
        color: var(--color-danger);
    }
    .empty {
        color: var(--color-text-muted);
    }
</style>
