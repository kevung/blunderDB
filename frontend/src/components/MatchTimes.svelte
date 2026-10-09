<script>
    // The time a match took, per player, and the decisions over the match.
    // Nothing is drawn for a match that recorded no time.
    import { t } from '../i18n';
    import { fmtDuration, fmtMean, timeBars } from '../utils/decisionTime.js';
    import MatchChart from './MatchChart.svelte';

    /** @type {{ summary: any, movePositions: any[], player1: string, player2: string, hovered?: number | null, onhover?: (moveId: number | null) => void, onselect?: (index: number) => void }} */
    let { summary, movePositions, player1, player2, hovered = null, onhover = () => {}, onselect = () => {} } = $props();

    // On the same decision axis as the loss charts.
    const H = 110;

    let bars = $derived(timeBars(movePositions));
    let peak = $derived(Math.max(1, ...bars.map((b) => b.ms)));
    let byIndex = $derived(new Map(bars.map((b) => [b.index, b])));
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
        <div class="times-chart">
            <MatchChart
                testid="times-plot"
                title={$t('match.timesPerDecision')}
                label={$t('match.timesChart')}
                height={H}
                {movePositions}
                ticks={[
                    { at: 1, text: fmtDuration(peak) },
                    { at: 0, text: '0' }
                ]}
                {focus}
                onfocus={setFocus}
                {onselect}
            >
                {#snippet legend()}
                    {#each players as { name }, i (i)}
                        <span><span class="key" class:player1={i === 0} class:player2={i === 1}></span>{name}</span>
                    {/each}
                {/snippet}
                {#snippet marks({ slot })}
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
                        />
                    {/each}
                {/snippet}
                {#snippet tip(i)}
                    {@const mp = movePositions[i]}
                    {@const b = byIndex.get(i)}
                    <div><b>{$t('match.lossWhere', { game: mp.game_number, move: mp.move_number })}</b> · {players[mp.player_on_roll === 1 ? 1 : 0].name}</div>
                    <div>{b ? fmtDuration(b.ms) : '—'}</div>
                {/snippet}
            </MatchChart>
        </div>
    </div>
{/if}

<style>
    .match-times {
        display: flex;
        flex-wrap: wrap;
        gap: 4px 12px;
        align-items: flex-end;
        padding: 6px 12px;
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
        flex: 1 1 100%;
        min-width: 0;
        margin: 0 -12px;
    }
    .key {
        display: inline-block;
        width: 10px;
        height: 10px;
        margin-right: 5px;
        vertical-align: middle;
    }
    .key.player1 {
        background: var(--player1-color, currentColor);
    }
    .key.player2 {
        background: var(--player2-color, currentColor);
        opacity: 0.6;
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
