<script>
    import { logger } from '../utils/logger.js';
    import { onMount, onDestroy } from 'svelte';
    import { SvelteSet } from 'svelte/reactivity';
    import { createReorder } from '../utils/reorder.js';
    import { createInlineEdit } from '../utils/inlineEdit.svelte.js';
    import { autofocus } from '../utils/autofocus.js';
    import { collectionsStore, selectedCollectionStore, collectionPositionsStore, activeCollectionStore } from '../stores/collectionStore';
    import { openPanels, PANEL, closePanel, statusBarTextStore, statusBarModeStore, currentPositionIndexStore } from '../stores/uiStore';
    import { databaseLoadedStore } from '../stores/databaseStore';
    import { positionStore } from '../stores/positionStore';
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
    } from '../../wailsjs/go/database/Database.js';
    import { t, tMsg } from '../i18n';
    import { confirmAction } from '../services/confirmService.js';
    import PanelTable from './panels/PanelTable.svelte';
    import { panelKeyGuard } from '../services/keyboardService.js';

    let { onOpenCollection } = $props();

    // Read-only mirrors of stores
    let collections = $derived($collectionsStore || []);
    let selectedCollection = $derived($selectedCollectionStore);
    let collectionPositions = $derived($collectionPositionsStore || []);
    let activeCollection = $derived($activeCollectionStore);
    let visible = $derived($openPanels.has(PANEL.COLLECTION));
    let currentPosition = $derived($positionStore);

    let mode = 'NORMAL';

    let positionCollectionIds = $state([]);

    // View: 'list' (all collections) or 'detail' (positions in active collection)
    let view = $state('list');

    // Collection editing (unified: name + description at the same time). The
    // fields of one row form a blur group: tabbing between them keeps editing.
    const collectionEdit = createInlineEdit({
        blurGroup: 'tr, .desc-bar',
        onSave: async (id, draft) => {
            const collection = collections.find((c) => c.id === id) || (activeCollection?.id === id ? activeCollection : null);
            if (!collection) return;
            const newName = draft.name.trim() || collection.name;
            const newDesc = draft.description.trim();
            if (newName === collection.name && newDesc === (collection.description || '')) return;
            if (newName !== collection.name && isDuplicateName(newName, collection.id)) {
                statusBarTextStore.set(tMsg('collection.alreadyExists', { name: newName }));
                return;
            }
            try {
                await UpdateCollection(collection.id, newName, newDesc);
                await loadCollections();
                if (activeCollection && activeCollection.id === collection.id) {
                    activeCollectionStore.set({ ...activeCollection, name: newName, description: newDesc });
                }
            } catch (error) {
                logger.error('Error updating collection:', error);
            }
        }
    });
    let inlineNewName = $state('');

    // Multi-select for positions: indices into collectionPositions. A SvelteSet
    // mutated in place — the template tracks this one instance, so there is
    // no clone-and-reassign to keep it reactive.
    const selectedPositionIndices = new SvelteSet();

    // Position index map (position_id -> 1-based index in DB)
    let positionIndexMap = $state({});

    // Inline new description
    let inlineNewDescription = $state('');

    const collectionColumns = $derived([
        { key: 'name', label: $t('collection.colName') },
        { key: 'positions', label: $t('collection.colPos'), narrow: true },
        { key: 'description', label: $t('collection.colDescription') },
        { key: 'modified', label: $t('collection.colModified'), narrow: true },
        { key: 'toggle', label: $t('collection.colPosCheck'), narrow: true, align: 'center', class: 'toggle-header', title: $t('collection.toggleHeaderTooltip') },
        { key: 'actions', actions: true }
    ]);

    const positionColumns = $derived([
        { key: 'index', label: '#', narrow: true },
        { key: 'id', label: $t('collection.colId'), narrow: true },
        { key: 'actions', actions: true }
    ]);

    // Sync view with activeCollection store
    $effect(() => {
        if ($activeCollectionStore) {
            view = 'detail';
        } else {
            view = 'list';
        }
    });

    // Load collections when database becomes available
    $effect(() => {
        if ($databaseLoadedStore) {
            loadCollections();
            loadPositionIndexMap();
        }
    });

    // Reload which collections contain the current position
    $effect(() => {
        const pos = $positionStore;
        positionCollectionIds = [];
        if (pos && pos.id) loadPositionCollections(pos.id);
    });

    // Sync mode; set view to detail when entering COLLECTION mode with an active collection
    $effect(() => {
        const v = $statusBarModeStore;
        mode = v;
        if (v === 'COLLECTION' && activeCollection) {
            view = 'detail';
        }
    });

    // Sync position selection when navigating with j/k in COLLECTION mode
    $effect(() => {
        const value = $currentPositionIndexStore;
        if (mode === 'COLLECTION' && activeCollection && value >= 0) {
            selectedPositionIndices.clear();
            selectedPositionIndices.add(value);
        }
    });

    async function loadCollections() {
        try {
            const loaded = await GetAllCollections();
            collectionsStore.set(loaded || []);
        } catch (error) {
            logger.error('Error loading collections:', error);
            statusBarTextStore.set(tMsg('collection.errorLoading'));
        }
    }

    async function loadPositionIndexMap() {
        try {
            positionIndexMap = (await GetPositionIndexMap()) || {};
        } catch (_error) {
            positionIndexMap = {};
        }
    }

    async function loadPositionCollections(positionId) {
        try {
            const colls = await GetPositionCollections(positionId);
            positionCollectionIds = (colls || []).map((c) => c.id);
        } catch (_error) {
            positionCollectionIds = [];
        }
    }

    function isDuplicateName(name, excludeId = null) {
        const lower = name.trim().toLowerCase();
        return collections.some((c) => c.name.toLowerCase() === lower && c.id !== excludeId);
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
        const pad = (n) => String(n).padStart(2, '0');
        return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
    }

    async function togglePositionInCollection(collectionId, event) {
        if (event) event.stopPropagation();
        if (!currentPosition || !currentPosition.id || currentPosition.id === 0) {
            statusBarTextStore.set(tMsg('collection.noPositionSelected'));
            return;
        }
        try {
            if (positionCollectionIds.includes(collectionId)) {
                await RemovePositionFromCollection(collectionId, currentPosition.id);
                positionCollectionIds = positionCollectionIds.filter((id) => id !== collectionId);
                statusBarTextStore.set(tMsg('collection.positionRemoved'));
            } else {
                await AddPositionToCollection(collectionId, currentPosition.id);
                positionCollectionIds = [...positionCollectionIds, collectionId];
                statusBarTextStore.set(tMsg('collection.positionAdded'));
            }
            await loadCollections();
            await loadPositionIndexMap();
            if (activeCollection && activeCollection.id === collectionId) {
                const positions = await GetCollectionPositions(collectionId);
                collectionPositionsStore.set(positions || []);
            }
        } catch (error) {
            logger.error('Error toggling position in collection:', error);
            statusBarTextStore.set(tMsg('collection.errorUpdating'));
        }
    }

    async function createCollectionInline() {
        if (!inlineNewName.trim()) return;
        if (isDuplicateName(inlineNewName)) {
            statusBarTextStore.set(tMsg('collection.alreadyExists', { name: inlineNewName.trim() }));
            return;
        }
        try {
            await CreateCollection(inlineNewName.trim(), inlineNewDescription.trim());
            await loadCollections();
            statusBarTextStore.set(tMsg('collection.created', { name: inlineNewName.trim() }));
            inlineNewName = '';
            inlineNewDescription = '';
        } catch (error) {
            logger.error('Error creating collection:', error);
            statusBarTextStore.set(tMsg('collection.errorCreating'));
        }
    }

    async function openCollection(collection) {
        if (collectionEdit.isEditing(collection.id)) return;
        try {
            const positions = await GetCollectionPositions(collection.id);
            if (!positions || positions.length === 0) {
                statusBarTextStore.set(tMsg('collection.isEmpty', { name: collection.name }));
                return;
            }
            selectedCollectionStore.set(collection);
            collectionPositionsStore.set(positions);
            activeCollectionStore.set(collection);
            selectedPositionIndices.clear();
            view = 'detail';
            await loadPositionIndexMap();
            if (onOpenCollection) {
                onOpenCollection(collection, positions);
            }
        } catch (error) {
            logger.error('Error opening collection:', error);
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
            logger.error('Error deleting collection:', error);
        }
    }

    function startEditing(collection, event) {
        if (event) event.stopPropagation();
        collectionEdit.start(collection.id, { name: collection.name, description: collection.description || '' });
    }

    // Collection reorder: ▲/▼ buttons and pointer drag share one helper.
    const collectionOrder = createReorder({
        get: () => collections,
        set: (next) => collectionsStore.set(next),
        persist: (next) => ReorderCollections(next.map((c) => c.id)),
        label: 'collections'
    });

    // Position reorder within the active collection; a selected row follows its move.
    const positionOrder = createReorder({
        get: () => (activeCollection ? collectionPositions : null),
        set: (next, from, to) => {
            collectionPositionsStore.set(next);
            if (selectedPositionIndices.has(from)) {
                selectedPositionIndices.delete(from);
                selectedPositionIndices.add(to);
            }
        },
        persist: (next) =>
            ReorderCollectionPositions(
                activeCollection.id,
                next.map((p) => p.id)
            ),
        label: 'positions'
    });

    // Select a position and display it
    async function selectAndDisplayPosition(index, event) {
        event.stopPropagation();
        if (event.shiftKey && selectedPositionIndices.size > 0) {
            const sorted = [...selectedPositionIndices].sort((a, b) => a - b);
            const last = sorted[sorted.length - 1];
            const start = Math.min(last, index);
            const end = Math.max(last, index);
            for (let i = start; i <= end; i++) {
                selectedPositionIndices.add(i);
            }
        } else if (event.ctrlKey || event.metaKey) {
            if (selectedPositionIndices.has(index)) {
                selectedPositionIndices.delete(index);
            } else {
                selectedPositionIndices.add(index);
            }
        } else {
            const wasOnlySelection = selectedPositionIndices.size === 1 && selectedPositionIndices.has(index);
            selectedPositionIndices.clear();
            if (!wasOnlySelection) selectedPositionIndices.add(index);
        }

        const position = collectionPositions[index];
        if (position) {
            navigateToPosition(position, index);
        }
    }

    async function removeSelectedFromCollection() {
        if (!activeCollection || selectedPositionIndices.size === 0) return;
        if (!(await confirmAction($t('collection.confirmRemove', { count: selectedPositionIndices.size }), { confirmLabel: $t('common.delete') }))) return;
        const sorted = [...selectedPositionIndices].sort((a, b) => b - a);
        try {
            for (const idx of sorted) {
                await RemovePositionFromCollection(activeCollection.id, collectionPositions[idx].id);
            }
            selectedPositionIndices.clear();
            const positions = await GetCollectionPositions(activeCollection.id);
            collectionPositionsStore.set(positions || []);
            await loadCollections();
            if (onOpenCollection && positions && positions.length > 0) {
                onOpenCollection(activeCollection, positions);
            }
        } catch (error) {
            logger.error('Error removing positions:', error);
        }
    }

    async function removePositionFromRow(index, event) {
        event.stopPropagation();
        if (!activeCollection) return;
        const positionId = collectionPositions[index].id;
        try {
            await RemovePositionFromCollection(activeCollection.id, positionId);
            // The rows after the removed one shift up by one.
            const shifted = [...selectedPositionIndices].filter((i) => i !== index).map((i) => (i > index ? i - 1 : i));
            selectedPositionIndices.clear();
            for (const i of shifted) selectedPositionIndices.add(i);
            const positions = await GetCollectionPositions(activeCollection.id);
            collectionPositionsStore.set(positions || []);
            await loadCollections();
            if (onOpenCollection && positions && positions.length > 0) {
                onOpenCollection(activeCollection, positions);
            }
        } catch (error) {
            logger.error('Error removing position:', error);
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
            logger.error('Error loading analysis:', error);
        }
    }

    // Pointer-based drag reorder for positions within a collection. A
    // multi-selection moves as a block; anything else is a single-item move.
    async function handlePositionReorder(fromIndex, toIndex) {
        if (!activeCollection) return;

        if (selectedPositionIndices.size > 1 && selectedPositionIndices.has(fromIndex)) {
            const sorted = [...selectedPositionIndices].sort((a, b) => a - b);
            const items = sorted.map((i) => collectionPositions[i]);
            const newOrder = collectionPositions.filter((_, i) => !selectedPositionIndices.has(i));
            let insertAt = toIndex;
            const removedBefore = sorted.filter((i) => i < toIndex).length;
            insertAt -= removedBefore;
            if (insertAt < 0) insertAt = 0;
            newOrder.splice(insertAt, 0, ...items);
            collectionPositionsStore.set(newOrder);
            selectedPositionIndices.clear();
            for (let i = 0; i < items.length; i++) {
                selectedPositionIndices.add(insertAt + i);
            }
            try {
                await ReorderCollectionPositions(
                    activeCollection.id,
                    newOrder.map((p) => p.id)
                );
            } catch (error) {
                logger.error('Error reordering positions:', error);
            }
        } else {
            await positionOrder.reorder(fromIndex, toIndex);
        }
    }

    function closeCollectionPanel() {
        closePanel(PANEL.COLLECTION);
    }

    function handleKeyDown(event) {
        if (!visible) return;

        // Let Ctrl/Meta combos, Space, '?', typing in an editable field, and
        // position-browsing keys (this panel has no in-panel list navigation of
        // its own) pass through to the global handler — see keyboardService.panelKeyGuard.
        if (panelKeyGuard(event, { allowNavKeys: true })) return;

        // Stop other keyboard events from propagating to global handlers
        event.stopPropagation();

        if (event.key === 'Escape') {
            if (view === 'detail' && mode !== 'COLLECTION') {
                view = 'list';
            } else if (selectedCollection) {
                selectedCollectionStore.set(null);
                collectionPositionsStore.set([]);
            } else {
                closeCollectionPanel();
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
        if (!(await confirmAction($t('collection.confirmRemove', { count: 1 }), { confirmLabel: $t('common.delete') }))) return;
        try {
            await RemovePositionFromCollection(activeCollection.id, positionId);
            const positions = await GetCollectionPositions(activeCollection.id);
            collectionPositionsStore.set(positions || []);
            await loadCollections();
            if (onOpenCollection && positions && positions.length > 0) {
                onOpenCollection(activeCollection, positions);
            }
        } catch (error) {
            logger.error('Error removing position:', error);
        }
    }

    onMount(async () => {
        document.addEventListener('keydown', handleKeyDown);
        if (mode === 'COLLECTION' && activeCollection && activeCollection.id) {
            view = 'detail';
            try {
                const positions = await GetCollectionPositions(activeCollection.id);
                collectionPositionsStore.set(positions || []);
            } catch (error) {
                logger.error('Error reloading active collection positions:', error);
            }
        }
    });

    onDestroy(() => {
        document.removeEventListener('keydown', handleKeyDown);
    });
</script>

<section class="collection-panel" id="collectionPanel" tabindex="-1" aria-label={$t('collection.title')}>
    {#if view === 'list'}
        <!-- Collections list -->
        <div class="table-wrapper">
            <PanelTable
                rows={collections}
                columns={collectionColumns}
                selectedKey={selectedCollection?.id}
                rowClass={(collection) => (positionCollectionIds.includes(collection.id) ? 'in-collection' : '')}
                onSelect={(collection, _index, e) => togglePositionInCollection(collection.id, e)}
                onActivate={(collection) => openCollection(collection)}
                onReorder={collectionOrder.reorder}
                emptyText={$t('collection.empty')}
            >
                {#snippet cells(collection, index)}
                    <td class="name-cell">
                        {#if collectionEdit.isEditing(collection.id)}
                            <input
                                class="inline-edit"
                                type="text"
                                bind:value={collectionEdit.draft.name}
                                onblur={collectionEdit.onBlur}
                                onkeydown={collectionEdit.onKeyDown}
                                onclick={(e) => e.stopPropagation()}
                                ondblclick={(e) => e.stopPropagation()}
                                use:autofocus
                            />
                        {:else}
                            <span title={collection.name}>{collection.name}</span>
                        {/if}
                    </td>
                    <td class="narrow-col count-cell">{collection.positionCount || 0}</td>
                    <td class="desc-cell">
                        {#if collectionEdit.isEditing(collection.id)}
                            <input
                                class="inline-edit"
                                type="text"
                                bind:value={collectionEdit.draft.description}
                                onblur={collectionEdit.onBlur}
                                onkeydown={collectionEdit.onKeyDown}
                                onclick={(e) => e.stopPropagation()}
                                ondblclick={(e) => e.stopPropagation()}
                                placeholder={$t('collection.descriptionPlaceholder')}
                            />
                        {:else}
                            <span class="desc-text" title={collection.description || ''}>{collection.description || ''}</span>
                        {/if}
                    </td>
                    <td class="narrow-col date-cell">{formatDate(collection.updatedAt)}</td>
                    <td class="narrow-col toggle-cell">
                        {#if currentPosition && currentPosition.id}
                            <input
                                type="checkbox"
                                checked={positionCollectionIds.includes(collection.id)}
                                onclick={(e) => {
                                    e.stopPropagation();
                                    ((e) => togglePositionInCollection(collection.id, e))(e);
                                }}
                                title={positionCollectionIds.includes(collection.id) ? $t('collection.removePositionTooltip') : $t('collection.addPositionTooltip')}
                            />
                        {/if}
                    </td>
                    <td class="actions-col">
                        <span class="item-actions">
                            <button
                                class="icon-btn"
                                onclick={(e) => {
                                    e.stopPropagation();
                                    collectionOrder.moveUp(index);
                                }}
                                disabled={index === 0}
                                title={$t('collection.moveUp')}>▲</button
                            >
                            <button
                                class="icon-btn"
                                onclick={(e) => {
                                    e.stopPropagation();
                                    collectionOrder.moveDown(index);
                                }}
                                disabled={index === collections.length - 1}
                                title={$t('collection.moveDown')}>▼</button
                            >
                            <button
                                class="icon-btn"
                                onclick={(e) => {
                                    e.stopPropagation();
                                    ((e) => startEditing(collection, e))(e);
                                }}
                                title={$t('common.edit')}>✎</button
                            >
                            <button
                                class="icon-btn delete"
                                onclick={(e) => {
                                    e.stopPropagation();
                                    ((e) => deleteCollection(collection, e))(e);
                                }}
                                title={$t('common.delete')}>×</button
                            >
                        </span>
                    </td>
                {/snippet}
            </PanelTable>
            <!-- Inline add row below table -->
            <div class="add-row">
                <input
                    class="add-input"
                    type="text"
                    bind:value={inlineNewName}
                    placeholder={$t('collection.newCollectionPlaceholder')}
                    onkeydown={(e) => {
                        e.stopPropagation();
                        ((e) => e.key === 'Enter' && createCollectionInline())(e);
                    }}
                />
                <input
                    class="add-input desc"
                    type="text"
                    bind:value={inlineNewDescription}
                    placeholder={$t('collection.descriptionInputPlaceholder')}
                    onkeydown={(e) => {
                        e.stopPropagation();
                        ((e) => e.key === 'Enter' && createCollectionInline())(e);
                    }}
                />
            </div>
        </div>
    {:else if view === 'detail' && activeCollection}
        <!-- Positions in active collection -->
        <div class="table-wrapper">
            <PanelTable
                rows={collectionPositions}
                columns={positionColumns}
                rowClass={(_position, index) => [$currentPositionIndexStore === index ? 'current' : '', selectedPositionIndices.has(index) ? 'multi-selected' : ''].join(' ')}
                onSelect={(_position, index, e) => selectAndDisplayPosition(index, e)}
                onReorder={handlePositionReorder}
                emptyText={$t('collection.emptyCollection')}
            >
                {#snippet header()}
                    <button class="back-btn" onclick={goBackToList} title={$t('collection.backToCollections')}>←</button>
                    <span class="detail-title" title={activeCollection.name}>{activeCollection.name}</span>
                    <span class="detail-count">{$t('collection.posCount', { count: collectionPositions.length })}</span>
                    {#if currentPosition && currentPosition.id}
                        <input
                            type="checkbox"
                            checked={positionCollectionIds.includes(activeCollection.id)}
                            onclick={(e) => togglePositionInCollection(activeCollection.id, e)}
                            title={positionCollectionIds.includes(activeCollection.id) ? $t('collection.removePositionTooltip') : $t('collection.addPositionTooltip')}
                        />
                    {/if}
                {/snippet}
                {#snippet subheader()}
                    {#if collectionEdit.isEditing(activeCollection.id)}
                        <div class="desc-bar">
                            <input
                                class="inline-edit full-width"
                                type="text"
                                bind:value={collectionEdit.draft.description}
                                onblur={collectionEdit.onBlur}
                                onkeydown={collectionEdit.onKeyDown}
                                placeholder={$t('collection.descriptionPlaceholder')}
                                use:autofocus
                            />
                        </div>
                    {:else}
                        <div class="desc-bar clickable" onclick={(e) => startEditing(activeCollection, e)}>
                            <span class="desc-text" title={$t('collection.clickToEdit')}>{activeCollection.description || $t('collection.addDescription')}</span>
                        </div>
                    {/if}
                {/snippet}
                {#snippet cells(position, index)}
                    <td class="narrow-col idx-cell">{index + 1}</td>
                    <td class="narrow-col id-cell">{positionIndexMap[position.id] || '?'}</td>
                    <td class="actions-col">
                        <span class="item-actions">
                            <button
                                class="icon-btn"
                                onclick={(e) => {
                                    e.stopPropagation();
                                    positionOrder.moveUp(index);
                                }}
                                disabled={index === 0}
                                title={$t('collection.moveUp')}>▲</button
                            >
                            <button
                                class="icon-btn"
                                onclick={(e) => {
                                    e.stopPropagation();
                                    positionOrder.moveDown(index);
                                }}
                                disabled={index === collectionPositions.length - 1}
                                title={$t('collection.moveDown')}>▼</button
                            >
                            <button
                                class="icon-btn delete"
                                onclick={(e) => {
                                    e.stopPropagation();
                                    ((e) => removePositionFromRow(index, e))(e);
                                }}
                                title={$t('collection.removeFromCollection')}>×</button
                            >
                        </span>
                    </td>
                {/snippet}
            </PanelTable>
        </div>
    {/if}
</section>

<style>
    .collection-panel {
        width: 100%;
        height: 100%;
        background: white;
        box-sizing: border-box;
        outline: none;
        overflow: hidden;
        user-select: none;
        -webkit-user-select: none;
    }
    .collection-panel * {
        user-select: none;
        -webkit-user-select: none;
    }
    .collection-panel input,
    .collection-panel textarea {
        user-select: text;
        -webkit-user-select: text;
    }

    .table-wrapper {
        height: 100%;
        display: flex;
        flex-direction: column;
        min-height: 0;
    }

    .date-cell {
        font-size: var(--font-size-small);
        color: #999;
    }

    /* Row states (the rows are PanelTable's elements) */
    .collection-panel :global(tr.in-collection) {
        border-left: 3px solid #4a8;
    }
    .collection-panel :global(tr.current),
    .collection-panel :global(tr.multi-selected) {
        background-color: #dce9f7;
    }

    /* Cell styles */
    .name-cell {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        max-width: 0;
    }
    .desc-cell {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        max-width: 0;
    }
    .desc-text {
        color: #888;
        font-style: italic;
        cursor: pointer;
        font-size: var(--font-size-small);
    }
    .desc-text:hover {
        color: #555;
    }
    .idx-cell {
        text-align: right;
        color: #999;
    }
    .id-cell {
        color: #666;
    }

    .inline-edit {
        width: 100%;
        font-size: var(--font-size-base);
        padding: 1px 4px;
        border: 1px solid #999;
        outline: none;
        box-sizing: border-box;
    }

    .collection-panel :global(th.toggle-header) {
        color: #4a8;
    }
    .toggle-cell {
        text-align: center;
    }
    .toggle-cell input[type='checkbox'] {
        cursor: pointer;
        margin: 0;
        width: 15px;
        height: 15px;
        accent-color: #4a8;
    }

    /* Add row */
    .add-row {
        padding: 4px 8px;
        background: #fafafa;
        border-top: 1px solid #e0e0e0;
        flex-shrink: 0;
    }
    .add-row {
        display: flex;
        gap: 4px;
    }
    .add-input {
        flex: 1;
        padding: 3px 6px;
        border: 1px solid #ccc;
        border-radius: 3px;
        font-size: var(--font-size-base);
        outline: none;
        box-sizing: border-box;
    }
    .add-input.desc {
        flex: 1;
        font-size: var(--font-size-small);
        color: #666;
    }
    .add-input:focus {
        border-color: #999;
    }

    /* Detail header (the strip itself is PanelTable's) */
    .back-btn {
        background: none;
        border: none;
        cursor: pointer;
        font-size: var(--font-size-title);
        color: #666;
        padding: 2px 6px;
        line-height: 1;
    }
    .back-btn:hover {
        color: #333;
    }
    .detail-title {
        font-size: var(--font-size-base);
        font-weight: 600;
        color: #333;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        flex: 1;
    }
    .detail-count {
        font-size: var(--font-size-small);
        color: #888;
        flex-shrink: 0;
    }

    /* Description bar */
    .desc-bar {
        padding: 3px 8px 3px 32px;
        border-bottom: 1px solid #eee;
        flex-shrink: 0;
    }
    .desc-bar.clickable {
        cursor: pointer;
    }
    .desc-bar .desc-text {
        font-size: var(--font-size-small);
    }
    .full-width {
        width: 100%;
    }
</style>
