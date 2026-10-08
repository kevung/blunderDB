<script>
    // The match winning chances each decision cost, per player: a bar per
    // decision and the running total, on the decision axis of MatchTimes. Two
    // plots rather than one: a decision's loss and a total are the same unit
    // but not the same scale. Nothing is drawn for a match without analysis.
    import { t } from '../i18n';
    import { indexAt, stepIndex } from '../utils/chartAxis.js';
    import { fmtExcess, fmtLoss, lossSeries, niceCeil } from '../utils/decisionLoss.js';

    /** @type {{ losses: any[] | null, movePositions: any[], player1: string, player2: string, hovered?: number | null, onhover?: (moveId: number | null) => void, onselect?: (index: number) => void }} */
    let { losses, movePositions, player1, player2, hovered = null, onhover = () => {}, onselect = () => {} } = $props();

    const W = 320;
    const H = 56;

    let series = $derived(lossSeries(movePositions, losses));
    let n = $derived(Math.max(1, movePositions.length));
    let slot = $derived(W / n);
    let peak = $derived(niceCeil(Math.max(0, ...series.items.map((d) => Math.max(d.loss ?? 0, d.difficulty ?? 0)))));
    let cumPeak = $derived(niceCeil(Math.max(series.totals[0], series.totals[1])));
    let names = $derived([player1, player2]);

    /** @param {number} loss @param {number} top */
    const barH = (loss, top) => (loss > 0 ? Math.max(1, (loss / top) * (H - 2)) : 0);
    /** @param {number} cum */
    const cumY = (cum) => H - 1 - (cum / cumPeak) * (H - 4);
    /** One polyline per player, through the decisions that player made. */
    let lines = $derived(
        [0, 1].map((p) => {
            const pts = series.items.filter((d) => d.player === p).map((d) => ({ x: (d.index + 0.5) * slot, y: cumY(d.cum) }));
            return { p, pts, points: pts.map(({ x, y }) => `${x.toFixed(1)},${y.toFixed(1)}`).join(' '), end: pts[pts.length - 1] };
        })
    );

    // The hovered decision is shared with the time chart by its Move, not its place.
    let hover = $derived.by(() => {
        if (hovered === null) return null;
        const i = movePositions.findIndex((mp) => mp.move_id === hovered);
        return i < 0 ? null : i;
    });
    /** @param {number | null} i */
    const setHover = (i) => onhover(i === null ? null : (movePositions[i]?.move_id ?? null));
    let tip = $derived(hover === null ? null : series.items[hover]);
    let tipMove = $derived(hover === null ? null : movePositions[hover]);

    /** @param {number} v */
    const axis = (v) => (v === 0 ? '0' : `${+(v * 100).toFixed(1)} %`);

    /** @param {KeyboardEvent} e */
    function onKey(e) {
        const next = stepIndex(e.key, hover, movePositions.length);
        if (next !== null) {
            e.preventDefault();
            setHover(next);
        } else if (e.key === 'Enter' && hover !== null) {
            e.preventDefault();
            onselect(hover);
        } else if (e.key === 'Escape') {
            setHover(null);
        }
    }
</script>

{#if series.any}
    <div class="match-losses" role="group" aria-label={$t('match.lossTitle')} data-testid="match-losses">
        <table class="loss-table">
            <thead>
                <tr>
                    <th></th>
                    <th>{$t('match.lossTotal')}</th>
                    <th>{$t('match.lossScored')}</th>
                    {#if series.anyDifficulty}
                        <th title={$t('match.difficultyColTooltip')}>{$t('match.difficultyTotal')}</th>
                        <th title={$t('match.excessTooltip')}>{$t('match.excess')}</th>
                        <th title={$t('match.ratioTooltip')}>{$t('match.ratio')}</th>
                        <th title={$t('match.avoidableTooltip')}>{$t('match.avoidableCount')}</th>
                    {/if}
                </tr>
            </thead>
            <tbody>
                {#each [0, 1] as p (p)}
                    <tr>
                        <td class="loss-player">
                            <svg class="swatch" viewBox="0 0 18 6" aria-hidden="true">
                                <line class="line" class:player1={p === 0} class:player2={p === 1} x1="1" y1="3" x2="17" y2="3" />
                            </svg>{names[p]}
                        </td>
                        <td data-testid="loss-total-{p}">{fmtLoss(series.totals[p])}</td>
                        <td>{series.scored[p]}</td>
                        {#if series.anyDifficulty}
                            {@const s = series.difficulty[p]}
                            <td data-testid="difficulty-total-{p}">{fmtLoss(s.difficulty)}</td>
                            <td data-testid="excess-{p}">{fmtExcess(s.excess)}</td>
                            <td data-testid="ratio-{p}">{s.ratio === null ? '—' : s.ratio.toFixed(2)}</td>
                            <td data-testid="avoidable-{p}">{s.avoidable}</td>
                        {/if}
                    </tr>
                {/each}
            </tbody>
        </table>

        <div class="plots">
            {#each [{ key: 'per', label: $t('match.lossPerDecision'), top: peak }, { key: 'cum', label: $t('match.lossCumulative'), top: cumPeak }] as plot (plot.key)}
                <div class="plot-label">{plot.label}</div>
                <!-- A chart: walked with the arrow keys, opened with Enter. -->
                <!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
                <div
                    class="plot"
                    role="group"
                    tabindex="0"
                    aria-label={$t('match.lossChart')}
                    title={$t('match.chartOpen')}
                    data-testid="loss-plot-{plot.key}"
                    onmousemove={(e) => setHover(indexAt(e, movePositions.length))}
                    onmouseleave={() => setHover(null)}
                    onclick={(e) => onselect(indexAt(e, movePositions.length))}
                    onkeydown={onKey}
                    onblur={() => setHover(null)}
                >
                    <svg viewBox="0 0 {W} {H}" role="img" aria-label={plot.label}>
                        <line class="grid" x1="0" y1="0.5" x2={W} y2="0.5" />
                        <line class="grid base" x1="0" y1={H - 0.5} x2={W} y2={H - 0.5} />
                        {#if plot.key === 'per'}
                            {#each series.items as d (d.index)}
                                {#if d.loss === null}
                                    <rect class="unscored" x={d.index * slot} y={H - 1.5} width={Math.max(1, slot - 0.5)} height="1.5" />
                                {:else if d.loss > 0}
                                    {@const h = barH(d.loss, plot.top)}
                                    <rect
                                        class="bar"
                                        class:player1={d.player === 0}
                                        class:player2={d.player === 1}
                                        class:hot={d.index === hover}
                                        x={d.index * slot}
                                        y={H - h}
                                        width={Math.max(1, slot - 0.5)}
                                        height={h}
                                    />
                                {/if}
                                {#if d.difficulty !== null && d.difficulty > 0}
                                    {@const y = H - barH(d.difficulty, plot.top)}
                                    <line class="difficulty" x1={d.index * slot} y1={y} x2={d.index * slot + Math.max(1, slot - 0.5)} y2={y} />
                                {/if}
                                {#if d.avoidable}
                                    <circle class="avoidable" data-testid="avoidable-dot" cx={(d.index + 0.5) * slot} cy="3" r="2" />
                                {/if}
                            {/each}
                        {:else}
                            {#each lines as l (l.p)}
                                {#if l.pts.length > 0}
                                    <polyline class="line" class:player1={l.p === 0} class:player2={l.p === 1} points={l.points} />
                                    <circle class="end" class:player1={l.p === 0} class:player2={l.p === 1} cx={l.end.x} cy={l.end.y} r="2.5" />
                                {/if}
                            {/each}
                        {/if}
                        {#if plot.key === 'cum' && hover !== null && series.items[hover].loss !== null}
                            <circle class="hot-dot" cx={(hover + 0.5) * slot} cy={cumY(series.items[hover].cum)} r="3" />
                        {/if}
                        {#if hover !== null}
                            <line class="cross" x1={(hover + 0.5) * slot} y1="0" x2={(hover + 0.5) * slot} y2={H} />
                        {/if}
                    </svg>
                    <span class="y-top">{axis(plot.top)}</span>
                    {#if tip && tipMove}
                        <div class="tip" class:flip={hover !== null && hover > movePositions.length / 2} style="left: {((hover ?? 0) + 0.5) * (100 / n)}%" role="tooltip">
                            <div>{$t('match.lossWhere', { game: tipMove.game_number, move: tipMove.move_number })} · {names[tip.player]}</div>
                            <div>
                                {tip.loss === null ? $t('match.lossUnscored') : fmtLoss(tip.loss)}
                                ({tipMove.move_type === 'cube' ? $t('match.lossKindCube') : $t('match.lossKindChecker')})
                            </div>
                            {#if tip.difficulty !== null}
                                <div>{$t('match.difficultyCol')} {tip.difficulty > 0 ? fmtLoss(tip.difficulty) : '0'}{tip.avoidable ? ' · ' + $t('match.avoidable') : ''}</div>
                            {/if}
                            {#if plot.key === 'cum'}<div>{fmtLoss(tip.cum)}</div>{/if}
                        </div>
                    {/if}
                </div>
            {/each}
        </div>
    </div>
{/if}

<style>
    .match-losses {
        display: flex;
        flex-wrap: wrap;
        gap: 12px;
        align-items: flex-end;
        padding: 6px 8px;
        font-size: var(--font-size-small);
    }
    .loss-table {
        border-collapse: collapse;
    }
    .loss-table th,
    .loss-table td {
        padding: 2px 8px;
        text-align: right;
    }
    .loss-table th {
        font-weight: 500;
        color: var(--color-text-muted, inherit);
    }
    .loss-player {
        text-align: left !important;
    }
    .swatch {
        width: 18px;
        height: 6px;
        margin-right: 6px;
        vertical-align: middle;
    }
    .plots {
        width: 320px;
        max-width: 100%;
    }
    .plot-label {
        color: var(--color-text-muted, inherit);
        margin-top: 4px;
    }
    .plot {
        position: relative;
        width: 100%;
        cursor: pointer;
        outline-offset: 2px;
    }
    .plot svg {
        display: block;
        width: 100%;
        height: 56px;
        overflow: visible;
    }
    .grid {
        stroke: var(--color-border, currentColor);
        stroke-width: 1;
        opacity: 0.6;
    }
    .grid.base {
        opacity: 1;
    }
    .cross {
        stroke: var(--color-text-muted, currentColor);
        stroke-width: 1;
    }
    .unscored {
        fill: var(--color-text-muted, currentColor);
        opacity: 0.5;
    }
    .bar.player1 {
        fill: var(--player1-color, currentColor);
    }
    .bar.player2 {
        fill: var(--player2-color, currentColor);
        opacity: 0.6;
    }
    /* The reference player's expected loss: a tick across the bar, so a bar
       far above its tick reads as an error the position did not excuse. */
    .difficulty {
        stroke: var(--color-text, currentColor);
        stroke-width: 1.5;
    }
    .avoidable {
        fill: var(--color-danger, currentColor);
    }
    .bar.hot {
        stroke: var(--color-text, currentColor);
        stroke-width: 1;
    }
    .hot-dot {
        fill: var(--color-text, currentColor);
        stroke: var(--color-surface, transparent);
        stroke-width: 2;
    }
    .line {
        fill: none;
        stroke-width: 2;
        stroke-linejoin: round;
        stroke-linecap: round;
    }
    .line.player1 {
        stroke: var(--player1-color, currentColor);
    }
    .line.player2 {
        stroke: var(--player2-color, currentColor);
        stroke-dasharray: 5 3;
    }
    .end {
        stroke: var(--color-surface, transparent);
        stroke-width: 2;
    }
    .end.player1 {
        fill: var(--player1-color, currentColor);
    }
    .end.player2 {
        fill: var(--player2-color, currentColor);
    }
    .y-top {
        position: absolute;
        top: 1px;
        left: 3px;
        color: var(--color-text-muted, inherit);
        pointer-events: none;
    }
    .tip {
        position: absolute;
        top: -4px;
        transform: translate(8px, -100%);
        padding: 4px 8px;
        background: var(--color-surface, transparent);
        color: var(--color-text, inherit);
        border: 1px solid var(--color-border, currentColor);
        border-radius: var(--radius, 4px);
        white-space: nowrap;
        pointer-events: none;
        z-index: 5;
    }
    .tip.flip {
        transform: translate(calc(-100% - 8px), -100%);
    }
</style>
