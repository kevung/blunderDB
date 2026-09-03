<script>
    import { t } from '../i18n';
    import { cubeRows, cubeInfoRows } from '../utils/analysisRows.js';

    // The cube Decision, in the one shape it has whatever regime produced it
    // (ADR-0020): three named options in rows, canonical order, never sorted,
    // plus the verdict. Mounted by EPCPanel (a live evaluation) and by
    // AnalysisPanel (a stored record, possibly several engines side by side) —
    // one rendering, two content rules, which is ADR-0017 rule 5.
    //
    // Win/gammon/backgammon chances and the cubeless equity are NOT here: they
    // are position facts and live in PositionFactsTable (ADR-0017 rule 1).
    // This component owns the decision and nothing else.
    //
    // `decision` comes from utils/cubeDecision.js — the single place the two
    // source shapes (race.Money, domain.DoublingCubeAnalysis) become one
    // object. `cubeAnalysis` is kept only to feed the depth/engine footer.
    //
    // The cells themselves — labels, formatted equities, the verdict text, the
    // played/best marks — come from utils/analysisRows.js, the same rows the
    // copied image paints: this component lays them out and nothing more.
    //
    // showInfo (ADR-0018 rule 4): EPCPanel hides that footer — depth and engine
    // are named once in its own strip, since every row of a live evaluation
    // shares them — while AnalysisPanel keeps it, a stored record's provenance
    // being shown nowhere else.
    //
    // masked (Défi, ADR-0020 rule 7): the values are replaced in place and the
    // structure stays. The best-row emphasis is suppressed with them — it is
    // the verdict's only other carrier, so leaving it on would let the exercise
    // be solved by looking for the bold line.
    //
    // isMoney (ADR-0016 point 6, #190/C.3): states the equity column's own
    // referential — undefined (the caller has no position to read one from)
    // keeps the plain, scale-silent header.
    //
    // jacoby / beaver (#190/C.3 point 5): the position's own two money-game
    // rule flags (domain.Position.HasJacoby/HasBeaver) were stored and hashed
    // but never shown next to the verdict they change the value of — a
    // reader had no way to tell a "no double" reached under Jacoby from one
    // reached without it. Undefined/false renders nothing, so a match-play
    // position (where the same XGID field means Crawford, not these rules)
    // is unaffected by a caller that simply never passes them.
    let {
        decision,
        cubeAnalysis = null,
        cubeValue = 0,
        isPlayedCubeAction = () => false,
        engineVersionFallback = '',
        showInfo = true,
        masked = false,
        isMoney = undefined,
        jacoby = false,
        beaver = false
    } = $props();

    let block = $derived(cubeRows(decision, { t: $t, cubeValue, isPlayedCubeAction, masked, isMoney }));
    let info = $derived(cubeInfoRows(cubeAnalysis, { t: $t, engineFallback: engineVersionFallback }));
    let rules = $derived([jacoby && $t('cube.jacoby'), beaver && $t('cube.beaver')].filter(Boolean));
</script>

<table class="cube-table">
    <thead>
        <tr>
            {#each block.header as label, i (i)}
                <th>{label}</th>
            {/each}
        </tr>
    </thead>
    <tbody>
        {#each block.rows as row (row.key)}
            <tr class:played={row.highlight} class:best={row.best}>
                <td class="option">{row.label}</td>
                <td class="equity">{row.cells[0]}</td>
                <td class="error">{row.cells[1]}</td>
            </tr>
        {/each}
        <tr class="verdict-row" class:unavailable={block.verdict.unavailable}>
            <td class="option">{block.verdict.label}</td>
            <td colspan="2" class="verdict" class:japanese-text={block.verdict.text.includes('ダブル')}>{block.verdict.text}</td>
        </tr>
    </tbody>
</table>
{#if rules.length > 0}
    <div class="cube-rules">
        {#each rules as rule (rule)}
            <span class="rule-badge">{rule}</span>
        {/each}
    </div>
{/if}
{#if showInfo && cubeAnalysis}
    <table class="info-table">
        <tbody>
            {#each info as row (row.label)}
                <tr>
                    <th>{row.label}</th>
                    <td>{row.cells[0]}</td>
                </tr>
            {/each}
        </tbody>
    </table>
{/if}

<style>
    /* Jacoby/Beaver (#190/C.3 point 5): small, quiet badges next to the table
       they change the value of — not a fourth column, since they apply to at
       most a money position and most positions carry neither. */
    .cube-rules {
        display: flex;
        gap: 6px;
        margin-top: 4px;
    }

    .rule-badge {
        font-size: var(--font-size-small);
        color: #777;
        border: 1px solid #ddd;
        border-radius: 3px;
        padding: 1px 6px;
        text-transform: uppercase;
        letter-spacing: 0.3px;
    }

    /* ADR-0018 rule 5's idiom, which ADR-0020 finishes applying here: no cell
       grid, hairline horizontal separators, small grey uppercase headers on a
       transparent ground, tabular figures. Hierarchy comes from weight and
       colour (ADR-0008). */
    .cube-table,
    .info-table {
        border-collapse: collapse;
        font-size: var(--font-size-base);
    }

    th,
    td {
        padding: 2px 10px;
        text-align: center;
        white-space: nowrap;
        font-variant-numeric: tabular-nums;
    }

    thead th,
    .info-table th {
        font-size: var(--font-size-small);
        color: #777;
        text-transform: uppercase;
        letter-spacing: 0.3px;
        font-weight: 600;
    }

    td {
        color: #222;
    }

    .option {
        text-align: left;
    }

    tbody tr + tr td,
    tbody tr + tr th {
        border-top: 1px solid #eee;
    }

    /* Two orthogonal channels on one row: `played` is a background, `best` is
       weight and colour, so a played action that is also the best reads as both
       (ADR-0020). */
    tbody tr.played td {
        background-color: #fff3cd;
    }

    tbody tr.best td {
        font-weight: 600;
        color: var(--color-primary);
    }

    /* The verdict is the one cell whose text can be a sentence rather than a
       label — "Too good to double, take" (ADR-0019) is twice the width of
       "Double, Take". Let it wrap instead of widening the whole table, which
       the Eval panel has no room to absorb (ADR-0018: no scroll). */
    .verdict-row td {
        border-top: 2px solid #e0e0e0;
        font-weight: 600;
        color: var(--color-primary);
        white-space: normal;
    }

    /* A named absence is not an answer: it is set apart from a verdict so the
       two are never read as the same kind of statement. */
    .verdict-row.unavailable td {
        font-weight: 400;
        font-style: italic;
        color: #777;
    }

    .japanese-text {
        font-family: 'Noto Sans JP', sans-serif;
    }
</style>
