<script>
    import { t } from '../i18n';
    import { CUBE_OPTIONS, DECISION_STATE } from '../utils/cubeDecision.js';

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
    // showInfo (ADR-0018 rule 4): EPCPanel hides that footer — depth and engine
    // are named once in its own strip, since every row of a live evaluation
    // shares them — while AnalysisPanel keeps it, a stored record's provenance
    // being shown nowhere else.
    //
    // masked (Défi, ADR-0020 rule 7): the values are replaced in place and the
    // structure stays. The best-row emphasis is suppressed with them — it is
    // the verdict's only other carrier, so leaving it on would let the exercise
    // be solved by looking for the bold line.
    let { decision, cubeAnalysis = null, cubeValue = 0, isPlayedCubeAction = () => false, engineVersionFallback = '', showInfo = true, masked = false } = $props();

    const HIDDEN = '···';

    const OPTION_LABELS = {
        no_double: ['analysis.noDouble', 'analysis.noRedouble'],
        double_take: ['analysis.doubleTake', 'analysis.redoubleTake'],
        double_pass: ['analysis.doublePass', 'analysis.redoublePass']
    };

    // cubeValue is the log2 exponent everywhere in blunderDB (see the XGID
    // contract), so >= 1 means the cube has already been turned at least once:
    // the options are redoubles. The race decision never made this switch and
    // wrote "no double" on a cube sitting at 2 — one of the defects ADR-0020
    // removes by having a single rendering.
    let labelKeys = $derived(Object.fromEntries(CUBE_OPTIONS.map((k) => [k, OPTION_LABELS[k][cubeValue >= 1 ? 1 : 0]])));

    let options = $derived(decision?.options ?? CUBE_OPTIONS.map((key) => ({ key, equity: null, error: null })));
    let state = $derived(decision?.state ?? DECISION_STATE.PENDING);
    let best = $derived(masked ? null : decision?.best);

    function formatEquity(value) {
        if (value == null) return '';
        return value >= 0 ? `+${value.toFixed(3)}` : value.toFixed(3);
    }

    function cell(value) {
        return masked ? HIDDEN : formatEquity(value);
    }

    // Legacy action strings: AnalysisPanel's isPlayedCubeAction speaks the
    // vocabulary stored in playedCubeAction ("Double", "Take", …), so the
    // canonical keys are translated back at the call rather than changing a
    // contract two panels and the board already share.
    function isPlayed(key) {
        if (key === 'no_double') return isPlayedCubeAction('No Double');
        if (key === 'double_take') return isPlayedCubeAction('Double') && isPlayedCubeAction('Take');
        return isPlayedCubeAction('Double') && isPlayedCubeAction('Pass');
    }

    // The single place the block's state is named (ADR-0020 rule 4). Empty
    // means "still computing" and nothing else — never a refusal, never a
    // regime that will never answer, never a dead cube.
    let verdictText = $derived.by(() => {
        if (masked) return HIDDEN;
        switch (state) {
            case DECISION_STATE.PENDING:
                return '';
            case DECISION_STATE.NO_DECISION:
                return $t('cube.noDecision');
            case DECISION_STATE.REFUSED:
                return $t('cube.refused');
            case DECISION_STATE.CUBE_OPPONENT:
                return $t('cube.cubeOpponent');
            case DECISION_STATE.CRAWFORD:
                return $t('cube.crawford');
            default:
                // A live evaluation names its verdict by key, so it is
                // translated and keeps "too good"; a stored record carries its
                // analysing engine's own words and is reported verbatim.
                return decision?.verdict ? $t('cube.verdicts.' + decision.verdict) : (decision?.verdictText ?? '');
        }
    });

    let verdictUnavailable = $derived(state !== DECISION_STATE.VERDICT && state !== DECISION_STATE.PENDING);
</script>

<table class="cube-table">
    <thead>
        <tr>
            <th>{$t('analysis.decision')}</th>
            <th>{$t('analysis.equity')}</th>
            <th>{$t('analysis.error')}</th>
        </tr>
    </thead>
    <tbody>
        {#each options as option (option.key)}
            <tr class:played={!masked && isPlayed(option.key)} class:best={option.key === best}>
                <td class="option">{$t(labelKeys[option.key])}</td>
                <td class="equity">{cell(option.equity)}</td>
                <td class="error">{cell(option.error)}</td>
            </tr>
        {/each}
        <tr class="verdict-row" class:unavailable={verdictUnavailable}>
            <td class="option">{$t('analysis.bestAction')}</td>
            <td colspan="2" class="verdict" class:japanese-text={verdictText.includes('ダブル')}>{verdictText}</td>
        </tr>
    </tbody>
</table>
{#if showInfo && cubeAnalysis}
    <table class="info-table">
        <tbody>
            <tr>
                <th>{$t('analysis.analysisDepth')}</th>
                <td>{cubeAnalysis.analysisDepth}</td>
            </tr>
            <tr>
                <th>{$t('analysis.engine')}</th>
                <td>{cubeAnalysis.analysisEngine || engineVersionFallback}</td>
            </tr>
        </tbody>
    </table>
{/if}

<style>
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
        color: #1a56c4;
    }

    /* The verdict is the one cell whose text can be a sentence rather than a
       label — "Too good to double, take" (ADR-0019) is twice the width of
       "Double, Take". Let it wrap instead of widening the whole table, which
       the Eval panel has no room to absorb (ADR-0018: no scroll). */
    .verdict-row td {
        border-top: 2px solid #e0e0e0;
        font-weight: 600;
        color: #1a56c4;
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
