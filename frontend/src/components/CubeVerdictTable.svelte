<script>
    import { t } from '../i18n';
    import { cubeRows, cubeInfoRows } from '../utils/analysisRows.js';

    // The cube Decision in its one shape (ADR-0020): three options in canonical
    // order plus the verdict, for EvalPanel and AnalysisPanel (ADR-0017 rule 5).
    // Position facts live in PositionFactsTable. `decision` comes from
    // utils/cubeDecision.js, cells from utils/analysisRows.js; `cubeAnalysis`
    // only feeds the depth/engine footer, which showInfo hides (ADR-0018 rule 4).
    //
    // masked (Défi, ADR-0020 rule 7): values replaced in place, best-row
    // emphasis suppressed too, or the bold line would give the answer away.
    // isMoney (ADR-0016 point 6): the equity header's referential; undefined
    // keeps it plain. jacoby / beaver: the position's money rule flags, shown
    // since they change the verdict; never passed at match play, where the XGID
    // field means Crawford. maxCube: log2 exponent of the source's ceiling, which
    // the built-in evaluator does not model — the one visible reason blunderDB
    // and XG can disagree; 0 renders nothing.
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
        beaver = false,
        maxCube = 0
    } = $props();

    let block = $derived(cubeRows(decision, { t: $t, cubeValue, isPlayedCubeAction, masked, isMoney }));
    let info = $derived(cubeInfoRows(cubeAnalysis, { t: $t, engineFallback: engineVersionFallback }));
    let rules = $derived([jacoby && $t('cube.jacoby'), beaver && $t('cube.beaver'), maxCube > 0 && $t('cube.maxCube', { value: 2 ** maxCube })].filter(Boolean));
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
    /* Quiet badges, not a column: most positions carry neither flag. */
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

    /* ADR-0018 rule 5 idiom (ADR-0020): hairlines, small grey headers, tabular figures. */
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
        color: var(--color-text-muted);
        text-transform: uppercase;
        letter-spacing: 0.3px;
        font-weight: 600;
    }

    td {
        color: var(--color-text);
    }

    .option {
        text-align: left;
    }

    tbody tr + tr td,
    tbody tr + tr th {
        border-top: 1px solid #eee;
    }

    /* `played` = background, `best` = weight and colour: both can show (ADR-0020). */
    tbody tr.played td {
        background-color: color-mix(in srgb, #ffc107 20%, var(--color-surface));
    }

    tbody tr.best td {
        font-weight: 600;
        color: var(--color-primary);
    }

    /* The verdict may be a sentence: wrap rather than widen (ADR-0018: no scroll). */
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
