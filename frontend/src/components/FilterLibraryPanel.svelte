<script>
    import { onMount } from 'svelte';
    import { filterLibraryStore } from '../stores/filterLibraryStore';
    import { showFilterLibraryPanelStore, statusBarTextStore, statusBarModeStore, currentPositionIndexStore } from '../stores/uiStore';
    import { SaveFilter, UpdateFilter, DeleteFilter, LoadFilters } from '../../wailsjs/go/main/Database.js';

    let filters = [];
    let filterName = '';
    let filterCommand = '';
    let selectedFilter = null;
    let visible = false;
    let filterExists = false;

    filterLibraryStore.subscribe(value => {
        filters = value;
    });

    showFilterLibraryPanelStore.subscribe(async value => {
        visible = value;
        if (visible) {
            await loadFilters();
            focusNameField(); // Focus on the name field when the panel is opened
            statusBarModeStore.set('EDIT'); // Switch to edit mode when the panel is opened
        } else {
            statusBarModeStore.set('NORMAL'); // Switch to normal mode when the panel is closed
            // Refresh board and display position associated with currentPositionIndexStore
            const currentIndex = $currentPositionIndexStore;
            currentPositionIndexStore.set(-1); // Temporarily set to a different value to force redraw
            currentPositionIndexStore.set(currentIndex); // Set back to the original value
        }
    });

    async function loadFilters() {
        try {
            const loadedFilters = await LoadFilters();
            filterLibraryStore.set(loadedFilters);
            filters = loadedFilters; // Ensure filters are set correctly
        } catch (error) {
            console.error('Error loading filters:', error);
            statusBarTextStore.set('Error loading filters');
        }
    }

    async function saveFilter() {
        if (filterName && filterCommand.startsWith('s ')) {
            const existingFilter = filters.find(filter => filter.name === filterName);
            if (existingFilter) {
                statusBarTextStore.set('Filter name already exists');
                return;
            }
            await SaveFilter(filterName, filterCommand);
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
        if (selectedFilter && filterName && filterCommand.startsWith('s ')) {
            await UpdateFilter(selectedFilter.id, filterName, filterCommand);
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
    }

    function selectFilter(filter) {
        selectedFilter = filter;
        filterName = filter.name;
        filterCommand = filter.command;
    }

    function closePanel() {
        statusBarModeStore.set('NORMAL'); // Switch to normal mode when the panel is closed
        showFilterLibraryPanelStore.set(false);
        
        // Refresh board and display position associated with currentPositionIndexStore
        const currentIndex = $currentPositionIndexStore;
        currentPositionIndexStore.set(-1); // Temporarily set to a different value to force redraw
        currentPositionIndexStore.set(currentIndex); // Set back to the original value
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

    function focusNameField() {
        const nameField = document.getElementById('filterName');
        if (nameField) {
            nameField.focus();
        }
    }
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
                    <button on:click={saveFilter} disabled={filterExists}>Add</button>
                    <button on:click={updateFilter} disabled={!filterExists}>Update</button>
                    <button on:click={deleteFilter} disabled={!selectedFilter}>Delete</button>
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
                            <tr id={`filter-${filter.id}`} class:highlight={filter.name === filterName} on:click={() => selectFilter(filter)}>
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
</style>
