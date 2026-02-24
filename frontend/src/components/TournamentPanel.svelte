<script>
    import { onMount, onDestroy } from 'svelte';
    import { 
        GetAllTournaments, 
        CreateTournament, 
        DeleteTournament, 
        UpdateTournament,
        GetTournamentMatches,
        RemoveMatchFromTournament,
        GetAllMatches,
        AddMatchToTournament,
        GetMatchMovePositions,
        LoadAnalysis,
        SwapMatchPlayers
    } from '../../wailsjs/go/main/Database.js';
    import { showTournamentPanelStore, statusBarTextStore, statusBarModeStore } from '../stores/uiStore';
    import { tournamentsStore, selectedTournamentStore, tournamentMatchesStore } from '../stores/tournamentStore';
    import { databaseLoadedStore } from '../stores/databaseStore';
    import { positionStore, matchContextStore, lastVisitedMatchStore } from '../stores/positionStore';
    import { analysisStore, selectedMoveStore } from '../stores/analysisStore';
    import { commentTextStore } from '../stores/uiStore';

    let tournaments = [];
    let selectedTournament = null;
    let tournamentMatches = [];
    let visible = false;
    let databaseLoaded = false;
    let sortBy = 'date';
    let sortOrder = 'desc';
    
    // New tournament form
    let newTournamentName = '';
    let newTournamentDate = '';
    let newTournamentLocation = '';
    
    // Edit tournament
    let editingTournament = null;
    let editName = '';
    let editDate = '';
    let editLocation = '';

    // Add match to tournament
    let addMatchSearch = '';
    let allMatches = [];
    let filteredMatches = [];
    let addMatchFocused = false;

    const unsubTournaments = tournamentsStore.subscribe(value => {
        tournaments = value || [];
    });

    const unsubSelected = selectedTournamentStore.subscribe(value => {
        selectedTournament = value;
    });

    const unsubMatches = tournamentMatchesStore.subscribe(value => {
        tournamentMatches = value || [];
    });

    const unsubDb = databaseLoadedStore.subscribe(value => {
        databaseLoaded = value;
    });

    const unsubVisible = showTournamentPanelStore.subscribe(async value => {
        const wasVisible = visible;
        visible = value;
        if (visible && !wasVisible) {
            await loadTournaments();
            selectedTournamentStore.set(null);
            tournamentMatchesStore.set([]);
        } else if (!visible && wasVisible) {
            selectedTournamentStore.set(null);
            tournamentMatchesStore.set([]);
            editingTournament = null;
            addMatchSearch = '';
            addMatchFocused = false;
        }
    });

    onDestroy(() => {
        unsubTournaments();
        unsubSelected();
        unsubMatches();
        unsubDb();
        unsubVisible();
    });

    async function loadTournaments() {
        try {
            const loaded = await GetAllTournaments();
            tournamentsStore.set(sortTournaments(loaded || []));
        } catch (error) {
            console.error('Error loading tournaments:', error);
        }
    }

    async function loadAllMatches() {
        try {
            allMatches = (await GetAllMatches()) || [];
        } catch (error) {
            console.error('Error loading matches:', error);
            allMatches = [];
        }
    }

    function sortTournaments(list) {
        return [...list].sort((a, b) => {
            if (sortBy === 'date') {
                const dateA = a.date || '';
                const dateB = b.date || '';
                return sortOrder === 'desc' ? dateB.localeCompare(dateA) : dateA.localeCompare(dateB);
            } else if (sortBy === 'location') {
                const locA = (a.location || '').toLowerCase();
                const locB = (b.location || '').toLowerCase();
                return sortOrder === 'desc' ? locB.localeCompare(locA) : locA.localeCompare(locB);
            }
            return 0;
        });
    }

    function toggleSort(field) {
        if (sortBy === field) {
            sortOrder = sortOrder === 'desc' ? 'asc' : 'desc';
        } else {
            sortBy = field;
            sortOrder = 'desc';
        }
        tournamentsStore.set(sortTournaments(tournaments));
    }

    async function selectTournament(tournament) {
        if (selectedTournament && selectedTournament.id === tournament.id) {
            selectedTournamentStore.set(null);
            tournamentMatchesStore.set([]);
            addMatchSearch = '';
            return;
        }
        selectedTournamentStore.set(tournament);
        addMatchSearch = '';
        await loadAllMatches();
        try {
            const matches = await GetTournamentMatches(tournament.id);
            tournamentMatchesStore.set(matches || []);
        } catch (error) {
            console.error('Error loading tournament matches:', error);
        }
    }

    async function createTournament() {
        if (!newTournamentName.trim()) return;
        try {
            await CreateTournament(newTournamentName.trim(), newTournamentDate, newTournamentLocation.trim());
            await loadTournaments();
            statusBarTextStore.set(`Tournament "${newTournamentName.trim()}" created`);
            newTournamentName = '';
            newTournamentDate = '';
            newTournamentLocation = '';
        } catch (error) {
            console.error('Error creating tournament:', error);
            statusBarTextStore.set('Error creating tournament');
        }
    }

    async function deleteTournamentEntry(tournament, event) {
        event.stopPropagation();
        if (!confirm(`Delete tournament "${tournament.name}"?`)) return;
        try {
            await DeleteTournament(tournament.id);
            await loadTournaments();
            if (selectedTournament && selectedTournament.id === tournament.id) {
                selectedTournamentStore.set(null);
                tournamentMatchesStore.set([]);
            }
            statusBarTextStore.set('Tournament deleted');
        } catch (error) {
            console.error('Error deleting tournament:', error);
        }
    }

    function startEdit(tournament, event) {
        event.stopPropagation();
        editingTournament = tournament;
        editName = tournament.name;
        editDate = tournament.date || '';
        editLocation = tournament.location || '';
    }

    async function saveEdit() {
        if (!editName.trim()) return;
        try {
            await UpdateTournament(editingTournament.id, editName.trim(), editDate, editLocation.trim());
            await loadTournaments();
            if (selectedTournament && selectedTournament.id === editingTournament.id) {
                selectedTournamentStore.set({ ...selectedTournament, name: editName.trim(), date: editDate, location: editLocation.trim() });
            }
            editingTournament = null;
        } catch (error) {
            console.error('Error updating tournament:', error);
        }
    }

    function cancelEdit() {
        editingTournament = null;
    }

    function updateFilteredMatches() {
        const matchIds = tournamentMatches.map(m => m.id);
        let available = allMatches.filter(m => !matchIds.includes(m.id));
        if (addMatchSearch.trim()) {
            const q = addMatchSearch.toLowerCase();
            available = available.filter(m =>
                (m.player1_name || '').toLowerCase().includes(q) ||
                (m.player2_name || '').toLowerCase().includes(q) ||
                String(m.match_length || '').includes(q)
            );
        }
        filteredMatches = available;
    }

    $: {
        // Re-filter when tournamentMatches or search changes
        const _m = tournamentMatches;
        const _s = addMatchSearch;
        updateFilteredMatches();
    }

    async function addMatchToTournament(matchId) {
        if (!selectedTournament) return;
        try {
            await AddMatchToTournament(selectedTournament.id, matchId);
            const matches = await GetTournamentMatches(selectedTournament.id);
            tournamentMatchesStore.set(matches || []);
            await loadTournaments();
            await loadAllMatches();
            updateFilteredMatches();
        } catch (error) {
            console.error('Error adding match:', error);
        }
    }

    async function removeMatch(matchId) {
        if (!selectedTournament) return;
        try {
            await RemoveMatchFromTournament(matchId);
            const matches = await GetTournamentMatches(selectedTournament.id);
            tournamentMatchesStore.set(matches || []);
            await loadTournaments();
        } catch (error) {
            console.error('Error removing match:', error);
        }
    }

    async function swapMatchPlayersInTournament(match) {
        try {
            await SwapMatchPlayers(match.id);
            // Reload tournament matches
            if (selectedTournament) {
                const matches = await GetTournamentMatches(selectedTournament.id);
                tournamentMatchesStore.set(matches || []);
            }

            // If we are currently viewing this match in match mode, update context
            let currentContext = null;
            const unsub = matchContextStore.subscribe(v => currentContext = v);
            unsub();
            if (currentContext && currentContext.isMatchMode && currentContext.matchID === match.id) {
                const movePositions = await GetMatchMovePositions(match.id);
                if (movePositions && movePositions.length > 0) {
                    const currentIndex = Math.min(currentContext.currentIndex, movePositions.length - 1);
                    matchContextStore.set({
                        isMatchMode: true,
                        matchID: match.id,
                        movePositions: movePositions,
                        currentIndex: currentIndex,
                        player1Name: movePositions[0].player1_name,
                        player2Name: movePositions[0].player2_name,
                    });
                    positionStore.set(movePositions[currentIndex].position);
                }
            }

            statusBarTextStore.set('Swapped players for match');
        } catch (error) {
            console.error('Error swapping match players:', error);
            statusBarTextStore.set('Error swapping match players');
        }
    }

    async function openMatch(match) {
        try {
            const movePositions = await GetMatchMovePositions(match.id);
            if (!movePositions || movePositions.length === 0) {
                statusBarTextStore.set('No moves found in this match');
                return;
            }

            let startIndex = 0;
            let lastVisitedMatch = null;
            const unsub = lastVisitedMatchStore.subscribe(v => lastVisitedMatch = v);
            unsub();
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
            try { analysis = await LoadAnalysis(startMovePos.position.id); } catch (e) {}

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
            closePanel();
        } catch (error) {
            console.error('Error opening match:', error);
            statusBarTextStore.set('Error opening match');
        }
    }

    function closePanel() {
        showTournamentPanelStore.set(false);
    }

    function handleKeyDown(event) {
        if (!visible) return;
        if (event.target.matches('input, textarea')) return;

        // Let Ctrl+key combos pass through to global handler
        if (event.ctrlKey) return;

        // Block all non-Ctrl keys from propagating (prevents position browsing)
        event.stopPropagation();

        if (event.key === 'Escape') {
            event.preventDefault();
            if (editingTournament) {
                cancelEdit();
            } else if (addMatchSearch) {
                addMatchSearch = '';
            } else if (selectedTournament) {
                selectedTournamentStore.set(null);
                tournamentMatchesStore.set([]);
            } else {
                closePanel();
            }
            return;
        }

        // j/k / ArrowUp/Down to browse tournament list
        if (tournaments.length > 0) {
            const currentIndex = selectedTournament 
                ? tournaments.findIndex(t => t.id === selectedTournament.id) 
                : -1;

            if (event.key === 'j' || event.key === 'ArrowDown') {
                event.preventDefault();
                const nextIndex = currentIndex < tournaments.length - 1 ? currentIndex + 1 : currentIndex;
                if (nextIndex >= 0 && nextIndex !== currentIndex) {
                    selectTournament(tournaments[nextIndex]);
                    scrollToTournament(nextIndex);
                } else if (currentIndex === -1 && tournaments.length > 0) {
                    selectTournament(tournaments[0]);
                    scrollToTournament(0);
                }
            } else if (event.key === 'k' || event.key === 'ArrowUp') {
                event.preventDefault();
                if (currentIndex > 0) {
                    selectTournament(tournaments[currentIndex - 1]);
                    scrollToTournament(currentIndex - 1);
                }
            }
        }
    }

    function scrollToTournament(index) {
        setTimeout(() => {
            const items = document.querySelectorAll('.tournament-panel .tournament-item');
            if (items[index]) {
                items[index].scrollIntoView({ behavior: 'smooth', block: 'nearest' });
            }
        }, 0);
    }

    $: if (visible) {
        setTimeout(() => {
            const panel = document.getElementById('tournamentPanel');
            if (panel) panel.focus();
        }, 100);
    }

    onMount(() => {
        document.addEventListener('keydown', handleKeyDown);
    });

    onDestroy(() => {
        document.removeEventListener('keydown', handleKeyDown);
    });
</script>

{#if visible}
    <section class="tournament-panel" role="dialog" aria-modal="true" id="tournamentPanel" tabindex="-1">
        <button type="button" class="close-icon" on:click={closePanel} aria-label="Close">×</button>
        
        <div class="panel-content">
            <div class="panels-container">
                <!-- Left: Tournaments list -->
                <div class="tournaments-list">
                    <div class="panel-header">
                        Tournaments ({tournaments.length})
                    </div>
                    <div class="col-header-row">
                        <span class="col-h col-name">Name</span>
                        <span class="col-h col-count">#</span>
                        <span class="col-h col-date sortable" on:click={() => toggleSort('date')}>
                            Date {sortBy === 'date' ? (sortOrder === 'desc' ? '↓' : '↑') : ''}
                        </span>
                        <span class="col-h col-location sortable" on:click={() => toggleSort('location')}>
                            Location {sortBy === 'location' ? (sortOrder === 'desc' ? '↓' : '↑') : ''}
                        </span>
                        <span class="col-h col-acts"></span>
                    </div>
                    <div class="list-container">
                        {#each tournaments as tournament, index}
                            {#if editingTournament && editingTournament.id === tournament.id}
                                <div class="tournament-item editing">
                                    <span class="col-cell col-name"><input type="text" bind:value={editName} class="edit-input" on:keydown|stopPropagation={(e) => e.key === 'Enter' && saveEdit()} /></span>
                                    <span class="col-cell col-count">{tournament.matchCount}</span>
                                    <span class="col-cell col-date"><input type="date" bind:value={editDate} class="edit-input" /></span>
                                    <span class="col-cell col-location"><input type="text" bind:value={editLocation} class="edit-input" placeholder="Location" on:keydown|stopPropagation={(e) => e.key === 'Enter' && saveEdit()} /></span>
                                    <span class="col-cell col-acts">
                                        <div class="item-actions">
                                            <button class="icon-btn" on:click={saveEdit} title="Save">✓</button>
                                            <button class="icon-btn" on:click={cancelEdit} title="Cancel">✕</button>
                                        </div>
                                    </span>
                                </div>
                            {:else}
                                <div 
                                    class="tournament-item" 
                                    class:selected={selectedTournament?.id === tournament.id}
                                    on:click={() => selectTournament(tournament)}
                                >
                                    <span class="col-cell col-name" title={tournament.name}>{tournament.name}</span>
                                    <span class="col-cell col-count">{tournament.matchCount}</span>
                                    <span class="col-cell col-date">{tournament.date || '—'}</span>
                                    <span class="col-cell col-location" title={tournament.location || ''}>{tournament.location || '—'}</span>
                                    <span class="col-cell col-acts">
                                        <div class="item-actions">
                                            <button class="icon-btn" on:click|stopPropagation={(e) => startEdit(tournament, e)} title="Edit">✎</button>
                                            <button class="icon-btn delete" on:click|stopPropagation={(e) => deleteTournamentEntry(tournament, e)} title="Delete">×</button>
                                        </div>
                                    </span>
                                </div>
                            {/if}
                        {/each}
                        {#if tournaments.length === 0}
                            <div class="empty-msg">No tournaments</div>
                        {/if}
                    </div>
                    <div class="new-item-row">
                        <input 
                            type="text" 
                            bind:value={newTournamentName} 
                            placeholder="Name"
                            on:keydown|stopPropagation={(e) => e.key === 'Enter' && createTournament()}
                        />
                        <input 
                            type="date" 
                            bind:value={newTournamentDate} 
                            class="new-date-input"
                            on:keydown|stopPropagation
                        />
                        <input 
                            type="text" 
                            bind:value={newTournamentLocation} 
                            placeholder="Location"
                            class="new-location-input"
                            on:keydown|stopPropagation={(e) => e.key === 'Enter' && createTournament()}
                        />
                        <button 
                            on:click={createTournament}
                            disabled={!newTournamentName.trim()}
                        >Add</button>
                    </div>
                </div>

                <!-- Right: tournament matches -->
                <div class="right-panel">
                    {#if selectedTournament}
                        <div class="panel-header">
                            <span>{selectedTournament.name} — {tournamentMatches.length} match{tournamentMatches.length !== 1 ? 'es' : ''}</span>
                        </div>
                        <div class="list-container">
                            {#if tournamentMatches.length === 0}
                                <div class="empty-msg">No matches in this tournament</div>
                            {/if}
                            {#each tournamentMatches as match}
                                <div class="match-item" on:dblclick={() => openMatch(match)}>
                                    <span class="match-players" title="{match.player1_name} vs {match.player2_name}">
                                        {match.player1_name} vs {match.player2_name}
                                    </span>
                                    <span class="match-detail">{match.match_length}pt</span>
                                    <button class="icon-btn" on:click|stopPropagation={() => swapMatchPlayersInTournament(match)} title="Swap players">⇄</button>
                                    <button class="icon-btn delete" on:click|stopPropagation={() => removeMatch(match.id)} title="Remove from tournament">×</button>
                                </div>
                            {/each}
                        </div>
                        <div class="add-match-row">
                            <div class="add-match-input-wrapper">
                                <input 
                                    type="text"
                                    bind:value={addMatchSearch}
                                    on:focus={() => { addMatchFocused = true; loadAllMatches().then(updateFilteredMatches); }}
                                    on:blur={() => setTimeout(() => { addMatchFocused = false; }, 150)}
                                    on:keydown|stopPropagation={(e) => { if (e.key === 'Escape') { addMatchSearch = ''; e.target.blur(); } }}
                                    placeholder="Search match to add (player, pts)…"
                                    class="add-match-input"
                                />
                                {#if addMatchFocused && filteredMatches.length > 0}
                                    <div class="add-match-dropdown">
                                        {#each filteredMatches as match}
                                            <div class="add-match-item" on:mousedown|preventDefault={() => addMatchToTournament(match.id)}>
                                                <span class="match-players">{match.player1_name} vs {match.player2_name}</span>
                                                <span class="match-detail">{match.match_length}pt</span>
                                            </div>
                                        {/each}
                                    </div>
                                {/if}
                            </div>
                        </div>
                    {:else}
                        <div class="panel-header">Matches</div>
                        <div class="list-container">
                            <div class="empty-msg">Select a tournament</div>
                        </div>
                    {/if}
                </div>
            </div>
        </div>
    </section>
{/if}

<style>
    .tournament-panel {
        position: fixed;
        width: 100%;
        bottom: 22px;
        left: 0;
        right: 0;
        height: 178px;
        background-color: white;
        border-top: 1px solid rgba(0, 0, 0, 0.1);
        padding: 4px 10px;
        box-sizing: border-box;
        z-index: 5;
        outline: none;
        overflow: hidden;
        user-select: none;
        -webkit-user-select: none;
        -moz-user-select: none;
        -ms-user-select: none;
    }

    .close-icon {
        position: absolute;
        top: -4px;
        right: 6px;
        font-size: 20px;
        font-weight: bold;
        cursor: pointer;
        color: #333;
        background: none;
        border: none;
        padding: 2px 6px;
        z-index: 10;
    }

    .close-icon:hover {
        color: #000;
    }

    .panel-content {
        font-size: 12px;
        color: #333;
        height: 100%;
        display: flex;
        flex-direction: column;
    }

    .panels-container {
        display: flex;
        gap: 6px;
        flex: 1;
        min-height: 0;
    }

    .tournaments-list {
        flex: 3;
        display: flex;
        flex-direction: column;
        border: 1px solid #ddd;
        min-width: 0;
        overflow: hidden;
    }

    .right-panel {
        flex: 2;
        min-width: 180px;
        max-width: 350px;
        display: flex;
        flex-direction: column;
        border: 1px solid #ddd;
        overflow: hidden;
    }

    .panel-header {
        background-color: #f2f2f2;
        padding: 2px 8px;
        font-weight: bold;
        font-size: 11px;
        border-bottom: 1px solid #ddd;
        flex-shrink: 0;
        display: flex;
        align-items: center;
        gap: 6px;
        white-space: nowrap;
    }

    .col-header-row {
        display: flex;
        align-items: center;
        background-color: #f8f8f8;
        border-bottom: 1px solid #ddd;
        padding: 1px 8px;
        flex-shrink: 0;
    }

    .col-h {
        font-size: 10px;
        font-weight: 600;
        color: #888;
        text-transform: uppercase;
        letter-spacing: 0.3px;
    }

    .col-h.sortable {
        cursor: pointer;
    }

    .col-h.sortable:hover {
        color: #333;
    }

    .col-name { width: 140px; min-width: 80px; flex-shrink: 0; text-align: left; }
    .col-count { width: 30px; flex-shrink: 0; text-align: right; padding-right: 8px; }
    .col-date { width: 90px; flex-shrink: 0; text-align: left; }
    .col-location { flex: 1; min-width: 60px; text-align: left; }
    .col-acts { width: 50px; flex-shrink: 0; text-align: right; }

    .list-container {
        flex: 1;
        overflow-y: auto;
        overflow-x: hidden;
    }

    .tournament-item {
        display: flex;
        align-items: center;
        padding: 2px 8px;
        cursor: pointer;
        border-bottom: 1px solid #f0f0f0;
        min-height: 22px;
    }

    .tournament-item:hover {
        background-color: #f5f5f5;
    }

    .tournament-item.selected {
        background-color: #e3f2fd;
    }

    .tournament-item.editing {
        background-color: #fff3cd;
    }

    .col-cell {
        font-size: 11px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .item-actions {
        display: flex;
        gap: 2px;
        visibility: hidden;
    }

    .tournament-item:hover .item-actions,
    .tournament-item.editing .item-actions {
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

    .edit-input {
        width: 100%;
        padding: 1px 4px;
        border: 1px solid #ccc;
        border-radius: 2px;
        font-size: 11px;
        box-sizing: border-box;
    }

    .empty-msg {
        text-align: center;
        color: #999;
        padding: 12px;
        font-size: 11px;
    }

    .new-item-row {
        display: flex;
        gap: 4px;
        padding: 3px 8px 8px 8px;
        border-top: 1px solid #eee;
        background: #fafafa;
        flex-shrink: 0;
    }

    .new-item-row input {
        flex: 1;
        padding: 2px 6px;
        border: 1px solid #ccc;
        border-radius: 3px;
        font-size: 11px;
        min-width: 0;
    }

    .new-date-input {
        flex: 0 0 auto !important;
        width: 110px;
    }

    .new-location-input {
        flex: 0.7 !important;
    }

    .new-item-row button {
        padding: 2px 10px;
        border: 1px solid #ccc;
        border-radius: 3px;
        font-size: 11px;
        cursor: pointer;
        background: white;
    }

    .new-item-row button:hover {
        background: #f0f0f0;
    }

    .new-item-row button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    /* Right panel: match items */
    .match-item {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 2px 8px;
        cursor: pointer;
        font-size: 11px;
        border-bottom: 1px solid #f0f0f0;
        min-height: 22px;
    }

    .match-item:hover {
        background: #f5f5f5;
    }

    .match-players {
        flex: 1;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .match-detail {
        color: #999;
        font-size: 10px;
        flex-shrink: 0;
    }

    .match-item .icon-btn {
        visibility: hidden;
    }

    .match-item:hover .icon-btn {
        visibility: visible;
    }

    /* Add match row */
    .add-match-row {
        border-top: 1px solid #eee;
        background: #fafafa;
        flex-shrink: 0;
        padding: 3px 8px 8px 8px;
    }

    .add-match-input-wrapper {
        position: relative;
    }

    .add-match-input {
        width: 100%;
        padding: 2px 6px;
        border: 1px solid #ccc;
        border-radius: 3px;
        font-size: 11px;
        box-sizing: border-box;
        outline: none;
    }

    .add-match-input:focus {
        border-color: #99c;
    }

    .add-match-dropdown {
        position: absolute;
        bottom: 100%;
        left: 0;
        right: 0;
        max-height: 90px;
        overflow-y: auto;
        background: white;
        border: 1px solid #ccc;
        border-bottom: none;
        border-radius: 3px 3px 0 0;
        box-shadow: 0 -2px 6px rgba(0,0,0,0.1);
        z-index: 20;
    }

    .add-match-item {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 2px 8px;
        cursor: pointer;
        font-size: 11px;
        border-bottom: 1px solid #f5f5f5;
    }

    .add-match-item:hover {
        background: #e3f2fd;
    }
</style>
