<script>
    import { onMount, onDestroy } from 'svelte';
    import { 
        GetAllMatches, 
        DeleteMatch, 
        UpdateMatch,
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

    // Sorting state
    let sortColumn = null;     // null | 'player1' | 'player2' | 'date' | 'length' | 'tournament'
    let sortDirection = 'asc'; // 'asc' | 'desc'

    // Inline tournament editing
    let editingTournamentMatchId = null;
    let editTournamentValue = '';
    let showTournamentDropdown = false;
    let filteredTournaments = [];

    // Inline match editing (player names, date)
    let editingMatchId = null;
    let editPlayer1Value = '';
    let editPlayer2Value = '';
    let editDateValue = '';

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
            matches = loadedMatches || [];
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
        editTournamentValue = match.tournament_name || match.event || '';
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

    function toDateInputValue(dateStr) {
        if (!dateStr) return '';
        try {
            const date = new Date(dateStr);
            if (isNaN(date.getTime())) return '';
            return date.toISOString().split('T')[0];
        } catch {
            return '';
        }
    }

    function startEditMatch(match, ev) {
        ev.stopPropagation();
        editingMatchId = match.id;
        editPlayer1Value = match.player1_name || '';
        editPlayer2Value = match.player2_name || '';
        editDateValue = toDateInputValue(match.match_date);
    }

    async function saveMatchEdit() {
        if (editingMatchId === null) return;
        try {
            await UpdateMatch(editingMatchId, editPlayer1Value, editPlayer2Value, editDateValue);
            await loadMatches();
            statusBarTextStore.set('Match updated');
        } catch (error) {
            console.error('Error updating match:', error);
            statusBarTextStore.set('Error updating match');
        }
        editingMatchId = null;
    }

    function cancelMatchEdit() {
        editingMatchId = null;
        editPlayer1Value = '';
        editPlayer2Value = '';
        editDateValue = '';
    }

    function handleMatchEditKeyDown(event) {
        event.stopPropagation();
        if (event.key === 'Enter') {
            event.preventDefault();
            saveMatchEdit();
        } else if (event.key === 'Escape') {
            event.preventDefault();
            cancelMatchEdit();
        }
    }

    // --- Sorting helpers ---
    function handleSort(column) {
        if (sortColumn === column) {
            if (sortDirection === 'asc') {
                sortDirection = 'desc';
            } else {
                // Reset sorting
                sortColumn = null;
                sortDirection = 'asc';
            }
        } else {
            sortColumn = column;
            sortDirection = 'asc';
        }
    }

    function compareValues(a, b) {
        if (a == null && b == null) return 0;
        if (a == null) return 1;
        if (b == null) return -1;
        if (typeof a === 'string') return a.localeCompare(b, undefined, { sensitivity: 'base' });
        return a - b;
    }

    function getSortValue(match, column) {
        switch (column) {
            case 'player1': return match.player1_name || '';
            case 'player2': return match.player2_name || '';
            case 'date': return match.match_date || '';
            case 'length': return match.match_length || 0;
            case 'tournament': return match.tournament_name || match.event || '';
            default: return '';
        }
    }

    $: sortedMatches = (() => {
        if (!sortColumn) return matches;
        const sorted = [...matches].sort((a, b) => {
            const valA = getSortValue(a, sortColumn);
            const valB = getSortValue(b, sortColumn);
            const cmp = compareValues(valA, valB);
            return sortDirection === 'asc' ? cmp : -cmp;
        });
        return sorted;
    })();

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
        if (isNaN(date.getTime())) return '-';
        const y = date.getFullYear();
        const m = String(date.getMonth() + 1).padStart(2, '0');
        const d = String(date.getDate()).padStart(2, '0');
        return `${y}/${m}/${d}`;
    }

    function closePanel() {
        showMatchPanelStore.set(false);
    }

    function handleKeyDown(event) {
        if (!visible) return;

        // Let Ctrl+key combos pass through to global handler (e.g. Ctrl+T to toggle panel)
        if (event.ctrlKey) return;

        // Let Space pass through so the command line can be opened from global handler
        if (event.code === 'Space') return;

        // Stop all keyboard events from propagating to global handlers
        event.stopPropagation();

        if (event.key === 'Escape') {
            if (editingMatchId !== null) {
                cancelMatchEdit();
                event.preventDefault();
            } else if (editingTournamentMatchId !== null) {
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

        if (selectedMatch && sortedMatches.length > 0) {
            const currentIndex = sortedMatches.findIndex(m => m.id === selectedMatch.id);

            if (event.key === 'j' || event.key === 'ArrowDown') {
                event.preventDefault();
                if (currentIndex >= 0 && currentIndex < sortedMatches.length - 1) {
                    selectMatch(sortedMatches[currentIndex + 1]);
                    setTimeout(() => {
                        const selectedRow = document.querySelector('.match-panel tr.selected');
                        if (selectedRow) selectedRow.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
                    }, 0);
                }
            } else if (event.key === 'k' || event.key === 'ArrowUp') {
                event.preventDefault();
                if (currentIndex > 0) {
                    selectMatch(sortedMatches[currentIndex - 1]);
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
        // Cancel match edit if clicking outside the editing row
        if (editingMatchId !== null && !event.target.closest('.match-editing-row')) {
            cancelMatchEdit();
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
                <div class="match-table-container">
                    <table class="match-table">
                        <thead>
                            <tr>
                                <th class="no-select narrow-col">#</th>
                                <th class="no-select sortable narrow-col" on:click={() => handleSort('date')}>Date {#if sortColumn === 'date'}<span class="sort-arrow">{sortDirection === 'asc' ? '▲' : '▼'}</span>{/if}</th>
                                <th class="no-select sortable" on:click={() => handleSort('tournament')}>Tournament {#if sortColumn === 'tournament'}<span class="sort-arrow">{sortDirection === 'asc' ? '▲' : '▼'}</span>{/if}</th>
                                <th class="no-select sortable" on:click={() => handleSort('player1')}>Player 1 {#if sortColumn === 'player1'}<span class="sort-arrow">{sortDirection === 'asc' ? '▲' : '▼'}</span>{/if}</th>
                                <th class="no-select sortable" on:click={() => handleSort('player2')}>Player 2 {#if sortColumn === 'player2'}<span class="sort-arrow">{sortDirection === 'asc' ? '▲' : '▼'}</span>{/if}</th>
                                <th class="no-select sortable narrow-col" on:click={() => handleSort('length')}>Length {#if sortColumn === 'length'}<span class="sort-arrow">{sortDirection === 'asc' ? '▲' : '▼'}</span>{/if}</th>
                                <th class="no-select actions-col"></th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each sortedMatches as match, index}
                                {#if editingMatchId === match.id}
                                    <tr class="match-editing-row" class:selected={selectedMatch === match}>
                                        <td class="index-cell narrow-col no-select">{index + 1}</td>
                                        <td class="narrow-col">
                                            <input
                                                type="date"
                                                class="match-edit-input"
                                                bind:value={editDateValue}
                                                on:keydown={handleMatchEditKeyDown}
                                            />
                                        </td>
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
                                                <span class="tournament-display" title="Click to edit">{match.tournament_name || match.event || '—'}</span>
                                            {/if}
                                        </td>
                                        <td>
                                            <input
                                                type="text"
                                                class="match-edit-input"
                                                bind:value={editPlayer1Value}
                                                on:keydown={handleMatchEditKeyDown}
                                                placeholder="Player 1"
                                            />
                                        </td>
                                        <td>
                                            <input
                                                type="text"
                                                class="match-edit-input"
                                                bind:value={editPlayer2Value}
                                                on:keydown={handleMatchEditKeyDown}
                                                placeholder="Player 2"
                                            />
                                        </td>
                                        <td class="narrow-col no-select">{match.match_length}</td>
                                        <td class="actions-col no-select">
                                            <span class="item-actions editing-actions">
                                                <button class="icon-btn" on:click|stopPropagation={saveMatchEdit} title="Save">✓</button>
                                                <button class="icon-btn" on:click|stopPropagation={cancelMatchEdit} title="Cancel">✕</button>
                                            </span>
                                        </td>
                                    </tr>
                                {:else}
                                    <tr 
                                        class:selected={selectedMatch === match}
                                        on:click={() => selectMatch(match)}
                                        on:dblclick={() => handleDoubleClick(match)}
                                    >
                                        <td class="index-cell narrow-col no-select">{index + 1}</td>
                                        <td class="narrow-col no-select">{formatDate(match.match_date)}</td>
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
                                                <span class="tournament-display" title="Click to edit">{match.tournament_name || match.event || '—'}</span>
                                            {/if}
                                        </td>
                                        <td class="no-select">{match.player1_name}</td>
                                        <td class="no-select">{match.player2_name}</td>
                                        <td class="narrow-col no-select">{match.match_length}</td>
                                        <td class="actions-col no-select">
                                            <span class="item-actions">
                                                <button class="icon-btn" on:click|stopPropagation={(e) => startEditMatch(match, e)} title="Edit">✎</button>
                                                <button class="icon-btn delete" on:click|stopPropagation={(e) => deleteMatchEntry(match, e)} title="Delete">×</button>
                                            </span>
                                        </td>
                                    </tr>
                                {/if}
                            {/each}
                        </tbody>
                    </table>
                </div>
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

    .match-table th.sortable {
        cursor: pointer;
    }

    .match-table th.sortable:hover {
        background-color: #e8e8e8;
    }

    .sort-arrow {
        font-size: 9px;
        margin-left: 3px;
        color: #1976d2;
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
        text-align: center;
        color: #999;
    }

    .narrow-col {
        width: 1px;  /* shrink to content */
        white-space: nowrap;
        padding-left: 6px;
        padding-right: 6px;
    }

    .actions-col {
        width: 36px;
        min-width: 36px;
        max-width: 36px;
        white-space: nowrap;
        text-align: center;
        padding: 0 4px;
    }

    /* Hover-based row actions (like TournamentPanel) */
    .item-actions {
        display: inline-flex;
        gap: 2px;
        visibility: hidden;
        vertical-align: middle;
    }

    .editing-actions {
        visibility: visible;
    }

    .match-table tbody tr:hover .item-actions {
        visibility: visible;
    }

    .icon-btn {
        background: none;
        border: none;
        cursor: pointer;
        font-size: 12px;
        color: #666;
        padding: 0 3px;
        line-height: 1;
    }

    .icon-btn:hover {
        color: #000;
    }

    .icon-btn.delete:hover {
        color: #c55;
    }

    .match-edit-input {
        width: 100%;
        padding: 1px 4px;
        border: 1px solid #1976d2;
        border-radius: 2px;
        font-size: 11px;
        box-sizing: border-box;
        outline: none;
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
        max-width: 120px;
        white-space: nowrap;
    }

    .tournament-display {
        color: #666;
        font-size: 11px;
        display: block;
        overflow: hidden;
        text-overflow: ellipsis;
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
        min-width: 160px;
        background: white;
        border: 1px solid #ccc;
        border-top: none;
        border-radius: 0 0 3px 3px;
        max-height: 120px;
        overflow-y: auto;
        z-index: 30;
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
