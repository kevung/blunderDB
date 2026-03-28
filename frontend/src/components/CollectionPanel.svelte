<script>
    import { onMount, onDestroy } from 'svelte';
    import { dragReorder } from '../utils/dragReorder.js';
    import { 
        collectionsStore, 
        selectedCollectionStore, 
        collectionPositionsStore,
        activeCollectionStore
    } from '../stores/collectionStore';
    import { 
        showCollectionPanelStore, 
        statusBarTextStore, 
        statusBarModeStore, 
        currentPositionIndexStore
    } from '../stores/uiStore';
    import { databaseLoadedStore } from '../stores/databaseStore';
    import { positionStore, positionsStore } from '../stores/positionStore';
    import { analysisStore } from '../stores/analysisStore';
    import {
        CreateCollection,
        GetAllCollections,
        DeleteCollection,
        AddPositionToCollection,
        RemovePositionFromCollection,
        GetCollectionPositions,
        ReorderCollectionPositions,
        ReorderCollections,
        UpdateCollection,
        GetPositionCollections,
        GetPositionIndexMap,
        LoadAnalysis
    } from '../../wailsjs/go/main/Database.js';

    export let onOpenCollection;

    let collections = [];
    let selectedCollection = null;
    let collectionPositions = [];
    let activeCollection = null;
    let visible = false;
    let databaseLoaded = false;
    let currentPosition = null;
    let mode = 'NORMAL';
    
    let positionCollectionIds = [];

    // View: 'list' (all collections) or 'detail' (positions in active collection)
    let view = 'list';

    // Collection editing (unified: name + description at the same time)
    let editingCollectionId = null;
    let editingName = '';
    let editingDescription = '';
    let inlineNewName = '';

    // Multi-select for positions
    let selectedPositionIndices = new Set();

    // Position index map (position_id -> 1-based index in DB)
    let positionIndexMap = {};

    // Inline new description
    let inlineNewDescription = '';



    const unsubCollections = collectionsStore.subscribe(value => {
        collections = value || [];
    });

    const unsubSelected = selectedCollectionStore.subscribe(value => {
        selectedCollection = value;
    });

    const unsubPositions = collectionPositionsStore.subscribe(value => {
        collectionPositions = value || [];
    });

    const unsubActive = activeCollectionStore.subscribe(value => {
        activeCollection = value;
        if (value) {
            view = 'detail';
        } else {
            view = 'list';
        }
    });

    const unsubDb = databaseLoadedStore.subscribe(async value => {
        const wasLoaded = databaseLoaded;
        databaseLoaded = value;
        if (value && !wasLoaded) {
            await loadCollections();
            await loadPositionIndexMap();
            if (currentPosition && currentPosition.id) {
                await loadPositionCollections(currentPosition.id);
            }
        }
    });

    const unsubPosition = positionStore.subscribe(value => {
        const prevId = currentPosition ? currentPosition.id : null;
        currentPosition = value;
        const newId = value ? value.id : null;
        if (newId && newId !== prevId) {
            // Reset immediately so checkboxes don't show stale state
            positionCollectionIds = [];
            loadPositionCollections(newId);
        } else if (!newId) {
            positionCollectionIds = [];
        }
    });

    const unsubMode = statusBarModeStore.subscribe(value => {
        mode = value;
        if (value === 'COLLECTION' && activeCollection) {
            view = 'detail';
        }
    });

    // Sync position selection when navigating with j/k in COLLECTION mode
    const unsubCurrentIdx = currentPositionIndexStore.subscribe(value => {
        if (mode === 'COLLECTION' && activeCollection && value >= 0) {
            selectedPositionIndices = new Set([value]);
        }
    });

    const unsubVisible = showCollectionPanelStore.subscribe(async value => {
        visible = value;
    });

    onDestroy(() => {
        unsubCollections();
        unsubSelected();
        unsubPositions();
        unsubActive();
        unsubDb();
        unsubPosition();
        unsubMode();
        unsubCurrentIdx();
        unsubVisible();
    });

    async function loadCollections() {
        try {
            const loaded = await GetAllCollections();
            collectionsStore.set(loaded || []);
        } catch (error) {
            console.error('Error loading collections:', error);
            statusBarTextStore.set('Error loading collections');
        }
    }

    async function loadPositionIndexMap() {
        try {
            positionIndexMap = await GetPositionIndexMap() || {};
        } catch (error) {
            positionIndexMap = {};
        }
    }

    async function loadPositionCollections(positionId) {
        try {
            const colls = await GetPositionCollections(positionId);
            positionCollectionIds = (colls || []).map(c => c.id);
        } catch (error) {
            positionCollectionIds = [];
        }
    }

    function isDuplicateName(name, excludeId = null) {
        const lower = name.trim().toLowerCase();
        return collections.some(c => c.name.toLowerCase() === lower && c.id !== excludeId);
    }

    function formatDate(dateStr) {
        if (!dateStr) return '';
        // Handle multiple date formats: "YYYY-MM-DD HH:MM:SS", "YYYY-MM-DDTHH:MM:SSZ", or Go time.Time string format
        let normalized = dateStr;
        if (!normalized.includes('T')) {
            // SQLite format "YYYY-MM-DD HH:MM:SS" — take only the first 19 chars to avoid timezone suffix
            normalized = normalized.substring(0, 19).replace(' ', 'T') + 'Z';
        }
        const d = new Date(normalized);
        if (isNaN(d.getTime())) return '';
        const pad = n => String(n).padStart(2, '0');
        return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
    }

    async function togglePositionInCollection(collectionId, event) {
        if (event) event.stopPropagation();
        if (!currentPosition || !currentPosition.id || currentPosition.id === 0) {
            statusBarTextStore.set('No position selected');
            return;
        }
        try {
            if (positionCollectionIds.includes(collectionId)) {
                await RemovePositionFromCollection(collectionId, currentPosition.id);
                positionCollectionIds = positionCollectionIds.filter(id => id !== collectionId);
                statusBarTextStore.set('Position removed from collection');
            } else {
                await AddPositionToCollection(collectionId, currentPosition.id);
                positionCollectionIds = [...positionCollectionIds, collectionId];
                statusBarTextStore.set('Position added to collection');
            }
            await loadCollections();
            await loadPositionIndexMap();
            if (activeCollection && activeCollection.id === collectionId) {
                const positions = await GetCollectionPositions(collectionId);
                collectionPositionsStore.set(positions || []);
            }
        } catch (error) {
            console.error('Error toggling position in collection:', error);
            statusBarTextStore.set('Error updating collection');
        }
    }

    async function createCollectionInline() {
        if (!inlineNewName.trim()) return;
        if (isDuplicateName(inlineNewName)) {
            statusBarTextStore.set(`A collection named "${inlineNewName.trim()}" already exists`);
            return;
        }
        try {
            await CreateCollection(inlineNewName.trim(), inlineNewDescription.trim());
            await loadCollections();
            statusBarTextStore.set(`Collection "${inlineNewName.trim()}" created`);
            inlineNewName = '';
            inlineNewDescription = '';
        } catch (error) {
            console.error('Error creating collection:', error);
            statusBarTextStore.set('Error creating collection');
        }
    }

    async function selectCollection(collection) {
        if (mode === 'COLLECTION' && activeCollection) return;
        if (selectedCollection && selectedCollection.id === collection.id) {
            selectedCollectionStore.set(null);
            collectionPositionsStore.set([]);
            return;
        }
        selectedCollectionStore.set(collection);
        try {
            const positions = await GetCollectionPositions(collection.id);
            collectionPositionsStore.set(positions || []);
        } catch (error) {
            console.error('Error loading collection positions:', error);
        }
    }

    async function openCollection(collection) {
        if (editingCollectionId === collection.id) return;
        try {
            const positions = await GetCollectionPositions(collection.id);
            if (!positions || positions.length === 0) {
                statusBarTextStore.set('Collection "' + collection.name + '" is empty');
                return;
            }
            selectedCollectionStore.set(collection);
            collectionPositionsStore.set(positions);
            activeCollectionStore.set(collection);
            selectedPositionIndices = new Set();
            view = 'detail';
            await loadPositionIndexMap();
            if (onOpenCollection) {
                onOpenCollection(collection, positions);
            }
        } catch (error) {
            console.error('Error opening collection:', error);
        }
    }

    function goBackToList() {
        view = 'list';
    }

    async function deleteCollection(collection, event) {
        event.stopPropagation();
        try {
            await DeleteCollection(collection.id);
            if (selectedCollection && selectedCollection.id === collection.id) {
                selectedCollectionStore.set(null);
                collectionPositionsStore.set([]);
            }
            if (activeCollection && activeCollection.id === collection.id) {
                activeCollectionStore.set(null);
                view = 'list';
            }
            await loadCollections();
            if (currentPosition && currentPosition.id) {
                await loadPositionCollections(currentPosition.id);
            }
        } catch (error) {
            console.error('Error deleting collection:', error);
        }
    }

    function startEditing(collection, event) {
        if (event) event.stopPropagation();
        editingCollectionId = collection.id;
        editingName = collection.name;
        editingDescription = collection.description || '';
    }

    async function finishEditing(collection) {
        if (editingCollectionId !== collection.id) return;
        const newName = editingName.trim() || collection.name;
        const newDesc = editingDescription.trim();
        if (newName !== collection.name || newDesc !== (collection.description || '')) {
            if (newName !== collection.name && isDuplicateName(newName, collection.id)) {
                statusBarTextStore.set(`A collection named "${newName}" already exists`);
                editingCollectionId = null;
                return;
            }
            try {
                await UpdateCollection(collection.id, newName, newDesc);
                await loadCollections();
                if (activeCollection && activeCollection.id === collection.id) {
                    activeCollectionStore.set({ ...activeCollection, name: newName, description: newDesc });
                }
            } catch (error) {
                console.error('Error updating collection:', error);
            }
        }
        editingCollectionId = null;
    }

    function handleEditingBlur(collection, event) {
        // Delay to check if focus moved to another input in the same editing row
        setTimeout(() => {
            const active = document.activeElement;
            const row = event.target.closest('tr') || event.target.closest('.desc-bar');
            if (row && row.contains(active)) return; // focus still in same row
            finishEditing(collection);
        }, 50);
    }

    function handleEditingKeyDown(event, collection) {
        if (event.key === 'Enter') {
            event.stopPropagation();
            finishEditing(collection);
        } else if (event.key === 'Escape') {
            event.stopPropagation();
            editingCollectionId = null;
        }
    }

    // Collection reorder
    async function moveCollectionUp(index, event) {
        event.stopPropagation();
        if (index <= 0) return;
        const newOrder = [...collections];
        [newOrder[index - 1], newOrder[index]] = [newOrder[index], newOrder[index - 1]];
        collectionsStore.set(newOrder);
        try {
            await ReorderCollections(newOrder.map(c => c.id));
        } catch (error) {
            console.error('Error reordering collections:', error);
        }
    }

    async function moveCollectionDown(index, event) {
        event.stopPropagation();
        if (index >= collections.length - 1) return;
        const newOrder = [...collections];
        [newOrder[index], newOrder[index + 1]] = [newOrder[index + 1], newOrder[index]];
        collectionsStore.set(newOrder);
        try {
            await ReorderCollections(newOrder.map(c => c.id));
        } catch (error) {
            console.error('Error reordering collections:', error);
        }
    }

    // Select a position and display it
    async function selectAndDisplayPosition(index, event) {
        event.stopPropagation();
        const newSet = new Set(selectedPositionIndices);
        if (event.shiftKey && selectedPositionIndices.size > 0) {
            const sorted = [...selectedPositionIndices].sort((a, b) => a - b);
            const last = sorted[sorted.length - 1];
            const start = Math.min(last, index);
            const end = Math.max(last, index);
            for (let i = start; i <= end; i++) {
                newSet.add(i);
            }
        } else if (event.ctrlKey || event.metaKey) {
            if (newSet.has(index)) {
                newSet.delete(index);
            } else {
                newSet.add(index);
            }
        } else {
            if (newSet.size === 1 && newSet.has(index)) {
                newSet.clear();
            } else {
                newSet.clear();
                newSet.add(index);
            }
        }
        selectedPositionIndices = newSet;

        const position = collectionPositions[index];
        if (position) {
            navigateToPosition(position, index);
        }
    }

    async function removeSelectedFromCollection() {
        if (!activeCollection || selectedPositionIndices.size === 0) return;
        const sorted = [...selectedPositionIndices].sort((a, b) => b - a);
        try {
            for (const idx of sorted) {
                await RemovePositionFromCollection(activeCollection.id, collectionPositions[idx].id);
            }
            selectedPositionIndices = new Set();
            const positions = await GetCollectionPositions(activeCollection.id);
            collectionPositionsStore.set(positions || []);
            await loadCollections();
            if (onOpenCollection && positions && positions.length > 0) {
                onOpenCollection(activeCollection, positions);
            }
        } catch (error) {
            console.error('Error removing positions:', error);
        }
    }

    async function movePositionUp(index, event) {
        event.stopPropagation();
        if (!activeCollection || index <= 0) return;
        const newOrder = [...collectionPositions];
        [newOrder[index - 1], newOrder[index]] = [newOrder[index], newOrder[index - 1]];
        collectionPositionsStore.set(newOrder);
        if (selectedPositionIndices.has(index)) {
            const newSet = new Set(selectedPositionIndices);
            newSet.delete(index);
            newSet.add(index - 1);
            selectedPositionIndices = newSet;
        }
        try {
            await ReorderCollectionPositions(activeCollection.id, newOrder.map(p => p.id));
        } catch (error) {
            console.error('Error reordering positions:', error);
        }
    }

    async function movePositionDown(index, event) {
        event.stopPropagation();
        if (!activeCollection || index >= collectionPositions.length - 1) return;
        const newOrder = [...collectionPositions];
        [newOrder[index], newOrder[index + 1]] = [newOrder[index + 1], newOrder[index]];
        collectionPositionsStore.set(newOrder);
        if (selectedPositionIndices.has(index)) {
            const newSet = new Set(selectedPositionIndices);
            newSet.delete(index);
            newSet.add(index + 1);
            selectedPositionIndices = newSet;
        }
        try {
            await ReorderCollectionPositions(activeCollection.id, newOrder.map(p => p.id));
        } catch (error) {
            console.error('Error reordering positions:', error);
        }
    }

    async function removePositionFromRow(index, event) {
        event.stopPropagation();
        if (!activeCollection) return;
        const positionId = collectionPositions[index].id;
        try {
            await RemovePositionFromCollection(activeCollection.id, positionId);
            selectedPositionIndices.delete(index);
            selectedPositionIndices = new Set([...selectedPositionIndices].map(i => i > index ? i - 1 : i));
            const positions = await GetCollectionPositions(activeCollection.id);
            collectionPositionsStore.set(positions || []);
            await loadCollections();
            if (onOpenCollection && positions && positions.length > 0) {
                onOpenCollection(activeCollection, positions);
            }
        } catch (error) {
            console.error('Error removing position:', error);
        }
    }

    async function navigateToPosition(position, index) {
        positionStore.set(position);
        currentPositionIndexStore.set(index);
        try {
            const analysis = await LoadAnalysis(position.id);
            if (analysis) {
                analysisStore.set(analysis);
            }
        } catch (error) {
            console.error('Error loading analysis:', error);
        }
    }

    // Pointer-based drag reorder for collections
    async function handleCollectionReorder(fromIndex, toIndex) {
        const newOrder = [...collections];
        const [moved] = newOrder.splice(fromIndex, 1);
        newOrder.splice(toIndex, 0, moved);
        collectionsStore.set(newOrder);
        try {
            await ReorderCollections(newOrder.map(c => c.id));
        } catch (error) {
            console.error('Error reordering collections:', error);
        }
    }

    // Pointer-based drag reorder for positions within a collection
    async function handlePositionReorder(fromIndex, toIndex) {
        if (!activeCollection) return;

        if (selectedPositionIndices.size > 1 && selectedPositionIndices.has(fromIndex)) {
            const sorted = [...selectedPositionIndices].sort((a, b) => a - b);
            const items = sorted.map(i => collectionPositions[i]);
            const newOrder = collectionPositions.filter((_, i) => !selectedPositionIndices.has(i));
            let insertAt = toIndex;
            const removedBefore = sorted.filter(i => i < toIndex).length;
            insertAt -= removedBefore;
            if (insertAt < 0) insertAt = 0;
            newOrder.splice(insertAt, 0, ...items);
            collectionPositionsStore.set(newOrder);
            const newSelected = new Set();
            for (let i = 0; i < items.length; i++) {
                newSelected.add(insertAt + i);
            }
            selectedPositionIndices = newSelected;
            try {
                await ReorderCollectionPositions(activeCollection.id, newOrder.map(p => p.id));
            } catch (error) {
                console.error('Error reordering positions:', error);
            }
        } else {
            const newOrder = [...collectionPositions];
            const [moved] = newOrder.splice(fromIndex, 1);
            newOrder.splice(toIndex, 0, moved);
            collectionPositionsStore.set(newOrder);
            if (selectedPositionIndices.has(fromIndex)) {
                selectedPositionIndices = new Set([toIndex]);
            }
            try {
                await ReorderCollectionPositions(activeCollection.id, newOrder.map(p => p.id));
            } catch (error) {
                console.error('Error reordering positions:', error);
            }
        }
    }

    function closePanel() {
        showCollectionPanelStore.set(false);
    }

    function handleKeyDown(event) {
        if (!visible) return;
        if (event.target.matches('input, textarea')) return;

        // Let Ctrl+key combos pass through to global handler
        if (event.ctrlKey) return;

        // Let navigation keys pass through to global handler for position browsing
        const isNavigationKey = (event.key === 'j' || event.key === 'k' || 
                                event.key === 'ArrowLeft' || event.key === 'ArrowRight' ||
                                event.key === 'h' || event.key === 'l' ||
                                event.key === 'PageUp' || event.key === 'PageDown');
        if (isNavigationKey) return;

        // Stop other keyboard events from propagating to global handlers
        event.stopPropagation();

        if (event.key === 'Escape') {
            if (view === 'detail' && mode !== 'COLLECTION') {
                view = 'list';
            } else if (selectedCollection) {
                selectedCollectionStore.set(null);
                collectionPositionsStore.set([]);
            } else {
                closePanel();
            }
            return;
        }

        if (mode === 'COLLECTION' && activeCollection && event.key === 'Delete') {
            if (selectedPositionIndices.size > 0) {
                event.preventDefault();
                removeSelectedFromCollection();
            } else {
                const idx = $currentPositionIndexStore;
                if (idx >= 0 && idx < collectionPositions.length) {
                    event.preventDefault();
                    removeFromCollectionSingle(collectionPositions[idx].id);
                }
            }
        }
    }

    async function removeFromCollectionSingle(positionId) {
        if (!activeCollection) return;
        try {
            await RemovePositionFromCollection(activeCollection.id, positionId);
            const positions = await GetCollectionPositions(activeCollection.id);
            collectionPositionsStore.set(positions || []);
            await loadCollections();
            if (onOpenCollection && positions && positions.length > 0) {
                onOpenCollection(activeCollection, positions);
            }
        } catch (error) {
            console.error('Error removing position:', error);
        }
    }

    onMount(async () => {
        document.addEventListener('keydown', handleKeyDown);
        // Load collections when component mounts (tab activated)
        await loadCollections();
        await loadPositionIndexMap();
        if (mode === 'COLLECTION' && activeCollection && activeCollection.id) {
            view = 'detail';
            try {
                const positions = await GetCollectionPositions(activeCollection.id);
                collectionPositionsStore.set(positions || []);
            } catch (error) {
                console.error('Error reloading active collection positions:', error);
            }
        }
        if (currentPosition && currentPosition.id) {
            await loadPositionCollections(currentPosition.id);
        }
    });

    onDestroy(() => {
        document.removeEventListener('keydown', handleKeyDown);
    });
</script>

    <section class="collection-panel" id="collectionPanel" tabindex="-1">
        {#if view === 'list'}
            <!-- Collections list -->
            <div class="table-wrapper">
                <table class="coll-table">
                    <thead>
                        <tr>
                            <th class="no-select">Name</th>
                            <th class="no-select narrow-col">Pos</th>
                            <th class="no-select">Description</th>
                            <th class="no-select narrow-col">Modified</th>
                            <th class="no-select narrow-col toggle-header" title="Add/remove current position">Pos ✓</th>
                            <th class="no-select actions-col"></th>
                        </tr>
                    </thead>
                    <tbody use:dragReorder={{ onReorder: handleCollectionReorder }}>
                        {#each collections as collection, index}
                            <tr
                                class:selected={selectedCollection?.id === collection.id}
                                class:in-collection={positionCollectionIds.includes(collection.id)}
                                on:click={(e) => togglePositionInCollection(collection.id, e)}
                                on:dblclick={() => openCollection(collection)}
                            >
                                <td class="name-cell">
                                    {#if editingCollectionId === collection.id}
                                        <input
                                            class="inline-edit"
                                            type="text"
                                            bind:value={editingName}
                                            on:blur={(e) => handleEditingBlur(collection, e)}
                                            on:keydown={(e) => handleEditingKeyDown(e, collection)}
                                            on:click|stopPropagation
                                            on:dblclick|stopPropagation
                                            autofocus
                                        />
                                    {:else}
                                        <span title={collection.name}>{collection.name}</span>
                                    {/if}
                                </td>
                                <td class="narrow-col count-cell">{collection.positionCount || 0}</td>
                                <td class="desc-cell">
                                    {#if editingCollectionId === collection.id}
                                        <input
                                            class="inline-edit"
                                            type="text"
                                            bind:value={editingDescription}
                                            on:blur={(e) => handleEditingBlur(collection, e)}
                                            on:keydown={(e) => handleEditingKeyDown(e, collection)}
                                            on:click|stopPropagation
                                            on:dblclick|stopPropagation
                                            placeholder="description…"
                                        />
                                    {:else}
                                        <span class="desc-text" title={collection.description || ''}>{collection.description || ''}</span>
                                    {/if}
                                </td>
                                <td class="narrow-col date-cell">{formatDate(collection.updatedAt)}</td>
                                <td class="narrow-col toggle-cell">
                                    {#if currentPosition && currentPosition.id}
                                        <input type="checkbox" checked={positionCollectionIds.includes(collection.id)} on:click|stopPropagation={(e) => togglePositionInCollection(collection.id, e)} title={positionCollectionIds.includes(collection.id) ? 'Remove position from collection' : 'Add position to collection'} />
                                    {/if}
                                </td>
                                <td class="actions-col">
                                    <span class="item-actions">
                                        <button class="icon-btn" on:click|stopPropagation={(e) => moveCollectionUp(index, e)} disabled={index === 0} title="Move up">▲</button>
                                        <button class="icon-btn" on:click|stopPropagation={(e) => moveCollectionDown(index, e)} disabled={index === collections.length - 1} title="Move down">▼</button>
                                        <button class="icon-btn" on:click|stopPropagation={(e) => startEditing(collection, e)} title="Edit">✎</button>
                                        <button class="icon-btn delete" on:click|stopPropagation={(e) => deleteCollection(collection, e)} title="Delete">×</button>
                                    </span>
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
                <!-- Inline add row below table -->
                <div class="add-row">
                    <input class="add-input" type="text" bind:value={inlineNewName} placeholder="New collection…" on:keydown|stopPropagation={(e) => e.key === 'Enter' && createCollectionInline()} />
                    <input class="add-input desc" type="text" bind:value={inlineNewDescription} placeholder="Description…" on:keydown|stopPropagation={(e) => e.key === 'Enter' && createCollectionInline()} />
                </div>
                {#if collections.length === 0}
                    <div class="empty-msg">No collections</div>
                {/if}
            </div>
        {:else if view === 'detail' && activeCollection}
            <!-- Positions in active collection -->
            <div class="table-wrapper">
                <div class="detail-header">
                    <button class="back-btn" on:click={goBackToList} title="Back to collections">←</button>
                    <span class="detail-title" title={activeCollection.name}>{activeCollection.name}</span>
                    <span class="detail-count">{collectionPositions.length} pos</span>
                    {#if currentPosition && currentPosition.id}
                        <input type="checkbox" checked={positionCollectionIds.includes(activeCollection.id)} on:click={(e) => togglePositionInCollection(activeCollection.id, e)} title={positionCollectionIds.includes(activeCollection.id) ? 'Remove position from collection' : 'Add position to collection'} />
                    {/if}
                </div>
                {#if editingCollectionId === activeCollection.id}
                    <div class="desc-bar">
                        <input
                            class="inline-edit full-width"
                            type="text"
                            bind:value={editingDescription}
                            on:blur={(e) => handleEditingBlur(activeCollection, e)}
                            on:keydown={(e) => handleEditingKeyDown(e, activeCollection)}
                            placeholder="description…"
                            autofocus
                        />
                    </div>
                {:else}
                    <div class="desc-bar clickable" on:click={(e) => startEditing(activeCollection, e)}>
                        <span class="desc-text" title="Click to edit">{activeCollection.description || 'Add a description…'}</span>
                    </div>
                {/if}
                <table class="coll-table">
                    <thead>
                        <tr>
                            <th class="no-select narrow-col">#</th>
                            <th class="no-select narrow-col">ID</th>
                            <th class="no-select actions-col"></th>
                        </tr>
                    </thead>
                    <tbody use:dragReorder={{ onReorder: handlePositionReorder }}>
                        {#each collectionPositions as position, index}
                            <tr
                                class:current={$currentPositionIndexStore === index}
                                class:multi-selected={selectedPositionIndices.has(index)}
                                on:click={(e) => selectAndDisplayPosition(index, e)}
                            >
                                <td class="narrow-col idx-cell">{index + 1}</td>
                                <td class="narrow-col id-cell">{positionIndexMap[position.id] || '?'}</td>
                                <td class="actions-col">
                                    <span class="item-actions">
                                        <button class="icon-btn" on:click|stopPropagation={(e) => movePositionUp(index, e)} disabled={index === 0} title="Move up">▲</button>
                                        <button class="icon-btn" on:click|stopPropagation={(e) => movePositionDown(index, e)} disabled={index === collectionPositions.length - 1} title="Move down">▼</button>
                                        <button class="icon-btn delete" on:click|stopPropagation={(e) => removePositionFromRow(index, e)} title="Remove from collection">×</button>
                                    </span>
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
                {#if collectionPositions.length === 0}
                    <div class="empty-msg">Empty collection</div>
                {/if}
            </div>
        {/if}
    </section>

<style>
    .collection-panel { width: 100%; height: 100%; background: white; box-sizing: border-box; outline: none; overflow: hidden; user-select: none; -webkit-user-select: none; }
    .collection-panel * { user-select: none; -webkit-user-select: none; }
    .collection-panel input, .collection-panel textarea { user-select: text; -webkit-user-select: text; }

    .table-wrapper { height: 100%; display: flex; flex-direction: column; min-height: 0; overflow-y: auto; overflow-x: hidden; }

    /* Table layout (same pattern as Match/Tournament panels) */
    .coll-table { width: 100%; border-collapse: collapse; font-size: 12px; }

    .coll-table thead { position: sticky; top: 0; background-color: #f5f5f5; z-index: 1; }

    .coll-table th, .coll-table td { padding: 4px 8px; text-align: left; border-bottom: 1px solid #e0e0e0; }

    .coll-table th { font-weight: 600; color: #333; font-size: 11px; }

    .narrow-col { width: 1px; white-space: nowrap; padding-left: 6px; padding-right: 6px; }

    .actions-col { width: 90px; min-width: 90px; max-width: 90px; white-space: nowrap; text-align: center; padding: 0 4px; }

    .date-cell { font-size: 10px; color: #999; }

    /* Row styles */
    .coll-table tbody tr { transition: background-color 0.1s; }
    .coll-table tbody tr:hover { background-color: #f9f9f9; }
    .coll-table tbody tr.selected { background-color: #e3f2fd; }
    .coll-table tbody tr.selected:hover { background-color: #bbdefb; }
    .coll-table tbody tr.in-collection { border-left: 3px solid #4a8; }
    .coll-table tbody :global(tr.drag-over) { border-top: 2px solid #999; }
    .coll-table tbody :global(tr.dragging) { opacity: 0.5; }
    .coll-table tbody tr.current { background-color: #dce9f7; }
    .coll-table tbody tr.multi-selected { background-color: #dce9f7; }

    /* Cell styles */
    .name-cell { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 0; }
    .count-cell { text-align: center; color: #666; }
    .desc-cell { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 0; }
    .desc-text { color: #888; font-style: italic; cursor: pointer; font-size: 11px; }
    .desc-text:hover { color: #555; }
    .idx-cell { text-align: right; color: #999; }
    .id-cell { color: #666; }

    .inline-edit { width: 100%; font-size: 12px; padding: 1px 4px; border: 1px solid #999; outline: none; box-sizing: border-box; }

    .toggle-header { text-align: center; color: #4a8; }
    .toggle-cell { text-align: center; }
    .toggle-cell input[type="checkbox"] { cursor: pointer; margin: 0; width: 15px; height: 15px; accent-color: #4a8; }

    /* Action buttons - always visible */
    .item-actions { display: inline-flex; gap: 2px; vertical-align: middle; }

    .icon-btn { background: none; border: none; cursor: pointer; font-size: 12px; color: #666; padding: 2px 4px; line-height: 1; }
    .icon-btn:hover:not(:disabled) { color: #000; }
    .icon-btn:disabled { opacity: 0.3; cursor: not-allowed; }
    .icon-btn.delete:hover:not(:disabled) { color: #c55; }


    /* Add row */
    .add-row { padding: 4px 8px; background: #fafafa; border-top: 1px solid #e0e0e0; flex-shrink: 0; }
    .add-row { display: flex; gap: 4px; }
    .add-input { flex: 1; padding: 3px 6px; border: 1px solid #ccc; border-radius: 3px; font-size: 12px; outline: none; box-sizing: border-box; }
    .add-input.desc { flex: 1; font-size: 11px; color: #666; }
    .add-input:focus { border-color: #999; }

    .empty-msg { text-align: center; color: #bbb; padding: 16px; font-size: 11px; font-style: italic; }

    /* Detail header */
    .detail-header { display: flex; align-items: center; gap: 8px; padding: 5px 8px; background: #f5f5f5; border-bottom: 1px solid #e0e0e0; flex-shrink: 0; }
    .back-btn { background: none; border: none; cursor: pointer; font-size: 16px; color: #666; padding: 2px 6px; line-height: 1; }
    .back-btn:hover { color: #333; }
    .detail-title { font-size: 13px; font-weight: 600; color: #333; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; }
    .detail-count { font-size: 11px; color: #888; flex-shrink: 0; }

    /* Description bar */
    .desc-bar { padding: 3px 8px 3px 32px; border-bottom: 1px solid #eee; flex-shrink: 0; }
    .desc-bar.clickable { cursor: pointer; }
    .desc-bar .desc-text { font-size: 11px; }
    .full-width { width: 100%; }

    .no-select { user-select: none; }
</style>
