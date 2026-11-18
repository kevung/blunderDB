<script>
    // How a match's decision times spread, per player: one band per duration
    // range of the time summary, its counts read from the summary. A click on a
    // band opens its decisions one after the other, in match order.
    import { t } from '../i18n';
    import { bandLabel, bandMoves } from '../utils/decisionTime.js';

    /** @type {{ summary: any, movePositions: any[], names: [string, string], shown?: [boolean, boolean], onselect?: (index: number) => void }} */
    let { summary, movePositions, names, shown = [true, true], onselect = () => {} } = $props();

    let bounds = $derived(/** @type {number[]} */ (summary?.histogram_bounds_ms ?? []));
    /** @type {number[][]} */
    let counts = $derived(
        [0, 1].map((p) => {
            const h = summary?.players?.[p]?.histogram;
            return Array.isArray(h) && h.length === bounds.length + 1 ? h : [];
        })
    );
    // An older server sends no histogram: nothing is drawn rather than zeros.
    let ready = $derived(bounds.length > 0 && counts.every((h) => h.length === bounds.length + 1));
    let peak = $derived(Math.max(1, ...counts.flatMap((h, p) => (shown[p] ? h : []))));
    let moves = $derived(bandMoves(movePositions, bounds));

    // The decision a band opened last, so the next click goes on.
    /** @type {Record<number, number>} */
    let cursor = $state({});
    /** @param {number} band */
    function open(band) {
        const list = moves[band]?.filter((m) => shown[m.player]) ?? [];
        if (list.length === 0) return;
        const next = ((cursor[band] ?? -1) + 1) % list.length;
        cursor[band] = next;
        onselect(list[next].index);
    }
    /** @param {number} band */
    const tip = (band) => [bandLabel(bounds, band), ...[0, 1].filter((p) => shown[p]).map((p) => `${names[p]}: ${counts[p][band]}`)].join('\n');
</script>

{#if ready}
    <div class="histogram" role="group" aria-label={$t('match.timesHistogram')} data-testid="times-histogram" style:--bands={bounds.length + 1}>
        {#each counts[0] as _, band (band)}
            {@const empty = !(shown[0] && counts[0][band]) && !(shown[1] && counts[1][band])}
            <button type="button" class="band" disabled={empty} title={tip(band)} aria-label={tip(band)} data-testid="times-band-{band}" onclick={() => open(band)}>
                {#each [0, 1] as p (p)}
                    {#if shown[p]}
                        <span class="bin" class:player1={p === 0} class:player2={p === 1} style:height="{(counts[p][band] / peak) * 100}%"></span>
                    {/if}
                {/each}
            </button>
        {/each}
        {#each counts[0] as _, band (band)}
            <span class="edge">{band < bounds.length ? Math.round(bounds[band] / 1000) : ''}</span>
        {/each}
    </div>
{/if}

<style>
    .histogram {
        flex: 1 1 160px;
        min-width: 0;
        max-width: 360px;
        display: grid;
        grid-template-columns: repeat(var(--bands), 1fr);
        grid-template-rows: 1fr auto;
        column-gap: 2px;
        min-height: 56px;
    }
    .band {
        display: flex;
        align-items: flex-end;
        gap: 1px;
        padding: 0;
        border: none;
        border-bottom: 1px solid var(--border-color, currentColor);
        background: none;
        cursor: pointer;
    }
    .band:disabled {
        cursor: default;
    }
    .band:not(:disabled):hover .bin,
    .band:focus-visible .bin {
        outline: 1px solid var(--color-text, currentColor);
    }
    .bin {
        flex: 1 1 0;
        min-height: 0;
    }
    .bin.player1 {
        background: var(--player1-color, currentColor);
    }
    .bin.player2 {
        background: var(--player2-color, currentColor);
        opacity: 0.6;
    }
    .edge {
        text-align: right;
        transform: translateX(calc(50% + 1px));
        color: var(--text-muted, inherit);
        white-space: nowrap;
    }
</style>
