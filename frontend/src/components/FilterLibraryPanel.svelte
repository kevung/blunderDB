<script>
    import { onMount } from 'svelte';
    import { filterLibraryStore } from '../stores/filterLibraryStore';
    import { showFilterLibraryPanelStore } from '../stores/uiStore';
    import { SaveFilter, UpdateFilter, DeleteFilter, LoadFilters } from '../../wailsjs/go/main/Database.js';

    let filters = [];
    let filterName = '';
    let filterCommand = '';
    let selectedFilter = null;
    let visible = false;
    let errorMessage = '';

    filterLibraryStore.subscribe(value => {
        filters = value;
    });

    showFilterLibraryPanelStore.subscribe(value => {
        visible = value;
    });

    async function loadFilters() {
        const loadedFilters = await LoadFilters();
        filterLibraryStore.set(loadedFilters);
    }

    async function saveFilter() {
        if (filterName && filterCommand.startsWith('s ')) {
            await SaveFilter(filterName, filterCommand);
            await loadFilters();
            resetForm();
            errorMessage = '';
        } else {
            errorMessage = 'Filter command must start with "s "';
        }
    }

    async function updateFilter() {
        if (selectedFilter && filterName && filterCommand.startsWith('s ')) {
            await UpdateFilter(selectedFilter.id, filterName, filterCommand);
            await loadFilters();
            resetForm();
            errorMessage = '';
        } else {
            errorMessage = 'Filter command must start with "s "';
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
        showFilterLibraryPanelStore.set(false);
    }

    onMount(() => {
        loadFilters();
    });
</script>

{#if visible}
    <section class="filter-library-panel" role="dialog" aria-modal="true" id="filterLibraryPanel">
        <button type="button" class="close-icon" on:click={closePanel} aria-label="Close">×</button>
        <div class="filter-library-content">
            <h2>Filter Library</h2>
            <div class="form-group">
                <label for="filterName">Filter Name</label>
                <input type="text" id="filterName" bind:value={filterName} />
            </div>
            <div class="form-group">
                <label for="filterCommand">Filter Command</label>
                <input type="text" id="filterCommand" bind:value={filterCommand} />
            </div>
            <div class="form-actions">
                <button on:click={saveFilter}>Save</button>
                <button on:click={updateFilter} disabled={!selectedFilter}>Update</button>
                <button on:click={deleteFilter} disabled={!selectedFilter}>Delete</button>
                <button on:click={resetForm}>Reset</button>
            </div>
            <div class="status-bar">
                {#if errorMessage}
                    <p class="error">{errorMessage}</p>
                {/if}
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
                            <tr on:click={() => selectFilter(filter)}>
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
        overflow-y: auto;
        background-color: white;
        border-top: 1px solid rgba(0, 0, 0, 0.1);
        padding: 10px;
        box-sizing: border-box;
        z-index: 5;
        outline: none;
        resize: none;
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
    }

    .close-icon:hover {
        color: #333;
    }

    .filter-library-content {
        font-size: 12px; /* Reduce font size */
        color: black; /* Set text color */
    }

    .form-group {
        margin-bottom: 10px;
    }

    .form-actions {
        display: flex;
        justify-content: space-between;
    }

    .filter-table-container {
        max-height: 200px;
        overflow-y: auto;
        margin-top: 10px;
    }

    .filter-table {
        width: 100%;
        border-collapse: collapse;
    }

    .filter-table th, .filter-table td {
        border: 1px solid #ccc;
        padding: 8px;
        text-align: left;
    }

    .filter-table tr {
        cursor: pointer;
    }

    .filter-table tr:hover {
        background: #f0f0f0;
    }

    .status-bar {
        margin-top: 10px;
        color: red;
    }

    .error {
        color: red;
    }
</style>
