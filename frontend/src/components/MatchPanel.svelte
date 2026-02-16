<script>
    import { onMount, onDestroy } from 'svelte';
    import { 
        GetAllMatches, 
        DeleteMatch, 
        GetMatchMovePositions, 
        LoadAnalysis,
        GetAllTournaments,
        SetMatchTournamentByName
    } from '../../wailsjs/go/main/Database.js';
    import { positionStore, matchContextStore, lastVisitedMatchStore } from '../stores/positionStore';
    import { statusBarModeStore, showMatchPanelStore, matchPanelRefreshTriggerStore, positionReloadTriggerStore, statusBarTextStore } from '../stores/uiStore';
    import { analysisStore, selectedMoveStore } from '../stores/analysisStore';
    import { commentTextStore } from '../stores/uiStore';
    import { tournamentsStore } from '../stores/tournamentStore';

    let matches = [];
    let selectedMatch = null;
    let visible = false;
    let lastVisitedMatch = null;
    let tournaments = [];

    // Inline tournament editing
    let editingTournamentMatchId = null;
    let editTournamentValue = '';
    let showTournamentDropdown = false;
    let filteredTournaments = [];

    lastVisitedMatchStore.subscribe(value => {
        lastVisitedMatch = value;
    });

    tournamentsStore.subscribe(value => {
        tournaments = value || [];
    });

    // Subscribe to refresh trigger to reload matches when a new match is imported
    matchPanelRefreshTriggerStore.subscribe(async () => {
        if (visible) {
            await loadMatches();
            // If we have a last visited match, try to re-select it
            if (lastVisitedMatch && lastVisitedMatch.matchID) {
                selectedMatch = matches.find(m => m.id === lastVisitedMatch.matchID);
            }
        }
    });

    showMatchPanelStore.subscribe(async value => {
        const wasVisible = visible;
        visible = value;
        if (visible && !wasVisible) {
            await loadMatches();
            if (lastVisitedMatch && lastVisitedMatch.matchID) {
                selectedMatch = matches.find(m => m.id === lastVisitedMatch.matchID);
            } else {
                selectedMatch = null;
            }
        } else if (!visible && wasVisible) {
            selectedMatch = null;
            editingTournamentMatchId = null;
        }
    });

    async function loadMatches() {
        try {
            const loadedMatches = await GetAllMatches();
            matches = (loadedMatches || []).reverse();
            await loadTournaments();
        } catch (error) {
            console.error('Error loading matches:', error);
            matches = [];
        }
    }

    async function loadTournaments() {
        try {
            const loaded = await GetAllTournaments();
            tournamentsStore.set(loaded || []);
        } catch (error) {
            console.error('Error loading tournaments:', error);
        }
    }

    function startEditTournament(match, event) {
        event.stopPropagation();
        editingTournamentMatchId = match.id;
        editTournamentValue = match.tournament_name || '';
        showTournamentDropdown = true;
        filteredTournaments = tournaments;
        setTimeout(() => {
            const input = document.querySelector('.tournament-edit-input');
            if (input) input.focus();
        }, 50);
    }

    function filterTournaments() {
        const val = editTournamentValue.toLowerCase();
        if (!val) {
            filteredTournaments = tournaments;
        } else {
            filteredTournaments = tournaments.filter(t => t.name.toLowerCase().includes(val));
        }
        showTournamentDropdown = true;
    }

    async function selectTournamentOption(name) {
        editTournamentValue = name;
        showTournamentDropdown = false;
        await saveTournamentEdit();
    }

    async function saveTournamentEdit() {
        if (editingTournamentMatchId === null) return;
        try {
            await SetMatchTournamentByName(editingTournamentMatchId, editTournamentValue.trim());
            await loadMatches();
            await loadTournaments();
            statusBarTextStore.set(editTournamentValue.trim() ? `Tournament set to "${editTournamentValue.trim()}"` : 'Tournament cleared');
        } catch (error) {
            console.error('Error setting tournament:', error);
            statusBarTextStore.set('Error setting tournament');
        }
        editingTournamentMatchId = null;
        editTournamentValue = '';
        showTournamentDropdown = false;
    }

    function cancelTournamentEdit() {
        editingTournamentMatchId = null;
        editTournamentValue = '';
        showTournamentDropdown = false;
    }

    function handleTournamentKeyDown(event) {
        event.stopPropagation();
        if (event.key === 'Enter') {
            event.preventDefault();
            saveTournamentEdit();
        } else if (event.key === 'Escape') {
            event.preventDefault();
            cancelTournamentEdit();
        }
    }

    function selectMatch(match) {
        if (selectedMatch === match) {
            selectedMatch = null;
        } else {
            selectedMatch = match;
        }
    }

    async function loadMatchPositions(match) {
        try {
            const movePositions = await GetMatchMovePositions(match.id);
            
            if (!movePositions || movePositions.length === 0) {
                statusBarTextStore.set('No moves found in this match');
                return;
            }

            let startIndex = 0;
            if (lastVisitedMatch && lastVisitedMatch.matchID === match.id) {
                if (lastVisitedMatch.currentIndex >= 0 && lastVisitedMatch.currentIndex < movePositions.length) {
                    startIndex = lastVisitedMatch.currentIndex;
                }
            }

            matchContextStore.set({
                isMatchMode: true,
                matchID: match.id,
                movePositions: movePositions,
                currentIndex: startIndex,
                player1Name: match.player1_name,
                player2Name: match.player2_name,
            });

            const startMovePos = movePositions[startIndex];
            positionStore.set(startMovePos.position);
            
            let analysis = null;
            try {
                analysis = await LoadAnalysis(startMovePos.position.id);
            } catch (error) {}
            
            const currentPlayedMove = startMovePos.checker_move || '';
            const currentPlayedCubeAction = startMovePos.cube_action || '';
            
            analysisStore.set({
                positionId: analysis?.positionId || null,
                xgid: analysis?.xgid || '',
                player1: analysis?.player1 || '',
                player2: analysis?.player2 || '',
                analysisType: analysis?.analysisType || '',
                analysisEngineVersion: analysis?.analysisEngineVersion || '',
                checkerAnalysis: analysis?.checkerAnalysis || { moves: [] },
                doublingCubeAnalysis: analysis?.doublingCubeAnalysis || {
                    analysisDepth: '', playerWinChances: 0, playerGammonChances: 0,
                    playerBackgammonChances: 0, opponentWinChances: 0, opponentGammonChances: 0,
                    opponentBackgammonChances: 0, cubelessNoDoubleEquity: 0, cubelessDoubleEquity: 0,
                    cubefulNoDoubleEquity: 0, cubefulNoDoubleError: 0, cubefulDoubleTakeEquity: 0,
                    cubefulDoubleTakeError: 0, cubefulDoublePassEquity: 0, cubefulDoublePassError: 0,
                    bestCubeAction: '', wrongPassPercentage: 0, wrongTakePercentage: 0
                },
                playedMove: currentPlayedMove,
                playedCubeAction: currentPlayedCubeAction,
                playedMoves: analysis?.playedMoves || [],
                playedCubeActions: analysis?.playedCubeActions || [],
                creationDate: analysis?.creationDate || '',
                lastModifiedDate: analysis?.lastModifiedDate || ''
            });
            
            commentTextStore.set('');
            selectedMoveStore.set(null);
            statusBarModeStore.set('MATCH');
            statusBarTextStore.set(`${match.player1_name} vs ${match.player2_name}`);
            
            lastVisitedMatchStore.set({
                matchID: match.id,
                currentIndex: startIndex,
                gameNumber: startMovePos.game_number
            });
            
        } catch (error) {
            console.error('Error loading match positions:', error);
            statusBarTextStore.set('Error loading match positions');
        }
    }

    function handleDoubleClick(match) {
        loadMatchPositions(match);
        closePanel();
    }

    async function deleteMatchEntry(match, event) {
        event.stopPropagation();
        if (!confirm(`Delete match between ${match.player1_name} and ${match.player2_name}?`)) return;
        try {
            await DeleteMatch(match.id);
            await loadMatches();
            if (selectedMatch && selectedMatch.id === match.id) {
                selectedMatch = null;
            }
            // Trigger match panel refresh to update all dependent components
            matchPanelRefreshTriggerStore.update(n => n + 1);
            // Trigger position reload to reflect deleted positions
            positionReloadTriggerStore.update(n => n + 1);
            statusBarTextStore.set('Match deleted');
        } catch (error) {
            console.error('Error deleting match:', error);
            statusBarTextStore.set('Error deleting match');
        }
    }

    function formatDate(dateStr) {
        if (!dateStr) return '-';
        const date = new Date(dateStr);
        return date.toLocaleDateString();
    }

    function closePanel() {
        showMatchPanelStore.set(false);
    }

    function handleKeyDown(event) {
        if (!visible) return;

        // Let Ctrl+key combos pass through to global handler (e.g. Ctrl+T to toggle panel)
        if (event.ctrlKey) return;

        // Stop all keyboard events from propagating to global handlers
        event.stopPropagation();

        if (event.key === 'Escape') {
            if (editingTournamentMatchId !== null) {
                cancelTournamentEdit();
                event.preventDefault();
            } else if (selectedMatch) {
                selectedMatch = null;
                event.preventDefault();
            } else {
                closePanel();
            }
            return;
        }

        if (selectedMatch && matches.length > 0) {
            const currentIndex = matches.findIndex(m => m.id === selectedMatch.id);

            if (event.key === 'j' || event.key === 'ArrowDown') {
                event.preventDefault();
                if (currentIndex >= 0 && currentIndex < matches.length - 1) {
                    selectMatch(matches[currentIndex + 1]);
                    setTimeout(() => {
                        const selectedRow = document.querySelector('.match-panel tr.selected');
                        if (selectedRow) selectedRow.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
                    }, 0);
                }
            } else if (event.key === 'k' || event.key === 'ArrowUp') {
                event.preventDefault();
                if (currentIndex > 0) {
                    selectMatch(matches[currentIndex - 1]);
                    setTimeout(() => {
                        const selectedRow = document.querySelector('.match-panel tr.selected');
                        if (selectedRow) selectedRow.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
                    }, 0);
                }
            } else if (event.key === 'Enter') {
                event.preventDefault();
                handleDoubleClick(selectedMatch);
            } else if (event.key === 'Delete') {
                event.preventDefault();
                const syntheticEvent = { stopPropagation: () => {} };
                deleteMatchEntry(selectedMatch, syntheticEvent);
            }
        }
    }

    function handleClickOutside(event) {
        // Close tournament dropdown if clicking outside
        if (editingTournamentMatchId !== null && !event.target.closest('.tournament-cell-edit')) {
            cancelTournamentEdit();
        }
        const panel = document.getElementById('matchPanel');
        if (panel && !panel.contains(event.target)) {
            document.activeElement.blur();
        }
    }

    $: {
        if (visible) {
            setTimeout(() => {
                const panel = document.getElementById('matchPanel');
                if (panel) panel.focus();
                if (selectedMatch) {
                    const selectedRow = document.querySelector('.match-panel tr.selected');
                    if (selectedRow) selectedRow.scrollIntoView({ behavior: 'smooth', block: 'center' });
                }
            }, 100);
        }
    }

    onMount(async () => {
        if (visible) await loadMatches();
        document.addEventListener('click', handleClickOutside);
        document.addEventListener('keydown', handleKeyDown);
    });

    onDestroy(() => {
        document.removeEventListener('click', handleClickOutside);
        document.removeEventListener('keydown', handleKeyDown);
    });
</script>

{#if visible}
    <section class="match-panel" role="dialog" aria-modal="true" id="matchPanel" tabindex="-1">
        <button type="button" class="close-icon" on:click={closePanel} aria-label="Close">×</button>
        <div class="match-panel-content">
            {#if matches.length === 0}
                <p class="empty-message">No matches imported yet. Import .xg files using Ctrl+I.</p>
            {:else}
                <div class="match-table-container">
                    <table class="match-table">
                        <thead>
                            <tr>
                                <th class="no-select">#</th>
                                <th class="no-select">Player 1</th>
                                <th class="no-select">Player 2</th>
                                <th class="no-select">Event</th>
                                <th class="no-select">Date</th>
                                <th class="no-select">Length</th>
                                <th class="no-select">Tournament</th>
                                <th class="no-select">Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each matches as match, index}
                                <tr 
                                    class:selected={selectedMatch === match}
                                    on:click={() => selectMatch(match)}
                                    on:dblclick={() => handleDoubleClick(match)}
                                >
                                    <td class="index-cell no-select">{index + 1}</td>
                                    <td class="no-select">{match.player1_name}</td>
                                    <td class="no-select">{match.player2_name}</td>
                                    <td class="no-select">{match.event || '-'}</td>
                                    <td class="no-select">{formatDate(match.match_date)}</td>
                                    <td class="no-select">{match.match_length}</td>
                                    <td class="tournament-cell no-select" on:click|stopPropagation={(e) => startEditTournament(match, e)}>
                                        {#if editingTournamentMatchId === match.id}
                                            <div class="tournament-cell-edit">
                                                <input 
                                                    type="text" 
                                                    class="tournament-edit-input"
                                                    bind:value={editTournamentValue}
                                                    on:input={filterTournaments}
                                                    on:keydown={handleTournamentKeyDown}
                                                    on:blur={() => setTimeout(cancelTournamentEdit, 200)}
                                                    placeholder="Tournament name"
                                                />
                                                {#if showTournamentDropdown && filteredTournaments.length > 0}
                                                    <div class="tournament-dropdown">
                                                        {#each filteredTournaments as t}
                                                            <div class="tournament-dropdown-item" on:mousedown|preventDefault={() => selectTournamentOption(t.name)}>
                                                                {t.name}
                                                            </div>
                                                        {/each}
                                                    </div>
                                                {/if}
                                            </div>
                                        {:else}
                                            <span class="tournament-display" title="Click to edit">{match.tournament_name || '—'}</span>
                                        {/if}
                                    </td>
                                    <td class="actions-cell">
                                        <button 
                                            class="action-btn delete-btn" 
                                            on:click|stopPropagation={(e) => deleteMatchEntry(match, e)}
                                            title="Delete match"
                                        >
                                            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
                                                <path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
                                            </svg>
                                        </button>
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {/if}
        </div>
    </section>
{/if}


<style>
    .match-panel {
        position: fixed;
        width: 100%;
        bottom: 22px;
        left: 0;
        right: 0;
        height: 178px;
        background-color: white;
        border-top: 1px solid rgba(0, 0, 0, 0.1);
        z-index: 5;
        outline: none;
        user-select: none;
        -webkit-user-select: none;
    }

    .close-icon {
        position: absolute;
        top: 6px;
        right: 12px;
        background: none;
        border: none;
        font-size: 24px;
        cursor: pointer;
        color: #666;
        line-height: 1;
        padding: 0;
        width: 24px;
        height: 24px;
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 10;
    }

    .close-icon:hover {
        color: #000;
    }

    .match-panel-content {
        height: 100%;
        overflow-y: auto;
        display: flex;
        flex-direction: row;
    }

    .empty-message {
        text-align: center;
        color: #666;
        padding: 20px;
        margin: 60px 0 0 0;
    }

    .match-table-container {
        flex: 1;
        height: 100%;
        overflow-y: auto;
    }

    .match-table {
        width: 100%;
        border-collapse: collapse;
        font-size: 12px;
    }

    .match-table thead {
        position: sticky;
        top: 0;
        background-color: #f5f5f5;
        z-index: 1;
    }

    .match-table th,
    .match-table td {
        padding: 8px 12px;
        text-align: left;
        border-bottom: 1px solid #e0e0e0;
    }

    .match-table th {
        font-weight: 600;
        color: #333;
    }

    .match-table tbody tr {
        cursor: pointer;
        transition: background-color 0.1s;
    }

    .match-table tbody tr:hover {
        background-color: #f9f9f9;
    }

    .match-table tbody tr.selected {
        background-color: #e3f2fd;
    }

    .match-table tbody tr.selected:hover {
        background-color: #bbdefb;
    }

    .index-cell {
        width: 50px;
        text-align: center;
        color: #999;
    }

    .actions-cell {
        width: 40px;
        text-align: center;
    }

    .action-btn {
        background: none;
        border: none;
        cursor: pointer;
        padding: 4px;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        color: #666;
        transition: color 0.2s;
    }

    .action-btn:hover {
        color: #000;
    }

    .delete-btn:hover {
        color: #d32f2f;
    }

    .action-btn svg {
        width: 18px;
        height: 18px;
    }

    .no-select {
        user-select: none;
        -webkit-user-select: none;
        -moz-user-select: none;
        -ms-user-select: none;
    }

    .tournament-cell {
        position: relative;
        cursor: pointer;
        min-width: 100px;
    }

    .tournament-display {
        color: #666;
        font-size: 11px;
    }

    .tournament-display:hover {
        color: #1976d2;
    }

    .tournament-cell-edit {
        position: relative;
    }

    .tournament-edit-input {
        width: 100%;
        padding: 2px 4px;
        border: 1px solid #1976d2;
        border-radius: 2px;
        font-size: 11px;
        box-sizing: border-box;
        outline: none;
    }

    .tournament-dropdown {
        position: absolute;
        top: 100%;
        left: 0;
        right: 0;
        background: white;
        border: 1px solid #ccc;
        border-top: none;
        border-radius: 0 0 3px 3px;
        max-height: 80px;
        overflow-y: auto;
        z-index: 20;
        box-shadow: 0 2px 4px rgba(0,0,0,0.15);
    }

    .tournament-dropdown-item {
        padding: 3px 6px;
        font-size: 11px;
        cursor: pointer;
    }

    .tournament-dropdown-item:hover {
        background: #e3f2fd;
    }
</style>
