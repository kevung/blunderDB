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

    // Sub-tab: 'list' (collections list) or 'positions' (positions in active collection) or 'add' (add to collection)
    let activeSubTab = (mode === 'COLLECTION' && activeCollection) ? 'positions' : 'list';

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
        if (value === 'COLLECTION' && activeCollection) {
            activeSubTab = 'positions';
        }
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

    // Inline add row in collections list
    let inlineNewName = '';
    async function createCollectionInline() {
        if (!inlineNewName.trim()) return;
        try {
            await CreateCollection(inlineNewName.trim(), '');
            await loadCollections();
            statusBarTextStore.set(`Collection "${inlineNewName.trim()}" created`);
            inlineNewName = '';
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
        if (event.key === 'Enter') {
            event.stopPropagation();
            finishRename(collection);
        } else if (event.key === 'Escape') {
            event.stopPropagation();
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
        if (event.key === 'Enter') {
            event.stopPropagation();
            saveDescription();
        } else if (event.key === 'Escape') {
            event.stopPropagation();
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
        if (event.key === 'Enter') {
            event.stopPropagation();
            saveListDescription(collection);
        } else if (event.key === 'Escape') {
            event.stopPropagation();
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

        // Let Ctrl+key combos pass through to global handler (e.g. Ctrl+K to toggle panel)
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

    <section class="collection-panel" id="collectionPanel" tabindex="-1">
        <div class="collection-content">
            <!-- Sub-tab sidebar -->
            <div class="sub-tab-sidebar">
                <button class="sub-tab-btn home-btn" class:active={activeSubTab === 'list'} on:click={() => activeSubTab = 'list'} title="Collections">⌂</button>
                {#if mode === 'COLLECTION' && activeCollection}
                    <div class="sub-tab-btn named-tab" class:active={activeSubTab === 'positions'} on:click={() => activeSubTab = 'positions'} on:keydown={() => {}} role="button" tabindex="-1">
                        <span class="tab-name" title={activeCollection.name}>{activeCollection.name}</span>
                        <button class="tab-close-btn" on:click|stopPropagation={() => { activeSubTab = 'list'; }} title="Close">×</button>
                    </div>
                {/if}
            </div>

            <div class="sub-tab-content">
                {#if activeSubTab === 'list'}
                    <!-- Collections list -->
                    <div class="list-view">
                        <div class="list-container">
                            {#each collections as collection, index}
                                <div
                                    class="row"
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
                                    {:else}
                                        <span class="row-name" title={collection.name}>{collection.name}</span>
                                    {/if}
                                    {#if collection.positionCount > 0}
                                        <span class="row-badge">{collection.positionCount}</span>
                                    {/if}
                                    {#if collection.description}
                                        <span class="row-desc" title={collection.description}>{collection.description}</span>
                                    {/if}
                                    <span class="row-acts">
                                        <button class="icon-btn" on:click|stopPropagation={(e) => startRename(collection, e)} title="Rename">✎</button>
                                        <button class="icon-btn delete" on:click|stopPropagation={(e) => deleteCollection(collection, e)} title="Delete">×</button>
                                    </span>
                                </div>
                            {/each}
                            <!-- Inline add row -->
                            <div class="row add-row">
                                <input class="row-input name" type="text" bind:value={inlineNewName} placeholder="New collection…" on:keydown|stopPropagation={(e) => e.key === 'Enter' && createCollectionInline()} />
                            </div>
                            {#if collections.length === 0}
                                <div class="empty-msg">No collections</div>
                            {/if}
                        </div>
                    </div>
                {:else if activeSubTab === 'positions' && mode === 'COLLECTION' && activeCollection}
                    <!-- Positions in active collection -->
                    <div class="list-view">
                        <div class="list-header">
                            <span>{activeCollection.name}</span>
                            {#if editingDescription}
                                <input
                                    class="desc-inline-input"
                                    type="text"
                                    bind:value={descriptionText}
                                    on:blur={saveDescription}
                                    on:keydown={handleDescriptionKeyDown}
                                    placeholder="description…"
                                    autofocus
                                />
                            {:else}
                                <span class="desc-inline" on:click={() => { editingDescription = true; }} title="Click to edit description">{descriptionText || ''}</span>
                            {/if}
                        </div>
                        <div class="list-container">
                            {#each collectionPositions as position, index}
                                <div
                                    class="row pos-row"
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
                                    <span class="pos-idx">{index + 1}</span>
                                    <span class="pos-id">id {positionIndexMap[position.id] || '?'}</span>
                                    <span class="row-acts">
                                        <button class="icon-btn" on:click|stopPropagation={(e) => movePositionUp(index, e)} disabled={index === 0}>▲</button>
                                        <button class="icon-btn" on:click|stopPropagation={(e) => movePositionDown(index, e)} disabled={index === collectionPositions.length - 1}>▼</button>
                                        <button class="icon-btn delete" on:click|stopPropagation={(e) => removePositionFromRow(index, e)}>×</button>
                                    </span>
                                </div>
                            {/each}
                            {#if collectionPositions.length === 0}
                                <div class="empty-msg">Empty collection</div>
                            {/if}
                        </div>
                    </div>
                {/if}
            </div>
        </div>
    </section>

<style>
    .collection-panel { width: 100%; height: 100%; background: white; box-sizing: border-box; outline: none; overflow: hidden; user-select: none; }
    .collection-content { font-size: 12px; color: #333; height: 100%; display: flex; }

    .sub-tab-sidebar { display: flex; flex-direction: column; width: 70px; flex-shrink: 0; background: #f5f5f5; border-right: 1px solid #ddd; }
    .sub-tab-btn { border: none; background: transparent; padding: 8px 4px; font-size: 11px; color: #666; cursor: pointer; border-left: 2px solid transparent; text-align: center; transition: background 0.15s; }
    .sub-tab-btn:hover { background: #e8e8e8; }
    .sub-tab-btn.active { color: #333; font-weight: 600; background: #fff; border-left-color: #555; }
    .sub-tab-btn.home-btn { font-size: 16px; padding: 6px 4px; }
    .sub-tab-btn.named-tab { display: flex; flex-direction: column; align-items: center; gap: 2px; padding: 6px 2px; position: relative; cursor: pointer; }
    .tab-name { font-size: 9px; line-height: 1.2; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 62px; display: block; }
    .tab-close-btn { border: none; background: none; font-size: 12px; color: #aaa; cursor: pointer; line-height: 1; padding: 0 2px; }
    .tab-close-btn:hover { color: #c55; }
    .sub-tab-content { flex: 1; min-width: 0; overflow: hidden; display: flex; }

    .list-view { flex: 1; display: flex; flex-direction: column; min-height: 0; overflow: hidden; }
    .list-header { padding: 3px 10px; font-size: 11px; font-weight: 600; color: #555; border-bottom: 1px solid #eee; flex-shrink: 0; background: #fafafa; display: flex; align-items: center; gap: 8px; }
    .list-container { flex: 1; overflow-y: auto; overflow-x: hidden; }

    .row {
        display: flex;
        align-items: center;
        padding: 2px 10px;
        cursor: pointer;
        border-bottom: 1px solid #f5f5f5;
        min-height: 24px;
        gap: 6px;
    }
    .row:hover { background: #f5f8ff; }
    .row.selected { background: #e3f2fd; }
    .row.active { background: #dce9f7; }
    .row.drag-over { border-top: 2px solid #999; }
    .row.add-row { cursor: default; background: #fafafa; border-bottom: none; }
    .row.add-row:hover { background: #fafafa; }

    .row-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; }
    .row-badge { font-size: 9px; color: #888; background: #eee; border-radius: 8px; padding: 0 5px; flex-shrink: 0; }
    .row-desc { font-size: 10px; color: #aaa; font-style: italic; flex-shrink: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 120px; }
    .row-acts { display: flex; gap: 2px; visibility: hidden; flex-shrink: 0; }
    .row:hover .row-acts { visibility: visible; }

    .rename-input { flex: 1; font-size: 12px; padding: 1px 4px; border: 1px solid #999; outline: none; box-sizing: border-box; }
    .row-input { padding: 1px 4px; border: 1px solid #ccc; border-radius: 2px; font-size: 11px; outline: none; box-sizing: border-box; }
    .row-input.name { flex: 1; min-width: 0; }

    .icon-btn { background: none; border: none; cursor: pointer; font-size: 10px; color: #888; padding: 0 3px; line-height: 1; }
    .icon-btn:hover:not(:disabled) { color: #333; }
    .icon-btn:disabled { opacity: 0.3; cursor: not-allowed; }
    .icon-btn.delete:hover:not(:disabled) { color: #c55; }

    .empty-msg { text-align: center; color: #bbb; padding: 16px; font-size: 11px; font-style: italic; }

    /* Positions */
    .pos-row { cursor: pointer; }
    .pos-row.current { background: #dce9f7; }
    .pos-row.multi-selected { background: #dce9f7; }
    .pos-row.drag-over { border-top: 2px solid #999; }
    .pos-idx { font-size: 10px; color: #aaa; width: 24px; text-align: right; flex-shrink: 0; }
    .pos-id { font-size: 11px; color: #666; flex: 1; }

    /* Description inline */
    .desc-inline { font-size: 10px; color: #aaa; font-style: italic; cursor: pointer; }
    .desc-inline:hover { color: #666; }
    .desc-inline-input { font-size: 10px; border: 1px solid #ccc; outline: none; padding: 1px 4px; flex: 1; min-width: 0; }
</style>
