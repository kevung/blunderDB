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
        SwapMatchPlayers,
        SaveLastVisitedPosition
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
    let matchDropdownStyle = '';

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

    function computeMatchDropdownPos(inputEl) {
        if (!inputEl) return;
        const rect = inputEl.getBoundingClientRect();
        const spaceAbove = rect.top;
        const maxH = 90;
        matchDropdownStyle = `position:fixed; bottom:${window.innerHeight - rect.top}px; left:${rect.left}px; width:${rect.width}px; max-height:${Math.min(maxH, spaceAbove)}px;`;
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
            // First check in-memory store for same session, then check DB-persisted value
            if (lastVisitedMatch && lastVisitedMatch.matchID === match.id) {
                if (lastVisitedMatch.currentIndex >= 0 && lastVisitedMatch.currentIndex < movePositions.length) {
                    startIndex = lastVisitedMatch.currentIndex;
                }
            } else if (match.last_visited_position >= 0 && match.last_visited_position < movePositions.length) {
                startIndex = match.last_visited_position;
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
            // Persist last visited position to database
            SaveLastVisitedPosition(match.id, startIndex).catch(e => {
                console.error('Error persisting last visited position:', e);
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

    <section class="tournament-panel" id="tournamentPanel" tabindex="-1">
        <div class="panel-content">
            <!-- Sub-tab sidebar -->
            <div class="sub-tab-sidebar">
                <button class="sub-tab-btn home-btn" class:active={!selectedTournament} on:click={() => { selectedTournamentStore.set(null); tournamentMatchesStore.set([]); addMatchSearch = ''; }} title="Tournaments">⌂</button>
                {#if selectedTournament}
                    <div class="sub-tab-btn named-tab" class:active={selectedTournament} role="button" tabindex="-1" on:click={() => {}} on:keydown={() => {}}>
                        <span class="tab-name" title={selectedTournament.name}>{selectedTournament.name}</span>
                        <button class="tab-close-btn" on:click|stopPropagation={() => { selectedTournamentStore.set(null); tournamentMatchesStore.set([]); addMatchSearch = ''; }} title="Close">×</button>
                    </div>
                {/if}
            </div>

            <div class="sub-tab-content">
                {#if !selectedTournament}
                    <!-- Tournaments list -->
                    <div class="list-view">
                        <div class="list-container">
                            {#each tournaments as tournament, index}
                                {#if editingTournament && editingTournament.id === tournament.id}
                                    <div class="row editing">
                                        <input class="row-input name" type="text" bind:value={editName} on:keydown={(e) => { if (e.key === 'Enter') { e.stopPropagation(); saveEdit(); } if (e.key === 'Escape') { e.stopPropagation(); cancelEdit(); } }} autofocus />
                                        <input class="row-input date" type="date" bind:value={editDate} on:keydown={(e) => { if (e.key === 'Escape') { e.stopPropagation(); cancelEdit(); } }} />
                                        <input class="row-input loc" type="text" bind:value={editLocation} placeholder="Location" on:keydown={(e) => { if (e.key === 'Enter') { e.stopPropagation(); saveEdit(); } if (e.key === 'Escape') { e.stopPropagation(); cancelEdit(); } }} />
                                        <span class="row-acts">
                                            <button class="icon-btn" on:click={saveEdit}>✓</button>
                                            <button class="icon-btn" on:click={cancelEdit}>✕</button>
                                        </span>
                                    </div>
                                {:else}
                                    <div class="row" on:click={() => selectTournament(tournament)} on:dblclick={() => selectTournament(tournament)}>
                                        <span class="row-name" title={tournament.name}>{tournament.name}</span>
                                        {#if tournament.matchCount > 0}
                                            <span class="row-badge">{tournament.matchCount}</span>
                                        {/if}
                                        <span class="row-date">{tournament.date || ''}</span>
                                        <span class="row-acts">
                                            <button class="icon-btn" on:click|stopPropagation={(e) => startEdit(tournament, e)} title="Edit">✎</button>
                                            <button class="icon-btn delete" on:click|stopPropagation={(e) => deleteTournamentEntry(tournament, e)} title="Delete">×</button>
                                        </span>
                                    </div>
                                {/if}
                            {/each}
                            <!-- Inline add row -->
                            <div class="row add-row">
                                <input class="row-input name" type="text" bind:value={newTournamentName} placeholder="New tournament…" on:keydown={(e) => { if (e.key === 'Enter') { e.stopPropagation(); createTournament(); } if (e.key === 'Escape') { e.stopPropagation(); e.currentTarget.blur(); } }} />
                                <input class="row-input date" type="date" bind:value={newTournamentDate} on:keydown={(e) => { if (e.key === 'Escape') { e.stopPropagation(); e.currentTarget.blur(); } }} />
                                <input class="row-input loc" type="text" bind:value={newTournamentLocation} placeholder="Location" on:keydown={(e) => { if (e.key === 'Enter') { e.stopPropagation(); createTournament(); } if (e.key === 'Escape') { e.stopPropagation(); e.currentTarget.blur(); } }} />
                            </div>
                        </div>
                    </div>
                {:else}
                    <!-- Matches for selected tournament -->
                    <div class="list-view">
                        <div class="list-header">{selectedTournament.name}</div>
                        <div class="list-container">
                            {#if tournamentMatches.length === 0}
                                <div class="empty-msg">No matches</div>
                            {/if}
                            {#each tournamentMatches as match}
                                <div class="row match-row" on:dblclick={() => openMatch(match)}>
                                    <span class="row-name">{match.player1_name} vs {match.player2_name}</span>
                                    <span class="row-detail">{match.match_length}pt</span>
                                    <span class="row-acts">
                                        <button class="icon-btn" on:click|stopPropagation={() => swapMatchPlayersInTournament(match)} title="Swap">⇄</button>
                                        <button class="icon-btn delete" on:click|stopPropagation={() => removeMatch(match.id)} title="Remove">×</button>
                                    </span>
                                </div>
                            {/each}
                        </div>
                        <div class="add-match-area">
                            <div class="add-match-wrap">
                                <input
                                    type="text"
                                    bind:value={addMatchSearch}
                                    on:focus={(e) => { addMatchFocused = true; computeMatchDropdownPos(e.currentTarget); loadAllMatches().then(updateFilteredMatches); }}
                                    on:blur={() => setTimeout(() => { addMatchFocused = false; }, 150)}
                                    on:keydown={(e) => { if (e.key === 'Escape') { e.stopPropagation(); addMatchSearch = ''; e.currentTarget.blur(); } }}
                                    placeholder="Add match…"
                                    class="add-match-input"
                                />
                                {#if addMatchFocused && filteredMatches.length > 0}
                                    <div class="match-dropdown" style={matchDropdownStyle}>
                                        {#each filteredMatches as match}
                                            <div class="dropdown-item" on:mousedown|preventDefault={() => addMatchToTournament(match.id)}>
                                                {match.player1_name} vs {match.player2_name} <span class="row-detail">{match.match_length}pt</span>
                                            </div>
                                        {/each}
                                    </div>
                                {/if}
                            </div>
                        </div>
                    </div>
                {/if}
            </div>
        </div>
    </section>

<style>
    .tournament-panel { width: 100%; height: 100%; background: white; box-sizing: border-box; outline: none; overflow: hidden; user-select: none; }
    .panel-content { font-size: 12px; color: #333; height: 100%; display: flex; }

    .sub-tab-sidebar { display: flex; flex-direction: column; width: 70px; flex-shrink: 0; background: #f5f5f5; border-right: 1px solid #ddd; }
    .sub-tab-btn { border: none; background: transparent; padding: 8px 4px; font-size: 11px; color: #666; cursor: pointer; border-left: 2px solid transparent; text-align: center; transition: background 0.15s; }
    .sub-tab-btn:hover { background: #e8e8e8; }
    .sub-tab-btn.active { color: #333; font-weight: 600; background: #fff; border-left-color: #555; }
    .sub-tab-btn.home-btn { font-size: 16px; padding: 6px 4px; }
    .sub-tab-btn.named-tab { display: flex; flex-direction: column; align-items: center; gap: 2px; padding: 6px 2px; position: relative; cursor: pointer; }
    .tab-name { font-size: 9px; line-height: 1.2; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 62px; display: block; }
    .tab-close-btn { border: none; background: none; font-size: 12px; color: #aaa; cursor: pointer; line-height: 1; padding: 0 2px; }
    .tab-close-btn:hover { color: #c55; }
    .sub-tab-content { flex: 1; min-width: 0; overflow: hidden; display: flex; }

    .list-view { flex: 1; display: flex; flex-direction: column; min-height: 0; overflow: hidden; }
    .list-header { padding: 3px 10px; font-size: 11px; font-weight: 600; color: #555; border-bottom: 1px solid #eee; flex-shrink: 0; background: #fafafa; }
    .list-container { flex: 1; overflow-y: auto; overflow-x: hidden; }

    .row {
        display: flex;
        align-items: center;
        padding: 2px 10px;
        cursor: pointer;
        border-bottom: 1px solid #f5f5f5;
        min-height: 24px;
        gap: 6px;
    }
    .row:hover { background: #f5f8ff; }
    .row.editing { background: #fefce8; cursor: default; }
    .row.add-row { cursor: default; background: #fafafa; border-bottom: none; }
    .row.add-row:hover { background: #fafafa; }

    .row-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; }
    .row-badge { font-size: 9px; color: #888; background: #eee; border-radius: 8px; padding: 0 5px; flex-shrink: 0; }
    .row-date { font-size: 10px; color: #999; flex-shrink: 0; width: 80px; text-align: right; }
    .row-detail { font-size: 10px; color: #999; flex-shrink: 0; }
    .row-acts { display: flex; gap: 2px; visibility: hidden; flex-shrink: 0; }
    .row:hover .row-acts, .row.editing .row-acts { visibility: visible; }

    .row-input { padding: 1px 4px; border: 1px solid #ccc; border-radius: 2px; font-size: 11px; outline: none; box-sizing: border-box; }
    .row-input.name { flex: 1; min-width: 0; }
    .row-input.date { width: 110px; flex-shrink: 0; }
    .row-input.loc { width: 90px; flex-shrink: 0; }

    .icon-btn { background: none; border: none; cursor: pointer; font-size: 12px; color: #888; padding: 0 3px; line-height: 1; }
    .icon-btn:hover { color: #333; }
    .icon-btn.delete:hover { color: #c55; }

    .empty-msg { text-align: center; color: #bbb; padding: 16px; font-size: 11px; font-style: italic; }

    .match-row { cursor: default; }
    .match-row:hover { background: #f5f8ff; }

    .add-match-area { border-top: 1px solid #eee; padding: 3px 10px 6px; flex-shrink: 0; background: #fafafa; }
    .add-match-wrap { position: relative; }
    .add-match-input { width: 100%; padding: 2px 6px; border: 1px solid #ccc; border-radius: 3px; font-size: 11px; box-sizing: border-box; outline: none; }
    .add-match-input:focus { border-color: #99c; }

    .match-dropdown { overflow-y: auto; background: white; border: 1px solid #ccc; border-radius: 3px; box-shadow: 0 2px 8px rgba(0,0,0,0.18); z-index: 9999; }
    .dropdown-item { padding: 3px 8px; cursor: pointer; font-size: 11px; border-bottom: 1px solid #f5f5f5; }
    .dropdown-item:hover { background: #e3f2fd; }
</style>
