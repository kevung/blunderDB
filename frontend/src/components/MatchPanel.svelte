<script>
    import { onMount, onDestroy } from 'svelte';
    import { GetAllMatches, GetGamesByMatch, GetMovesByGame, DeleteMatch } from '../../wailsjs/go/main/Database.js';
    import { positionStore } from '../stores/positionStore';
    import { statusBarModeStore, showMatchPanelStore, statusBarTextStore } from '../stores/uiStore';

    let matches = [];
    let selectedMatch = null;
    let visible = false;

    showMatchPanelStore.subscribe(async value => {
        const wasVisible = visible;
        visible = value;
        if (visible && !wasVisible) {
            // Panel just opened
            await loadMatches();
            selectedMatch = null;
        } else if (!visible && wasVisible) {
            // Panel closed - reset selection
            selectedMatch = null;
        }
    });

    async function loadMatches() {
        try {
            const loadedMatches = await GetAllMatches();
            // Reverse to show most recent first (chronological order with recent at top)
            matches = (loadedMatches || []).reverse();
        } catch (error) {
            console.error('Error loading matches:', error);
            matches = [];
        }
    }

    function selectMatch(match) {
        if (selectedMatch === match) {
            // Deselect if clicking the same match
            selectedMatch = null;
        } else {
            selectedMatch = match;
        }
    }

    async function loadMatchPositions(match) {
        try {
            // Switch to MATCH mode
            statusBarModeStore.set('MATCH');
            
            // Load all games for this match
            const games = await GetGamesByMatch(match.id);
            if (!games || games.length === 0) {
                console.error('No games found for match');
                statusBarTextStore.set('No games found for this match');
                return;
            }

            // Load first game's first position
            const firstGame = games[0];
            const moves = await GetMovesByGame(firstGame.id);
            
            if (!moves || moves.length === 0) {
                console.error('No moves found for game');
                statusBarTextStore.set('No moves found in this game');
                return;
            }

            // Load the first position
            // TODO: Need to implement GetPositionByID in backend
            // For now, this is a placeholder
            console.log('Loading first position of match:', match);
            console.log('First move:', moves[0]);
            statusBarTextStore.set(`Loaded match: ${match.player1_name} vs ${match.player2_name}`);
            
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
        
        if (!confirm(`Delete match between ${match.player1_name} and ${match.player2_name}?`)) {
            return;
        }

        try {
            await DeleteMatch(match.id);
            await loadMatches();
            if (selectedMatch && selectedMatch.id === match.id) {
                selectedMatch = null;
            }
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

        if (event.key === 'Escape') {
            if (selectedMatch) {
                // Deselect if a match is selected
                selectedMatch = null;
                event.preventDefault();
                event.stopPropagation();
            } else {
                closePanel();
            }
            return;
        }

        // Handle j/k and arrow keys for match navigation ONLY when a match is selected
        if (selectedMatch && matches.length > 0) {
            const currentIndex = matches.findIndex(m => m.id === selectedMatch.id);

            if (event.key === 'j' || event.key === 'ArrowDown') {
                event.preventDefault();
                event.stopPropagation();
                // Select next match (down in the list)
                if (currentIndex >= 0 && currentIndex < matches.length - 1) {
                    selectMatch(matches[currentIndex + 1]);
                }
            } else if (event.key === 'k' || event.key === 'ArrowUp') {
                event.preventDefault();
                event.stopPropagation();
                // Select previous match (up in the list)
                if (currentIndex > 0) {
                    selectMatch(matches[currentIndex - 1]);
                }
            } else if (event.key === 'Enter') {
                event.preventDefault();
                event.stopPropagation();
                handleDoubleClick(selectedMatch);
            } else if (event.key === 'Delete') {
                event.preventDefault();
                event.stopPropagation();
                // Create a synthetic event object
                const syntheticEvent = { stopPropagation: () => {} };
                deleteMatchEntry(selectedMatch, syntheticEvent);
            }
        }
    }

    function handleClickOutside(event) {
        const panel = document.getElementById('matchPanel');
        if (panel && !panel.contains(event.target)) {
            document.activeElement.blur();
        }
    }

    // Focus the panel when it becomes visible
    $: {
        if (visible) {
            setTimeout(() => {
                const panel = document.getElementById('matchPanel');
                if (panel) {
                    panel.focus();
                }
            }, 0);
        }
    }

    onMount(async () => {
        if (visible) {
            await loadMatches();
        }
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
        bottom: 0;
        left: 0;
        right: 0;
        height: 178px;
        background-color: white;
        border-top: 1px solid rgba(0, 0, 0, 0.1);
        z-index: 5;
        outline: none;
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
    }

    .empty-message {
        text-align: center;
        color: #666;
        padding: 20px;
        margin: 60px 0 0 0;
    }

    .match-table-container {
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
        width: 80px;
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
</style>
