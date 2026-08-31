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
    // bottom/top: {win, gammon, backgammon, cubeless} | null — probabilities
    // as fractions [0,1], cubeless as an equity. null renders a blank row
    // (structure is never conditioned on whether a value has landed yet —
    // ADR-0017 rule 3), not a hidden one.
    // bottomEPC/topEPC: race.EPCResult-shaped {epc, pipCount, wastage,
    // meanRolls, stdDev} | null — the race columns appear only when at
    // least one is present.
    let { bottom = null, top = null, bottomEPC = null, topEPC = null, maskedBottom = false, maskedTop = false, onRevealBottom = () => {}, onRevealTop = () => {}, showProbabilities = true } = $props();

    let showRace = $derived(!!(bottomEPC || topEPC));
    let bothShown = $derived(!maskedBottom && !maskedTop);

    const HIDDEN = '···';
    const DASH = '—';
    const show = (masked, v) => (masked ? HIDDEN : (v ?? DASH));
    const pct = (x) => (x == null ? null : (100 * x).toFixed(2));
    const eq = (x) => (x == null ? null : (x >= 0 ? '+' : '') + x.toFixed(3));
    const sd = (x, digits) => (x == null ? null : (x >= 0 ? '+' : '') + x.toFixed(digits));

    function delta(a, b, fmt) {
        if (a == null || b == null) return null;
        return fmt(a - b);
    }
</script>

<table class="facts-table">
    <thead>
        <tr>
            <th></th>
            {#if showProbabilities}
                <th>{$t('epc.facts.gain')}</th>
                <th>{$t('epc.facts.gammon')}</th>
                <th>{$t('epc.facts.backgammon')}</th>
                <th>{$t('epc.facts.cubelessEquity')}</th>
            {/if}
            {#if showRace}
                <th class="race-col-start">{$t('epc.epc')}</th>
                <th>{$t('epc.pipCount')}</th>
                <th>{$t('epc.wastage')}</th>
                <th>{$t('epc.avgRolls')}</th>
                <th>{$t('epc.stdDev')}</th>
            {/if}
        </tr>
    </thead>
    <tbody>
        <tr class:masked={maskedBottom} onclick={() => maskedBottom && onRevealBottom()} title={maskedBottom ? $t('epc.clickToReveal') : $t('epc.bottomBlack')}>
            <td class="row-label"><span class="player-indicator bottom"></span></td>
            {#if showProbabilities}
                <td class="main-value">{show(maskedBottom, pct(bottom?.win))}</td>
                <td>{show(maskedBottom, pct(bottom?.gammon))}</td>
                <td>{show(maskedBottom, pct(bottom?.backgammon))}</td>
                <td>{show(maskedBottom, eq(bottom?.cubeless))}</td>
            {/if}
            {#if showRace}
                <td class="race-col-start">{show(maskedBottom, bottomEPC?.epc?.toFixed(2))}</td>
                <td>{show(maskedBottom, bottomEPC?.pipCount)}</td>
                <td>{show(maskedBottom, bottomEPC?.wastage?.toFixed(2))}</td>
                <td>{show(maskedBottom, bottomEPC?.meanRolls?.toFixed(3))}</td>
                <td>{show(maskedBottom, bottomEPC?.stdDev?.toFixed(3))}</td>
            {/if}
        </tr>
        <tr class:masked={maskedTop} onclick={() => maskedTop && onRevealTop()} title={maskedTop ? $t('epc.clickToReveal') : $t('epc.topWhite')}>
            <td class="row-label"><span class="player-indicator top"></span></td>
            {#if showProbabilities}
                <td class="main-value">{show(maskedTop, pct(top?.win))}</td>
                <td>{show(maskedTop, pct(top?.gammon))}</td>
                <td>{show(maskedTop, pct(top?.backgammon))}</td>
                <td>{show(maskedTop, eq(top?.cubeless))}</td>
            {/if}
            {#if showRace}
                <td class="race-col-start">{show(maskedTop, topEPC?.epc?.toFixed(2))}</td>
                <td>{show(maskedTop, topEPC?.pipCount)}</td>
                <td>{show(maskedTop, topEPC?.wastage?.toFixed(2))}</td>
                <td>{show(maskedTop, topEPC?.meanRolls?.toFixed(3))}</td>
                <td>{show(maskedTop, topEPC?.stdDev?.toFixed(3))}</td>
            {/if}
        </tr>
        {#if bottom || top || bottomEPC || topEPC}
            <tr class="delta-row" title={$t('epc.comparison')}>
                <td class="row-label">Δ</td>
                {#if showProbabilities}
                    <td class="main-value"
                        >{show(
                            !bothShown,
                            delta(bottom?.win, top?.win, (v) => sd(100 * v, 2))
                        )}</td
                    >
                    <td
                        >{show(
                            !bothShown,
                            delta(bottom?.gammon, top?.gammon, (v) => sd(100 * v, 2))
                        )}</td
                    >
                    <td
                        >{show(
                            !bothShown,
                            delta(bottom?.backgammon, top?.backgammon, (v) => sd(100 * v, 2))
                        )}</td
                    >
                    <td
                        >{show(
                            !bothShown,
                            delta(bottom?.cubeless, top?.cubeless, (v) => sd(v, 3))
                        )}</td
                    >
                {/if}
                {#if showRace}
                    <td class="race-col-start"
                        >{show(
                            !bothShown,
                            delta(bottomEPC?.epc, topEPC?.epc, (v) => sd(v, 2))
                        )}</td
                    >
                    <td
                        >{show(
                            !bothShown,
                            delta(bottomEPC?.pipCount, topEPC?.pipCount, (v) => sd(v, 0))
                        )}</td
                    >
                    <td
                        >{show(
                            !bothShown,
                            delta(bottomEPC?.wastage, topEPC?.wastage, (v) => sd(v, 2))
                        )}</td
                    >
                    <td>—</td>
                    <td>—</td>
                {/if}
            </tr>
        {/if}
    </tbody>
</table>

<style>
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

    tbody tr + tr td {
        border-top: 1px solid #eee;
    }

    .race-col-start {
        border-left: 2px solid #e0e0e0;
    }

    .row-label {
        font-size: var(--font-size-small);
        font-weight: 600;
        color: #444;
    }

    .main-value {
        font-weight: 600;
        color: #1a56c4;
        font-variant-numeric: tabular-nums;
    }

    td {
        font-variant-numeric: tabular-nums;
        color: #222;
    }

    .delta-row td {
        color: #666;
    }

    .delta-row .main-value {
        color: #666;
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
