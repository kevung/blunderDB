<script>
    import { t } from '../i18n';

    // Position facts (ADR-0017, CONTEXT.md), shared by EvalPanel and
    // AnalysisPanel. Two blocks (ADR-0018): the race block, always per side,
    // and the pre-roll vector, per side only when showProbabilities (otherwise
    // it is CandidateMovesTable's Baseline row). ADR-0021 stacks them as two
    // <tbody>s of ONE table so they share a column grid; side by side they
    // pushed the cube block under the numbers at the default width.
    //
    // bottom/top: {win, gammon, backgammon, cubeless} | null — fractions and an
    // equity; null renders a blank row, never a hidden one (ADR-0017 rule 3).
    // bottomEPC/topEPC: race.EPCResult-shaped | null; the race block shows when
    // at least one is present.
    let {
        bottom = null,
        top = null,
        bottomEPC = null,
        topEPC = null,
        bottomPoints = 0,
        topPoints = 0,
        maskedBottom = false,
        maskedTop = false,
        onRevealBottom = () => {},
        onRevealTop = () => {},
        showProbabilities = true
    } = $props();

    let showRace = $derived(!!(bottomEPC || topEPC));
    let bothShown = $derived(!maskedBottom && !maskedTop);

    // Five value columns with the race block, four otherwise; the probability
    // block ends in an empty cell that keeps the alignment.
    let dataCols = $derived(showRace ? 5 : 4);
    let probHeaders = $derived([$t('epc.facts.gain'), $t('epc.facts.gammon'), $t('epc.facts.backgammon'), $t('epc.facts.cubelessEquity')]);
    let raceHeaders = $derived([$t('epc.epc'), $t('epc.pipCount'), $t('epc.wastage'), $t('epc.avgRolls'), $t('epc.stdDev')]);

    // The one-sided table used, shown only beyond six points (ADR-0027 §9), in
    // the race block's empty corner cell so no column moves (ADR-0021).
    let raceDomain = $derived.by(() => {
        const width = Math.max(bottomPoints ?? 0, topPoints ?? 0);
        return width > 6 ? `OS-${String(width).padStart(2, '0')}` : '';
    });

    const HIDDEN = '···';
    const DASH = '—';
    const show = (/** @type {boolean} */ masked, /** @type {any} */ v) => (masked ? HIDDEN : (v ?? DASH));
    const pct = (/** @type {number|null|undefined} */ x) => (x == null ? null : (100 * x).toFixed(2));
    const eq = (/** @type {number|null|undefined} */ x) => (x == null ? null : (x >= 0 ? '+' : '') + x.toFixed(3));
    const sd = (x, digits) => (x == null ? null : (x >= 0 ? '+' : '') + x.toFixed(digits));

    function delta(a, b, fmt) {
        if (a == null || b == null) return null;
        return fmt(a - b);
    }

    // Three rows per block (side, side, Δ), rendered by one snippet. The race Δ
    // row ends in dashes: differencing rolls or deviations means nothing.
    let probCells = $derived({
        bottom: [pct(bottom?.win), pct(bottom?.gammon), pct(bottom?.backgammon), eq(bottom?.cubeless)],
        top: [pct(top?.win), pct(top?.gammon), pct(top?.backgammon), eq(top?.cubeless)],
        delta: [
            delta(bottom?.win, top?.win, (v) => sd(100 * v, 2)),
            delta(bottom?.gammon, top?.gammon, (v) => sd(100 * v, 2)),
            delta(bottom?.backgammon, top?.backgammon, (v) => sd(100 * v, 2)),
            delta(bottom?.cubeless, top?.cubeless, (v) => sd(v, 3))
        ]
    });

    let raceCells = $derived({
        bottom: [bottomEPC?.epc?.toFixed(2), bottomEPC?.pipCount, bottomEPC?.wastage?.toFixed(2), bottomEPC?.meanRolls?.toFixed(3), bottomEPC?.stdDev?.toFixed(3)],
        top: [topEPC?.epc?.toFixed(2), topEPC?.pipCount, topEPC?.wastage?.toFixed(2), topEPC?.meanRolls?.toFixed(3), topEPC?.stdDev?.toFixed(3)],
        delta: [
            delta(bottomEPC?.epc, topEPC?.epc, (v) => sd(v, 2)),
            delta(bottomEPC?.pipCount, topEPC?.pipCount, (v) => sd(v, 0)),
            delta(bottomEPC?.wastage, topEPC?.wastage, (v) => sd(v, 2)),
            DASH,
            DASH
        ]
    });
</script>

{#snippet cells(/** @type {any[]} */ row, /** @type {boolean} */ masked)}
    {#each Array(dataCols) as _, i (i)}
        {#if i < row.length}
            <td class:main-value={i === 0}>{show(masked, row[i])}</td>
        {:else}
            <td class="filler"></td>
        {/if}
    {/each}
{/snippet}

{#snippet block(/** @type {any[]} */ headers, /** @type {any} */ rows, /** @type {string} */ corner = '')}
    <tbody class="facts-block">
        <tr class="head-row">
            <th class="corner" title={corner ? $t('epc.raceDomainTooltip', { domain: corner }) : undefined}>{corner}</th>
            {#each Array(dataCols) as _, i (i)}
                <th>{headers[i] ?? ''}</th>
            {/each}
        </tr>
        <tr class:masked={maskedBottom} onclick={() => maskedBottom && onRevealBottom()} title={maskedBottom ? $t('epc.clickToReveal') : $t('epc.bottomBlack')}>
            <td class="row-label"><span class="player-indicator bottom"></span></td>
            {@render cells(rows.bottom, maskedBottom)}
        </tr>
        <tr class:masked={maskedTop} onclick={() => maskedTop && onRevealTop()} title={maskedTop ? $t('epc.clickToReveal') : $t('epc.topWhite')}>
            <td class="row-label"><span class="player-indicator top"></span></td>
            {@render cells(rows.top, maskedTop)}
        </tr>
        <tr class="delta-row" title={$t('epc.comparison')}>
            <td class="row-label">Δ</td>
            {@render cells(rows.delta, !bothShown)}
        </tr>
    </tbody>
{/snippet}

<!-- One table, one column grid, two blocks (ADR-0021). -->
<table class="facts-table">
    {#if showProbabilities}
        {@render block(probHeaders, probCells)}
    {/if}
    {#if showRace}
        {@render block(raceHeaders, raceCells, raceDomain)}
    {/if}
</table>

<style>
    /* One grid for both blocks (ADR-0021): same left edge, same column stops,
       one column of side markers. */
    .facts-table {
        border-collapse: collapse;
        font-size: var(--font-size-base);
    }

    th,
    td {
        padding: 2px 10px;
        text-align: center;
        white-space: nowrap;
    }

    th {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        text-transform: uppercase;
        letter-spacing: 0.3px;
        font-weight: 600;
    }

    /* Hairlines separate values from values, never a header from the block it
       heads — and the second block opens on a little air instead of a rule. */
    tbody tr + tr td {
        border-top: 1px solid var(--color-border);
    }

    .head-row + tr td {
        border-top: none;
    }

    .facts-block + .facts-block th {
        padding-top: 10px;
    }

    .row-label {
        width: 22px;
        padding-left: 0;
        padding-right: 6px;
        font-size: var(--font-size-small);
        font-weight: 600;
        color: var(--color-text);
    }

    .main-value {
        font-weight: 600;
        color: var(--color-primary);
        font-variant-numeric: tabular-nums;
    }

    td {
        font-variant-numeric: tabular-nums;
        color: var(--color-text);
    }

    .delta-row td {
        color: var(--color-text-muted);
    }

    .delta-row .main-value {
        color: var(--color-text-muted);
    }

    tr.masked {
        cursor: pointer;
    }

    tr.masked td:not(.row-label) {
        color: var(--color-text-muted);
        letter-spacing: 2px;
    }

    tr.masked:hover td {
        background: var(--color-surface-alt);
    }

    .player-indicator {
        display: inline-block;
        width: 10px;
        height: 10px;
        border-radius: 50%;
        flex-shrink: 0;
        vertical-align: middle;
    }

    .player-indicator.bottom {
        background: #333;
        border: 1px solid #555;
    }

    .player-indicator.top {
        background: #fff;
        border: 1px solid #999;
    }
</style>
