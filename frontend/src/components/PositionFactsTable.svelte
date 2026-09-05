<script>
    import { t } from '../i18n';

    // The "position fact" half of ADR-0017's layout rule: quantities that
    // belong to the board itself, never to a choice a player might make.
    // Shared by EPCPanel (live evaluation) and AnalysisPanel (a stored
    // record) — one rendering, two callers, so the two never drift on what
    // a "position fact" looks like on screen (CONTEXT.md).
    //
    // Two kinds of fact live here, and ADR-0018 gives them different axes:
    // - the race block (bottomEPC/topEPC) is per side by nature and always
    //   read in bottom/top/Δ rows;
    // - the pre-roll vector (bottom/top) is per side ONLY when there is no
    //   list to read it against — showProbabilities is false whenever the
    //   caller instead renders it as a Baseline row inside a candidate list
    //   (CandidateMovesTable's own `baseline` prop). The two never show at
    //   once for the same position.
    //
    // ADR-0021: those two kinds are two BLOCKS stacked in one table — two
    // `<tbody>`s, each with its own header row — not ten columns on one line.
    // Welded into one line, the facts plus the cube block needed 963 px (fr) to
    // 1125 px (el) against the 996 px the panel has at blunderDB's default
    // window, so in seven languages out of nine the cube block, last in the
    // flex row, was pushed under the numbers it answers, and in the other two a
    // turned cube's longer labels were enough to do it. Stacked, the facts plus
    // the cube block need 674 px (en) to 824 px (el), and the height is free:
    // a cube decision is only ever shown when there are no dice, hence no
    // candidate list.
    //
    // One table rather than two stacked tables, because two tables size their
    // columns independently: the blocks came out different widths with nothing
    // lining up under anything. Sharing one column grid makes the two blocks
    // read as one object — same left edge, same right edge, same column stops,
    // the side markers in a single column.
    //
    // bottom/top: {win, gammon, backgammon, cubeless} | null — probabilities
    // as fractions [0,1], cubeless as an equity. null renders a blank row
    // (structure is never conditioned on whether a value has landed yet —
    // ADR-0017 rule 3), not a hidden one.
    // bottomEPC/topEPC: race.EPCResult-shaped {epc, pipCount, wastage,
    // meanRolls, stdDev} | null — the race block appears only when at least
    // one is present.
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

    // The shared grid is as wide as the widest block: five value columns when
    // the race block is there, four otherwise. The probability block then ends
    // in one empty cell rather than in a column of its own width — that empty
    // cell IS the alignment.
    let dataCols = $derived(showRace ? 5 : 4);
    let probHeaders = $derived([$t('epc.facts.gain'), $t('epc.facts.gammon'), $t('epc.facts.backgammon'), $t('epc.facts.cubelessEquity')]);
    let raceHeaders = $derived([$t('epc.epc'), $t('epc.pipCount'), $t('epc.wastage'), $t('epc.avgRolls'), $t('epc.stdDev')]);

    // The one-sided table these numbers came from, shown only when it is not
    // the ordinary six points (ADR-0027 §9). At six the label would be noise;
    // beyond it, it is the difference between "this side is home" and "this
    // side has a chequer on the 8-point and was answered anyway", which the
    // reader has no other way of knowing. It goes in the race block's empty
    // corner cell, so no column moves (ADR-0021).
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

    // Each block is three rows of the same shape — a side, the other side,
    // their difference — so the cells are computed here and the markup is
    // written once, in the snippet below. The Δ row of the race block ends in
    // two dashes: averaging rolls or differencing standard deviations across
    // sides means nothing.
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

<!-- One table, one column grid, two blocks: the probability vector and the
     race numbers share their column stops and their side-marker column, so the
     pair reads as a single object rather than as two tables that happen to sit
     one above the other (ADR-0021). -->
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
        color: #777;
        text-transform: uppercase;
        letter-spacing: 0.3px;
        font-weight: 600;
    }

    /* Hairlines separate values from values, never a header from the block it
       heads — and the second block opens on a little air instead of a rule. */
    tbody tr + tr td {
        border-top: 1px solid #eee;
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
        color: #444;
    }

    .main-value {
        font-weight: 600;
        color: var(--color-primary);
        font-variant-numeric: tabular-nums;
    }

    td {
        font-variant-numeric: tabular-nums;
        color: #222;
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
        color: #aaa;
        letter-spacing: 2px;
    }

    tr.masked:hover td {
        background: #f5f5f5;
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
