<script>
    // The time a match took, per player, and the decisions over the match.
    // Nothing is drawn for a match that recorded no time.
    import { t } from '../i18n';
    import { indexAt, stepIndex } from '../utils/chartAxis.js';
    import { fmtDuration, fmtMean, timeBars } from '../utils/decisionTime.js';

    /** @type {{ summary: any, movePositions: any[], player1: string, player2: string, hovered?: number | null, onhover?: (moveId: number | null) => void, onselect?: (index: number) => void }} */
    let { summary, movePositions, player1, player2, hovered = null, onhover = () => {}, onselect = () => {} } = $props();

    const W = 320;
    const H = 56;

    let bars = $derived(timeBars(movePositions));
    let peak = $derived(Math.max(1, ...bars.map((b) => b.ms)));
    let slot = $derived(W / Math.max(1, movePositions.length));
    let players = $derived([
        { name: player1, p: summary?.players?.[0] },
        { name: player2, p: summary?.players?.[1] }
    ]);
    // The hovered decision is shared with the loss chart by its Move, not its place.
    let focus = $derived.by(() => {
        if (hovered === null) return null;
        const i = movePositions.findIndex((mp) => mp.move_id === hovered);
        return i < 0 ? null : i;
    });
    /** @param {number | null} i */
    const setFocus = (i) => onhover(i === null ? null : (movePositions[i]?.move_id ?? null));

    /** @param {KeyboardEvent} e */
    function onKey(e) {
        const next = stepIndex(e.key, focus, movePositions.length);
        if (next !== null) {
            e.preventDefault();
            setFocus(next);
        } else if (e.key === 'Enter' && focus !== null) {
            e.preventDefault();
            onselect(focus);
        } else if (e.key === 'Escape') {
            setFocus(null);
        }
    }
    let known = $derived(players.some(({ p }) => p && p.checker_count + p.cube_count > 0));
</script>

{#if known}
    <div class="match-times" role="group" aria-label={$t('match.timesTitle')} data-testid="match-times">
        <table class="times-table">
            <thead>
                <tr>
                    <th></th>
                    <th>{$t('match.timesTotal')}</th>
                    <th>{$t('match.timesChecker')}</th>
                    <th>{$t('match.timesCube')}</th>
                    {#if summary.has_cadence}<th title={$t('match.timesOverrunTooltip')}>{$t('match.timesOverrun')}</th>{/if}
                </tr>
            </thead>
            <tbody>
                {#each players as { name, p }, i (i)}
                    {#if p}
                        <tr>
                            <td class="times-player" class:player1={i === 0} class:player2={i === 1}>{name}</td>
                            <td>{fmtDuration(p.checker_count + p.cube_count > 0 ? p.total_ms : null)}</td>
                            <td>{fmtMean(p.checker_total_ms, p.checker_count)}</td>
                            <td>{fmtMean(p.cube_total_ms, p.cube_count)}</td>
                            {#if summary.has_cadence}
                                <td data-testid="overrun-{i}">{p.over_time ? '●' : ''}</td>
                            {/if}
                        </tr>
                    {/if}
                {/each}
            </tbody>
        </table>
        <!-- A chart: walked with the arrow keys, opened with Enter. -->
        <!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
        <div
            class="times-plot"
            role="group"
            tabindex="0"
            aria-label={$t('match.timesChart')}
            title={$t('match.chartOpen')}
            data-testid="times-plot"
            onmousemove={(e) => setFocus(indexAt(e, movePositions.length))}
            onmouseleave={() => setFocus(null)}
            onclick={(e) => onselect(indexAt(e, movePositions.length))}
            onkeydown={onKey}
            onblur={() => setFocus(null)}
        >
            <svg class="times-chart" viewBox="0 0 {W} {H}" role="img" aria-label={$t('match.timesChart')}>
                {#each bars as b (b.index)}
                    {@const h = Math.max(1, (b.ms / peak) * (H - 2))}
                    <rect
                        class="bar"
                        class:player1={b.player === 0}
                        class:player2={b.player === 1}
                        class:hot={b.index === focus}
                        x={b.index * slot}
                        y={H - h}
                        width={Math.max(1, slot - 0.5)}
                        height={h}
                    >
                        <title>{fmtDuration(b.ms)}</title>
                    </rect>
                {/each}
                {#if focus !== null}
                    <line class="cross" x1={(focus + 0.5) * slot} y1="0" x2={(focus + 0.5) * slot} y2={H} />
                {/if}
            </svg>
        </div>
    </div>
{/if}

<style>
    .match-times {
        display: flex;
        flex-wrap: wrap;
        gap: 12px;
        align-items: flex-end;
        padding: 6px 8px;
        font-size: var(--font-size-small);
    }
    .times-table {
        border-collapse: collapse;
    }
    .times-table th,
    .times-table td {
        padding: 2px 8px;
        text-align: right;
    }
    .times-table th {
        font-weight: 500;
        color: var(--text-muted, inherit);
    }
    .times-player {
        text-align: left;
    }
    .times-chart {
        width: 320px;
        max-width: 100%;
        height: 56px;
    }
    .times-plot {
        width: 320px;
        max-width: 100%;
        cursor: pointer;
        outline-offset: 2px;
    }
    .cross {
        stroke: var(--color-text-muted, currentColor);
        stroke-width: 1;
    }
    .bar.hot {
        stroke: var(--color-text, currentColor);
        stroke-width: 1;
    }
    .bar.player1 {
        fill: var(--player1-color, currentColor);
    }
    .bar.player2 {
        fill: var(--player2-color, currentColor);
        opacity: 0.6;
    }
</style>
