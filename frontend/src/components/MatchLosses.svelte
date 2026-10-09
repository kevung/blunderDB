<script>
    // The match winning chances each decision cost, per player: a bar per
    // decision and the running total, on the decision axis of MatchTimes. Two
    // plots rather than one: a decision's loss and a total are the same unit
    // but not the same scale. Nothing is drawn for a match without analysis.
    // The per-player totals are in the review's details.
    import { t } from '../i18n';
    import { fmtLoss, lossSeries, niceCeil } from '../utils/decisionLoss.js';
    import MatchChart from './MatchChart.svelte';

    /** @type {{ losses: any[] | null, movePositions: any[], player1: string, player2: string, hovered?: number | null, onhover?: (moveId: number | null) => void, onselect?: (index: number) => void }} */
    let { losses, movePositions, player1, player2, hovered = null, onhover = () => {}, onselect = () => {} } = $props();

    // The least height a plot keeps; the Charts tab shares the rest of its height out.
    const MIN_H = 140;

    let series = $derived(lossSeries(movePositions, losses));
    let peak = $derived(niceCeil(Math.max(0, ...series.items.map((d) => Math.max(d.loss ?? 0, d.difficulty ?? 0)))));
    let cumPeak = $derived(niceCeil(Math.max(series.totals[0], series.totals[1])));
    let names = $derived([player1, player2]);

    /** @param {number} loss @param {number} top @param {number} H */
    const barH = (loss, top, H) => (loss > 0 ? Math.max(1, (loss / top) * (H - 2)) : 0);
    /** @param {number} cum @param {number} H */
    const cumY = (cum, H) => H - 1 - (cum / cumPeak) * (H - 4);

    // The hovered decision is shared with the time chart by its Move, not its place.
    let hover = $derived.by(() => {
        if (hovered === null) return null;
        const i = movePositions.findIndex((mp) => mp.move_id === hovered);
        return i < 0 ? null : i;
    });
    /** @param {number | null} i */
    const setHover = (i) => onhover(i === null ? null : (movePositions[i]?.move_id ?? null));

    /** @param {number} v */
    const axis = (v) => (v === 0 ? '0' : `${+(v * 100).toFixed(2)} %`);
    /** @param {number} top */
    const ticks = (top) => [
        { at: 1, text: axis(top) },
        { at: 0.5, text: axis(top / 2) },
        { at: 0, text: '0' }
    ];
</script>

{#if series.any}
    <div class="match-losses" role="group" aria-label={$t('match.lossTitle')} data-testid="match-losses">
        <MatchChart
            testid="loss-plot-per"
            title={$t('match.lossPerDecision')}
            label={$t('match.lossChart')}
            minHeight={MIN_H}
            {movePositions}
            ticks={ticks(peak)}
            focus={hover}
            onfocus={setHover}
            {onselect}
        >
            {#snippet legend()}
                <span><span class="key bar-key player1"></span>{names[0]}</span>
                <span><span class="key bar-key player2"></span>{names[1]}</span>
                <span title={$t('match.difficultyColTooltip')}><span class="key tick-key"></span>{$t('match.legendDifficulty')}</span>
                <span><span class="key dot-key"></span>{$t('match.avoidable')}</span>
            {/snippet}
            {#snippet marks({ slot, H })}
                {#each series.items as d (d.index)}
                    {#if d.loss === null}
                        <rect class="unscored" x={d.index * slot} y={H - 1.5} width={Math.max(1, slot - 0.5)} height="1.5" />
                    {:else if d.loss > 0}
                        {@const h = barH(d.loss, peak, H)}
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
                        {@const y = H - barH(d.difficulty, peak, H)}
                        <line class="difficulty" x1={d.index * slot} y1={y} x2={d.index * slot + Math.max(1, slot - 0.5)} y2={y} />
                    {/if}
                    {#if d.avoidable}
                        <circle class="avoidable" data-testid="avoidable-dot" cx={(d.index + 0.5) * slot} cy="4" r="3" />
                    {/if}
                {/each}
            {/snippet}
            {#snippet tip(i)}
                {@render decision(i)}
            {/snippet}
        </MatchChart>
        <MatchChart
            testid="loss-plot-cum"
            title={$t('match.lossCumulative')}
            label={$t('match.lossChart')}
            minHeight={MIN_H}
            {movePositions}
            ticks={ticks(cumPeak)}
            focus={hover}
            onfocus={setHover}
            {onselect}
        >
            {#snippet legend()}
                <span><svg class="line-key" viewBox="0 0 22 6"><line class="line player1" x1="1" y1="3" x2="21" y2="3" /></svg>{names[0]}</span>
                <span><svg class="line-key" viewBox="0 0 22 6"><line class="line player2" x1="1" y1="3" x2="21" y2="3" /></svg>{names[1]}</span>
            {/snippet}
            {#snippet marks({ slot, H })}
                {#each [0, 1] as p (p)}
                    {@const pts = series.items.filter((d) => d.player === p).map((d) => ({ x: (d.index + 0.5) * slot, y: cumY(d.cum, H) }))}
                    {#if pts.length > 0}
                        {@const end = pts[pts.length - 1]}
                        <polyline class="line" class:player1={p === 0} class:player2={p === 1} points={pts.map(({ x, y }) => `${x.toFixed(1)},${y.toFixed(1)}`).join(' ')} />
                        <circle class="end" class:player1={p === 0} class:player2={p === 1} cx={end.x} cy={end.y} r="3" />
                    {/if}
                {/each}
                {#if hover !== null && series.items[hover].loss !== null}
                    <circle class="hot-dot" cx={(hover + 0.5) * slot} cy={cumY(series.items[hover].cum, H)} r="4" />
                {/if}
            {/snippet}
            {#snippet tip(i)}
                {@render decision(i)}
                <div>{$t('match.lossSoFar', { loss: fmtLoss(series.items[i].cum) || '0' })}</div>
            {/snippet}
        </MatchChart>
    </div>
{/if}

{#snippet decision(/** @type {number} */ i)}
    {@const d = series.items[i]}
    {@const mp = movePositions[i]}
    <div><b>{$t('match.lossWhere', { game: mp.game_number, move: mp.move_number })}</b> · {names[d.player]}</div>
    <div>
        {d.loss === null ? $t('match.lossUnscored') : fmtLoss(d.loss) || '0'}
        ({mp.move_type === 'cube' ? $t('match.lossKindCube') : $t('match.lossKindChecker')})
    </div>
    {#if d.difficulty !== null}
        <div>{$t('match.difficultyCol')} {d.difficulty > 0 ? fmtLoss(d.difficulty) : '0'}{d.avoidable ? ' · ' + $t('match.avoidable') : ''}</div>
    {/if}
{/snippet}

<style>
    .match-losses {
        display: flex;
        flex-direction: column;
        flex: 5 1 0;
    }
    .key {
        display: inline-block;
        margin-right: 5px;
        vertical-align: middle;
    }
    .bar-key {
        width: 10px;
        height: 10px;
    }
    .bar-key.player1 {
        background: var(--player1-color, currentColor);
    }
    .bar-key.player2 {
        background: var(--player2-color, currentColor);
        opacity: 0.6;
    }
    .tick-key {
        width: 12px;
        height: 0;
        border-top: 2px solid var(--color-text, currentColor);
    }
    .dot-key {
        width: 7px;
        height: 7px;
        border-radius: 50%;
        background: var(--color-danger, currentColor);
    }
    .line-key {
        width: 22px;
        height: 6px;
        margin-right: 5px;
        vertical-align: middle;
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
        stroke-width: 2;
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
</style>
