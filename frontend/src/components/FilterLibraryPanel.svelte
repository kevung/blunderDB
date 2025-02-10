<script>
    import { onMount, onDestroy } from 'svelte';
    import { filterLibraryStore } from '../stores/filterLibraryStore';
    import { showFilterLibraryPanelStore, statusBarTextStore, statusBarModeStore, currentPositionIndexStore, showCommentStore } from '../stores/uiStore';
    import { databaseLoadedStore } from '../stores/databaseStore';
    import { SaveFilter, UpdateFilter, DeleteFilter, LoadFilters, SaveEditPosition, LoadEditPosition } from '../../wailsjs/go/main/Database.js';
    import { positionStore } from '../stores/positionStore';
    
    export let onLoadPositionsByFilters; // Accept the prop

    let filters = [];
    let filterName = '';
    let filterCommand = '';
    let selectedFilter = null;
    let visible = false;
    let filterExists = false;
    let databaseLoaded = false;
    let editPosition = ''; // Add editPosition variable

    filterLibraryStore.subscribe(value => {
        filters = value || [];
    });

    databaseLoadedStore.subscribe(value => {
        databaseLoaded = value;
    });

    showFilterLibraryPanelStore.subscribe(async value => {
        visible = value;
        if (visible) {
            await loadFilters();
        }
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

    async function selectFilter(filter) {
        selectedFilter = filter;
        filterName = filter.name;
        filterCommand = filter.command;
        editPosition = await LoadEditPosition(filter.name); // Load edit position
        if (editPosition) {
            positionStore.set(JSON.parse(editPosition)); // Restore positionStore from JSON string
        }
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
            mirrorPositionFilter
        );

        // Do not close the filter library panel after executing the filter search
    }

    function closePanel() {
        showFilterLibraryPanelStore.set(false);
    }

    function handleClickOutside(event) {
        const panel = document.getElementById('filterLibraryPanel');
        if (panel && !panel.contains(event.target)) {
            document.activeElement.blur(); // Remove focus from the active element
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
    });

    onDestroy(() => {
        document.removeEventListener('click', handleClickOutside);
    });
</script>

{#if visible}
    <section class="filter-library-panel" role="dialog" aria-modal="true" id="filterLibraryPanel">
        <button type="button" class="close-icon" on:click={closePanel} aria-label="Close">×</button>
        <div class="filter-library-content">
            <div class="form-row">
                <div class="form-group name-group">
                    <input type="text" id="filterName" bind:value={filterName} placeholder=" Name " />
                </div>
                <div class="form-group command-group">
                    <input type="text" id="filterCommand" bind:value={filterCommand} placeholder=" Filter Command " />
                </div>
                <div class="form-actions">
                    <button on:click={saveFilter} disabled={filterExists || $statusBarModeStore !== 'EDIT'}>Add</button>
                    <button on:click={updateFilter} disabled={!filterExists || $statusBarModeStore !== 'EDIT'}>Update</button>
                    <button on:click={deleteFilter} disabled={!selectedFilter || $statusBarModeStore !== 'EDIT'}>Delete</button>
                </div>
            </div>
            <div class="filter-table-container">
                <table class="filter-table">
                    <thead>
                        <tr>
                            <th>Name</th>
                            <th>Filter</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each filters as filter}
                            <tr id={`filter-${filter.id}`} class:highlight={filter.name === filterName} on:click={() => selectFilter(filter)} on:dblclick={() => executeFilterCommand(filter)}>
                                <td>{filter.name}</td>
                                <td>{filter.command}</td>
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
        background: #f0f0f0;
    }

    .highlight {
        background-color: #f0f0f0; /* Very light highlight color */
        border: 1px solid #ccc; /* Subtle border */
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
