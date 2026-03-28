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
        SaveLastVisitedPosition,
        UpdateMatchComment,
        UpdateTournamentComment,
        ReorderTournamentMatches
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
    let sortBy = null;
    let sortOrder = 'asc';
    
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

    // Match comment editing
    let editingMatchCommentId = null;
    let editingMatchComment = '';

    // Tournament comment editing
    let editingTournamentComment = false;
    let tournamentCommentText = '';

    // Drag-and-drop reordering
    let dragIndex = null;
    let dragOverIndex = null;

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
        if (!sortBy) return list;
        return [...list].sort((a, b) => {
            let valA, valB;
            if (sortBy === 'name') {
                valA = (a.name || '').toLowerCase();
                valB = (b.name || '').toLowerCase();
            } else if (sortBy === 'date') {
                valA = a.date || '';
                valB = b.date || '';
            } else if (sortBy === 'location') {
                valA = (a.location || '').toLowerCase();
                valB = (b.location || '').toLowerCase();
            } else if (sortBy === 'matches') {
                valA = a.matchCount || 0;
                valB = b.matchCount || 0;
            } else {
                return 0;
            }
            const cmp = typeof valA === 'number' ? valA - valB : valA.localeCompare(valB);
            return sortOrder === 'asc' ? cmp : -cmp;
        });
    }

    function handleSort(column) {
        if (sortBy === column) {
            if (sortOrder === 'asc') {
                sortOrder = 'desc';
            } else {
                sortBy = null;
                sortOrder = 'asc';
            }
        } else {
            sortBy = column;
            sortOrder = 'asc';
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
        // Only show matches not assigned to any tournament
        let available = allMatches.filter(m => !m.tournament_id);
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

    // Match reordering
    async function moveMatchUp(index) {
        if (index <= 0 || !selectedTournament) return;
        const newList = [...tournamentMatches];
        [newList[index - 1], newList[index]] = [newList[index], newList[index - 1]];
        tournamentMatchesStore.set(newList);
        try {
            await ReorderTournamentMatches(selectedTournament.id, newList.map(m => m.id));
        } catch (error) {
            console.error('Error reordering matches:', error);
        }
    }

    async function moveMatchDown(index) {
        if (index >= tournamentMatches.length - 1 || !selectedTournament) return;
        const newList = [...tournamentMatches];
        [newList[index], newList[index + 1]] = [newList[index + 1], newList[index]];
        tournamentMatchesStore.set(newList);
        try {
            await ReorderTournamentMatches(selectedTournament.id, newList.map(m => m.id));
        } catch (error) {
            console.error('Error reordering matches:', error);
        }
    }

    // Match comment
    function startEditMatchComment(match, event) {
        event.stopPropagation();
        editingMatchCommentId = match.id;
        editingMatchComment = match.comment || '';
    }

    async function saveMatchComment() {
        if (editingMatchCommentId == null) return;
        try {
            await UpdateMatchComment(editingMatchCommentId, editingMatchComment);
            // Update local state
            const updated = tournamentMatches.map(m =>
                m.id === editingMatchCommentId ? { ...m, comment: editingMatchComment } : m
            );
            tournamentMatchesStore.set(updated);
            editingMatchCommentId = null;
        } catch (error) {
            console.error('Error saving match comment:', error);
        }
    }

    function cancelMatchComment() {
        editingMatchCommentId = null;
        editingMatchComment = '';
    }

    // Tournament comment
    function startEditTournamentComment() {
        editingTournamentComment = true;
        tournamentCommentText = selectedTournament?.comment || '';
    }

    async function saveTournamentComment() {
        if (!selectedTournament) return;
        try {
            await UpdateTournamentComment(selectedTournament.id, tournamentCommentText);
            selectedTournamentStore.set({ ...selectedTournament, comment: tournamentCommentText });
            editingTournamentComment = false;
        } catch (error) {
            console.error('Error saving tournament comment:', error);
        }
    }

    function cancelTournamentComment() {
        editingTournamentComment = false;
    }

    // Drag-and-drop
    function handleDragStart(index, event) {
        dragIndex = index;
        event.dataTransfer.effectAllowed = 'move';
        event.dataTransfer.setData('text/plain', String(index));
    }

    function handleDragOver(index, event) {
        event.preventDefault();
        event.dataTransfer.dropEffect = 'move';
        dragOverIndex = index;
    }

    function handleDragLeave() {
        dragOverIndex = null;
    }

    async function handleDrop(index, event) {
        event.preventDefault();
        if (dragIndex === null || dragIndex === index || !selectedTournament) {
            dragIndex = null;
            dragOverIndex = null;
            return;
        }
        const newList = [...tournamentMatches];
        const [moved] = newList.splice(dragIndex, 1);
        newList.splice(index, 0, moved);
        tournamentMatchesStore.set(newList);
        dragIndex = null;
        dragOverIndex = null;
        try {
            await ReorderTournamentMatches(selectedTournament.id, newList.map(m => m.id));
        } catch (error) {
            console.error('Error reordering matches:', error);
        }
    }

    function handleDragEnd() {
        dragIndex = null;
        dragOverIndex = null;
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
            const rows = document.querySelectorAll('.tournament-panel .tournament-table tbody tr');
            if (rows[index]) {
                rows[index].scrollIntoView({ behavior: 'smooth', block: 'nearest' });
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
            {#if !selectedTournament}
                <!-- Tournaments list -->
                <div class="tournament-list-pane">
                    <div class="table-container">
                        <table class="tournament-table">
                            <thead>
                                <tr>
                                    <th class="no-select sortable" on:click={() => handleSort('name')}>Name {#if sortBy === 'name'}<span class="sort-arrow">{sortOrder === 'asc' ? '▲' : '▼'}</span>{/if}</th>
                                    <th class="no-select sortable narrow-col" on:click={() => handleSort('matches')}>Matches {#if sortBy === 'matches'}<span class="sort-arrow">{sortOrder === 'asc' ? '▲' : '▼'}</span>{/if}</th>
                                    <th class="no-select sortable narrow-col" on:click={() => handleSort('date')}>Date {#if sortBy === 'date'}<span class="sort-arrow">{sortOrder === 'asc' ? '▲' : '▼'}</span>{/if}</th>
                                    <th class="no-select sortable" on:click={() => handleSort('location')}>Location {#if sortBy === 'location'}<span class="sort-arrow">{sortOrder === 'asc' ? '▲' : '▼'}</span>{/if}</th>
                                    <th class="no-select actions-col"></th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each tournaments as tournament, index}
                                    {#if editingTournament && editingTournament.id === tournament.id}
                                        <tr class="editing-row">
                                            <td><input class="edit-input" type="text" bind:value={editName} on:keydown={(e) => { if (e.key === 'Enter') { e.stopPropagation(); saveEdit(); } if (e.key === 'Escape') { e.stopPropagation(); cancelEdit(); } }} autofocus /></td>
                                            <td class="narrow-col"></td>
                                            <td class="narrow-col"><input class="edit-input" type="date" bind:value={editDate} on:keydown={(e) => { if (e.key === 'Escape') { e.stopPropagation(); cancelEdit(); } }} /></td>
                                            <td><input class="edit-input" type="text" bind:value={editLocation} placeholder="Location" on:keydown={(e) => { if (e.key === 'Enter') { e.stopPropagation(); saveEdit(); } if (e.key === 'Escape') { e.stopPropagation(); cancelEdit(); } }} /></td>
                                            <td class="actions-col no-select">
                                                <span class="item-actions editing-actions">
                                                    <button class="icon-btn" on:click={saveEdit} title="Save">✓</button>
                                                    <button class="icon-btn" on:click={cancelEdit} title="Cancel">✕</button>
                                                </span>
                                            </td>
                                        </tr>
                                    {:else}
                                        <tr on:click={() => selectTournament(tournament)} on:dblclick={() => selectTournament(tournament)}>
                                            <td class="no-select">{tournament.name}</td>
                                            <td class="narrow-col no-select count-cell">{tournament.matchCount || 0}</td>
                                            <td class="narrow-col no-select">{tournament.date || ''}</td>
                                            <td class="no-select">{tournament.location || ''}</td>
                                            <td class="actions-col no-select">
                                                <span class="item-actions">
                                                    <button class="icon-btn" on:click|stopPropagation={(e) => startEdit(tournament, e)} title="Edit">✎</button>
                                                    <button class="icon-btn delete" on:click|stopPropagation={(e) => deleteTournamentEntry(tournament, e)} title="Delete">×</button>
                                                </span>
                                            </td>
                                        </tr>
                                    {/if}
                                {/each}
                            </tbody>
                        </table>
                        {#if tournaments.length === 0}
                            <div class="empty-state">No tournaments</div>
                        {/if}
                    </div>
                    <div class="add-area">
                        <input class="add-input name" type="text" bind:value={newTournamentName} placeholder="New tournament…" on:keydown={(e) => { if (e.key === 'Enter') { e.stopPropagation(); createTournament(); } if (e.key === 'Escape') { e.stopPropagation(); e.currentTarget.blur(); } }} />
                        <input class="add-input date" type="date" bind:value={newTournamentDate} on:keydown={(e) => { if (e.key === 'Escape') { e.stopPropagation(); e.currentTarget.blur(); } }} />
                        <input class="add-input loc" type="text" bind:value={newTournamentLocation} placeholder="Location" on:keydown={(e) => { if (e.key === 'Enter') { e.stopPropagation(); createTournament(); } if (e.key === 'Escape') { e.stopPropagation(); e.currentTarget.blur(); } }} />
                    </div>
                </div>
            {:else}
                <!-- Matches for selected tournament -->
                <div class="tournament-list-pane">
                    <div class="detail-header">
                        <button class="back-btn" on:click={() => { selectedTournamentStore.set(null); tournamentMatchesStore.set([]); addMatchSearch = ''; editingTournamentComment = false; }} title="Back to tournaments">←</button>
                        <span class="header-name" title={selectedTournament.name}>{selectedTournament.name}</span>
                        {#if selectedTournament.date || selectedTournament.location}
                            <span class="header-meta">
                                {#if selectedTournament.date}{selectedTournament.date}{/if}
                                {#if selectedTournament.date && selectedTournament.location} · {/if}
                                {#if selectedTournament.location}{selectedTournament.location}{/if}
                            </span>
                        {/if}
                        <button class="icon-btn edit-header-btn" on:click|stopPropagation={(e) => startEdit(selectedTournament, e)} title="Edit">✎</button>
                        <span class="header-spacer"></span>
                        {#if editingTournamentComment}
                            <input
                                class="tournament-comment-inline"
                                type="text"
                                bind:value={tournamentCommentText}
                                on:keydown={(e) => { if (e.key === 'Enter') { e.stopPropagation(); saveTournamentComment(); } if (e.key === 'Escape') { e.stopPropagation(); cancelTournamentComment(); } }}
                                on:blur={saveTournamentComment}
                                placeholder="Notes…"
                                autofocus
                            />
                        {:else}
                            <span class="tournament-comment-text" class:has-comment={selectedTournament.comment} on:click|stopPropagation={startEditTournamentComment} title={selectedTournament.comment || 'Click to add notes'}>
                                {selectedTournament.comment || 'Notes…'}
                            </span>
                        {/if}
                    </div>
                    <div class="table-container">
                        <table class="tournament-table">
                            <thead>
                                <tr>
                                    <th class="no-select narrow-col">#</th>
                                    <th class="no-select">Player 1</th>
                                    <th class="no-select">Player 2</th>
                                    <th class="no-select narrow-col">Pts</th>
                                    <th class="no-select comment-col">Comment</th>
                                    <th class="no-select actions-col"></th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each tournamentMatches as match, index}
                                    <tr
                                        on:dblclick={() => openMatch(match)}
                                        draggable="true"
                                        on:dragstart={(e) => handleDragStart(index, e)}
                                        on:dragover={(e) => handleDragOver(index, e)}
                                        on:dragleave={handleDragLeave}
                                        on:drop={(e) => handleDrop(index, e)}
                                        on:dragend={handleDragEnd}
                                        class:drag-over={dragOverIndex === index && dragIndex !== index}
                                    >
                                        <td class="index-cell narrow-col no-select">{index + 1}</td>
                                        <td class="no-select">{match.player1_name}</td>
                                        <td class="no-select">{match.player2_name}</td>
                                        <td class="narrow-col no-select">{match.match_length}</td>
                                        <td class="comment-col no-select">
                                            {#if editingMatchCommentId === match.id}
                                                <input
                                                    class="edit-input"
                                                    type="text"
                                                    bind:value={editingMatchComment}
                                                    on:keydown={(e) => { if (e.key === 'Enter') { e.stopPropagation(); saveMatchComment(); } if (e.key === 'Escape') { e.stopPropagation(); cancelMatchComment(); } }}
                                                    on:blur={saveMatchComment}
                                                    autofocus
                                                />
                                            {:else}
                                                <span class="comment-text" class:has-comment={match.comment} on:click|stopPropagation={(e) => startEditMatchComment(match, e)} title={match.comment || 'Click to add comment'}>
                                                    {match.comment || ''}
                                                </span>
                                            {/if}
                                        </td>
                                        <td class="actions-col no-select">
                                            <span class="item-actions">
                                                <button class="icon-btn" on:click|stopPropagation={() => moveMatchUp(index)} disabled={index === 0} title="Move up">▲</button>
                                                <button class="icon-btn" on:click|stopPropagation={() => moveMatchDown(index)} disabled={index === tournamentMatches.length - 1} title="Move down">▼</button>
                                                <button class="icon-btn" on:click|stopPropagation={() => swapMatchPlayersInTournament(match)} title="Swap">⇄</button>
                                                <button class="icon-btn delete" on:click|stopPropagation={() => removeMatch(match.id)} title="Remove">×</button>
                                            </span>
                                        </td>
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                        {#if tournamentMatches.length === 0}
                            <div class="empty-state">No matches</div>
                        {/if}
                    </div>
                    <div class="add-area">
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
                                            {match.player1_name} vs {match.player2_name} <span class="match-pts">{match.match_length}pt</span>
                                        </div>
                                    {/each}
                                </div>
                            {/if}
                        </div>
                    </div>
                </div>
            {/if}
        </div>
    </section>

<style>
    .tournament-panel { width: 100%; height: 100%; background: white; box-sizing: border-box; outline: none; overflow: hidden; user-select: none; -webkit-user-select: none; }
    .panel-content { font-size: 12px; color: #333; height: 100%; display: flex; overflow: hidden; }

    .tournament-list-pane { flex: 1; min-width: 0; height: 100%; overflow: hidden; display: flex; flex-direction: column; }

    .table-container { flex: 1; overflow-y: auto; }

    .tournament-table { width: 100%; border-collapse: collapse; font-size: 12px; }
    .tournament-table thead { position: sticky; top: 0; background-color: #f5f5f5; z-index: 1; }
    .tournament-table th,
    .tournament-table td { padding: 4px 8px; text-align: left; border-bottom: 1px solid #e0e0e0; }
    .tournament-table th { font-weight: 600; color: #333; font-size: 11px; }
    .tournament-table th.sortable { cursor: pointer; }
    .tournament-table th.sortable:hover { background-color: #e8e8e8; }
    .sort-arrow { font-size: 9px; margin-left: 3px; color: #1976d2; }

    .tournament-table tbody tr { cursor: pointer; transition: background-color 0.1s; }
    .tournament-table tbody tr:hover { background-color: #f9f9f9; }
    .tournament-table tbody tr.editing-row { background-color: #fefce8; cursor: default; }

    .index-cell { text-align: center; color: #999; }
    .count-cell { text-align: center; color: #888; }
    .narrow-col { width: 1px; white-space: nowrap; padding-left: 6px; padding-right: 6px; }
    .actions-col { width: 80px; min-width: 80px; max-width: 80px; white-space: nowrap; text-align: center; padding: 0 4px; }

    .item-actions { display: inline-flex; gap: 2px; visibility: hidden; vertical-align: middle; }
    .editing-actions { visibility: visible; }
    .tournament-table tbody tr:hover .item-actions { visibility: visible; }

    .icon-btn { background: none; border: none; cursor: pointer; font-size: 12px; color: #666; padding: 0 3px; line-height: 1; }
    .icon-btn:hover:not(:disabled) { color: #000; }
    .icon-btn:disabled { opacity: 0.3; cursor: not-allowed; }
    .icon-btn.delete:hover:not(:disabled) { color: #c55; }

    .edit-input { width: 100%; padding: 1px 4px; border: 1px solid #1976d2; border-radius: 2px; font-size: 11px; box-sizing: border-box; outline: none; }

    .no-select { user-select: none; -webkit-user-select: none; }
    .empty-state { text-align: center; color: #999; padding: 24px; font-size: 12px; }

    /* Detail header for match view */
    .detail-header { padding: 3px 8px; font-size: 11px; color: #555; border-bottom: 1px solid #e0e0e0; flex-shrink: 0; background: #f5f5f5; display: flex; align-items: center; gap: 6px; min-height: 24px; }
    .back-btn { border: none; background: none; cursor: pointer; font-size: 14px; color: #666; padding: 0 4px; line-height: 1; flex-shrink: 0; }
    .back-btn:hover { color: #333; }
    .header-name { font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .header-meta { font-size: 10px; color: #999; flex-shrink: 0; }
    .edit-header-btn { visibility: hidden; flex-shrink: 0; }
    .detail-header:hover .edit-header-btn { visibility: visible; }
    .header-spacer { flex: 1; }

    /* Inline tournament comment in header */
    .tournament-comment-inline { flex: 1; min-width: 80px; padding: 1px 4px; border: 1px solid #1976d2; border-radius: 2px; font-size: 11px; box-sizing: border-box; outline: none; }
    .tournament-comment-text { flex-shrink: 1; font-size: 10px; color: #bbb; cursor: pointer; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 200px; font-style: italic; }
    .tournament-comment-text.has-comment { color: #888; }
    .tournament-comment-text:hover { color: #1976d2; }

    /* Add area */
    .add-area { border-top: 1px solid #eee; padding: 3px 8px 4px; flex-shrink: 0; background: #fafafa; display: flex; gap: 4px; align-items: center; }
    .add-input { padding: 2px 4px; border: 1px solid #ccc; border-radius: 2px; font-size: 11px; outline: none; box-sizing: border-box; }
    .add-input:focus { border-color: #99c; }
    .add-input.name { flex: 1; min-width: 0; }
    .add-input.date { width: 110px; flex-shrink: 0; }
    .add-input.loc { width: 90px; flex-shrink: 0; }

    /* Add match dropdown */
    .add-match-wrap { position: relative; flex: 1; }
    .add-match-input { width: 100%; padding: 2px 6px; border: 1px solid #ccc; border-radius: 3px; font-size: 11px; box-sizing: border-box; outline: none; }
    .add-match-input:focus { border-color: #99c; }
    .match-dropdown { overflow-y: auto; background: white; border: 1px solid #ccc; border-radius: 3px; box-shadow: 0 2px 8px rgba(0,0,0,0.18); z-index: 9999; }
    .dropdown-item { padding: 3px 8px; cursor: pointer; font-size: 11px; border-bottom: 1px solid #f5f5f5; }
    .dropdown-item:hover { background: #e3f2fd; }
    .match-pts { font-size: 10px; color: #999; }

    /* Match comment column */
    .comment-col { max-width: 140px; overflow: hidden; }
    .comment-text { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 11px; color: #aaa; cursor: pointer; min-height: 16px; }
    .comment-text.has-comment { color: #666; }
    .comment-text:hover { color: #1976d2; }

    /* Drag-and-drop */
    .tournament-table tbody tr.drag-over { border-top: 2px solid #1976d2; }
    .tournament-table tbody tr[draggable="true"] { cursor: grab; }
    .tournament-table tbody tr[draggable="true"]:active { cursor: grabbing; }
</style>
