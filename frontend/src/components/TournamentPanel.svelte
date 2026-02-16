<script>
    import { onMount, onDestroy } from 'svelte';
    import { 
        GetAllTournaments, 
        CreateTournament, 
        DeleteTournament, 
        UpdateTournament,
        GetTournamentMatches,
        RemoveMatchFromTournament
    } from '../../wailsjs/go/main/Database.js';
    import { showTournamentPanelStore, statusBarTextStore } from '../stores/uiStore';
    import { tournamentsStore, selectedTournamentStore, tournamentMatchesStore } from '../stores/tournamentStore';
    import { databaseLoadedStore } from '../stores/databaseStore';

    let tournaments = [];
    let selectedTournament = null;
    let tournamentMatches = [];
    let visible = false;
    let databaseLoaded = false;
    let sortBy = 'date'; // 'date' or 'location'
    let sortOrder = 'desc'; // 'desc' (most recent first) or 'asc'
    
    // New tournament form
    let newTournamentName = '';
    let newTournamentDate = '';
    let newTournamentLocation = '';
    let showNewForm = false;
    
    // Edit tournament
    let editingTournament = null;
    let editName = '';
    let editDate = '';
    let editLocation = '';

    // Subscribe to stores
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
            showNewForm = false;
            editingTournament = null;
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
            let sorted = loaded || [];
            
            // Apply sorting
            sorted = sortTournaments(sorted);
            
            tournamentsStore.set(sorted);
        } catch (error) {
            console.error('Error loading tournaments:', error);
            statusBarTextStore.set('Error loading tournaments');
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
            // Deselect
            selectedTournamentStore.set(null);
            tournamentMatchesStore.set([]);
            return;
        }

        selectedTournamentStore.set(tournament);
        try {
            const matches = await GetTournamentMatches(tournament.id);
            tournamentMatchesStore.set(matches || []);
        } catch (error) {
            console.error('Error loading tournament matches:', error);
            statusBarTextStore.set('Error loading tournament matches');
        }
    }

    async function createTournament() {
        if (!newTournamentName.trim()) {
            statusBarTextStore.set('Tournament name is required');
            return;
        }

        try {
            await CreateTournament(newTournamentName.trim(), newTournamentDate, newTournamentLocation.trim());
            await loadTournaments();
            statusBarTextStore.set(`Tournament "${newTournamentName}" created`);
            newTournamentName = '';
            newTournamentDate = '';
            newTournamentLocation = '';
            showNewForm = false;
        } catch (error) {
            console.error('Error creating tournament:', error);
            statusBarTextStore.set('Error creating tournament');
        }
    }

    async function deleteTournamentEntry(tournament, event) {
        event.stopPropagation();
        
        if (!confirm(`Delete tournament "${tournament.name}"? Matches will be unlinked but not deleted.`)) {
            return;
        }

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
            statusBarTextStore.set('Error deleting tournament');
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
        if (!editName.trim()) {
            statusBarTextStore.set('Tournament name is required');
            return;
        }

        try {
            await UpdateTournament(editingTournament.id, editName.trim(), editDate, editLocation.trim());
            await loadTournaments();
            statusBarTextStore.set('Tournament updated');
            editingTournament = null;
        } catch (error) {
            console.error('Error updating tournament:', error);
            statusBarTextStore.set('Error updating tournament');
        }
    }

    function cancelEdit() {
        editingTournament = null;
    }

    async function removeMatchFromTournament(matchId, event) {
        event.stopPropagation();
        
        try {
            await RemoveMatchFromTournament(matchId);
            const matches = await GetTournamentMatches(selectedTournament.id);
            tournamentMatchesStore.set(matches || []);
            await loadTournaments(); // Update match counts
            statusBarTextStore.set('Match removed from tournament');
        } catch (error) {
            console.error('Error removing match from tournament:', error);
            statusBarTextStore.set('Error removing match');
        }
    }

    function formatDate(dateStr) {
        if (!dateStr) return '-';
        return dateStr;
    }

    function closePanel() {
        showTournamentPanelStore.set(false);
    }

    function handleKeyDown(event) {
        if (!visible) return;

        if (event.key === 'Escape') {
            if (editingTournament) {
                cancelEdit();
                event.preventDefault();
                event.stopPropagation();
            } else if (showNewForm) {
                showNewForm = false;
                event.preventDefault();
                event.stopPropagation();
            } else if (selectedTournament) {
                selectedTournamentStore.set(null);
                tournamentMatchesStore.set([]);
                event.preventDefault();
                event.stopPropagation();
            } else {
                closePanel();
            }
            return;
        }

        // Handle j/k and arrow keys for tournament navigation when selected
        if (selectedTournament && !tournamentMatches.length && tournaments.length > 0) {
            const currentIndex = tournaments.findIndex(t => t.id === selectedTournament.id);

            if (event.key === 'j' || event.key === 'ArrowDown') {
                event.preventDefault();
                event.stopPropagation();
                if (currentIndex >= 0 && currentIndex < tournaments.length - 1) {
                    selectTournament(tournaments[currentIndex + 1]);
                }
            } else if (event.key === 'k' || event.key === 'ArrowUp') {
                event.preventDefault();
                event.stopPropagation();
                if (currentIndex > 0) {
                    selectTournament(tournaments[currentIndex - 1]);
                }
            }
        }
    }

    // Focus panel when visible
    $: {
        if (visible) {
            setTimeout(() => {
                const panel = document.getElementById('tournamentPanel');
                if (panel) {
                    panel.focus();
                }
            }, 100);
        }
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
        
        <div class="tournament-panel-header">
            <h3>Tournaments</h3>
            <button class="add-btn" on:click={() => showNewForm = !showNewForm} title="New Tournament">
                {showNewForm ? '−' : '+'}
            </button>
        </div>

        {#if showNewForm}
            <div class="new-tournament-form">
                <input 
                    type="text" 
                    bind:value={newTournamentName} 
                    placeholder="Tournament name *" 
                    class="form-input"
                />
                <input 
                    type="date" 
                    bind:value={newTournamentDate} 
                    class="form-input"
                />
                <input 
                    type="text" 
                    bind:value={newTournamentLocation} 
                    placeholder="Location" 
                    class="form-input"
                />
                <div class="form-buttons">
                    <button class="btn-cancel" on:click={() => showNewForm = false}>Cancel</button>
                    <button class="btn-create" on:click={createTournament}>Create</button>
                </div>
            </div>
        {/if}

        <div class="tournament-panel-content">
            {#if tournaments.length === 0}
                <p class="empty-message">No tournaments yet. Create one to organize your matches.</p>
            {:else}
                <div class="tournament-table-container">
                    <table class="tournament-table">
                        <thead>
                            <tr>
                                <th class="no-select">#</th>
                                <th class="no-select">Name</th>
                                <th class="no-select sortable" on:click={() => toggleSort('date')}>
                                    Date {sortBy === 'date' ? (sortOrder === 'desc' ? '↓' : '↑') : ''}
                                </th>
                                <th class="no-select sortable" on:click={() => toggleSort('location')}>
                                    Location {sortBy === 'location' ? (sortOrder === 'desc' ? '↓' : '↑') : ''}
                                </th>
                                <th class="no-select">Matches</th>
                                <th class="no-select">Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each tournaments as tournament, index}
                                {#if editingTournament && editingTournament.id === tournament.id}
                                    <tr class="editing">
                                        <td class="index-cell no-select">{index + 1}</td>
                                        <td><input type="text" bind:value={editName} class="edit-input" /></td>
                                        <td><input type="date" bind:value={editDate} class="edit-input" /></td>
                                        <td><input type="text" bind:value={editLocation} class="edit-input" /></td>
                                        <td class="no-select">{tournament.matchCount}</td>
                                        <td class="actions-cell">
                                            <button class="action-btn save-btn" on:click={saveEdit} title="Save">✓</button>
                                            <button class="action-btn cancel-btn" on:click={cancelEdit} title="Cancel">✕</button>
                                        </td>
                                    </tr>
                                {:else}
                                    <tr 
                                        class:selected={selectedTournament && selectedTournament.id === tournament.id}
                                        on:click={() => selectTournament(tournament)}
                                    >
                                        <td class="index-cell no-select">{index + 1}</td>
                                        <td class="no-select">{tournament.name}</td>
                                        <td class="no-select">{formatDate(tournament.date)}</td>
                                        <td class="no-select">{tournament.location || '-'}</td>
                                        <td class="no-select">{tournament.matchCount}</td>
                                        <td class="actions-cell">
                                            <button 
                                                class="action-btn edit-btn" 
                                                on:click|stopPropagation={(e) => startEdit(tournament, e)}
                                                title="Edit tournament"
                                            >
                                                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-4">
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="m16.862 4.487 1.687-1.688a1.875 1.875 0 1 1 2.652 2.652L6.832 19.82a4.5 4.5 0 0 1-1.897 1.13l-2.685.8.8-2.685a4.5 4.5 0 0 1 1.13-1.897L16.863 4.487Z" />
                                                </svg>
                                            </button>
                                            <button 
                                                class="action-btn delete-btn" 
                                                on:click|stopPropagation={(e) => deleteTournamentEntry(tournament, e)}
                                                title="Delete tournament"
                                            >
                                                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-4">
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
                                                </svg>
                                            </button>
                                        </td>
                                    </tr>
                                {/if}
                            {/each}
                        </tbody>
                    </table>
                </div>

                {#if selectedTournament && tournamentMatches.length > 0}
                    <div class="matches-section">
                        <h4>Matches in "{selectedTournament.name}"</h4>
                        <div class="matches-list">
                            {#each tournamentMatches as match}
                                <div class="match-item">
                                    <span class="match-players">{match.player1_name} vs {match.player2_name}</span>
                                    <span class="match-length">{match.match_length}pt</span>
                                    <button 
                                        class="remove-btn" 
                                        on:click={(e) => removeMatchFromTournament(match.id, e)}
                                        title="Remove from tournament"
                                    >×</button>
                                </div>
                            {/each}
                        </div>
                    </div>
                {/if}
            {/if}
        </div>
    </section>
{/if}


<style>
    .tournament-panel {
        position: fixed;
        width: 100%;
        bottom: 0;
        left: 0;
        right: 0;
        height: 220px;
        background-color: white;
        border-top: 1px solid rgba(0, 0, 0, 0.1);
        z-index: 5;
        outline: none;
        display: flex;
        flex-direction: column;
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

    .tournament-panel-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 8px 40px 8px 12px;
        border-bottom: 1px solid #eee;
    }

    .tournament-panel-header h3 {
        margin: 0;
        font-size: 14px;
        font-weight: 600;
    }

    .add-btn {
        background: #333;
        color: white;
        border: none;
        width: 24px;
        height: 24px;
        border-radius: 4px;
        cursor: pointer;
        font-size: 18px;
        line-height: 1;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .add-btn:hover {
        background: #555;
    }

    .new-tournament-form {
        display: flex;
        gap: 8px;
        padding: 8px 12px;
        background: #f9f9f9;
        border-bottom: 1px solid #eee;
        align-items: center;
    }

    .form-input {
        padding: 4px 8px;
        border: 1px solid #ccc;
        border-radius: 4px;
        font-size: 12px;
    }

    .form-input:first-child {
        flex: 2;
    }

    .form-buttons {
        display: flex;
        gap: 4px;
    }

    .btn-cancel, .btn-create {
        padding: 4px 12px;
        border: 1px solid #ccc;
        border-radius: 4px;
        font-size: 12px;
        cursor: pointer;
    }

    .btn-cancel {
        background: white;
    }

    .btn-create {
        background: #333;
        color: white;
        border-color: #333;
    }

    .btn-create:hover {
        background: #555;
    }

    .tournament-panel-content {
        flex: 1;
        overflow-y: auto;
        display: flex;
        flex-direction: row;
    }

    .empty-message {
        text-align: center;
        color: #666;
        padding: 20px;
        font-size: 13px;
    }

    .tournament-table-container {
        flex: 2;
        overflow-y: auto;
    }

    .tournament-table {
        width: 100%;
        border-collapse: collapse;
        font-size: 12px;
    }

    .tournament-table thead {
        position: sticky;
        top: 0;
        background: white;
        z-index: 1;
    }

    .tournament-table th {
        padding: 6px 8px;
        text-align: left;
        font-weight: 600;
        border-bottom: 1px solid #ddd;
        color: #666;
    }

    .tournament-table th.sortable {
        cursor: pointer;
    }

    .tournament-table th.sortable:hover {
        color: #333;
    }

    .tournament-table td {
        padding: 6px 8px;
        border-bottom: 1px solid #eee;
    }

    .tournament-table tbody tr {
        cursor: pointer;
    }

    .tournament-table tbody tr:hover {
        background-color: #f5f5f5;
    }

    .tournament-table tbody tr.selected {
        background-color: #e8f0fe;
    }

    .tournament-table tbody tr.editing {
        background-color: #fff3cd;
    }

    .index-cell {
        color: #999;
        width: 30px;
    }

    .actions-cell {
        display: flex;
        gap: 4px;
        white-space: nowrap;
    }

    .action-btn {
        background: none;
        border: 1px solid #ddd;
        border-radius: 4px;
        cursor: pointer;
        padding: 2px 6px;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .action-btn:hover {
        background: #f0f0f0;
    }

    .action-btn svg {
        width: 14px;
        height: 14px;
    }

    .delete-btn:hover {
        background: #fee;
        border-color: #fcc;
    }

    .save-btn {
        color: #2a5;
    }

    .cancel-btn {
        color: #c55;
    }

    .edit-input {
        width: 100%;
        padding: 2px 4px;
        border: 1px solid #ccc;
        border-radius: 2px;
        font-size: 12px;
    }

    .no-select {
        user-select: none;
    }

    .matches-section {
        flex: 1;
        border-left: 1px solid #eee;
        padding: 8px;
        overflow-y: auto;
    }

    .matches-section h4 {
        margin: 0 0 8px 0;
        font-size: 12px;
        font-weight: 600;
        color: #666;
    }

    .matches-list {
        display: flex;
        flex-direction: column;
        gap: 4px;
    }

    .match-item {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 4px 6px;
        background: #f9f9f9;
        border-radius: 4px;
        font-size: 11px;
    }

    .match-players {
        flex: 1;
    }

    .match-length {
        color: #666;
    }

    .remove-btn {
        background: none;
        border: none;
        color: #999;
        cursor: pointer;
        font-size: 14px;
        line-height: 1;
        padding: 0 4px;
    }

    .remove-btn:hover {
        color: #c55;
    }

    .size-4 {
        width: 14px;
        height: 14px;
    }
</style>
