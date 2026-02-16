<script>
    import { onMount, onDestroy } from 'svelte';
    import { searchHistoryStore } from '../stores/searchHistoryStore';
    import { positionStore, positionBeforeFilterLibraryStore, positionIndexBeforeFilterLibraryStore } from '../stores/positionStore';
    import { statusBarModeStore, statusBarTextStore, showSearchHistoryPanelStore, currentPositionIndexStore } from '../stores/uiStore';
    import { LoadSearchHistory, DeleteSearchHistoryEntry, LoadFilters } from '../../wailsjs/go/main/Database.js';

    export let onLoadPositionsByFilters;
    export let onAddToFilterLibrary; // Function to add search to filter library

    let searchHistory = [];
    let selectedSearch = null;
    let showSaveDialog = false;
    let filterName = '';
    let filterLibrary = []; // Store loaded filters
    let visible = false;

    searchHistoryStore.subscribe(value => {
        searchHistory = value;
    });

    showSearchHistoryPanelStore.subscribe(async value => {
        const wasVisible = visible;
        visible = value;
        if (visible && !wasVisible) {
            // Panel just opened
            await loadHistory();
            await loadFilterLibrary();
        } else if (!visible && wasVisible) {
            // Panel just closed - restore previous position if a search was selected
            if (selectedSearch) {
                // Restore the position and index that was displayed before selecting any search
                if ($positionBeforeFilterLibraryStore) {
                    positionStore.set($positionBeforeFilterLibraryStore);
                }
                if ($positionIndexBeforeFilterLibraryStore >= 0) {
                    const savedIndex = $positionIndexBeforeFilterLibraryStore;
                    currentPositionIndexStore.set(-1); // Force redraw
                    currentPositionIndexStore.set(savedIndex);
                }
            }
            // Clear saved position/index and selection
            positionBeforeFilterLibraryStore.set(null);
            positionIndexBeforeFilterLibraryStore.set(-1);
            selectedSearch = null;
        }
    });

    // Update saved position when browsing positions (only if no search is selected)
    currentPositionIndexStore.subscribe(value => {
        if (visible && !selectedSearch && value >= 0) {
            // Update the saved position as user browses
            positionBeforeFilterLibraryStore.set(JSON.parse(JSON.stringify($positionStore)));
            positionIndexBeforeFilterLibraryStore.set(value);
        }
    });

    async function loadHistory() {
        try {
            const history = await LoadSearchHistory();
            searchHistoryStore.set(history || []);
        } catch (error) {
            console.error('Error loading search history:', error);
        }
    }

    async function loadFilterLibrary() {
        try {
            const filters = await LoadFilters();
            filterLibrary = filters || [];
        } catch (error) {
            console.error('Error loading filter library:', error);
            filterLibrary = [];
        }
    }

    function isInFilterLibrary(search) {
        return filterLibrary.some(filter => filter.command === search.command);
    }

    function selectSearch(search) {
        if (selectedSearch === search) {
            // Deselect - restore position and index that was displayed before selecting any search
            selectedSearch = null;
            if ($positionBeforeFilterLibraryStore) {
                positionStore.set($positionBeforeFilterLibraryStore);
            }
            if ($positionIndexBeforeFilterLibraryStore >= 0) {
                const savedIndex = $positionIndexBeforeFilterLibraryStore;
                currentPositionIndexStore.set(-1); // Force redraw
                currentPositionIndexStore.set(savedIndex);
            }
        } else {
            // Save the current position BEFORE selecting a new search (only if not already saved)
            if (!selectedSearch && !$positionBeforeFilterLibraryStore) {
                positionBeforeFilterLibraryStore.set(JSON.parse(JSON.stringify($positionStore)));
                positionIndexBeforeFilterLibraryStore.set($currentPositionIndexStore);
            }
            
            // Select new search - show its position
            selectedSearch = search;
            if (search.position) {
                positionStore.set(JSON.parse(search.position));
            }
            currentPositionIndexStore.set(-1); // Set current position index to -1
        }
    }

    function executeSearch(search) {
        // Restore the position from the search history
        if (search.position) {
            positionStore.set(JSON.parse(search.position));
        }

        // Parse and execute the search command
        const command = search.command;
        if (command.startsWith('s ') || command === 's') {
            // Parse the command to extract filters
            const filters = command === 's' ? [] : command.slice(2).trim().split(' ').map(filter => filter.trim());
            
            const includeCube = filters.includes('cube') || filters.includes('cu') || filters.includes('c') || filters.includes('cub');
            const includeScore = filters.includes('score') || filters.includes('sco') || filters.includes('sc') || filters.includes('s');
            const noContactFilter = filters.includes('nc');
            const decisionTypeFilter = filters.includes('d');
            const diceRollFilter = filters.includes('D');
            const mirrorPositionFilter = filters.includes('M');
            const pipCountFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('p>') || filter.startsWith('p<') || filter.startsWith('p')));
            const winRateFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('w>') || filter.startsWith('w<') || filter.startsWith('w')));
            const gammonRateFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('g>') || filter.startsWith('g<') || filter.startsWith('g')));
            const backgammonRateFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('b>') || filter.startsWith('b<') || (filter.startsWith('b') && !filter.startsWith('bo'))) && !filter.startsWith('bj'));
            const player2WinRateFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('W>') || filter.startsWith('W<') || filter.startsWith('W')));
            const player2GammonRateFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('G>') || filter.startsWith('G<') || filter.startsWith('G')));
            const player2BackgammonRateFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('B>') || filter.startsWith('B<') || filter.startsWith('B') && !filter.startsWith('BO')) && !filter.startsWith('BJ'));
            
            let player1CheckerOffFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('o>') || filter.startsWith('o<') || filter.startsWith('o')));
            if (player1CheckerOffFilter && !player1CheckerOffFilter.includes(',') && !player1CheckerOffFilter.includes('>') && !player1CheckerOffFilter.includes('<')) {
                player1CheckerOffFilter = `${player1CheckerOffFilter},${player1CheckerOffFilter.slice(1)}`;
            }
            
            let player2CheckerOffFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('O>') || filter.startsWith('O<') || filter.startsWith('O')));
            if (player2CheckerOffFilter && !player2CheckerOffFilter.includes(',') && !player2CheckerOffFilter.includes('>') && !player2CheckerOffFilter.includes('<')) {
                player2CheckerOffFilter = `${player2CheckerOffFilter},${player2CheckerOffFilter.slice(1)}`;
            }
            
            let player1BackCheckerFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('k>') || filter.startsWith('k<') || filter.startsWith('k')));
            if (player1BackCheckerFilter && !player1BackCheckerFilter.includes(',') && !player1BackCheckerFilter.includes('>') && !player1BackCheckerFilter.includes('<')) {
                player1BackCheckerFilter = `${player1BackCheckerFilter},${player1BackCheckerFilter.slice(1)}`;
            }
            
            let player2BackCheckerFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('K>') || filter.startsWith('K<') || filter.startsWith('K')));
            if (player2BackCheckerFilter && !player2BackCheckerFilter.includes(',') && !player2BackCheckerFilter.includes('>') && !player2BackCheckerFilter.includes('<')) {
                player2BackCheckerFilter = `${player2BackCheckerFilter},${player2BackCheckerFilter.slice(1)}`;
            }
            
            let player1CheckerInZoneFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('z>') || filter.startsWith('z<') || filter.startsWith('z')));
            if (player1CheckerInZoneFilter && !player1CheckerInZoneFilter.includes(',') && !player1CheckerInZoneFilter.includes('>') && !player1CheckerInZoneFilter.includes('<')) {
                player1CheckerInZoneFilter = `${player1CheckerInZoneFilter},${player1CheckerInZoneFilter.slice(1)}`;
            }
            
            let player2CheckerInZoneFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('Z>') || filter.startsWith('Z<') || filter.startsWith('Z')));
            if (player2CheckerInZoneFilter && !player2CheckerInZoneFilter.includes(',') && !player2CheckerInZoneFilter.includes('>') && !player2CheckerInZoneFilter.includes('<')) {
                player2CheckerInZoneFilter = `${player2CheckerInZoneFilter},${player2CheckerInZoneFilter.slice(1)}`;
            }
            
            const player1AbsolutePipCountFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('P>') || filter.startsWith('P<') || filter.startsWith('P')));
            const equityFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('e>') || filter.startsWith('e<') || filter.startsWith('e')));
            const dateFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('T>') || filter.startsWith('T<') || filter.startsWith('T')));
            const movePatternMatch = command.match(/m["'][^"']*["']/);
            const movePatternFilter = movePatternMatch ? movePatternMatch[0] : '';
            const searchTextMatch = command.match(/t["'][^"']*["']/);
            const searchText = searchTextMatch ? searchTextMatch[0] : '';
            const player1OutfieldBlotFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('bo>') || filter.startsWith('bo<') || filter.startsWith('bo')));
            const player2OutfieldBlotFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('BO>') || filter.startsWith('BO<') || filter.startsWith('BO')));
            const player1JanBlotFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('bj>') || filter.startsWith('bj<') || filter.startsWith('bj')));
            const player2JanBlotFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('BJ>') || filter.startsWith('BJ<') || filter.startsWith('BJ')));
            const moveErrorFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('E>') || filter.startsWith('E<') || (filter.startsWith('E') && /^E\d/.test(filter))));

            onLoadPositionsByFilters(
                filters,
                includeCube,
                includeScore,
                pipCountFilter,
                winRateFilter,
                gammonRateFilter,
                backgammonRateFilter,
                player2WinRateFilter,
                player2GammonRateFilter,
                player2BackgammonRateFilter,
                player1CheckerOffFilter,
                player2CheckerOffFilter,
                player1BackCheckerFilter,
                player2BackCheckerFilter,
                player1CheckerInZoneFilter,
                player2CheckerInZoneFilter,
                searchText,
                player1AbsolutePipCountFilter,
                equityFilter,
                decisionTypeFilter,
                diceRollFilter,
                movePatternFilter,
                dateFilter,
                player1OutfieldBlotFilter,
                player2OutfieldBlotFilter,
                player1JanBlotFilter,
                player2JanBlotFilter,
                noContactFilter,
                mirrorPositionFilter,
                moveErrorFilter,
                command  // Pass the original search command for session tracking
            );
        }
    }

    function handleDoubleClick(search) {
        executeSearch(search);
        closePanel();
    }

    function showAddToLibraryDialog(search) {
        selectedSearch = search;
        showSaveDialog = true;
        filterName = '';
    }

    function cancelSaveDialog() {
        showSaveDialog = false;
        filterName = '';
        selectedSearch = null;
    }

    async function saveToFilterLibrary() {
        if (!filterName || !selectedSearch) {
            statusBarTextStore.set('Please enter a filter name');
            return;
        }
        
        // Call the parent function to add to filter library
        if (onAddToFilterLibrary) {
            await onAddToFilterLibrary(filterName, selectedSearch.command, selectedSearch.position);
            await loadFilterLibrary(); // Reload filter library to update icon colors
            statusBarTextStore.set('Filter saved to library');
        }
        
        cancelSaveDialog();
    }

    async function deleteSearch(search, event) {
        event.stopPropagation();
        try {
            await DeleteSearchHistoryEntry(search.timestamp);
            await loadHistory();
            statusBarTextStore.set('Search deleted from history');
        } catch (error) {
            console.error('Error deleting search:', error);
            statusBarTextStore.set('Error deleting search');
        }
    }

    function formatTimestamp(timestamp) {
        const date = new Date(timestamp);
        return date.toLocaleString();
    }

    function closePanel() {
        showSearchHistoryPanelStore.set(false);
    }

    function handleKeyDown(event) {
        if (!visible) return;
        
        // Skip events from input fields to allow normal input field behavior
        if (event.target.matches('input, textarea')) {
            event.stopPropagation(); // Prevent global shortcuts from intercepting input
            return;
        }

        // Let Ctrl+key combos pass through to global handler
        if (event.ctrlKey) return;
        
        // Stop all keyboard events from propagating to global handlers
        // This prevents global shortcuts (j, k, space, etc.) from interfering with search history panel
        event.stopPropagation();
        
        if (event.key === 'Escape') {
            if (showSaveDialog) {
                cancelSaveDialog();
            } else if (selectedSearch) {
                // Deselect if a search is selected
                selectSearch(selectedSearch);
                event.preventDefault();
                event.stopPropagation();
            } else {
                closePanel();
            }
            return;
        }

        // Handle j/k and arrow keys for search navigation ONLY when a search is selected
        if (selectedSearch && !showSaveDialog && searchHistory.length > 0) {
            const currentIndex = searchHistory.findIndex(s => s.timestamp === selectedSearch.timestamp);

            if (event.key === 'j' || event.key === 'ArrowDown') {
                event.preventDefault();
                event.stopPropagation();
                // Select next search (older, down in the list)
                if (currentIndex >= 0 && currentIndex < searchHistory.length - 1) {
                    selectSearch(searchHistory[currentIndex + 1]);
                }
            } else if (event.key === 'k' || event.key === 'ArrowUp') {
                event.preventDefault();
                event.stopPropagation();
                // Select previous search (younger, up in the list)
                if (currentIndex > 0) {
                    selectSearch(searchHistory[currentIndex - 1]);
                }
            }
        }
    }

    function handleClickOutside(event) {
        // Don't close if the save dialog is open and click is within save dialog or overlay
        if (showSaveDialog) {
            const saveDialogOverlay = document.querySelector('.save-dialog-overlay');
            if (saveDialogOverlay && saveDialogOverlay.contains(event.target)) {
                return;
            }
        }
        
        const panel = document.getElementById('searchHistoryPanel');
        if (panel && !panel.contains(event.target)) {
            document.activeElement.blur(); // Remove focus from the active element
        }
    }

    // Focus the panel when it becomes visible
    $: {
        if (visible) {
            setTimeout(() => {
                const panel = document.getElementById('searchHistoryPanel');
                if (panel) {
                    panel.focus();
                }
            }, 0);
        }
    }

    onMount(async () => {
        document.addEventListener('click', handleClickOutside);
        document.addEventListener('keydown', handleKeyDown);
    });

    onDestroy(() => {
        document.removeEventListener('click', handleClickOutside);
        document.removeEventListener('keydown', handleKeyDown);
    });
</script>

{#if visible}
    <section class="search-history-panel" role="dialog" aria-modal="true" id="searchHistoryPanel" tabindex="-1">
        <button type="button" class="close-icon" on:click={closePanel} aria-label="Close">×</button>
        <div class="search-history-content">
            {#if searchHistory.length === 0}
                <p class="empty-message">No search history yet. Position searches starting with 's ' will appear here.</p>
            {:else}
                <div class="history-table-container">
                    <table class="history-table">
                        <thead>
                            <tr>
                                <th class="no-select">Date</th>
                                <th class="no-select">Command</th>
                                <th class="no-select">Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each searchHistory as search}
                                <tr 
                                    class:selected={selectedSearch === search}
                                    on:click={() => selectSearch(search)}
                                    on:dblclick={() => handleDoubleClick(search)}
                                >
                                    <td class="date-cell no-select">{formatTimestamp(search.timestamp)}</td>
                                    <td class="command-cell no-select">{search.command}</td>
                                    <td class="actions-cell">
                                        <button 
                                            class="action-btn add-btn"
                                            class:in-library={isInFilterLibrary(search)}
                                            on:click|stopPropagation={() => showAddToLibraryDialog(search)}
                                            title="Add to filter library"
                                        >
                                            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
                                                <path stroke-linecap="round" stroke-linejoin="round" d="M17.593 3.322c1.1.128 1.907 1.077 1.907 2.185V21L12 17.25 4.5 21V5.507c0-1.108.806-2.057 1.907-2.185a48.507 48.507 0 0 1 11.186 0Z" />
                                            </svg>
                                        </button>
                                        <button 
                                            class="action-btn delete-btn" 
                                            on:click|stopPropagation={(e) => deleteSearch(search, e)}
                                            title="Delete from history"
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

    {#if showSaveDialog}
        <div class="save-dialog-overlay" on:click={(e) => {
            // Close save dialog if clicking on overlay background
            if (e.target.classList.contains('save-dialog-overlay')) {
                cancelSaveDialog();
            }
        }}>
            <div class="save-dialog">
                <h3>Save to Filter Library</h3>
                <p class="command-preview">Command: {selectedSearch?.command || ''}</p>
                <div class="form-group">
                    <label for="filterName">Filter Name:</label>
                    <input 
                        type="text" 
                        id="filterName" 
                        bind:value={filterName} 
                        placeholder="Enter filter name"
                        on:keydown={(e) => e.key === 'Enter' && saveToFilterLibrary()}
                    />
                </div>
                <div class="dialog-actions">
                    <button class="btn-primary" on:click|stopPropagation={saveToFilterLibrary}>Save</button>
                    <button class="btn-secondary" on:click|stopPropagation={cancelSaveDialog}>Cancel</button>
                </div>
            </div>
        </div>
    {/if}
{/if}

<style>
    .search-history-panel {
        position: fixed;
        width: 100%;
        bottom: 0;
        left: 0;
        right: 0;
        height: 178px; /* Match FilterLibraryPanel height */
        background-color: white;
        border-top: 1px solid rgba(0, 0, 0, 0.1);
        padding: 10px;
        box-sizing: border-box;
        z-index: 5;
        outline: none;
        resize: none;
        overflow: hidden;
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
        z-index: 10;
    }

    .close-icon:hover {
        color: #333;
    }

    .search-history-content {
        font-size: 12px;
        color: black;
        height: 100%;
        display: flex;
        flex-direction: column;
    }

    .empty-message {
        text-align: center;
        color: #666;
        font-size: 12px;
        padding: 20px;
    }

    .history-table-container {
        flex: 1;
        overflow-y: auto;
        margin-bottom: 30px; /* Increased to prevent overlap with status bar */
    }

    .history-table {
        width: 100%;
        border-collapse: collapse;
    }

    .history-table thead {
        position: sticky;
        top: 0;
        background: #f2f2f2;
        z-index: 1;
    }

    .history-table th {
        padding: 2px;
        text-align: center;
        font-weight: bold;
        border: 1px solid #ddd;
        user-select: none;
        -webkit-user-select: none;
        -moz-user-select: none;
        -ms-user-select: none;
    }

    .history-table td {
        padding: 2px;
        border: 1px solid #ddd;
        text-align: center;
    }

    .history-table tbody tr {
        cursor: pointer;
        transition: background-color 0.2s;
    }

    .history-table tbody tr:hover {
        background-color: #e6f2ff; /* Light blue hover effect */
    }

    .history-table tbody tr.selected {
        background-color: #b3d9ff !important; /* Light blue highlight for selected row */
    }

    .date-cell {
        width: 180px;
        white-space: nowrap;
    }

    .command-cell {
        font-family: monospace;
        font-size: 12px;
    }

    .actions-cell {
        width: 100px;
    }

    .action-btn {
        background: none;
        border: none;
        font-size: 18px;
        cursor: pointer;
        padding: 2px 4px;
        margin: 0 2px;
        transition: transform 0.2s;
        display: inline-flex;
        align-items: center;
        justify-content: center;
    }

    .action-btn svg {
        width: 16px;
        height: 16px;
        stroke: currentColor;
    }

    .action-btn:hover {
        transform: scale(1.1);
    }

    .add-btn {
        color: #666;
    }

    .add-btn:hover {
        color: #333;
    }

    .add-btn.in-library {
        color: #333;
    }

    .add-btn.in-library:hover {
        color: #000;
    }

    .delete-btn {
        color: #666;
    }

    .delete-btn:hover {
        color: #333;
    }

    .no-select {
        user-select: none;
        -webkit-user-select: none;
        -moz-user-select: none;
        -ms-user-select: none;
    }

    .save-dialog-overlay {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.7);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 1001;
    }

    .save-dialog {
        background: white;
        border-radius: 8px;
        padding: 30px;
        width: 90%;
        max-width: 500px;
        box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
    }

    .save-dialog h3 {
        margin-top: 0;
        margin-bottom: 20px;
    }

    .command-preview {
        background: #f5f5f5;
        padding: 10px;
        border-radius: 4px;
        font-family: monospace;
        font-size: 14px;
        margin-bottom: 20px;
        word-break: break-all;
    }

    .form-group {
        margin-bottom: 20px;
    }

    .form-group label {
        display: block;
        margin-bottom: 8px;
        font-weight: bold;
    }

    .form-group input {
        width: 100%;
        padding: 10px;
        border: 1px solid #ddd;
        border-radius: 4px;
        font-size: 16px;
        box-sizing: border-box;
    }

    .dialog-actions {
        display: flex;
        justify-content: flex-end;
        gap: 10px;
    }

    .btn-primary, .btn-secondary {
        padding: 10px 20px;
        border: none;
        border-radius: 4px;
        cursor: pointer;
        font-size: 16px;
        transition: background-color 0.2s;
    }

    .btn-primary {
        background-color: #6c757d;
        color: white;
    }

    .btn-primary:hover {
        background-color: #5a6268;
    }

    .btn-secondary {
        background-color: #ccc;
        color: #333;
    }

    .btn-secondary:hover {
        background-color: #999;
    }
</style>
