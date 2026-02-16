<script>
    import { onMount, onDestroy } from 'svelte';
    import { filterLibraryStore } from '../stores/filterLibraryStore';
    import { showFilterLibraryPanelStore, statusBarTextStore, statusBarModeStore, currentPositionIndexStore, showCommentStore } from '../stores/uiStore';
    import { databaseLoadedStore } from '../stores/databaseStore';
    import { SaveFilter, UpdateFilter, DeleteFilter, LoadFilters, SaveEditPosition, LoadEditPosition } from '../../wailsjs/go/main/Database.js';
    import { positionStore, positionBeforeFilterLibraryStore, positionIndexBeforeFilterLibraryStore } from '../stores/positionStore';
    import { commandHistoryStore } from '../stores/commandHistoryStore'; // Import command history store
    import { searchHistoryStore } from '../stores/searchHistoryStore'; // Import search history store
    
    export let onLoadPositionsByFilters; // Accept the prop

    let filters = [];
    let filterName = '';
    let filterCommand = '';
    let selectedFilter = null;
    let visible = false;
    let filterExists = false;
    let databaseLoaded = false;
    let editPosition = ''; // Add editPosition variable
    let commandHistory = [];
    let searchHistory = [];
    let historyIndex = -1;

    filterLibraryStore.subscribe(value => {
        filters = value || [];
    });

    databaseLoadedStore.subscribe(value => {
        databaseLoaded = value;
    });

    showFilterLibraryPanelStore.subscribe(async value => {
        const wasVisible = visible;
        visible = value;
        if (visible && !wasVisible) {
            // Panel just opened
            await loadFilters();
        } else if (!visible && wasVisible) {
            // Panel just closed - restore previous position if a filter was selected
            if (selectedFilter) {
                // Restore the position and index that was displayed before selecting any filter
                if ($positionBeforeFilterLibraryStore) {
                    positionStore.set($positionBeforeFilterLibraryStore);
                }
                if ($positionIndexBeforeFilterLibraryStore >= 0) {
                    const savedIndex = $positionIndexBeforeFilterLibraryStore;
                    currentPositionIndexStore.set(-1); // Force redraw
                    currentPositionIndexStore.set(savedIndex);
                }
            }
            // Clear selection and saved position/index
            selectedFilter = null;
            filterName = '';
            filterCommand = '';
            editPosition = '';
            positionBeforeFilterLibraryStore.set(null);
            positionIndexBeforeFilterLibraryStore.set(-1);
        }
    });

    // Update saved position when browsing positions (only if no filter is selected)
    currentPositionIndexStore.subscribe(value => {
        if (visible && !selectedFilter && value >= 0) {
            // Update the saved position as user browses
            positionBeforeFilterLibraryStore.set(JSON.parse(JSON.stringify($positionStore)));
            positionIndexBeforeFilterLibraryStore.set(value);
        }
    });

    commandHistoryStore.subscribe(value => {
        commandHistory = value.filter(command => command.startsWith('s ') || command === 's'); // Filter commands
    });

    searchHistoryStore.subscribe(value => {
        searchHistory = value;
    });

    async function loadFilters() {
        try {
            const loadedFilters = await LoadFilters();
            filterLibraryStore.set(loadedFilters);
            filters = loadedFilters || []; // Ensure filters are set correctly
        } catch (error) {
            console.error('Error loading filters:', error);
            statusBarTextStore.set('Error loading filters');
        }
    }

    async function saveFilter() {
        if (filterName && (filterCommand.startsWith('s ') || filterCommand === 's')) {
            const existingFilter = filters.find(filter => filter.name === filterName);
            if (existingFilter) {
                statusBarTextStore.set('Filter name already exists');
                return;
            }
            editPosition = JSON.stringify($positionStore); // Save positionStore as JSON string
            await SaveFilter(filterName, filterCommand);
            await SaveEditPosition(filterName, editPosition); // Save edit position
            await loadFilters();
            const newFilter = filters.find(filter => filter.name === filterName);
            if (newFilter) {
                scrollToFilter(newFilter);
                highlightFilter(newFilter);
            }
            resetForm();
            statusBarTextStore.set('');
        } else {
            statusBarTextStore.set('Filter command must start with "s "');
        }
    }

    async function updateFilter() {
        if (selectedFilter && filterName && (filterCommand.startsWith('s ') || filterCommand === 's')) {
            editPosition = JSON.stringify($positionStore); // Save positionStore as JSON string
            await UpdateFilter(selectedFilter.id, filterName, filterCommand);
            await SaveEditPosition(filterName, editPosition); // Save edit position
            await loadFilters();
            resetForm();
            statusBarTextStore.set('');
        } else {
            statusBarTextStore.set('Filter command must start with "s "');
        }
    }

    async function deleteFilter() {
        if (selectedFilter) {
            await DeleteFilter(selectedFilter.id);
            await loadFilters();
            resetForm();
        }
    }

    function resetForm() {
        filterName = '';
        filterCommand = '';
        selectedFilter = null;
        editPosition = ''; // Reset edit position
    }

    async function saveLastSearch() {
        if (searchHistory.length === 0) {
            statusBarTextStore.set('No search history available');
            return;
        }
        
        const lastSearch = searchHistory[0]; // Get the most recent search
        filterCommand = lastSearch.command;
        editPosition = lastSearch.position;
        
        if (editPosition) {
            positionStore.set(JSON.parse(editPosition)); // Restore positionStore from JSON string
        }
        
        statusBarTextStore.set('Last search loaded. Enter a name and click Add to save.');
    }

    async function selectFilter(filter) {
        // If clicking on already selected filter, deselect it
        if (selectedFilter && selectedFilter.id === filter.id) {
            selectedFilter = null;
            filterName = '';
            filterCommand = '';
            editPosition = '';
            // Restore the position and index that was displayed before selecting any filter
            if ($positionBeforeFilterLibraryStore) {
                positionStore.set($positionBeforeFilterLibraryStore);
            }
            if ($positionIndexBeforeFilterLibraryStore >= 0) {
                const savedIndex = $positionIndexBeforeFilterLibraryStore;
                currentPositionIndexStore.set(-1); // Force redraw
                currentPositionIndexStore.set(savedIndex);
            }
            return;
        }
        
        // Save the current position BEFORE selecting a new filter (only if not already saved)
        if (!selectedFilter && !$positionBeforeFilterLibraryStore) {
            positionBeforeFilterLibraryStore.set(JSON.parse(JSON.stringify($positionStore)));
            positionIndexBeforeFilterLibraryStore.set($currentPositionIndexStore);
        }
        
        selectedFilter = filter;
        filterName = filter.name;
        filterCommand = filter.command;
        editPosition = await LoadEditPosition(filter.name); // Load edit position
        if (editPosition) {
            positionStore.set(JSON.parse(editPosition)); // Restore positionStore from JSON string
        }
        currentPositionIndexStore.set(-1); // Set current position index to -1
    }

    async function executeFilterCommand(filter) {
        const editPosition = await LoadEditPosition(filter.name); // Load edit position
        if (editPosition) {
            positionStore.set(JSON.parse(editPosition)); // Restore positionStore from JSON string
        }

        const command = filter.command;
        const filters = command.slice(1).trim().split(' ').map(filter => filter.trim());
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
            player1CheckerOffFilter = `${player1CheckerOffFilter},${player1CheckerOffFilter.slice(1)}`; // Handle case where 'ox' means 'ox,x'
        }
        let player2CheckerOffFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('O>') || filter.startsWith('O<') || filter.startsWith('O')));
        if (player2CheckerOffFilter && !player2CheckerOffFilter.includes(',') && !player2CheckerOffFilter.includes('>') && !player2CheckerOffFilter.includes('<')) {
            player2CheckerOffFilter = `${player2CheckerOffFilter},${player2CheckerOffFilter.slice(1)}`; // Handle case where 'Ox' means 'Ox,x'
        }
        let player1BackCheckerFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('k>') || filter.startsWith('k<') || filter.startsWith('k')));
        if (player1BackCheckerFilter && !player1BackCheckerFilter.includes(',') && !player1BackCheckerFilter.includes('>') && !player1BackCheckerFilter.includes('<')) {
            player1BackCheckerFilter = `${player1BackCheckerFilter},${player1BackCheckerFilter.slice(1)}`; // Handle case where 'kx' means 'kx,x'
        }
        let player2BackCheckerFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('K>') || filter.startsWith('K<') || filter.startsWith('K')));
        if (player2BackCheckerFilter && !player2BackCheckerFilter.includes(',') && !player2BackCheckerFilter.includes('>') && !player2BackCheckerFilter.includes('<')) {
            player2BackCheckerFilter = `${player2BackCheckerFilter},${player2BackCheckerFilter.slice(1)}`; // Handle case where 'Kx' means 'Kx,x'
        }
        let player1CheckerInZoneFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('z>') || filter.startsWith('z<') || filter.startsWith('z')));
        if (player1CheckerInZoneFilter && !player1CheckerInZoneFilter.includes(',') && !player1CheckerInZoneFilter.includes('>') && !player1CheckerInZoneFilter.includes('<')) {
            player1CheckerInZoneFilter = `${player1CheckerInZoneFilter},${player1CheckerInZoneFilter.slice(1)}`; // Handle case where 'zx' means 'zx,x'
        }
        let player2CheckerInZoneFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('Z>') || filter.startsWith('Z<') || filter.startsWith('Z')));
        if (player2CheckerInZoneFilter && !player2CheckerInZoneFilter.includes(',') && !player2CheckerInZoneFilter.includes('>') && !player2CheckerInZoneFilter.includes('<')) {
            player2CheckerInZoneFilter = `${player2CheckerInZoneFilter},${player2CheckerInZoneFilter.slice(1)}`; // Handle case where 'Zx' means 'Zx,x'
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

        // Do not close the filter library panel after executing the filter search
    }

    function closePanel() {
        showFilterLibraryPanelStore.set(false);
    }

    function handleKeyDown(event) {
        if (!visible) return;
        // Skip events from input fields to allow normal input field behavior
        if (event.target.matches('input')) {
            event.stopPropagation(); // Prevent global shortcuts from intercepting input
            return;
        }

        // Let Ctrl+key combos pass through to global handler
        if (event.ctrlKey) return;
        
        // Stop all keyboard events from propagating to global handlers
        // This prevents global shortcuts (j, k, space, etc.) from interfering with filter library panel
        event.stopPropagation();
        
        if (event.key === 'Escape') {
            if (selectedFilter) {
                // Deselect if a filter is selected
                selectedFilter = null;
                filterName = '';
                filterCommand = '';
                editPosition = '';
                event.preventDefault();
                event.stopPropagation();
                // Restore the position and index that was displayed before selecting any filter
                if ($positionBeforeFilterLibraryStore) {
                    positionStore.set($positionBeforeFilterLibraryStore);
                }
                if ($positionIndexBeforeFilterLibraryStore >= 0) {
                    const savedIndex = $positionIndexBeforeFilterLibraryStore;
                    currentPositionIndexStore.set(-1); // Force redraw
                    currentPositionIndexStore.set(savedIndex);
                }
            } else {
                closePanel();
            }
            return;
        }

        // Handle j/k and arrow keys for filter navigation ONLY when a filter is selected
        if (selectedFilter && filters.length > 0) {
            const currentIndex = filters.findIndex(f => f.id === selectedFilter.id);

            if (event.key === 'j' || event.key === 'ArrowDown') {
                event.preventDefault();
                event.stopPropagation();
                // Select next filter (down in the list)
                if (currentIndex >= 0 && currentIndex < filters.length - 1) {
                    selectFilter(filters[currentIndex + 1]);
                }
            } else if (event.key === 'k' || event.key === 'ArrowUp') {
                event.preventDefault();
                event.stopPropagation();
                // Select previous filter (up in the list)
                if (currentIndex > 0) {
                    selectFilter(filters[currentIndex - 1]);
                }
            }
        }
    }

    function handleClickOutside(event) {
        const panel = document.getElementById('filterLibraryPanel');
        if (panel && !panel.contains(event.target)) {
            document.activeElement.blur(); // Remove focus from the active element
        }
    }

    function handleCommandKeyDown(event) {
        if (event.code === 'ArrowUp') {
            if (historyIndex < commandHistory.length - 1) {
                historyIndex++;
                filterCommand = commandHistory[historyIndex];
            }
        } else if (event.code === 'ArrowDown') {
            if (historyIndex > 0) {
                historyIndex--;
                filterCommand = commandHistory[historyIndex];
            } else {
                historyIndex = -1;
                filterCommand = '';
            }
        }
    }

    $: filterExists = filters.some(filter => filter.name === filterName);

    $: {
        if (filterExists) {
            const existingFilter = filters.find(filter => filter.name === filterName);
            if (existingFilter) {
                scrollToFilter(existingFilter);
                highlightFilter(existingFilter);
            }
        }
    }

    function scrollToFilter(filter) {
        const filterTable = document.querySelector('.filter-table-container');
        const filterRow = document.getElementById(`filter-${filter.id}`);
        if (filterTable && filterRow) {
            filterTable.scrollTop = filterRow.offsetTop - filterTable.offsetTop;
        }
    }

    function highlightFilter(filter) {
        const filterRow = document.getElementById(`filter-${filter.id}`);
        if (filterRow) {
            filterRow.classList.add('highlight');
        }
    }

    onMount(() => {
        document.addEventListener('click', handleClickOutside);
        document.addEventListener('keydown', handleKeyDown);
    });

    onDestroy(() => {
        document.removeEventListener('click', handleClickOutside);
        document.removeEventListener('keydown', handleKeyDown);
    });
</script>

{#if visible}
    <section class="filter-library-panel" role="dialog" aria-modal="true" id="filterLibraryPanel" tabindex="-1">
        <button type="button" class="close-icon" on:click={closePanel} aria-label="Close">×</button>
        <div class="filter-library-content">
            <div class="form-row">
                <div class="form-group name-group">
                    <input type="text" id="filterName" bind:value={filterName} placeholder=" Name " disabled={$statusBarModeStore !== 'EDIT'} />
                </div>
                <div class="form-group command-group">
                    <input type="text" id="filterCommand" bind:value={filterCommand} placeholder=" Filter Command " disabled={$statusBarModeStore !== 'EDIT'} on:keydown={handleCommandKeyDown} />
                </div>
                <div class="form-actions">
                    <button on:click={saveFilter} disabled={filterExists || $statusBarModeStore !== 'EDIT'}>Add</button>
                    <button on:click={updateFilter} disabled={!filterExists || $statusBarModeStore !== 'EDIT'}>Update</button>
                    <button on:click={deleteFilter} disabled={!selectedFilter}>Delete</button>
                    <button on:click={saveLastSearch} disabled={$statusBarModeStore !== 'EDIT'} title="Load last search">Last Search</button>
                </div>
            </div>
            <div class="filter-table-container">
                <table class="filter-table">
                    <thead>
                        <tr>
                            <th class="no-select">Name</th>
                            <th class="no-select">Filter</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each filters as filter}
                            <tr id={`filter-${filter.id}`} class:highlight={filter.name === filterName} on:click={() => selectFilter(filter)} on:dblclick={() => executeFilterCommand(filter)}>
                                <td class="no-select">{filter.name}</td>
                                <td class="no-select">{filter.command}</td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        </div>
    </section>
{/if}

<style>
    .filter-library-panel {
        position: fixed;
        width: 100%;
        bottom: 0;
        left: 0;
        right: 0;
        height: 178px; /* Set a fixed height */
        background-color: white;
        border-top: 1px solid rgba(0, 0, 0, 0.1);
        padding: 10px;
        box-sizing: border-box;
        z-index: 5;
        outline: none;
        resize: none;
        overflow: hidden; /* Prevent overflow on the panel itself */
    }

    .close-icon {
        position: absolute;
        top: -6px; /* Same top position as analysis panel */
        right: 4px; /* Same right position as analysis panel */
        font-size: 24px;
        font-weight: bold;
        color: #666;
        cursor: pointer;
        background: none;
        border: none;
        padding: 0;
        transition: color 0.3s ease;
        z-index: 10; /* Ensure the close button is on top */
    }

    .close-icon:hover {
        color: #333;
    }

    .filter-library-content {
        font-size: 12px; /* Reduce font size */
        color: black; /* Set text color */
    }

    .form-row {
        display: flex;
        align-items: center;
        position: sticky;
        top: 0;
        background-color: white;
        z-index: 5;
        padding-bottom: 10px;
        gap: 5px; /* Ensure consistent gap between elements */
    }

    .form-group {
        display: flex;
        flex-direction: column;
    }

    .name-group {
        flex: 1;
        max-width: 200px; /* Smaller width for name field */
        margin-right: 5px;
    }

    .command-group {
        flex: 2; /* Larger width for command field */
        margin-right: 5px;
        max-width: 470px; /* Increase width for command field */
    }

    .form-actions {
        display: flex;
        gap: 5px;
        margin-left: auto; /* Align buttons to the right */
        transform: translateX(-20px); /* Move buttons slightly to the left */
    }

    .form-actions button {
        font-size: 12px; /* Match font size with analysis panel */
        padding: 2px 5px; /* Match padding with analysis panel */
        border: 1px solid #ddd; /* Match border with analysis panel */
        background-color: #f2f2f2; /* Match background color with analysis panel */
        cursor: pointer;
        transition: background-color 0.3s ease;
    }

    .form-actions button:hover {
        background-color: #e0e0e0; /* Match hover background color with analysis panel */
    }

    .filter-table-container {
        max-height: 100px; /* Adjust height to ensure the last row and scrollbar are visible */
        overflow-y: auto;
        margin-bottom: 10px; /* Add margin to create a gap between the table and the status bar */
    }

    .filter-table {
        width: 100%;
        border-collapse: collapse;
    }

    .filter-table th, .filter-table td {
        border: 1px solid #ddd;
        padding: 2px; /* Reduce padding */
        text-align: center;
        user-select: none; /* Prevent text selection */
        -webkit-user-select: none; /* Prevent text selection for Safari */
        -moz-user-select: none; /* Prevent text selection for Firefox */
        -ms-user-select: none; /* Prevent text selection for IE */
    }

    .filter-table th {
        background-color: #f2f2f2;
    }

    .filter-table th:nth-child(1) {
        width: 200px; /* Restore previous width for name column */
    }

    .filter-table th:nth-child(2) {
        width: 470px; /* Restore previous width for filter column */
    }

    .filter-table tr {
        cursor: pointer;
    }

    .filter-table tr:hover {
        background-color: #e6f2ff; /* Light blue hover effect */
    }

    .highlight {
        background-color: #b3d9ff !important; /* Light blue highlight for selected row */
    }

    .no-select {
        user-select: none; /* Prevent text selection */
        -webkit-user-select: none; /* Prevent text selection for Safari */
        -moz-user-select: none; /* Prevent text selection for Firefox */
        -ms-user-select: none; /* Prevent text selection for IE */
    }

    .status-bar {
        margin-top: 10px;
        color: red;
    }

    .error {
        color: red;
    }

    input::placeholder {
        color: rgba(0, 0, 0, 0.5); /* Subtle placeholder color */
    }

    input {
        font-size: 12px; /* Match font size with analysis panel */
        padding: 2px 5px; /* Match padding with analysis panel */
        border: 1px solid #ddd; /* Match border with analysis panel */
        background-color: #f2f2f2; /* Match background color with analysis panel */
    }
</style>
