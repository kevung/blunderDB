<script>
    // The score map: the position's best decision at every score of a match of N points. Only the
    // view on screen computes (the map shares the panel's search slot); hovering a cell shows its
    // play on the board, clicking one opens its detail and, from there, evaluates at that score.
    import { onMount, onDestroy, untrack } from 'svelte';
    import { EventsOn } from '../../wailsjs/runtime/runtime.js';
    import { StartScoreMap, CancelScoreMap } from '../../wailsjs/go/gui/App.js';
    import { GetGammonNetDisplayPly, GetGammonNetPruneK, GetMapMatchLength } from '../../wailsjs/go/main/Config.js';
    import { positionStore } from '../stores/positionStore';
    import { selectedMoveStore } from '../stores/analysisStore';
    import { evalViewStore, mapMatchLengthStore, EVAL_VIEW } from '../stores/mapStore.js';
    import { sendPositionToEval } from '../services/modeMachine.js';
    import { gridCells, playGroups, cellKey, cellMargin, tint, positionCell, positionAtScore, VERDICTS, VERDICT_LABELS } from '../utils/scoreMap.js';
    import { signed } from '../utils/rollMap.js';
    import { logger } from '../utils/logger.js';
    import { t } from '../i18n';

    let { active = true } = $props();

    const MAX_PLY = 2;
    const LENGTHS = [3, 5, 7, 9, 11, 13, 15];

    let configured = $state(9);
    let matchLength = $derived($mapMatchLengthStore ?? configured);
    let cells = $state(/** @type {Array<any>} */ ([]));
    let decision = $state(/** @type {string} */ (''));
    let error = $state('');
    let targetPly = $state(MAX_PLY);
    let focus = $state(/** @type {string | null} */ (null));
    let detail = $state(/** @type {any} */ (null));
    let tag = 0;

    let request = $derived(JSON.stringify([$positionStore, matchLength]));

    $effect(() => {
        const key = request;
        const on = active;
        if (!on || !key) return;
        untrack(start);
        return () => {
            tag++;
            CancelScoreMap().catch(() => {});
            selectedMoveStore.set(null);
        };
    });

    async function start() {
        const mine = ++tag;
        cells = Array(matchLength * matchLength).fill(null);
        decision = '';
        error = '';
        focus = null;
        detail = null;
        let ply = MAX_PLY;
        let pruneK = 0;
        try {
            ply = Math.min(MAX_PLY, Math.max(0, await GetGammonNetDisplayPly()));
            pruneK = await GetGammonNetPruneK();
        } catch (e) {
            logger.error('score map settings:', e);
        }
        if (mine !== tag) return;
        targetPly = ply;
        try {
            await StartScoreMap(/** @type {any} */ ($positionStore), matchLength, ply, pruneK, mine);
        } catch (e) {
            if (mine === tag) error = String(e);
        }
    }

    /** @type {Array<() => void>} */
    let off = [];
    onMount(() => {
        GetMapMatchLength()
            .then((n) => (configured = n))
            .catch(() => {});
        off = [
            EventsOn('gammonnet-scoremap:cell', (e) => {
                if (e?.tag !== tag) return;
                cells[e.index] = e.cell;
            }),
            EventsOn('gammonnet-scoremap:done', (e) => {
                if (e?.tag !== tag) return;
                decision = e.map?.decision ?? '';
                cells = e.map?.cells ?? cells;
            }),
            EventsOn('gammonnet-scoremap:cancelled', () => {}),
            EventsOn('gammonnet-scoremap:error', (e) => {
                if (e?.tag !== tag) return;
                error = String(e?.message ?? e);
            })
        ];
    });
    onDestroy(() => {
        tag++;
        off.forEach((f) => f && f());
        CancelScoreMap().catch(() => {});
        selectedMoveStore.set(null);
    });

    // A position with dice is a checker decision; the done event confirms what the cells say.
    let kind = $derived(decision || (($positionStore?.dice?.[0] ?? 0) > 0 && ($positionStore?.dice?.[1] ?? 0) > 0 ? 'checker' : 'cube'));
    let present = $derived(cells.filter(Boolean));
    let groups = $derived(playGroups(present, kind));
    let byKey = $derived(new Map(groups.map((g) => [g.key, g])));
    let grid = $derived(gridCells(cells, matchLength));
    let here = $derived(positionCell($positionStore, matchLength));
    let refined = $derived(present.filter((c) => c.ply >= targetPly).length);

    /** @param {number} away */
    const axisLabel = (away) => String(away);

    /** @param {any} cell */
    function groupOf(cell) {
        return byKey.get(cellKey(cell, kind));
    }

    /** The cell's own label: the play's letter, or the verdict's short form.
     * @param {any} cell */
    function glyph(cell) {
        if (cell.refused) return '—';
        return kind === 'cube' ? $t(VERDICT_LABELS[cell.verdict].short) : (groupOf(cell)?.letter ?? '');
    }

    /** @param {any} cell */
    function slotStyle(cell) {
        const group = groupOf(cell);
        if (!group) return '';
        const colour = group.slot >= 0 ? `var(--map-play-${group.slot + 1})` : 'var(--map-play-other)';
        const share = { strong: 60, soft: 30, close: 12 }[tint(cellMargin(cell, kind))];
        return `background: color-mix(in srgb, ${colour} ${share}%, var(--color-surface))`;
    }

    /** @param {any} cell */
    const dimmed = (cell) => focus !== null && cellKey(cell, kind) !== focus;
    /** @param {any} cell */
    const isHere = (cell) => !!here && cell.awayOnRoll === here.mine && cell.awayOpponent === here.theirs;
    /** @param {any} cell */
    const close = (cell) => tint(cellMargin(cell, kind)) === 'close';

    /** @param {any} cell */
    function hover(cell) {
        selectedMoveStore.set(kind === 'checker' && cell?.move ? cell.move : null);
    }

    /** @param {string} key */
    function toggleFocus(key) {
        focus = focus === key ? null : key;
    }

    /** @param {any} cell */
    function evaluateAt(cell) {
        const next = positionAtScore($positionStore, cell);
        evalViewStore.set(EVAL_VIEW.DECISION);
        sendPositionToEval(next);
    }

    /** @param {Event & { currentTarget: HTMLSelectElement }} e */
    function setLength(e) {
        mapMatchLengthStore.set(Number(e.currentTarget.value));
    }
</script>

<div class="score-map">
    <div class="head">
        <label>
            {$t('maps.scores.match', { n: '' }).trim()}
            <select value={matchLength} onchange={setLength}>
                {#each LENGTHS as n (n)}
                    <option value={n}>{n}</option>
                {/each}
            </select>
        </label>
        <span class="progress" aria-live="polite">{$t('maps.scores.progress', { ply: targetPly, n: refined, total: matchLength * matchLength })}</span>
    </div>

    {#if error}
        <div class="error" role="alert">{$t('maps.scores.error')} : {error}</div>
    {:else}
        <div class="body">
            <table class="grid" aria-label={$t('maps.view.scores')}>
                <thead>
                    <tr>
                        <th class="corner">{$t('maps.scores.me')} ↓ · {$t('maps.scores.opponent')} →</th>
                        {#each grid[0] ?? [] as _, j (j)}
                            <th scope="col">{axisLabel(matchLength - j)}{matchLength - j === 1 ? ' C' : ''}</th>
                        {/each}
                    </tr>
                </thead>
                <tbody>
                    {#each grid as line, i (i)}
                        <tr>
                            <th scope="row">{axisLabel(matchLength - i)}{matchLength - i === 1 ? ' C' : ''}</th>
                            {#each line as cell, j (j)}
                                {#if cell}
                                    <td>
                                        <button
                                            type="button"
                                            class="cell"
                                            class:here={isHere(cell)}
                                            class:pending={cell.ply < targetPly}
                                            class:dim={dimmed(cell)}
                                            style={slotStyle(cell)}
                                            onmouseenter={() => hover(cell)}
                                            onmouseleave={() => hover(null)}
                                            onfocus={() => hover(cell)}
                                            onblur={() => hover(null)}
                                            onclick={() => (detail = cell)}
                                            title={`${cell.awayOnRoll}-${cell.awayOpponent}${cell.crawford ? ' ' + $t('maps.scores.crawford') : ''}`}
                                        >
                                            <span class="glyph">{close(cell) && !cell.refused ? '≈ ' : ''}{glyph(cell)}</span>
                                            {#if kind === 'checker' && !cell.refused}
                                                <span class="equity">{signed(cell.equity, 2)}</span>
                                            {/if}
                                        </button>
                                    </td>
                                {:else}
                                    <td><span class="cell blank"></span></td>
                                {/if}
                            {/each}
                        </tr>
                    {/each}
                </tbody>
            </table>

            <div class="side">
                <ul class="legend" aria-label={$t('maps.scores.legend')}>
                    {#each groups as group (group.key)}
                        <li>
                            <button type="button" class="key" aria-pressed={focus === group.key} onclick={() => toggleFocus(group.key)}>
                                <span
                                    class="chip"
                                    style={`background: color-mix(in srgb, ${group.slot >= 0 ? `var(--map-play-${group.slot + 1})` : 'var(--map-play-other)'} 60%, var(--color-surface))`}
                                >
                                    {kind === 'cube' ? $t(VERDICT_LABELS[group.key].short) : group.letter}
                                </span>
                                <span class="name">{kind === 'cube' ? $t(VERDICT_LABELS[group.key].long) : group.key}</span>
                                <span class="count">{group.count}</span>
                            </button>
                        </li>
                    {/each}
                </ul>

                {#if detail}
                    <div class="detail">
                        <div class="detail-head">
                            <b>{detail.awayOnRoll}-{detail.awayOpponent}{detail.crawford ? ` ${$t('maps.scores.crawford')}` : ''}</b>
                            <button type="button" onclick={() => (detail = null)} aria-label={$t('maps.scores.close')}>×</button>
                        </div>
                        {#if detail.refused}
                            <div>{$t('maps.scores.refused')}</div>
                        {:else if kind === 'cube'}
                            {#each VERDICTS.slice(0, 3) as v (v)}
                                {@const value = v === 'no_double' ? detail.noDouble : v === 'double_take' ? detail.doubleTake : detail.doublePass}
                                <div class:best={detail.verdict === v}>{$t(VERDICT_LABELS[v].long)} <span class="num">{value.toFixed(3)}</span></div>
                            {/each}
                            {#if detail.verdict === 'too_good'}<div class="best">{$t('maps.verdict.long.too_good')}</div>{/if}
                        {:else}
                            <div class="best">{$t('maps.scores.best')} {detail.move} <span class="num">{detail.equity.toFixed(3)}</span></div>
                            {#if detail.secondMove}
                                <div>{$t('maps.scores.second')} {detail.secondMove} <span class="num">{detail.secondEquity?.toFixed(3)}</span></div>
                            {/if}
                        {/if}
                        <button type="button" class="evaluate" onclick={() => evaluateAt(detail)}>{$t('maps.scores.evaluate')}</button>
                    </div>
                {/if}
            </div>
        </div>
    {/if}
</div>

<style>
    .score-map {
        display: flex;
        flex-direction: column;
        gap: var(--space-2);
        min-height: 0;
        overflow: auto;
        padding: var(--space-2);
        color: var(--color-text);
        font-size: var(--font-size-base);
    }
    .head {
        display: flex;
        gap: var(--space-3);
        align-items: center;
    }
    .head select {
    }
    .progress {
        margin-left: auto;
        color: var(--color-text-muted);
    }
    .body {
        display: flex;
        flex-wrap: wrap;
        gap: var(--space-3);
        align-items: flex-start;
    }
    .grid {
        border-collapse: separate;
        border-spacing: 2px;
        font-size: var(--font-size-small);
    }
    .grid th {
        font-weight: 600;
        color: var(--color-text-muted);
        padding: 0 4px;
    }
    .corner {
        text-align: left;
        font-weight: 400;
    }
    .cell {
        font-size: var(--font-size-small);
        width: 44px;
        height: 38px;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        border: 2px solid transparent;
        border-radius: var(--radius);
        background: var(--color-surface-alt);
        color: var(--color-text);
        padding: 0;
        cursor: pointer;
    }
    .cell.blank {
        cursor: default;
    }
    .cell.pending {
        border-style: dashed;
        border-color: var(--color-text-muted);
    }
    .cell.here {
        border: 2px solid var(--color-text);
        outline: 2px solid var(--color-text);
        outline-offset: 1px;
    }
    .cell.dim {
        opacity: 0.3;
    }
    .glyph {
        font-weight: 700;
    }
    .equity {
        font-family: var(--font-family-mono);
        color: var(--color-text-muted);
    }
    .side {
        display: flex;
        flex-direction: column;
        gap: var(--space-3);
        min-width: 160px;
    }
    .legend {
        list-style: none;
        margin: 0;
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
    }
    .key {
        display: flex;
        gap: var(--space-2);
        align-items: center;
        width: 100%;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        padding: 2px var(--space-2);
        cursor: pointer;
        text-align: left;
    }
    .key[aria-pressed='true'] {
        outline: 2px solid var(--color-primary);
        font-weight: 700;
    }
    .chip {
        min-width: 28px;
        text-align: center;
        font-weight: 700;
        border-radius: var(--radius);
        padding: 0 2px;
    }
    .name {
        flex: 1;
    }
    .count {
        color: var(--color-text-muted);
    }
    .detail {
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        padding: var(--space-2);
    }
    .detail-head {
        display: flex;
        justify-content: space-between;
    }
    .detail-head button,
    .evaluate {
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }
    .best {
        font-weight: 700;
    }
    .num {
        font-family: var(--font-family-mono);
    }
    .error {
        color: var(--color-danger);
    }
</style>
