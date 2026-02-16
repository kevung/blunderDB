<script>
    import { onMount, onDestroy } from 'svelte';
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
    let newCollectionName = '';

    // Description editing (for COLLECTION mode header)
    let editingDescription = false;
    let descriptionText = '';

    // Description editing in collection list (NORMAL mode)
    let editingDescriptionId = null;
    let editingDescriptionText = '';

    // Renaming
    let renamingCollectionId = null;
    let renamingName = '';

    // Multi-select for positions
    let selectedPositionIndices = new Set();

    // Position index map (position_id -> 1-based index in DB)
    let positionIndexMap = {};

    // Drag state
    let dragType = null;
    let dragStartIndex = -1;
    let dragOverIndex = -1;

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
            descriptionText = value.description || '';
        }
    });

    const unsubDb = databaseLoadedStore.subscribe(value => {
        databaseLoaded = value;
    });

    const unsubPosition = positionStore.subscribe(value => {
        currentPosition = value;
        if (visible && value && value.id) {
            loadPositionCollections(value.id);
        }
    });

    const unsubMode = statusBarModeStore.subscribe(value => {
        mode = value;
    });

    // Sync position selection when navigating with j/k in COLLECTION mode
    const unsubCurrentIdx = currentPositionIndexStore.subscribe(value => {
        if (mode === 'COLLECTION' && activeCollection && value >= 0) {
            selectedPositionIndices = new Set([value]);
        }
    });

    const unsubVisible = showCollectionPanelStore.subscribe(async value => {
        const wasVisible = visible;
        visible = value;
        if (visible && !wasVisible) {
            await loadCollections();
            await loadPositionIndexMap();
            // In COLLECTION mode, reload the active collection's positions
            if (mode === 'COLLECTION' && activeCollection && activeCollection.id) {
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
        } else if (!visible && wasVisible) {
            selectedCollectionStore.set(null);
            collectionPositionsStore.set([]);
            renamingCollectionId = null;
            selectedPositionIndices = new Set();
        }
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

    async function togglePositionInCollection(collectionId) {
        if (!currentPosition || !currentPosition.id || currentPosition.id === 0) {
            statusBarTextStore.set('No position selected');
            return;
        }
        try {
            if (positionCollectionIds.includes(collectionId)) {
                await RemovePositionFromCollection(collectionId, currentPosition.id);
                positionCollectionIds = positionCollectionIds.filter(id => id !== collectionId);
            } else {
                await AddPositionToCollection(collectionId, currentPosition.id);
                positionCollectionIds = [...positionCollectionIds, collectionId];
            }
            await loadCollections();
        } catch (error) {
            console.error('Error toggling position in collection:', error);
            statusBarTextStore.set('Error updating collection');
        }
    }

    async function createAndAddToCollection() {
        if (!newCollectionName.trim()) return;
        if (!currentPosition || !currentPosition.id || currentPosition.id === 0) {
            statusBarTextStore.set('No position selected');
            return;
        }
        try {
            await CreateCollection(newCollectionName.trim(), '');
            await loadCollections();
            const newColl = collections.find(c => c.name === newCollectionName.trim());
            if (newColl) {
                await AddPositionToCollection(newColl.id, currentPosition.id);
                positionCollectionIds = [...positionCollectionIds, newColl.id];
                await loadCollections();
            }
            newCollectionName = '';
        } catch (error) {
            console.error('Error creating collection:', error);
            statusBarTextStore.set('Error creating collection');
        }
    }

    async function selectCollection(collection) {
        // In COLLECTION mode, don't allow deselecting the active collection
        if (mode === 'COLLECTION' && activeCollection) {
            return;
        }
        // Toggle off if clicking the same collection
        if (selectedCollection && selectedCollection.id === collection.id) {
            selectedCollectionStore.set(null);
            collectionPositionsStore.set([]);
            return;
        }
        // Deselect any other collection first
        selectedCollectionStore.set(null);
        collectionPositionsStore.set([]);
        // Now select the new one
        selectedCollectionStore.set(collection);
        try {
            const positions = await GetCollectionPositions(collection.id);
            collectionPositionsStore.set(positions || []);
        } catch (error) {
            console.error('Error loading collection positions:', error);
        }
    }

    async function openCollection(collection) {
        // Don't open if we're in rename mode for this collection
        if (renamingCollectionId === collection.id) return;
        try {
            const positions = await GetCollectionPositions(collection.id);
            if (!positions || positions.length === 0) {
                statusBarTextStore.set('Collection "' + collection.name + '" is empty');
                return;
            }
            selectedCollectionStore.set(collection);
            collectionPositionsStore.set(positions);
            activeCollectionStore.set(collection);
            descriptionText = collection.description || '';
            selectedPositionIndices = new Set();
            await loadPositionIndexMap();
            if (onOpenCollection) {
                onOpenCollection(collection, positions);
            }
        } catch (error) {
            console.error('Error opening collection:', error);
        }
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
            }
            await loadCollections();
            if (currentPosition && currentPosition.id) {
                await loadPositionCollections(currentPosition.id);
            }
        } catch (error) {
            console.error('Error deleting collection:', error);
        }
    }

    function startRename(collection, event) {
        event.stopPropagation();
        event.preventDefault();
        renamingCollectionId = collection.id;
        renamingName = collection.name;
    }

    async function finishRename(collection) {
        if (renamingCollectionId !== collection.id) return;
        if (renamingName.trim() && renamingName.trim() !== collection.name) {
            try {
                await UpdateCollection(collection.id, renamingName.trim(), collection.description || '');
                await loadCollections();
                if (activeCollection && activeCollection.id === collection.id) {
                    activeCollectionStore.set({ ...activeCollection, name: renamingName.trim() });
                }
            } catch (error) {
                console.error('Error renaming collection:', error);
            }
        }
        renamingCollectionId = null;
    }

    function handleRenameKeyDown(event, collection) {
        event.stopPropagation();
        if (event.key === 'Enter') {
            finishRename(collection);
        } else if (event.key === 'Escape') {
            renamingCollectionId = null;
        }
    }

    async function saveDescription() {
        if (!activeCollection) return;
        try {
            await UpdateCollection(activeCollection.id, activeCollection.name, descriptionText);
            activeCollectionStore.set({ ...activeCollection, description: descriptionText });
            editingDescription = false;
            await loadCollections();
        } catch (error) {
            console.error('Error saving description:', error);
        }
    }

    function handleDescriptionKeyDown(event) {
        event.stopPropagation();
        if (event.key === 'Enter') {
            saveDescription();
        } else if (event.key === 'Escape') {
            descriptionText = activeCollection?.description || '';
            editingDescription = false;
        }
    }

    // Description editing for collections in the list (NORMAL mode)
    function startEditDescription(collection, event) {
        event.stopPropagation();
        editingDescriptionId = collection.id;
        editingDescriptionText = collection.description || '';
    }

    async function saveListDescription(collection) {
        if (editingDescriptionId !== collection.id) return;
        if (editingDescriptionText.trim() !== (collection.description || '')) {
            try {
                await UpdateCollection(collection.id, collection.name, editingDescriptionText.trim());
                await loadCollections();
                if (activeCollection && activeCollection.id === collection.id) {
                    activeCollectionStore.set({ ...activeCollection, description: editingDescriptionText.trim() });
                    descriptionText = editingDescriptionText.trim();
                }
            } catch (error) {
                console.error('Error saving description:', error);
            }
        }
        editingDescriptionId = null;
    }

    function handleListDescriptionKeyDown(event, collection) {
        event.stopPropagation();
        if (event.key === 'Enter') {
            saveListDescription(collection);
        } else if (event.key === 'Escape') {
            editingDescriptionId = null;
        }
    }

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

        // Always navigate to the clicked position to display it
        const position = collectionPositions[index];
        if (position) {
            navigateToPosition(position, index);
        }
    }

    async function moveSelectedUp() {
        if (!activeCollection || selectedPositionIndices.size === 0) return;
        const sorted = [...selectedPositionIndices].sort((a, b) => a - b);
        if (sorted[0] <= 0) return;
        const newOrder = [...collectionPositions];
        for (const idx of sorted) {
            [newOrder[idx - 1], newOrder[idx]] = [newOrder[idx], newOrder[idx - 1]];
        }
        collectionPositionsStore.set(newOrder);
        selectedPositionIndices = new Set(sorted.map(i => i - 1));
        try {
            await ReorderCollectionPositions(activeCollection.id, newOrder.map(p => p.id));
        } catch (error) {
            console.error('Error reordering positions:', error);
        }
    }

    async function moveSelectedDown() {
        if (!activeCollection || selectedPositionIndices.size === 0) return;
        const sorted = [...selectedPositionIndices].sort((a, b) => b - a);
        if (sorted[0] >= collectionPositions.length - 1) return;
        const newOrder = [...collectionPositions];
        for (const idx of sorted) {
            [newOrder[idx], newOrder[idx + 1]] = [newOrder[idx + 1], newOrder[idx]];
        }
        collectionPositionsStore.set(newOrder);
        selectedPositionIndices = new Set([...selectedPositionIndices].map(i => i + 1));
        try {
            await ReorderCollectionPositions(activeCollection.id, newOrder.map(p => p.id));
        } catch (error) {
            console.error('Error reordering positions:', error);
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

    // Move a single position up (from row button)
    async function movePositionUp(index, event) {
        event.stopPropagation();
        if (!activeCollection || index <= 0) return;
        const newOrder = [...collectionPositions];
        [newOrder[index - 1], newOrder[index]] = [newOrder[index], newOrder[index - 1]];
        collectionPositionsStore.set(newOrder);
        // Update selection if this position was selected
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

    // Move a single position down (from row button)
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

    // Remove single position from row button
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

    // Drag & Drop collections
    function onCollectionDragStart(event, index) {
        dragType = 'collection';
        dragStartIndex = index;
        dragOverIndex = index;
        event.dataTransfer.effectAllowed = 'move';
        event.dataTransfer.setData('text/plain', String(index));
    }

    function onCollectionDragOver(event, index) {
        if (dragType !== 'collection') return;
        event.preventDefault();
        event.dataTransfer.dropEffect = 'move';
        dragOverIndex = index;
    }

    async function onCollectionDrop(event, index) {
        event.preventDefault();
        if (dragType !== 'collection' || dragStartIndex === index) {
            resetDrag();
            return;
        }
        const newOrder = [...collections];
        const [moved] = newOrder.splice(dragStartIndex, 1);
        newOrder.splice(index, 0, moved);
        collectionsStore.set(newOrder);
        resetDrag();
        try {
            await ReorderCollections(newOrder.map(c => c.id));
        } catch (error) {
            console.error('Error reordering collections:', error);
        }
    }

    // Drag & Drop positions
    function onPositionDragStart(event, index) {
        dragType = 'position';
        dragStartIndex = index;
        dragOverIndex = index;
        event.dataTransfer.effectAllowed = 'move';
        event.dataTransfer.setData('text/plain', String(index));
    }

    function onPositionDragOver(event, index) {
        if (dragType !== 'position') return;
        event.preventDefault();
        event.dataTransfer.dropEffect = 'move';
        dragOverIndex = index;
    }

    async function onPositionDrop(event, index) {
        event.preventDefault();
        if (dragType !== 'position' || !activeCollection || dragStartIndex === index) {
            resetDrag();
            return;
        }

        if (selectedPositionIndices.size > 1 && selectedPositionIndices.has(dragStartIndex)) {
            const sorted = [...selectedPositionIndices].sort((a, b) => a - b);
            const items = sorted.map(i => collectionPositions[i]);
            const newOrder = collectionPositions.filter((_, i) => !selectedPositionIndices.has(i));
            let insertAt = index;
            const removedBefore = sorted.filter(i => i < index).length;
            insertAt -= removedBefore;
            if (insertAt < 0) insertAt = 0;
            newOrder.splice(insertAt, 0, ...items);
            collectionPositionsStore.set(newOrder);
            const newSelected = new Set();
            for (let i = 0; i < items.length; i++) {
                newSelected.add(insertAt + i);
            }
            selectedPositionIndices = newSelected;
            resetDrag();
            try {
                await ReorderCollectionPositions(activeCollection.id, newOrder.map(p => p.id));
            } catch (error) {
                console.error('Error reordering positions:', error);
            }
        } else {
            const newOrder = [...collectionPositions];
            const [moved] = newOrder.splice(dragStartIndex, 1);
            newOrder.splice(index, 0, moved);
            collectionPositionsStore.set(newOrder);
            if (selectedPositionIndices.has(dragStartIndex)) {
                selectedPositionIndices = new Set([index]);
            }
            resetDrag();
            try {
                await ReorderCollectionPositions(activeCollection.id, newOrder.map(p => p.id));
            } catch (error) {
                console.error('Error reordering positions:', error);
            }
        }
    }

    function resetDrag() {
        dragType = null;
        dragStartIndex = -1;
        dragOverIndex = -1;
    }

    function onDragEnd() {
        resetDrag();
    }

    function formatDate(dateStr) {
        if (!dateStr) return '';
        try {
            const d = new Date(dateStr);
            return d.toLocaleDateString('fr-FR', { day: '2-digit', month: '2-digit', year: '2-digit' });
        } catch {
            return '';
        }
    }

    function closePanel() {
        showCollectionPanelStore.set(false);
    }

    function handleKeyDown(event) {
        if (!visible) return;
        if (event.target.matches('input, textarea')) return;

        if (event.key === 'Escape') {
            if (selectedCollection) {
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

    onMount(() => {
        document.addEventListener('keydown', handleKeyDown);
    });

    onDestroy(() => {
        document.removeEventListener('keydown', handleKeyDown);
    });
</script>

{#if visible}
    <section class="collection-panel" role="dialog" aria-modal="true" id="collectionPanel" tabindex="-1">
        <button type="button" class="close-icon" on:click={closePanel} aria-label="Close">×</button>
        
        <div class="collection-content">
            <div class="panels-container">
                <!-- Left: Collections list -->
                <div class="collections-list">
                    <div class="panel-header">Collections ({collections.length})</div>
                    <div class="col-header-row">
                        <span class="col-h col-name">Name</span>
                        <span class="col-h col-count">#</span>
                        <span class="col-h col-desc">Description</span>
                        <span class="col-h col-acts"></span>
                    </div>
                    <div class="list-container">
                        {#each collections as collection, index}
                            <div 
                                class="collection-item" 
                                class:selected={selectedCollection?.id === collection.id}
                                class:active={activeCollection?.id === collection.id}
                                class:drag-over={dragType === 'collection' && dragOverIndex === index && dragStartIndex !== index}
                                on:click={() => selectCollection(collection)}
                                on:dblclick={() => openCollection(collection)}
                                draggable="true"
                                on:dragstart={(e) => onCollectionDragStart(e, index)}
                                on:dragover={(e) => onCollectionDragOver(e, index)}
                                on:drop={(e) => onCollectionDrop(e, index)}
                                on:dragend={onDragEnd}
                            >
                                {#if renamingCollectionId === collection.id}
                                    <span class="col-cell col-name">
                                        <input
                                            class="rename-input"
                                            type="text"
                                            bind:value={renamingName}
                                            on:blur={() => finishRename(collection)}
                                            on:keydown={(e) => handleRenameKeyDown(e, collection)}
                                            on:click|stopPropagation
                                            on:dblclick|stopPropagation
                                            autofocus
                                        />
                                    </span>
                                {:else}
                                    <span class="col-cell col-name" title={collection.name}>{collection.name}</span>
                                {/if}
                                <span class="col-cell col-count">{collection.positionCount}</span>
                                <span class="col-cell col-desc">
                                    {#if editingDescriptionId === collection.id}
                                        <input
                                            class="desc-input"
                                            type="text"
                                            bind:value={editingDescriptionText}
                                            on:blur={() => saveListDescription(collection)}
                                            on:keydown={(e) => handleListDescriptionKeyDown(e, collection)}
                                            on:click|stopPropagation
                                            on:dblclick|stopPropagation
                                            placeholder="Add description..."
                                            autofocus
                                        />
                                    {:else}
                                        <span 
                                            class="desc-text"
                                            on:click|stopPropagation={(e) => startEditDescription(collection, e)}
                                            title={collection.description || 'Click to edit'}
                                        >{collection.description || '—'}</span>
                                    {/if}
                                </span>
                                <span class="col-cell col-acts">
                                    <div class="collection-actions">
                                        <button class="icon-btn" on:click|stopPropagation={(e) => startRename(collection, e)} title="Rename">✎</button>
                                        <button class="icon-btn" on:click|stopPropagation={(e) => moveCollectionUp(index, e)} disabled={index === 0} title="Move up">▲</button>
                                        <button class="icon-btn" on:click|stopPropagation={(e) => moveCollectionDown(index, e)} disabled={index === collections.length - 1} title="Move down">▼</button>
                                        <button class="icon-btn delete" on:click|stopPropagation={(e) => deleteCollection(collection, e)} title="Delete">×</button>
                                    </div>
                                </span>
                            </div>
                        {/each}
                        {#if collections.length === 0}
                            <div class="empty-msg">No collections</div>
                        {/if}
                    </div>
                </div>

                <!-- Right panel -->
                <div class="right-panel">
                    {#if mode === 'COLLECTION' && activeCollection}
                        <div class="panel-header">
                            <span>{activeCollection.name} — {collectionPositions.length} pos.</span>
                        </div>
                        <div class="description-row">
                            {#if editingDescription}
                                <input
                                    class="desc-input"
                                    type="text"
                                    bind:value={descriptionText}
                                    on:blur={saveDescription}
                                    on:keydown={handleDescriptionKeyDown}
                                    placeholder="Add description..."
                                    autofocus
                                />
                            {:else}
                                <span 
                                    class="desc-text desc-editable" 
                                    on:click={() => { editingDescription = true; }}
                                    title="Click to edit description"
                                >{descriptionText || '— click to add description'}</span>
                            {/if}
                        </div>
                        <div class="col-header-row">
                            <span class="col-h col-pos-order">#</span>
                            <span class="col-h col-pos-id">ID</span>
                            <span class="col-h col-pos-fill"></span>
                        </div>
                        <div class="list-container">
                            {#each collectionPositions as position, index}
                                <div 
                                    class="position-item"
                                    class:current={$currentPositionIndexStore === index}
                                    class:multi-selected={selectedPositionIndices.has(index)}
                                    class:drag-over={dragType === 'position' && dragOverIndex === index && dragStartIndex !== index}
                                    on:click={(e) => selectAndDisplayPosition(index, e)}
                                    draggable="true"
                                    on:dragstart={(e) => onPositionDragStart(e, index)}
                                    on:dragover={(e) => onPositionDragOver(e, index)}
                                    on:drop={(e) => onPositionDrop(e, index)}
                                    on:dragend={onDragEnd}
                                >
                                    <span class="col-cell col-pos-order">{index + 1}</span>
                                    <span class="col-cell col-pos-id">{positionIndexMap[position.id] || '?'}</span>
                                    <span class="col-cell col-pos-fill"></span>
                                    <div class="position-actions">
                                        <button class="icon-btn" on:click|stopPropagation={(e) => movePositionUp(index, e)} disabled={index === 0} title="Move up">▲</button>
                                        <button class="icon-btn" on:click|stopPropagation={(e) => movePositionDown(index, e)} disabled={index === collectionPositions.length - 1} title="Move down">▼</button>
                                        <button class="icon-btn delete" on:click|stopPropagation={(e) => removePositionFromRow(index, e)} title="Remove">×</button>
                                    </div>
                                </div>
                            {/each}
                            {#if collectionPositions.length === 0}
                                <div class="empty-msg">Empty collection</div>
                            {/if}
                        </div>
                    {:else}
                        <div class="panel-header">Add to collection</div>
                        <div class="list-container">
                            <div class="add-section">
                                {#each collections as collection}
                                    <label class="add-checkbox">
                                        <input 
                                            type="checkbox" 
                                            checked={positionCollectionIds.includes(collection.id)}
                                            on:change={() => togglePositionInCollection(collection.id)}
                                            disabled={!currentPosition || !currentPosition.id}
                                        />
                                        <span class="add-label">{collection.name}</span>
                                        <span class="add-count">({collection.positionCount})</span>
                                    </label>
                                {/each}
                                <div class="new-collection-row">
                                    <input 
                                        type="text" 
                                        bind:value={newCollectionName} 
                                        placeholder="New collection..."
                                        on:keydown|stopPropagation={(e) => e.key === 'Enter' && createAndAddToCollection()}
                                    />
                                    <button 
                                        on:click={createAndAddToCollection}
                                        disabled={!newCollectionName.trim() || !currentPosition || !currentPosition.id}
                                    >Add</button>
                                </div>
                            </div>
                        </div>
                    {/if}
                </div>
            </div>
        </div>
    </section>
{/if}

<style>
    .collection-panel {
        position: fixed;
        width: 100%;
        bottom: 22px;
        left: 0;
        right: 0;
        height: 178px;
        background-color: white;
        border-top: 1px solid rgba(0, 0, 0, 0.1);
        padding: 4px 10px;
        box-sizing: border-box;
        z-index: 5;
        outline: none;
        overflow: hidden;
        user-select: none;
        -webkit-user-select: none;
        -moz-user-select: none;
        -ms-user-select: none;
    }

    .close-icon {
        position: absolute;
        top: -4px;
        right: 6px;
        font-size: 20px;
        font-weight: bold;
        color: #888;
        cursor: pointer;
        background: none;
        border: none;
        padding: 0;
        z-index: 10;
    }

    .close-icon:hover {
        color: #333;
    }

    .collection-content {
        font-size: 12px;
        color: #333;
        height: 100%;
        display: flex;
        flex-direction: column;
    }

    .panels-container {
        display: flex;
        gap: 6px;
        flex: 1;
        min-height: 0;
    }

    .collections-list {
        flex: 3;
        display: flex;
        flex-direction: column;
        border: 1px solid #ddd;
        min-width: 0;
        overflow: hidden;
    }

    .right-panel {
        flex: 1;
        min-width: 180px;
        max-width: 280px;
        display: flex;
        flex-direction: column;
        border: 1px solid #ddd;
        overflow: hidden;
    }

    .panel-header {
        background-color: #f2f2f2;
        padding: 2px 8px;
        font-weight: bold;
        font-size: 11px;
        border-bottom: 1px solid #ddd;
        flex-shrink: 0;
        display: flex;
        align-items: center;
        gap: 6px;
        white-space: nowrap;
    }

    /* Column header row */
    .col-header-row {
        display: flex;
        align-items: center;
        background-color: #f8f8f8;
        border-bottom: 1px solid #ddd;
        padding: 1px 8px;
        flex-shrink: 0;
    }

    .col-h {
        font-size: 10px;
        font-weight: 600;
        color: #888;
        text-transform: uppercase;
        letter-spacing: 0.3px;
    }

    /* Column widths — collections */
    .col-name { width: 140px; min-width: 80px; flex-shrink: 0; text-align: left; }
    .col-count { width: 30px; flex-shrink: 0; text-align: right; padding-right: 8px; }
    .col-desc { flex: 1; min-width: 0; text-align: left; }
    .col-acts { width: 70px; flex-shrink: 0; }

    /* Column widths — positions */
    .col-pos-order { width: 32px; flex-shrink: 0; text-align: right; padding-right: 8px; }
    .col-pos-id { width: 50px; flex-shrink: 0; text-align: right; padding-right: 8px; }
    .col-pos-fill { flex: 1; }

    .col-cell {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    /* Description */
    .description-row {
        padding: 2px 8px;
        border-bottom: 1px solid #eee;
        flex-shrink: 0;
    }

    .desc-text {
        color: #666;
        font-style: italic;
        font-size: 11px;
        cursor: default;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        display: block;
        text-align: left;
    }

    .desc-editable {
        cursor: pointer;
    }

    .desc-editable:hover {
        color: #333;
    }

    .desc-input {
        width: 100%;
        font-size: 11px;
        padding: 1px 4px;
        border: 1px solid #999;
        outline: none;
        box-sizing: border-box;
        user-select: text;
        -webkit-user-select: text;
    }

    .list-container {
        flex: 1;
        overflow-y: auto;
        overflow-x: hidden;
        min-height: 0;
    }

    /* Collection items */
    .collection-item {
        display: flex;
        align-items: center;
        padding: 1px 8px;
        cursor: pointer;
        border-bottom: 1px solid #f0f0f0;
    }

    .collection-item:hover {
        background-color: #f0f5ff;
    }

    .collection-item.selected {
        background-color: #dce9f7;
    }

    .collection-item.active {
        background-color: #dce9f7;
    }

    .collection-item.selected .desc-text,
    .collection-item.active .desc-text {
        color: #444;
    }

    .collection-item.drag-over {
        border-top: 2px solid #999;
    }

    .collection-actions {
        display: flex;
        gap: 1px;
        visibility: hidden;
        flex-shrink: 0;
    }

    .collection-item:hover .collection-actions {
        visibility: visible;
    }

    .rename-input {
        width: 100%;
        font-size: 12px;
        padding: 1px 4px;
        border: 1px solid #999;
        outline: none;
        box-sizing: border-box;
        user-select: text;
        -webkit-user-select: text;
    }

    /* Icon buttons */
    .icon-btn {
        font-size: 10px;
        padding: 0 3px;
        border: 1px solid #ddd;
        background-color: #f5f5f5;
        cursor: pointer;
        line-height: 1.4;
        color: #666;
    }

    .icon-btn:hover:not(:disabled) {
        background-color: #e0e0e0;
        color: #333;
    }

    .icon-btn:disabled {
        opacity: 0.3;
        cursor: not-allowed;
    }

    .icon-btn.delete:hover:not(:disabled) {
        background-color: #fdd;
        border-color: #c00;
        color: #c00;
    }

    /* Position items */
    .position-item {
        display: flex;
        align-items: center;
        padding: 1px 8px;
        cursor: pointer;
        border-bottom: 1px solid #f0f0f0;
    }

    .position-item:hover {
        background-color: #f0f5ff;
    }

    .position-item.current {
        background-color: #dce9f7;
    }

    .position-item.multi-selected {
        background-color: #dce9f7;
    }

    .position-item.drag-over {
        border-top: 2px solid #999;
    }

    .position-actions {
        display: flex;
        gap: 1px;
        visibility: hidden;
        margin-left: auto;
    }

    .position-item:hover .position-actions {
        visibility: visible;
    }

    /* Add section */
    .add-section {
        padding: 4px;
    }

    .add-checkbox {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 2px 4px;
        cursor: pointer;
        font-size: 12px;
    }

    .add-checkbox:hover {
        background-color: #f5f5f5;
    }

    .add-checkbox input[type="checkbox"] {
        width: 13px;
        height: 13px;
        cursor: pointer;
    }

    .add-label {
        flex: 1;
    }

    .add-count {
        color: #999;
        font-size: 11px;
    }

    .new-collection-row {
        display: flex;
        gap: 4px;
        padding: 4px 4px 0;
        margin-top: 4px;
        border-top: 1px solid #eee;
    }

    .new-collection-row input {
        flex: 1;
        font-size: 12px;
        padding: 2px 4px;
        border: 1px solid #ddd;
        user-select: text;
        -webkit-user-select: text;
    }

    .new-collection-row button {
        font-size: 12px;
        padding: 2px 8px;
        border: 1px solid #ddd;
        background-color: #f5f5f5;
        cursor: pointer;
        white-space: nowrap;
    }

    .new-collection-row button:hover:not(:disabled) {
        background-color: #e0e0e0;
    }

    .new-collection-row button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .empty-msg {
        padding: 10px;
        text-align: center;
        color: #999;
        font-style: italic;
        font-size: 11px;
    }
</style>
