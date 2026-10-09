<script>
    // One chart of the match sheet, on the decision axis the charts share: a
    // title and its legend, the value axis on the left, the games named along
    // the bottom, and a tooltip that stays whole inside the plot whatever its
    // width. The marks and the tooltip's text are the caller's.
    import { t } from '../i18n';
    import { gameSpans, indexAt, stepIndex } from '../utils/chartAxis.js';
    import { placeTip } from '../utils/chartTip.js';
    import { trackSize } from '../utils/trackWidth.js';

    /**
     * @typedef {{
     *   testid: string,
     *   title: string,
     *   label: string,
     *   minHeight: number,
     *   movePositions: any[],
     *   ticks: { at: number, text: string }[],
     *   focus: number | null,
     *   onfocus: (index: number | null) => void,
     *   onselect: (index: number) => void,
     *   legend?: import('svelte').Snippet,
     *   marks: import('svelte').Snippet<[{ W: number, H: number, slot: number }]>,
     *   tip?: import('svelte').Snippet<[number]>
     * }} Props
     */
    /** @type {Props} */
    let { testid, title, label, minHeight, movePositions, ticks, focus, onfocus, onselect, legend, marks, tip } = $props();

    // Drawn at the plot's pixel size, so a dot stays round: the width is the
    // sheet's, the height what the Charts tab gives this chart, never below
    // minHeight.
    let plotWidth = $state(0);
    let plotHeight = $state(0);
    let W = $derived(Math.max(160, Math.round(plotWidth) || 320));
    let H = $derived(Math.max(minHeight, Math.round(plotHeight) || 0));
    let n = $derived(Math.max(1, movePositions.length));
    let slot = $derived(W / n);
    // Short game names only where the stretch is wide enough to hold one.
    let spans = $derived(gameSpans(movePositions).map((s) => ({ ...s, named: (s.to - s.from) * slot >= 26 })));

    // The tooltip shows on the chart the user is on, beside the pointer, or
    // over the decision the arrow keys reached.
    let active = $state(false);
    /** @type {{ x: number, y: number } | null} */
    let pointer = $state(null);
    let tipW = $state(0);
    let tipH = $state(0);
    let anchor = $derived(pointer ?? { x: ((focus ?? 0) + 0.5) * slot, y: H / 3 });
    let at = $derived(placeTip(anchor, { w: tipW, h: tipH }, { w: W, h: H }));

    // Measured whenever it shows another decision, its text having changed.
    /** @param {HTMLElement} node @param {number | null} _decision */
    function measure(node, _decision) {
        const read = () => {
            tipW = node.offsetWidth;
            tipH = node.offsetHeight;
        };
        read();
        return { update: read };
    }

    /** @param {MouseEvent} e */
    function onMove(e) {
        const r = /** @type {HTMLElement} */ (e.currentTarget).getBoundingClientRect();
        active = true;
        pointer = { x: e.clientX - r.left, y: e.clientY - r.top };
        onfocus(indexAt(e, movePositions.length));
    }

    function leave() {
        active = false;
        pointer = null;
        onfocus(null);
    }

    /** @param {KeyboardEvent} e */
    function onKey(e) {
        const next = stepIndex(e.key, focus, movePositions.length);
        if (next !== null) {
            e.preventDefault();
            active = true;
            pointer = null;
            onfocus(next);
        } else if (e.key === 'Enter' && focus !== null) {
            e.preventDefault();
            onselect(focus);
        } else if (e.key === 'Escape') {
            leave();
        }
    }
</script>

<figure class="chart">
    <figcaption class="head">
        <span class="title">{title}</span>
        {#if legend}<span class="legend">{@render legend()}</span>{/if}
    </figcaption>
    <div class="body">
        <div class="y-axis" aria-hidden="true">
            {#each ticks as tk (tk.at)}
                <span class="tick" style="top: {(1 - tk.at) * 100}%">{tk.text}</span>
            {/each}
        </div>
        <!-- A chart: walked with the arrow keys, opened with Enter. -->
        <!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
        <div
            class="plot"
            role="group"
            tabindex="0"
            aria-label={label}
            title={active ? undefined : $t('match.chartOpen')}
            data-testid={testid}
            style="min-height: {minHeight}px"
            use:trackSize={(w, h) => {
                plotWidth = w;
                plotHeight = h;
            }}
            onmousemove={onMove}
            onmouseleave={leave}
            onclick={(e) => onselect(indexAt(e, movePositions.length))}
            onkeydown={onKey}
            onblur={leave}
        >
            <svg viewBox="0 0 {W} {H}" role="img" aria-label={label}>
                {#each ticks as tk (tk.at)}
                    {@const y = tk.at === 0 ? H - 0.5 : (1 - tk.at) * H + 0.5}
                    <line class="grid" class:base={tk.at === 0} x1="0" y1={y} x2={W} y2={y} />
                {/each}
                {#each spans.slice(1) as s (s.game)}
                    <line class="game-edge" x1={s.from * slot} y1="0" x2={s.from * slot} y2={H} />
                {/each}
                {@render marks({ W, H, slot })}
                {#if focus !== null}
                    <line class="cross" x1={(focus + 0.5) * slot} y1="0" x2={(focus + 0.5) * slot} y2={H} />
                {/if}
            </svg>
            {#if tip && active && focus !== null}
                <div class="tip" role="tooltip" use:measure={focus} style="left: {at.left}px; top: {at.top}px; visibility: {tipW ? 'visible' : 'hidden'}">
                    {@render tip(focus)}
                </div>
            {/if}
        </div>
        <div class="x-axis" aria-hidden="true">
            {#each spans as s (s.game)}
                {#if s.named}
                    <span class="game" style="left: {((s.from + s.to) / 2) * (100 / n)}%">{$t('match.gameShort', { n: s.game })}</span>
                {/if}
            {/each}
        </div>
    </div>
</figure>

<style>
    /* A chart grows with the room its parent gives it; the plot takes the growth. */
    .chart {
        flex: 1 1 0;
        display: flex;
        flex-direction: column;
        margin: 0;
        padding: 8px 12px 4px;
        font-size: var(--font-size-small);
        text-align: left;
    }
    .head {
        display: flex;
        flex-wrap: wrap;
        align-items: baseline;
        gap: 2px 16px;
        margin-bottom: 12px;
    }
    .title {
        font-weight: 600;
        font-size: var(--font-size-base);
        color: var(--color-text);
    }
    .legend {
        display: flex;
        flex-wrap: wrap;
        gap: 2px 14px;
        color: var(--color-text-muted);
    }
    /* The value axis in its own column, so no label sits over a mark. */
    .body {
        flex: 1 1 auto;
        display: grid;
        grid-template-columns: 52px minmax(0, 1fr);
        grid-template-rows: minmax(0, 1fr) auto;
        column-gap: 6px;
    }
    .y-axis {
        position: relative;
        color: var(--color-text-muted);
        font-variant-numeric: tabular-nums;
    }
    .tick {
        position: absolute;
        right: 0;
        transform: translateY(-50%);
        white-space: nowrap;
    }
    .plot {
        position: relative;
        min-width: 0;
        cursor: pointer;
        outline-offset: 2px;
    }
    /* Out of the flow, so the drawing never feeds back into the size it is drawn at. */
    .plot svg {
        position: absolute;
        inset: 0;
        display: block;
        width: 100%;
        height: 100%;
        overflow: visible;
    }
    .x-axis {
        grid-column: 2;
        position: relative;
        height: 1.4em;
        color: var(--color-text-muted);
    }
    .game {
        position: absolute;
        top: 2px;
        transform: translateX(-50%);
        white-space: nowrap;
    }
    .grid {
        stroke: var(--color-border, currentColor);
        stroke-width: 1;
        opacity: 0.6;
    }
    .grid.base {
        opacity: 1;
    }
    .game-edge {
        stroke: var(--color-text-muted, currentColor);
        stroke-width: 1;
        stroke-dasharray: 3 3;
        opacity: 0.6;
    }
    .cross {
        stroke: var(--color-text-muted, currentColor);
        stroke-width: 1;
    }
    /* Its own width, not what is left right of where it sits, so that it is
       measured the same wherever it is placed. */
    .tip {
        position: absolute;
        width: max-content;
        max-width: 100%;
        box-sizing: border-box;
        padding: 4px 8px;
        background: var(--color-surface, transparent);
        color: var(--color-text, inherit);
        border: 1px solid var(--color-border, currentColor);
        border-radius: var(--radius, 4px);
        box-shadow: 0 2px 6px color-mix(in srgb, var(--color-text) 15%, transparent);
        pointer-events: none;
        z-index: 5;
    }
</style>
