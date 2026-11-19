<script>
    // Player 1's match winning chances over the match (MatchWinTrace), on the
    // decision axis of the other charts: across each decision the curve goes
    // from the chances with the best play to those after the play made, and
    // between two decisions it jumps by the dice. A decision without analysis
    // breaks the line; an error is a dot. Nothing is drawn at money.
    import { t } from '../i18n';
    import { winTraceSeries, winTraceRuns } from '../utils/decisionLoss.js';
    import MatchChart from './MatchChart.svelte';

    /** @type {{ trace: any, movePositions: any[], player1: string, player2: string, hovered?: number | null, onhover?: (moveId: number | null) => void, onselect?: (index: number) => void }} */
    let { trace, movePositions, player1, player2, hovered = null, onhover = () => {}, onselect = () => {} } = $props();

    const MIN_H = 140;

    let series = $derived(winTraceSeries(movePositions, trace));
    let runs = $derived(winTraceRuns(series.items));
    let names = $derived([player1, player2]);

    /** @param {number} v @param {number} H */
    const y = (v, H) => H - 2 - Math.min(1, Math.max(0, v)) * (H - 4);

    let hover = $derived.by(() => {
        if (hovered === null) return null;
        const i = movePositions.findIndex((mp) => mp.move_id === hovered);
        return i < 0 ? null : i;
    });
    /** @param {number | null} i */
    const setHover = (i) => onhover(i === null ? null : (movePositions[i]?.move_id ?? null));

    /** @param {number} v */
    const pct = (v) => `${(v * 100).toFixed(1)} %`;
    const ticks = [
        { at: 1, text: '100 %' },
        { at: 0.5, text: '50 %' },
        { at: 0, text: '0' }
    ];
</script>

{#if series.any}
    <div class="match-win-trace" role="group" aria-label={$t('match.winTraceTitle', { name: player1 })} data-testid="match-win-trace">
        <MatchChart
            testid="win-trace-plot"
            title={$t('match.winTraceTitle', { name: player1 })}
            label={$t('match.winTraceChart')}
            minHeight={MIN_H}
            {movePositions}
            {ticks}
            focus={hover}
            onfocus={setHover}
            {onselect}
        >
            {#snippet legend()}
                <span><svg class="line-key" viewBox="0 0 22 6"><line class="line" x1="1" y1="3" x2="21" y2="3" /></svg>{player1}</span>
                <span><span class="dot-key"></span>{$t('match.winTraceErrors')}</span>
            {/snippet}
            {#snippet marks({ W, slot, H })}
                <line class="even" x1="0" y1={y(0.5, H)} x2={W} y2={y(0.5, H)} />
                {#each runs as run, r (r)}
                    <polyline class="line" points={run.map((p) => `${(p.x * slot).toFixed(1)},${y(p.v, H).toFixed(1)}`).join(' ')} />
                {/each}
                {#each series.items.filter((d) => d.error && d.before !== null) as d (d.index)}
                    <circle class="error" data-testid="win-trace-error" cx={(d.index + 0.5) * slot} cy={y(d.after ?? d.before ?? 0, H)} r="3" />
                {/each}
                {#if hover !== null && series.items[hover].before !== null}
                    <circle class="hot-dot" cx={(hover + 0.5) * slot} cy={y(series.items[hover].after ?? series.items[hover].before ?? 0, H)} r="4" />
                {/if}
            {/snippet}
            {#snippet tip(i)}
                {@const d = series.items[i]}
                {@const mp = movePositions[i]}
                <div><b>{$t('match.lossWhere', { game: mp.game_number, move: mp.move_number })}</b> · {names[d.player]}</div>
                <div>
                    {#if d.before === null}
                        {$t('match.winTraceNone')}
                    {:else}
                        {player1} : {pct(d.before)}{d.after !== null ? ` → ${pct(d.after)}` : ''}
                    {/if}
                </div>
            {/snippet}
        </MatchChart>
    </div>
{/if}

<style>
    .match-win-trace {
        display: flex;
        flex-direction: column;
        flex: 3 1 0;
    }
    .line-key {
        width: 22px;
        height: 6px;
        margin-right: 5px;
        vertical-align: middle;
    }
    .dot-key {
        display: inline-block;
        width: 7px;
        height: 7px;
        margin-right: 5px;
        border-radius: 50%;
        vertical-align: middle;
        background: var(--color-danger, currentColor);
    }
    .line {
        fill: none;
        stroke: var(--player1-color, currentColor);
        stroke-width: 2;
        stroke-linejoin: round;
        stroke-linecap: round;
    }
    .even {
        stroke: var(--color-text-muted, currentColor);
        stroke-dasharray: 3 3;
        opacity: 0.6;
    }
    .error {
        fill: var(--color-danger, currentColor);
        stroke: var(--color-surface, transparent);
        stroke-width: 1;
    }
    .hot-dot {
        fill: var(--color-text, currentColor);
        stroke: var(--color-surface, transparent);
        stroke-width: 2;
    }
</style>
