<script>
    export let visible = false;
    export let onClose;
    export let analysisData = {};

    $: if (visible) {
        console.log('Panel is now visible');
        console.log('Received Analysis Data:', analysisData);
        console.log('Analysis Type:', analysisData.analysisType);
        console.log('Checker Analysis:', Array.isArray(analysisData.checkerAnalysis) ? analysisData.checkerAnalysis : []);
    } else {
        console.log('Panel is not visible');
    }

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            onClose();
        }
    }

    $: if (visible) {
        setTimeout(() => {
            const analysisEl = document.getElementById('analysisPanel');
            if (analysisEl) {
                analysisEl.focus();
            }
        }, 0);
    }

    function formatEquity(value) {
        return value >= 0 ? `+${value.toFixed(3)}` : value.toFixed(3);
    }
</script>

{#if visible}
    <div class="analysis-panel" tabindex="0" id="analysisPanel" on:keydown={handleKeyDown}>
        <div class="close-icon" on:click={onClose}>×</div>
        <div class="analysis-content">
            {#if analysisData.analysisType === 'DoublingCube' && analysisData.doublingCubeAnalysis}
                <div class="tables-container">
                    <table class="left-table">
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
                    </table>
                    <table class="right-table">
                        <tr>
                            <th>Decision</th>
                            <th>Equity</th>
                            <th>Error</th>
                        </tr>
                        <tr>
                            <td>No Double</td>
                            <td>{formatEquity(analysisData.doublingCubeAnalysis.cubefulNoDoubleEquity || 0)}</td>
                            <td>{formatEquity(analysisData.doublingCubeAnalysis.cubefulNoDoubleError || 0)}</td>
                        </tr>
                        <tr>
                            <td>Double/Take</td>
                            <td>{formatEquity(analysisData.doublingCubeAnalysis.cubefulDoubleTakeEquity || 0)}</td>
                            <td>{formatEquity(analysisData.doublingCubeAnalysis.cubefulDoubleTakeError || 0)}</td>
                        </tr>
                        <tr>
                            <td>Double/Pass</td>
                            <td>{formatEquity(analysisData.doublingCubeAnalysis.cubefulDoublePassEquity || 0)}</td>
                            <td>{formatEquity(analysisData.doublingCubeAnalysis.cubefulDoublePassError || 0)}</td>
                        </tr>
                        <tr class="best-action-row">
                            <td>Best Action</td>
                            <td colspan="2">{analysisData.doublingCubeAnalysis.bestCubeAction}</td>
                        </tr>
                    </table>
                    <table class="info-table">
                        <tr>
                            <th>XGID</th>
                            <td>{analysisData.xgid}</td>
                        </tr>
                        <tr>
                            <th>Engine Version</th>
                            <td>{analysisData.analysisEngineVersion}</td>
                        </tr>
                        <tr>
                            <th>Analysis Depth</th>
                            <td>{analysisData.doublingCubeAnalysis.analysisDepth}</td>
                        </tr>
                    </table>
                </div>
            {/if}

            {#if analysisData.analysisType === 'CheckerMove' && analysisData.checkerAnalysis}
                <table class="checker-table">
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
                </table>
            {/if}
        </div>
    </div>
{/if}

<style>
    .analysis-panel {
        position: absolute;
        width: 100%;
        bottom: 0;
        left: 0;
        right: 0;
        height: 23vh; /* Reduce height */
        overflow-y: auto;
        background-color: white;
        border-top: 1px solid rgba(0, 0, 0, 0.1);
        border-radius: 0px;
        padding: 10px; /* Reduce padding to optimize space */
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
        transition: background-color 0.3s ease, opacity 0.3s ease;
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

    .best-action-row {
        font-weight: bold;
        color: #000000; /* Subtle color change for emphasis */
    }

    .best-action {
        margin-top: 10px;
        text-align: center;
        width: 100%;
    }
</style>

