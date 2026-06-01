<script>
    import { logger } from '../utils/logger.js';
    import { onMount, onDestroy } from 'svelte';
    import { filterLibraryStore } from '../stores/filterLibraryStore';
    import { openPanels, PANEL, closePanel, statusBarTextStore, activeTabStore, currentPositionIndexStore } from '../stores/uiStore';
    import { SaveFilter, UpdateFilter, DeleteFilter, LoadFilters, SaveEditPosition, LoadEditPosition } from '../../wailsjs/go/database/Database.js';
    import { positionStore, positionBeforeFilterLibraryStore, positionIndexBeforeFilterLibraryStore } from '../stores/positionStore';
    import { commandHistoryStore } from '../stores/commandHistoryStore'; // Import command history store
    import { searchHistoryStore } from '../stores/searchHistoryStore'; // Import search history store
    import { t, tMsg } from '../i18n';

    let { onLoadPositionsByFilters } = $props();

    // Read-only mirrors of stores
    let filters = $derived($filterLibraryStore || []);
    let visible = $derived($openPanels.has(PANEL.FILTER_LIBRARY));
    let commandHistory = $derived($commandHistoryStore.filter((command) => command.startsWith('s ') || command === 's'));
    let searchHistory = $derived($searchHistoryStore);

    let filterName = $state('');
    let filterCommand = $state('');
    let selectedFilter = $state(null);
    let editPosition = '';
    let historyIndex = -1;

    // Load filters when panel opens; restore position when it closes
    let _prevVisible = false;
    $effect(() => {
        const v = $openPanels.has(PANEL.FILTER_LIBRARY);
        if (v !== _prevVisible) {
            if (v) {
                loadFilters();
            } else {
                if (selectedFilter) {
                    if ($positionBeforeFilterLibraryStore) {
                        positionStore.set($positionBeforeFilterLibraryStore);
                    }
                    if ($positionIndexBeforeFilterLibraryStore >= 0) {
                        const savedIndex = $positionIndexBeforeFilterLibraryStore;
                        currentPositionIndexStore.set(-1);
                        currentPositionIndexStore.set(savedIndex);
                    }
                }
                selectedFilter = null;
                filterName = '';
                filterCommand = '';
                editPosition = '';
                positionBeforeFilterLibraryStore.set(null);
                positionIndexBeforeFilterLibraryStore.set(-1);
            }
            _prevVisible = v;
        }
    });

    // Update saved position when browsing positions (only if no filter is selected)
    $effect(() => {
        const value = $currentPositionIndexStore;
        if (visible && !selectedFilter && value >= 0) {
            positionBeforeFilterLibraryStore.set(JSON.parse(JSON.stringify($positionStore)));
            positionIndexBeforeFilterLibraryStore.set(value);
        }
    });

    async function loadFilters() {
        try {
            const loadedFilters = await LoadFilters();
            filterLibraryStore.set(loadedFilters);
            filters = loadedFilters || []; // Ensure filters are set correctly
        } catch (error) {
            logger.error('Error loading filters:', error);
            statusBarTextStore.set(tMsg('filterLibrary.errorLoading'));
        }
    }

    async function saveFilter() {
        if (filterName && (filterCommand.startsWith('s ') || filterCommand === 's')) {
            const existingFilter = filters.find((filter) => filter.name === filterName);
            if (existingFilter) {
                statusBarTextStore.set(tMsg('filterLibrary.nameExists'));
                return;
            }
            editPosition = JSON.stringify($positionStore); // Save positionStore as JSON string
            await SaveFilter(filterName, filterCommand);
            await SaveEditPosition(filterName, editPosition); // Save edit position
            await loadFilters();
            const newFilter = filters.find((filter) => filter.name === filterName);
            if (newFilter) {
                scrollToFilter(newFilter);
                highlightFilter(newFilter);
            }
            resetForm();
            statusBarTextStore.set('');
        } else {
            statusBarTextStore.set(tMsg('filterLibrary.mustStartWithS'));
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
            statusBarTextStore.set(tMsg('filterLibrary.mustStartWithS'));
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
            statusBarTextStore.set(tMsg('filterLibrary.noHistoryStatus'));
            return;
        }

        const lastSearch = searchHistory[0]; // Get the most recent search
        filterCommand = lastSearch.command;
        editPosition = lastSearch.position;

        if (editPosition) {
            positionStore.set(JSON.parse(editPosition)); // Restore positionStore from JSON string
        }

        statusBarTextStore.set(tMsg('filterLibrary.lastSearchLoaded'));
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
        const filters = command
            .slice(1)
            .trim()
            .split(' ')
            .map((filter) => filter.trim());
        const includeCube = filters.includes('cube') || filters.includes('cu') || filters.includes('c') || filters.includes('cub');
        const includeScore = filters.includes('score') || filters.includes('sco') || filters.includes('sc') || filters.includes('s');
        const noContactFilter = filters.includes('nc');
        const decisionTypeFilter = filters.includes('d');
        const diceRollFilter = filters.includes('D') || filters.includes('D1');
        const diceRollMode = filters.includes('D1') ? 'first' : 'both';
        const mirrorPositionFilter = filters.includes('M');
        const pipCountFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('p>') || filter.startsWith('p<') || filter.startsWith('p')));
        const winRateFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('w>') || filter.startsWith('w<') || filter.startsWith('w')));
        const gammonRateFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('g>') || filter.startsWith('g<') || filter.startsWith('g')));
        const backgammonRateFilter = filters.find(
            (filter) => typeof filter === 'string' && (filter.startsWith('b>') || filter.startsWith('b<') || (filter.startsWith('b') && !filter.startsWith('bo'))) && !filter.startsWith('bj')
        );
        const player2WinRateFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('W>') || filter.startsWith('W<') || filter.startsWith('W')));
        const player2GammonRateFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('G>') || filter.startsWith('G<') || filter.startsWith('G')));
        const player2BackgammonRateFilter = filters.find(
            (filter) => typeof filter === 'string' && (filter.startsWith('B>') || filter.startsWith('B<') || (filter.startsWith('B') && !filter.startsWith('BO'))) && !filter.startsWith('BJ')
        );
        let player1CheckerOffFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('o>') || filter.startsWith('o<') || filter.startsWith('o')));
        if (player1CheckerOffFilter && !player1CheckerOffFilter.includes(',') && !player1CheckerOffFilter.includes('>') && !player1CheckerOffFilter.includes('<')) {
            player1CheckerOffFilter = `${player1CheckerOffFilter},${player1CheckerOffFilter.slice(1)}`; // Handle case where 'ox' means 'ox,x'
        }
        let player2CheckerOffFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('O>') || filter.startsWith('O<') || filter.startsWith('O')));
        if (player2CheckerOffFilter && !player2CheckerOffFilter.includes(',') && !player2CheckerOffFilter.includes('>') && !player2CheckerOffFilter.includes('<')) {
            player2CheckerOffFilter = `${player2CheckerOffFilter},${player2CheckerOffFilter.slice(1)}`; // Handle case where 'Ox' means 'Ox,x'
        }
        let player1BackCheckerFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('k>') || filter.startsWith('k<') || filter.startsWith('k')));
        if (player1BackCheckerFilter && !player1BackCheckerFilter.includes(',') && !player1BackCheckerFilter.includes('>') && !player1BackCheckerFilter.includes('<')) {
            player1BackCheckerFilter = `${player1BackCheckerFilter},${player1BackCheckerFilter.slice(1)}`; // Handle case where 'kx' means 'kx,x'
        }
        let player2BackCheckerFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('K>') || filter.startsWith('K<') || filter.startsWith('K')));
        if (player2BackCheckerFilter && !player2BackCheckerFilter.includes(',') && !player2BackCheckerFilter.includes('>') && !player2BackCheckerFilter.includes('<')) {
            player2BackCheckerFilter = `${player2BackCheckerFilter},${player2BackCheckerFilter.slice(1)}`; // Handle case where 'Kx' means 'Kx,x'
        }
        let player1CheckerInZoneFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('z>') || filter.startsWith('z<') || filter.startsWith('z')));
        if (player1CheckerInZoneFilter && !player1CheckerInZoneFilter.includes(',') && !player1CheckerInZoneFilter.includes('>') && !player1CheckerInZoneFilter.includes('<')) {
            player1CheckerInZoneFilter = `${player1CheckerInZoneFilter},${player1CheckerInZoneFilter.slice(1)}`; // Handle case where 'zx' means 'zx,x'
        }
        let player2CheckerInZoneFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('Z>') || filter.startsWith('Z<') || filter.startsWith('Z')));
        if (player2CheckerInZoneFilter && !player2CheckerInZoneFilter.includes(',') && !player2CheckerInZoneFilter.includes('>') && !player2CheckerInZoneFilter.includes('<')) {
            player2CheckerInZoneFilter = `${player2CheckerInZoneFilter},${player2CheckerInZoneFilter.slice(1)}`; // Handle case where 'Zx' means 'Zx,x'
        }
        const player1AbsolutePipCountFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('P>') || filter.startsWith('P<') || filter.startsWith('P')));
        const equityFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('e>') || filter.startsWith('e<') || filter.startsWith('e')));
        const dateFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('T>') || filter.startsWith('T<') || filter.startsWith('T')));
        const movePatternMatch = command.match(/m["'][^"']*["']/);
        const movePatternFilter = movePatternMatch ? movePatternMatch[0] : '';
        const searchTextMatch = command.match(/t["'][^"']*["']/);
        const searchText = searchTextMatch ? searchTextMatch[0] : '';
        const player1OutfieldBlotFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('bo>') || filter.startsWith('bo<') || filter.startsWith('bo')));
        const player2OutfieldBlotFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('BO>') || filter.startsWith('BO<') || filter.startsWith('BO')));
        const player1JanBlotFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('bj>') || filter.startsWith('bj<') || filter.startsWith('bj')));
        const player2JanBlotFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('BJ>') || filter.startsWith('BJ<') || filter.startsWith('BJ')));
        const moveErrorFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('E>') || filter.startsWith('E<') || (filter.startsWith('E') && /^E\d/.test(filter))));

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
            command, // Pass the original search command for session tracking
            '',
            '',
            '',
            false,
            diceRollMode
        );

        // Do not close the filter library panel after executing the filter search
    }

    function closeFilterLibraryPanel() {
        closePanel(PANEL.FILTER_LIBRARY);
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
                closeFilterLibraryPanel();
            }
            return;
        }

        // Handle j/k and arrow keys for filter navigation ONLY when a filter is selected
        if (selectedFilter && filters.length > 0) {
            const currentIndex = filters.findIndex((f) => f.id === selectedFilter.id);

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

    let filterExists = $derived(filters.some((filter) => filter.name === filterName));
    $effect(() => {
        if (filterExists) {
            const existingFilter = filters.find((filter) => filter.name === filterName);
            if (existingFilter) {
                scrollToFilter(existingFilter);
                highlightFilter(existingFilter);
            }
        }
    });

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

<section class="filter-library-panel" role="dialog" aria-modal="true" aria-label={$t('filterLibrary.title')} id="filterLibraryPanel" tabindex="-1">
    <div class="filter-library-content">
        <div class="form-row">
            <div class="form-group name-group">
                <input type="text" id="filterName" bind:value={filterName} placeholder={$t('filterLibrary.namePlaceholder')} disabled={$activeTabStore !== 'search'} />
            </div>
            <div class="form-group command-group">
                <input
                    type="text"
                    id="filterCommand"
                    bind:value={filterCommand}
                    placeholder={$t('filterLibrary.commandPlaceholder')}
                    disabled={$activeTabStore !== 'search'}
                    onkeydown={handleCommandKeyDown}
                />
            </div>
            <div class="form-actions">
                <button onclick={saveFilter} disabled={filterExists || $activeTabStore !== 'search'}>{$t('filterLibrary.add')}</button>
                <button onclick={updateFilter} disabled={!filterExists || $activeTabStore !== 'search'}>{$t('filterLibrary.update')}</button>
                <button onclick={deleteFilter} disabled={!selectedFilter}>{$t('common.delete')}</button>
                <button onclick={saveLastSearch} disabled={$activeTabStore !== 'search'} title={$t('filterLibrary.lastSearchTooltip')}>{$t('filterLibrary.lastSearch')}</button>
            </div>
        </div>
        <div class="filter-table-container">
            <table class="filter-table">
                <thead>
                    <tr>
                        <th class="no-select">{$t('filterLibrary.colName')}</th>
                        <th class="no-select">{$t('filterLibrary.colFilter')}</th>
                    </tr>
                </thead>
                <tbody>
                    {#each filters as filter (filter.id)}
                        <tr id={`filter-${filter.id}`} class:highlight={filter.name === filterName} onclick={() => selectFilter(filter)} ondblclick={() => executeFilterCommand(filter)}>
                            <td class="no-select">{filter.name}</td>
                            <td class="no-select">{filter.command}</td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        </div>
    </div>
</section>

<style>
    .filter-library-panel {
        width: 100%;
        height: 100%;
        background-color: white;
        padding: 10px;
        box-sizing: border-box;
        outline: none;
        resize: none;
        overflow: hidden;
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

    .filter-table th,
    .filter-table td {
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
