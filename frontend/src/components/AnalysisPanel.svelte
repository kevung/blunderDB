<script>
    import { analysisStore } from '../stores/analysisStore'; // Import analysisStore
    import { positionStore } from '../stores/positionStore'; // Import positionStore
    import { showAnalysisStore, showFilterLibraryPanelStore, showCommentStore, statusBarModeStore } from '../stores/uiStore'; // Import showAnalysisStore
    export let visible = false;
    export let onClose;

    let analysisData;
    let cubeValue;

    // Subscribe to analysisStore to get the analysis data
    analysisStore.subscribe(value => {
        analysisData = value;
    });

    // Subscribe to positionStore to get the cube value
    positionStore.subscribe(value => {
        cubeValue = value.cube.value;
    });

    showAnalysisStore.subscribe(value => {
        visible = value;
        if(visible) {
            console.log('Panel is now visible');
            console.log('Received Analysis Data:', analysisData);
            console.log('Analysis Type:', analysisData.analysisType);
            console.log('Checker Analysis:', Array.isArray(analysisData.checkerAnalysis) ? analysisData.checkerAnalysis : []);
            setTimeout(() => {
                const analysisEl = document.getElementById('analysisPanel');
                if (analysisEl) {
                    analysisEl.focus();
                }
            }, 0);
        } else {
            console.log('Panel is not visible');
        }
    });

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            onClose();
        }
    }

    function formatEquity(value) {
        return value >= 0 ? `+${value.toFixed(3)}` : value.toFixed(3);
    }

    function getDecisionLabel(decision) {
        if (cubeValue >= 1) {
            return decision.replace('Double', 'Redouble');
        }
        return decision;
    }
</script>

{#if visible}
    <section class="analysis-panel" role="dialog" aria-modal="true" id="analysisPanel" on:keydown={handleKeyDown}>
        <button type="button" class="close-icon" on:click={onClose} aria-label="Close" on:keydown={handleKeyDown}>×</button>
        <div class="analysis-content">
            {#if analysisData.analysisType === 'DoublingCube' && analysisData.doublingCubeAnalysis}
                <div class="tables-container">
                    <table class="left-table">
                        <tbody>
                            <tr>
                                <th></th>
                                <th>P</th>
                                <th>O</th>
                            </tr>
                            <tr>
                                <td>W</td>
                                <td>{(analysisData.doublingCubeAnalysis.playerWinChances || 0).toFixed(2)}</td>
                                <td>{(analysisData.doublingCubeAnalysis.opponentWinChances || 0).toFixed(2)}</td>
                            </tr>
                            <tr>
                                <td>G</td>
                                <td>{(analysisData.doublingCubeAnalysis.playerGammonChances || 0).toFixed(2)}</td>
                                <td>{(analysisData.doublingCubeAnalysis.opponentGammonChances || 0).toFixed(2)}</td>
                            </tr>
                            <tr>
                                <td>B</td>
                                <td>{(analysisData.doublingCubeAnalysis.playerBackgammonChances || 0).toFixed(2)}</td>
                                <td>{(analysisData.doublingCubeAnalysis.opponentBackgammonChances || 0).toFixed(2)}</td>
                            </tr>
                            <tr>
                                <td>ND Eq</td>
                                <td colspan="2">{formatEquity(analysisData.doublingCubeAnalysis.cubelessNoDoubleEquity || 0)}</td>
                            </tr>
                            <tr>
                                <td>D Eq</td>
                                <td colspan="2">{formatEquity(analysisData.doublingCubeAnalysis.cubelessDoubleEquity || 0)}</td>
                            </tr>
                        </tbody>
                    </table>
                    <table class="right-table">
                        <tbody>
                            <tr>
                                <th>Decision</th>
                                <th>Equity</th>
                                <th>Error</th>
                            </tr>
                            <tr>
                                <td>{getDecisionLabel('No Double')}</td>
                                <td>{formatEquity(analysisData.doublingCubeAnalysis.cubefulNoDoubleEquity || 0)}</td>
                                <td>{formatEquity(analysisData.doublingCubeAnalysis.cubefulNoDoubleError || 0)}</td>
                            </tr>
                            <tr>
                                <td>{getDecisionLabel('Double/Take')}</td>
                                <td>{formatEquity(analysisData.doublingCubeAnalysis.cubefulDoubleTakeEquity || 0)}</td>
                                <td>{formatEquity(analysisData.doublingCubeAnalysis.cubefulDoubleTakeError || 0)}</td>
                            </tr>
                            <tr>
                                <td>{getDecisionLabel('Double/Pass')}</td>
                                <td>{formatEquity(analysisData.doublingCubeAnalysis.cubefulDoublePassEquity || 0)}</td>
                                <td>{formatEquity(analysisData.doublingCubeAnalysis.cubefulDoublePassError || 0)}</td>
                            </tr>
                            <tr class="best-action-row">
                                <td>Best Action</td>
                                <td colspan="2">{analysisData.doublingCubeAnalysis.bestCubeAction}</td>
                            </tr>
                        </tbody>
                    </table>
                    <table class="info-table">
                        <tbody>
                            <tr>
                                <th>Analysis Depth</th>
                                <td>{analysisData.doublingCubeAnalysis.analysisDepth}</td>
                            </tr>
                            <tr>
                                <th>Engine Version</th>
                                <td>{analysisData.analysisEngineVersion}</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            {/if}

            {#if analysisData.analysisType === 'CheckerMove' && analysisData.checkerAnalysis}
                <table class="checker-table">
                    <tbody>
                        <tr>
                            <th>Move</th>
                            <th>Equity</th>
                            <th>Error</th>
                            <th>P W</th>
                            <th>P G</th>
                            <th>P B</th>
                            <th>O W</th>
                            <th>O G</th>
                            <th>O B</th>
                            <th>Depth</th>
                        </tr>
                        {#each analysisData.checkerAnalysis.moves as move}
                            <tr>
                                <td>{move.move}</td>
                                <td>{formatEquity(move.equity || 0)}</td>
                                <td>{formatEquity(move.equityError || 0)}</td>
                                <td>{(move.playerWinChance || 0).toFixed(2)}</td>
                                <td>{(move.playerGammonChance || 0).toFixed(2)}</td>
                                <td>{(move.playerBackgammonChance || 0).toFixed(2)}</td>
                                <td>{(move.opponentWinChance || 0).toFixed(2)}</td>
                                <td>{(move.opponentGammonChance || 0).toFixed(2)}</td>
                                <td>{(move.opponentBackgammonChance || 0).toFixed(2)}</td>
                                <td>{move.analysisDepth}</td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            {/if}
        </div>
    </section>
{/if}

<style>
    .analysis-panel {
        position: fixed;
        width: 100%;
        bottom: 0;
        left: 0;
        right: 0;
        height: 178px; /* Set a fixed height */
        overflow-y: auto;
        background-color: white;
        border-top: 1px solid rgba(0, 0, 0, 0.1);
        padding: 10px;
        box-sizing: border-box;
        z-index: 5;
        outline: none;
        resize: none;
    }

    .close-icon {
        position: absolute;
        top: -6px;
        right: 4px;
        font-size: 24px;
        font-weight: bold;
        color: #666;
        cursor: pointer;
        background: none;
        border: none;
        padding: 0;
        transition: color 0.3s ease;
    }

    .close-icon:hover {
        color: #333;
    }

    .analysis-content {
        font-size: 12px; /* Reduce font size */
        color: black; /* Set text color */
    }

    .tables-container {
        display: flex;
        justify-content: space-between;
    }

    .left-table, .right-table, .info-table {
        width: 28%; /* Reduce width for the first and third tables */
        border-collapse: collapse;
        font-size: 12px; /* Ensure same font size */
    }

    .left-table th:nth-child(1) {
        width: 20px; /* Reduce width for the first column */
    }

    .right-table th:nth-child(1) {
        width: 60px; /* Reduce width for the decision column */
    }

    .info-table th, .info-table td {
        border: 1px solid #ddd;
        padding: 2px; /* Reduce padding */
        text-align: center;
    }

    .checker-table {
        margin: 0 auto; /* Center the table */
        width: 100%;
        font-size: 12px; /* Ensure same font size */
        border-spacing: 0; /* Remove space between cells */
    }

    th, td {
        border: 1px solid #ddd;
        padding: 2px; /* Reduce padding */
        text-align: center;
    }

    th {
        background-color: #f2f2f2;
    }

    th:nth-child(1) {
        width: 150px; /* Sufficiently large for move column */
    }

    th:nth-child(n+2) {
        width: 60px; /* Fixed width for equity and percentage columns */
    }

    .checker-table th:nth-child(1),
    .checker-table td:nth-child(1) {
        border-right: 2px solid #ccc; /* More discreet border between move and equity columns */
    }

    .checker-table th:nth-child(3),
    .checker-table td:nth-child(3) {
        border-right: 2px solid #ccc; /* More discreet border between error and PW columns */
    }

    .checker-table th:nth-child(6),
    .checker-table td:nth-child(6) {
        border-right: 2px solid #ccc; /* More discreet border between PB and OW columns */
    }

    .checker-table th:nth-child(9),
    .checker-table td:nth-child(9) {
        border-right: 2px solid #ccc; /* More discreet border between OB and depth columns */
    }

    .checker-table tr:nth-child(even) {
        background-color: #fdfdfd; /* More discreet alternating row color */
    }

    .checker-table tr:nth-child(odd) {
        background-color: #ffffff; /* More discreet alternating row color */
    }

    .best-action-row {
        font-weight: bold;
        color: #000000; /* Subtle color change for emphasis */
    }

</style>

