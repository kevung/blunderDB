<script>
    import { analysisStore, selectedMoveStore } from '../stores/analysisStore'; // Import analysisStore and selectedMoveStore
    import { positionStore, matchContextStore } from '../stores/positionStore'; // Import positionStore and matchContextStore
    import { showAnalysisStore, showFilterLibraryPanelStore, showCommentStore, statusBarModeStore } from '../stores/uiStore'; // Import showAnalysisStore
    export let visible = false;
    export let onClose;

    let analysisData;
    let cubeValue;
    let activeTab = 'checker'; // 'checker' or 'cube'
    let matchCtx;

    // Sorting state for checker analysis table
    let sortColumn = 'equity';  // default sort by equity
    let sortDirection = 'desc'; // default highest to lowest

    // Subscribe to matchContextStore
    matchContextStore.subscribe(value => {
        matchCtx = value;
        // Auto-switch tab based on current move type in match mode
        // But only if no move is currently selected (to avoid interfering with move navigation)
        if ($statusBarModeStore === 'MATCH' && matchCtx.isMatchMode && matchCtx.movePositions.length > 0 && !$selectedMoveStore) {
            const currentMovePos = matchCtx.movePositions[matchCtx.currentIndex];
            if (currentMovePos) {
                // If it's the first position of a game (move_number 0 or 1), force checker tab and clear cube data
                if (currentMovePos.move_number === 0 || currentMovePos.move_number === 1) {
                    activeTab = 'checker';
                    // Clear any existing cube analysis data
                    analysisStore.update(current => ({
                        ...current,
                        doublingCubeAnalysis: null,
                        playedCubeAction: ''
                    }));
                } else if (currentMovePos.move_type) {
                    activeTab = currentMovePos.move_type;
                }
            }
        }
    });

    // Subscribe to analysisStore to get the analysis data
    analysisStore.subscribe(value => {
        analysisData = value;
    });

    // Subscribe to positionStore to get the cube value
    positionStore.subscribe(value => {
        cubeValue = value.cube.value;
    });

    showAnalysisStore.subscribe(async value => {
        visible = value;
        if(visible) {
            // Pre-load cube analysis in match mode if current position is checker
            if ($statusBarModeStore === 'MATCH' && matchCtx.isMatchMode) {
                const currentMovePos = matchCtx.movePositions[matchCtx.currentIndex];
                if (currentMovePos && currentMovePos.move_type === 'checker') {
                    // If it's the first position of a game (move_number 0 or 1), clear cube data immediately
                    if (currentMovePos.move_number === 0 || currentMovePos.move_number === 1) {
                        analysisStore.update(current => ({
                            ...current,
                            doublingCubeAnalysis: null,
                            playedCubeAction: ''
                        }));
                    } else {
                        // Load cube analysis in background (only if there's one in the same game)
                        const hasCubeInGame = await loadCubeAnalysisForCurrentPosition();
                        // If no cube analysis found in current game, clear any previous cube data
                        if (!hasCubeInGame) {
                            analysisStore.update(current => ({
                                ...current,
                                doublingCubeAnalysis: null,
                                playedCubeAction: ''
                            }));
                        }
                    }
                }
            }
            
            setTimeout(() => {
                const analysisEl = document.getElementById('analysisPanel');
                if (analysisEl) {
                    analysisEl.focus();
                }
            }, 0);
        } else {
            // Clear selected move when panel is closed
            selectedMoveStore.set(null);
        }
    });

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            // Clear selection first if a move is selected
            if ($selectedMoveStore) {
                selectedMoveStore.set(null);
            } else {
                onClose();
            }
            return;
        }

        // Handle tab switching with 'd' (doubling/cube) key to toggle
        // Only allow if showTabs is true (not first position of game)
        if (showTabs && (event.key === 'd' || event.key === 'D')) {
            event.preventDefault();
            const newTab = activeTab === 'checker' ? 'cube' : 'checker';
            handleTabSwitch(newTab);
            return;
        }

        // Handle j/k and arrow keys for move navigation when a move is selected
        // This should work regardless of which tab is active
        if ($selectedMoveStore) {
            if (!sortedMoves || sortedMoves.length === 0) {
                return; // No moves to navigate
            }
            
            const currentIndex = sortedMoves.findIndex(m => m.move === $selectedMoveStore);
            
            if (event.key === 'j' || event.key === 'ArrowDown') {
                event.preventDefault();
                if (currentIndex >= 0 && currentIndex < sortedMoves.length - 1) {
                    selectedMoveStore.set(sortedMoves[currentIndex + 1].move);
                }
                return;
            } else if (event.key === 'k' || event.key === 'ArrowUp') {
                event.preventDefault();
                if (currentIndex > 0) {
                    selectedMoveStore.set(sortedMoves[currentIndex - 1].move);
                }
                return;
            }
        }
    }

    // Column definitions for sorting
    const sortableColumns = {
        'move': { key: 'move', type: 'string' },
        'equity': { key: 'equity', type: 'number' },
        'error': { key: 'equityError', type: 'number' },
        'pw': { key: 'playerWinChance', type: 'number' },
        'pg': { key: 'playerGammonChance', type: 'number' },
        'pb': { key: 'playerBackgammonChance', type: 'number' },
        'ow': { key: 'opponentWinChance', type: 'number' },
        'og': { key: 'opponentGammonChance', type: 'number' },
        'ob': { key: 'opponentBackgammonChance', type: 'number' },
        'depth': { key: 'analysisDepth', type: 'number' }
    };

    function handleSort(column) {
        if (sortColumn === column) {
            sortDirection = sortDirection === 'asc' ? 'desc' : 'asc';
        } else {
            sortColumn = column;
            // Default direction: desc for numeric, asc for string
            sortDirection = sortableColumns[column].type === 'string' ? 'asc' : 'desc';
        }
    }

    // Reactive sorted moves array
    $: sortedMoves = (() => {
        if (!analysisData?.checkerAnalysis?.moves) return [];
        const moves = [...analysisData.checkerAnalysis.moves];
        const col = sortableColumns[sortColumn];
        if (!col) return moves;
        return moves.sort((a, b) => {
            let va = a[col.key];
            let vb = b[col.key];
            if (col.type === 'number') {
                va = va || 0;
                vb = vb || 0;
                return sortDirection === 'asc' ? va - vb : vb - va;
            } else {
                va = va || '';
                vb = vb || '';
                const cmp = va.localeCompare(vb);
                return sortDirection === 'asc' ? cmp : -cmp;
            }
        });
    })();

    function getSortIndicator(column) {
        if (sortColumn !== column) return '';
        return sortDirection === 'asc' ? ' ▲' : ' ▼';
    }

    function handleMoveRowClick(move) {
        // Toggle selection: if clicking the same move, deselect it
        if ($selectedMoveStore === move.move) {
            selectedMoveStore.set(null);
        } else {
            selectedMoveStore.set(move.move);
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

    // Normalize a move string for comparison by sorting individual moves
    // "5/2 5/4" and "5/4 5/2" are the same move but in different order
    function normalizeMoveString(move) {
        if (!move) return '';
        // Split by spaces and sort the individual moves
        return move.split(' ').sort().join(' ');
    }

    function isPlayedMove(move) {
        if (!move.move) return false;
        
        const normalizedMoveStr = normalizeMoveString(move.move);
        
        // In MATCH mode, only highlight the current match's specific move
        if ($statusBarModeStore === 'MATCH' && matchCtx.isMatchMode) {
            // Use the single playedMove field which contains the current match's move
            if (analysisData.playedMove) {
                return normalizeMoveString(analysisData.playedMove) === normalizedMoveStr;
            }
            return false;
        }
        
        // In normal mode (browsing positions), highlight all played moves
        // Check the playedMoves array first
        if (analysisData.playedMoves && analysisData.playedMoves.length > 0) {
            for (const playedMove of analysisData.playedMoves) {
                if (normalizeMoveString(playedMove) === normalizedMoveStr) {
                    return true;
                }
            }
        }
        
        // Fallback to old single playedMove field for backward compatibility
        if (analysisData.playedMove) {
            return normalizeMoveString(analysisData.playedMove) === normalizedMoveStr;
        }
        
        return false;
    }

    function isPlayedCubeAction(action) {
        // In MATCH mode, only highlight the current match's specific cube action
        if ($statusBarModeStore === 'MATCH' && matchCtx.isMatchMode) {
            if (analysisData.playedCubeAction) {
                const normalizedAction = action.toLowerCase().replace(/\s+/g, '');
                const normalizedPlayed = analysisData.playedCubeAction.toLowerCase().replace(/\s+/g, '');
                return normalizedAction.includes(normalizedPlayed) || normalizedPlayed.includes(normalizedAction);
            }
            return false;
        }
        
        // In normal mode, highlight all played cube actions
        // Check the playedCubeActions array first
        if (analysisData.playedCubeActions && analysisData.playedCubeActions.length > 0) {
            // Normalize the action for comparison
            const normalizedAction = action.toLowerCase().replace(/\s+/g, '');
            
            for (const playedAction of analysisData.playedCubeActions) {
                const normalizedPlayed = playedAction.toLowerCase().replace(/\s+/g, '');
                if (normalizedAction.includes(normalizedPlayed) || normalizedPlayed.includes(normalizedAction)) {
                    return true;
                }
            }
        }
        
        // Fallback to old single playedCubeAction field for backward compatibility
        if (analysisData.playedCubeAction) {
            const normalizedAction = action.toLowerCase().replace(/\s+/g, '');
            const normalizedPlayed = analysisData.playedCubeAction.toLowerCase().replace(/\s+/g, '');
            return normalizedAction.includes(normalizedPlayed) || normalizedPlayed.includes(normalizedAction);
        }
        
        return false;
    }

    async function switchTab(tab) {
        activeTab = tab;
        
        // When switching to cube tab in match mode, load the cube analysis and update position
        if (tab === 'cube' && $statusBarModeStore === 'MATCH' && matchCtx.isMatchMode) {
            await loadCubeAnalysisForCurrentPosition(true); // true = update position to show cube position
        }
    }

    // Load cube analysis for current checker position (find previous cube position)
    // Returns true if cube analysis was found in the same game, false otherwise
    async function loadCubeAnalysisForCurrentPosition(updatePosition = false) {
        if (!matchCtx.isMatchMode) return false;
        
        const currentIndex = matchCtx.currentIndex;
        const movePositions = matchCtx.movePositions;
        const currentMovePos = movePositions[currentIndex];
        
        // If we're on the first position of a game (move_number 0 or 1), no cube decision is possible
        if (currentMovePos && (currentMovePos.move_number === 0 || currentMovePos.move_number === 1)) {
            return false;
        }
        
        const currentGameNumber = movePositions[currentIndex].game_number;
        
        // Find the most recent cube decision before current position IN THE SAME GAME
        for (let i = currentIndex - 1; i >= 0; i--) {
            // Stop if we've gone to a different game
            if (movePositions[i].game_number !== currentGameNumber) {
                break;
            }
            
            if (movePositions[i].move_type === 'cube') {
                // Load analysis for this cube position
                try {
                    const { LoadAnalysis } = await import('../../wailsjs/go/main/Database.js');
                    const cubeAnalysis = await LoadAnalysis(movePositions[i].position.id);
                    if (cubeAnalysis && cubeAnalysis.doublingCubeAnalysis) {
                        // Update only the cube analysis part
                        analysisStore.update(current => ({
                            ...current,
                            doublingCubeAnalysis: cubeAnalysis.doublingCubeAnalysis,
                            playedCubeAction: cubeAnalysis.playedCubeAction || ''
                        }));
                        
                        // Only update position if explicitly requested (when clicking cube tab)
                        if (updatePosition) {
                            const cubePosition = {...movePositions[i].position};
                            cubePosition.dice = [0, 0];
                            positionStore.set(cubePosition);
                        }
                        return true; // Found cube analysis in current game
                    }
                } catch (error) {
                    console.error('Error loading cube analysis:', error);
                }
                return false;
            }
        }
        return false; // No cube analysis found in current game
    }

    // When switching back to checker tab, restore checker position
    async function restoreCheckerPosition() {
        if ($statusBarModeStore === 'MATCH' && matchCtx.isMatchMode) {
            const currentMovePos = matchCtx.movePositions[matchCtx.currentIndex];
            if (currentMovePos && currentMovePos.move_type === 'checker') {
                positionStore.set(currentMovePos.position);
            }
        }
    }

    // Enhanced switch with position restore
    async function handleTabSwitch(tab) {
        if (tab === activeTab) return;
        
        if (tab === 'checker') {
            await restoreCheckerPosition();
        }
        
        await switchTab(tab);
    }

    // Handle click in analysis content to toggle between checker and cube
    function handleContentClick(event) {
        // Only toggle if both analyses available (MATCH mode)
        if (!showTabs) return;
        
        // Don't toggle if clicking on close button
        if (event.target.closest('.close-icon')) {
            return;
        }
        
        // Check if clicking on table header (TH) or on a header row
        const clickedTH = event.target.closest('th');
        const clickedRow = event.target.closest('tr');
        const clickedDataRow = clickedRow && clickedRow.parentElement.tagName === 'TBODY' && !clickedTH;
        
        // If clicking a sortable checker table header, don't toggle tabs
        if (clickedTH && clickedTH.closest('.checker-table')) {
            return;
        }
        
        // Toggle if clicking on header OR anywhere outside data rows
        if (clickedTH || !clickedDataRow) {
            // Toggle between checker and cube
            const newTab = activeTab === 'checker' ? 'cube' : 'checker';
            handleTabSwitch(newTab);
        }
        // If clicking on data row (not header), don't toggle - let the row click handler do its job
    }

    // Determine if both analyses are available
    $: hasCheckerAnalysis = analysisData && analysisData.checkerAnalysis && 
                            analysisData.checkerAnalysis.moves && 
                            analysisData.checkerAnalysis.moves.length > 0;
    // For cube analysis in MATCH mode, it must not be null and have actual data
    // Check both that the object exists and that it has actual analysis content
    $: hasCubeAnalysis = analysisData && 
                         analysisData.doublingCubeAnalysis !== null && 
                         analysisData.doublingCubeAnalysis !== undefined && 
                         typeof analysisData.doublingCubeAnalysis === 'object' &&
                         (analysisData.doublingCubeAnalysis.bestCubeAction || 
                          analysisData.doublingCubeAnalysis.cubefulNoDoubleEquity !== undefined);
    // Check if current position is the first position of a game (no cube decision possible)
    // First position can be move_number 0 or 1
    $: isFirstPositionOfGame = matchCtx.isMatchMode && 
                                matchCtx.movePositions.length > 0 && 
                                (matchCtx.movePositions[matchCtx.currentIndex]?.move_number === 0 ||
                                 matchCtx.movePositions[matchCtx.currentIndex]?.move_number === 1);
    // Only show tabs in MATCH mode where checker and cube are separate positions
    // BUT not on the first position of a game (cube decision not possible)
    $: showTabs = hasCheckerAnalysis && hasCubeAnalysis && $statusBarModeStore === 'MATCH' && matchCtx.isMatchMode && !isFirstPositionOfGame;
</script>

{#if visible}
    <section class="analysis-panel" role="dialog" aria-modal="true" id="analysisPanel" tabindex="-1" on:keydown={handleKeyDown}>
        <button type="button" class="close-icon" on:click={onClose} aria-label="Close" on:keydown={handleKeyDown}>×</button>
        
        <div class="analysis-content" on:click={handleContentClick} on:keydown={() => {}} role="button" tabindex="-1">
            {#if (activeTab === 'cube' || (!showTabs && analysisData.analysisType === 'DoublingCube')) && analysisData.doublingCubeAnalysis}
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
                            <tr class:played={isPlayedCubeAction('No Double')}>
                                <td>{getDecisionLabel('No Double')}</td>
                                <td>{formatEquity(analysisData.doublingCubeAnalysis.cubefulNoDoubleEquity || 0)}</td>
                                <td>{formatEquity(analysisData.doublingCubeAnalysis.cubefulNoDoubleError || 0)}</td>
                            </tr>
                            <tr class:played={isPlayedCubeAction('Double') && isPlayedCubeAction('Take')}>
                                <td>{getDecisionLabel('Double/Take')}</td>
                                <td>{formatEquity(analysisData.doublingCubeAnalysis.cubefulDoubleTakeEquity || 0)}</td>
                                <td>{formatEquity(analysisData.doublingCubeAnalysis.cubefulDoubleTakeError || 0)}</td>
                            </tr>
                            <tr class:played={isPlayedCubeAction('Double') && isPlayedCubeAction('Pass')}>
                                <td>{getDecisionLabel('Double/Pass')}</td>
                                <td>{formatEquity(analysisData.doublingCubeAnalysis.cubefulDoublePassEquity || 0)}</td>
                                <td>{formatEquity(analysisData.doublingCubeAnalysis.cubefulDoublePassError || 0)}</td>
                            </tr>
                            <tr class="best-action-row {analysisData.doublingCubeAnalysis.bestCubeAction.includes('ダブル') ? 'japanese-text' : ''}">
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

            {#if (activeTab === 'checker' || (!showTabs && analysisData.analysisType === 'CheckerMove')) && analysisData.checkerAnalysis && analysisData.checkerAnalysis.moves && analysisData.checkerAnalysis.moves.length > 0}
                <table class="checker-table">
                    <thead>
                        <tr>
                            <th class="sortable" class:active-sort={sortColumn === 'move'} on:click|stopPropagation={() => handleSort('move')}>Move{getSortIndicator('move')}</th>
                            <th class="sortable" class:active-sort={sortColumn === 'equity'} on:click|stopPropagation={() => handleSort('equity')}>Equity</th>
                            <th class="sortable" class:active-sort={sortColumn === 'error'} on:click|stopPropagation={() => handleSort('error')}>Error{getSortIndicator('error')}</th>
                            <th class="sortable" class:active-sort={sortColumn === 'pw'} on:click|stopPropagation={() => handleSort('pw')}>P W{getSortIndicator('pw')}</th>
                            <th class="sortable" class:active-sort={sortColumn === 'pg'} on:click|stopPropagation={() => handleSort('pg')}>P G{getSortIndicator('pg')}</th>
                            <th class="sortable" class:active-sort={sortColumn === 'pb'} on:click|stopPropagation={() => handleSort('pb')}>P B{getSortIndicator('pb')}</th>
                            <th class="sortable" class:active-sort={sortColumn === 'ow'} on:click|stopPropagation={() => handleSort('ow')}>O W{getSortIndicator('ow')}</th>
                            <th class="sortable" class:active-sort={sortColumn === 'og'} on:click|stopPropagation={() => handleSort('og')}>O G{getSortIndicator('og')}</th>
                            <th class="sortable" class:active-sort={sortColumn === 'ob'} on:click|stopPropagation={() => handleSort('ob')}>O B{getSortIndicator('ob')}</th>
                            <th class="sortable" class:active-sort={sortColumn === 'depth'} on:click|stopPropagation={() => handleSort('depth')}>Depth{getSortIndicator('depth')}</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each sortedMoves as move}
                            <tr 
                                class:selected={$selectedMoveStore === move.move}
                                class:played={isPlayedMove(move)}
                                on:click={() => handleMoveRowClick(move)}
                                style="cursor: pointer;"
                            >
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

    .checker-table tr.selected {
        background-color: #b3d9ff !important; /* Highlight selected row with light blue */
        font-weight: bold;
    }

    .checker-table tr.played {
        background-color: #fff3cd !important; /* Light yellow background for played move */
    }

    .checker-table tr.played.selected {
        background-color: #a3c9ef !important; /* Mixed color when both played and selected */
    }

    .right-table tr.played {
        background-color: #fff3cd !important; /* Light yellow background for played cube action */
    }

    .checker-table tbody tr:hover {
        background-color: #e6f2ff; /* Light hover effect for move rows */
    }

    .sortable {
        cursor: pointer;
        user-select: none;
        position: relative;
    }

    .sortable:hover {
        background-color: #e0e0e0;
    }

    .active-sort {
        background-color: #dde8f0;
    }

    .best-action-row {
        font-weight: bold;
        color: #000000; /* Subtle color change for emphasis */
    }

    .japanese-text {
        font-family: 'Noto Sans JP', sans-serif;
    }

    /* Make analysis content interactive in MATCH mode (toggle on click) */
    .analysis-content {
        cursor: default;
    }
</style>

