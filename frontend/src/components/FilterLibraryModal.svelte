<script>
    import { onMount } from 'svelte';
    import { filterLibraryStore } from '../stores/filterLibraryStore';
    import { showFilterLibraryModalStore } from '../stores/uiStore';
    import { SaveFilter, UpdateFilter, DeleteFilter, LoadFilters } from '../../wailsjs/go/main/Database.js';

    let filters = [];
    let filterName = '';
    let filterCommand = '';
    let selectedFilter = null;
    let visible = false;

    filterLibraryStore.subscribe(value => {
        filters = value;
    });

    showFilterLibraryModalStore.subscribe(value => {
        visible = value;
    });

    async function loadFilters() {
        const loadedFilters = await LoadFilters();
        filterLibraryStore.set(loadedFilters);
    }

    async function saveFilter() {
        if (filterName && filterCommand) {
            await SaveFilter(filterName, filterCommand);
            await loadFilters();
            resetForm();
        }
    }

    async function updateFilter() {
        if (selectedFilter && filterName && filterCommand) {
            await UpdateFilter(selectedFilter.id, filterName, filterCommand);
            await loadFilters();
            resetForm();
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

    function closeModal() {
        showFilterLibraryModalStore.set(false);
    }

    onMount(() => {
        loadFilters();
    });
</script>

<div class="modal {visible ? 'visible' : ''}">
    <div class="modal-content">
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
        <ul class="filter-list">
            {#each filters as filter}
                <li on:click={() => selectFilter(filter)}>{filter.name}</li>
            {/each}
        </ul>
        <button class="close" on:click={closeModal}>Close</button>
    </div>
</div>

<style>
    .modal {
        display: none;
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.5);
        justify-content: center;
        align-items: center;
    }
    .modal.visible {
        display: flex;
    }
    .modal-content {
        background: white;
        padding: 20px;
        border-radius: 5px;
        width: 400px;
    }
    .form-group {
        margin-bottom: 10px;
    }
    .form-actions {
        display: flex;
        justify-content: space-between;
    }
    .filter-list {
        list-style: none;
        padding: 0;
    }
    .filter-list li {
        cursor: pointer;
        padding: 5px;
        border-bottom: 1px solid #ccc;
    }
    .filter-list li:hover {
        background: #f0f0f0;
    }
    .close {
        margin-top: 10px;
    }
</style>
