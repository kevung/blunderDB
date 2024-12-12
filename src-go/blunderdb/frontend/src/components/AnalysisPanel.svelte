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
</script>

{#if visible}
    <div class="analysis-panel" tabindex="0" id="analysisPanel" on:keydown={handleKeyDown}>
        <div class="close-icon" on:click={onClose}>×</div>
        <div class="analysis-content">
            {#if analysisData.analysisType === 'DoublingCube'}
                <h4>Doubling Cube Analysis</h4>
                <table>
                    <tr>
                        <th></th>
                        <th>P</th>
                        <th>O</th>
                    </tr>
                    <tr>
                        <td>W</td>
                        <td>{analysisData.doublingCubeAnalysis.playerWinChances}%</td>
                        <td>{analysisData.doublingCubeAnalysis.opponentWinChances}%</td>
                    </tr>
                    <tr>
                        <td>G</td>
                        <td>{analysisData.doublingCubeAnalysis.playerGammonChances}%</td>
                        <td>{analysisData.doublingCubeAnalysis.opponentGammonChances}%</td>
                    </tr>
                    <tr>
                        <td>B</td>
                        <td>{analysisData.doublingCubeAnalysis.playerBackgammonChances}%</td>
                        <td>{analysisData.doublingCubeAnalysis.opponentBackgammonChances}%</td>
                    </tr>
                    <tr>
                        <td>Cubeless No Double Equity</td>
                        <td colspan="2">{analysisData.doublingCubeAnalysis.cubelessNoDoubleEquity}</td>
                    </tr>
                    <tr>
                        <td>Cubeless Double Equity</td>
                        <td colspan="2">{analysisData.doublingCubeAnalysis.cubelessDoubleEquity}</td>
                    </tr>
                    <tr>
                        <td>Cubeful No Double Equity</td>
                        <td colspan="2">{analysisData.doublingCubeAnalysis.cubefulNoDoubleEquity} (Error: {analysisData.doublingCubeAnalysis.cubefulNoDoubleError})</td>
                    </tr>
                    <tr>
                        <td>Cubeful Double Take Equity</td>
                        <td colspan="2">{analysisData.doublingCubeAnalysis.cubefulDoubleTakeEquity} (Error: {analysisData.doublingCubeAnalysis.cubefulDoubleTakeError})</td>
                    </tr>
                    <tr>
                        <td>Cubeful Double Pass Equity</td>
                        <td colspan="2">{analysisData.doublingCubeAnalysis.cubefulDoublePassEquity} (Error: {analysisData.doublingCubeAnalysis.cubefulDoublePassError})</td>
                    </tr>
                    <tr>
                        <td>Best Cube Action</td>
                        <td colspan="2">{analysisData.doublingCubeAnalysis.bestCubeAction}</td>
                    </tr>
                    <tr>
                        <td>Wrong Pass Percentage</td>
                        <td colspan="2">{analysisData.doublingCubeAnalysis.wrongPassPercentage}%</td>
                    </tr>
                    <tr>
                        <td>Wrong Take Percentage</td>
                        <td colspan="2">{analysisData.doublingCubeAnalysis.wrongTakePercentage}%</td>
                    </tr>
                </table>
            {/if}

            {#if analysisData.analysisType === 'CheckerMove'}
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
                    </tr>
                    {#each analysisData.checkerAnalysis.moves as move}
                        <tr>
                            <td>{move.move}</td>
                            <td>{move.equity.toFixed(3)}</td>
                            <td>{move.equityError.toFixed(3)}</td>
                            <td>{move.playerWinChance.toFixed(2)}</td>
                            <td>{move.playerGammonChance.toFixed(2)}</td>
                            <td>{move.playerBackgammonChance.toFixed(2)}</td>
                            <td>{move.opponentWinChance.toFixed(2)}</td>
                            <td>{move.opponentGammonChance.toFixed(2)}</td>
                            <td>{move.opponentBackgammonChance.toFixed(2)}</td>
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
        padding: 15px; /* Reduce padding to optimize space */
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
        font-size: 14px; /* Reduce font size */
        color: black; /* Set text color */
    }

    table {
        width: 100%;
        border-collapse: collapse;
        margin-top: 10px;
    }

    .checker-table {
        margin: 0 auto; /* Center the table */
    }

    th, td {
        border: 1px solid #ddd;
        padding: 0px; /* Reduce padding to make row height smaller */
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
</style>

