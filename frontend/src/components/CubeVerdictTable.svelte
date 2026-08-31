<script>
    import { t } from '../i18n';

    // Rendering component for one engine's cube-decision verdict (ND/DT/DP,
    // equities, error, best action) — mounted by AnalysisPanel (a stored
    // record, possibly several engines side by side) and by EPCPanel (a live
    // evaluation, wired in #125). Which action counts as "played" is a
    // caller concern (MATCH mode vs. browsing several played actions across
    // positions) so it stays a callback, exactly like CandidateMovesTable's
    // isPlayedMove.
    //
    // Win/gammon/backgammon chances and the cubeless equity moved out to
    // PositionFactsTable (ADR-0017): they are a position fact, not part of
    // this decision, and showing them here duplicated a table the same
    // panels mount right next to this one. This component owns only the
    // decision itself — ND/DT/DP, their equities and error, the best
    // action — plus the depth/engine footer.
    let { cubeAnalysis, cubeValue = 0, isPlayedCubeAction = () => false, engineVersionFallback = '' } = $props();

    function formatEquity(value) {
        return value >= 0 ? `+${value.toFixed(3)}` : value.toFixed(3);
    }
</script>

<table class="right-table">
    <tbody>
        <tr>
            <th>{$t('analysis.decision')}</th>
            <th>{$t('analysis.equity')}</th>
            <th>{$t('analysis.error')}</th>
        </tr>
        <tr class:played={isPlayedCubeAction('No Double')}>
            <td>{$t(cubeValue >= 1 ? 'analysis.noRedouble' : 'analysis.noDouble')}</td>
            <td>{formatEquity(cubeAnalysis.cubefulNoDoubleEquity || 0)}</td>
            <td>{formatEquity(cubeAnalysis.cubefulNoDoubleError || 0)}</td>
        </tr>
        <tr class:played={isPlayedCubeAction('Double') && isPlayedCubeAction('Take')}>
            <td>{$t(cubeValue >= 1 ? 'analysis.redoubleTake' : 'analysis.doubleTake')}</td>
            <td>{formatEquity(cubeAnalysis.cubefulDoubleTakeEquity || 0)}</td>
            <td>{formatEquity(cubeAnalysis.cubefulDoubleTakeError || 0)}</td>
        </tr>
        <tr class:played={isPlayedCubeAction('Double') && isPlayedCubeAction('Pass')}>
            <td>{$t(cubeValue >= 1 ? 'analysis.redoublePass' : 'analysis.doublePass')}</td>
            <td>{formatEquity(cubeAnalysis.cubefulDoublePassEquity || 0)}</td>
            <td>{formatEquity(cubeAnalysis.cubefulDoublePassError || 0)}</td>
        </tr>
        <tr class="best-action-row {cubeAnalysis.bestCubeAction && cubeAnalysis.bestCubeAction.includes('ダブル') ? 'japanese-text' : ''}">
            <td>{$t('analysis.bestAction')}</td>
            <td colspan="2">{cubeAnalysis.bestCubeAction || ''}</td>
        </tr>
    </tbody>
</table>
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

<style>
    .right-table,
    .info-table {
        width: 28%;
        border-collapse: collapse;
        font-size: var(--font-size-base);
    }

    .right-table th:nth-child(1) {
        width: 60px;
    }

    .info-table th,
    .info-table td {
        border: 1px solid #ddd;
        padding: 2px;
        text-align: center;
    }

    th,
    td {
        border: 1px solid #ddd;
        padding: 2px;
        text-align: center;
    }

    th {
        background-color: #f2f2f2;
    }

    .right-table tr.played {
        background-color: #fff3cd !important;
    }

    .best-action-row {
        font-weight: bold;
        color: #000000;
    }

    .japanese-text {
        font-family: 'Noto Sans JP', sans-serif;
    }

    @container (max-width: 600px) {
        .right-table,
        .info-table {
            width: 100%;
        }
    }
</style>
