<script>
    import { onMount, onDestroy } from 'svelte';
    import {
        ankiDecksStore,
        selectedAnkiDeckStore,
        ankiReviewCardStore,
        ankiDeckStatsStore,
        ankiViewModeStore,
        ankiReviewActionStore,
        ankiPausedSessionStore
    } from '../stores/ankiStore';
    import {
        statusBarTextStore,
        activeTabStore,
        currentPositionIndexStore
    } from '../stores/uiStore';
    import { databaseLoadedStore } from '../stores/databaseStore';
    import { collectionsStore } from '../stores/collectionStore';
    import { positionsStore, positionStore } from '../stores/positionStore';
    import { lastSearchStore } from '../stores/searchHistoryStore';
    import { parseFilters } from '../commandProcessor';
    import {
        CreateAnkiDeck,
        GetAllAnkiDecks,
        UpdateAnkiDeck,
        UpdateAnkiDeckParams,
        DeleteAnkiDeck,
        SyncAnkiDeck,
        SyncAnkiDeckWithPositions,
        GetAnkiDeckStats,
        GetAnkiDeckPositions,
        GetNextAnkiCard,
        ReviewAnkiCard,
        ResetAnkiDeck,
        GetAllCollections,
        LoadPositionsByFilters
    } from '../../wailsjs/go/main/Database.js';

    let decks = [];
    let selectedDeck = null;
    let reviewCard = null;
    let stats = null;
    let viewMode = 'list';
    let databaseLoaded = false;
    let collections = [];
    let positions = [];

    // Create deck form
    let newDeckName = '';
    let newDeckSourceType = 'collection';
    let newDeckSourceId = 0;
    let showCreateForm = false;

    // Edit deck
    let editingDeckId = null;
    let editingName = '';
    let editingDescription = '';

    // Settings
    let settingsRetention = 0.9;
    let settingsMaxInterval = 36500;
    let settingsFuzz = true;

    // Review state
    let reviewSessionCount = 0;
    let lastSearch = null;
    let pausedSession = null;

    const unsubDecks = ankiDecksStore.subscribe(v => { decks = v || []; });
    const unsubSelected = selectedAnkiDeckStore.subscribe(v => { selectedDeck = v; });
    const unsubReview = ankiReviewCardStore.subscribe(v => { reviewCard = v; });
    const unsubStats = ankiDeckStatsStore.subscribe(v => { stats = v; });
    const unsubMode = ankiViewModeStore.subscribe(v => { viewMode = v; });
    const unsubDb = databaseLoadedStore.subscribe(v => { databaseLoaded = v; });
    const unsubCollections = collectionsStore.subscribe(v => { collections = v || []; });
    const unsubPositions = positionsStore.subscribe(v => { positions = v || []; });
    const unsubLastSearch = lastSearchStore.subscribe(v => { lastSearch = v; });
    const unsubPausedSession = ankiPausedSessionStore.subscribe(v => { pausedSession = v; });

    // Listen for review key actions routed from App.svelte
    const unsubReviewAction = ankiReviewActionStore.subscribe(v => {
        if (v !== null) {
            ankiReviewActionStore.set(null);
            if (v === 'back') {
                backToList();
            } else if (typeof v === 'number' && v >= 1 && v <= 4) {
                submitReview(v);
            }
        }
    });

    onMount(() => {
        if (databaseLoaded) loadDecks();
    });

    onDestroy(() => {
        unsubDecks(); unsubSelected(); unsubReview(); unsubStats();
        unsubMode(); unsubDb(); unsubCollections(); unsubPositions();
        unsubLastSearch(); unsubReviewAction(); unsubPausedSession();
    });

    async function loadDecks() {
        try {
            const result = await GetAllAnkiDecks();
            ankiDecksStore.set(result || []);
            // Also load collections for the create form
            const colls = await GetAllCollections();
            collectionsStore.set(colls || []);
        } catch (e) {
            console.error('Error loading anki decks:', e);
        }
    }

    async function createDeck() {
        if (!newDeckName.trim()) return;
        try {
            let sourceType = newDeckSourceType;
            let sourceId = newDeckSourceId;
            let sourceCommand = '';

            if (sourceType === 'search') {
                // Store the search command + position + current IDs so we can re-execute later
                const currentIds = positions.map(p => p.id);
                if (lastSearch && lastSearch.command) {
                    sourceCommand = JSON.stringify({ command: lastSearch.command, position: lastSearch.position, ids: currentIds });
                } else {
                    // Fallback: store position IDs only
                    sourceCommand = JSON.stringify({ ids: currentIds });
                }
                sourceId = 0;
            }

            const deckId = await CreateAnkiDeck(newDeckName.trim(), '', sourceType, sourceId, sourceCommand);

            // Sync cards from source
            if (sourceType === 'collection') {
                await SyncAnkiDeck(deckId);
            } else {
                // For search decks, execute the search and sync with results
                const ids = await executeSearchForDeckSync(sourceCommand);
                if (ids.length > 0) {
                    await SyncAnkiDeckWithPositions(deckId, ids);
                }
            }

            newDeckName = '';
            newDeckSourceType = 'collection';
            newDeckSourceId = 0;
            showCreateForm = false;
            await loadDecks();
            statusBarTextStore.set(`Deck created`);
        } catch (e) {
            statusBarTextStore.set(`Error: ${e}`);
        }
    }

    async function deleteDeck(deck, event) {
        event.stopPropagation();
        try {
            await DeleteAnkiDeck(deck.id);
            if (selectedDeck && selectedDeck.id === deck.id) {
                ankiSelectedDeckStore_reset();
            }
            await loadDecks();
            statusBarTextStore.set(`Deck "${deck.name}" deleted`);
        } catch (e) {
            statusBarTextStore.set(`Error: ${e}`);
        }
    }

    function ankiSelectedDeckStore_reset() {
        selectedAnkiDeckStore.set(null);
        ankiReviewCardStore.set(null);
        ankiDeckStatsStore.set(null);
        ankiViewModeStore.set('list');
    }

    async function selectDeck(deck) {
        selectedAnkiDeckStore.set(deck);
        try {
            const s = await GetAnkiDeckStats(deck.id);
            ankiDeckStatsStore.set(s);
            // Load deck positions into positionsStore for statusbar count
            const deckPositions = await GetAnkiDeckPositions(deck.id);
            positionsStore.set(deckPositions || []);
            if (deckPositions && deckPositions.length > 0) {
                currentPositionIndexStore.set(0);
            }
        } catch (e) {
            console.error(e);
        }
    }

    function updatePositionIndex(positionId) {
        const idx = positions.findIndex(p => p.id === positionId);
        if (idx >= 0) {
            currentPositionIndexStore.set(idx);
        }
    }

    async function startReview() {
        if (!selectedDeck) return;
        try {
            // Re-sync to pick up new positions
            if (selectedDeck.sourceType === 'search' && selectedDeck.sourceCommand) {
                const ids = await executeSearchForDeckSync(selectedDeck.sourceCommand);
                if (ids.length > 0) {
                    await SyncAnkiDeckWithPositions(selectedDeck.id, ids);
                }
            } else {
                await SyncAnkiDeck(selectedDeck.id);
            }
            // Load deck positions for statusbar navigation
            const deckPositions = await GetAnkiDeckPositions(selectedDeck.id);
            positionsStore.set(deckPositions || []);
            const card = await GetNextAnkiCard(selectedDeck.id);
            if (!card) {
                statusBarTextStore.set('No cards due for review');
                ankiPausedSessionStore.set(null);
                return;
            }
            ankiReviewCardStore.set(card);
            // Display the card's position on the board
            positionStore.set(JSON.parse(JSON.stringify(card.position)));
            // Update position index in statusbar
            updatePositionIndex(card.position.id);
            // Restore session count if resuming the same deck
            if (pausedSession && pausedSession.deckId === selectedDeck.id) {
                reviewSessionCount = pausedSession.sessionCount;
            } else {
                reviewSessionCount = 0;
            }
            ankiPausedSessionStore.set(null);
            ankiViewModeStore.set('review');
        } catch (e) {
            statusBarTextStore.set(`Error: ${e}`);
        }
    }

    async function submitReview(rating) {
        if (!reviewCard) return;
        try {
            const nextCard = await ReviewAnkiCard(reviewCard.card.id, rating);
            reviewSessionCount++;
            if (nextCard) {
                ankiReviewCardStore.set(nextCard);
                // Display the next card's position on the board
                positionStore.set(JSON.parse(JSON.stringify(nextCard.position)));
                // Update position index in statusbar
                updatePositionIndex(nextCard.position.id);
            } else {
                ankiReviewCardStore.set(null);
                ankiPausedSessionStore.set(null);
                ankiViewModeStore.set('list');
                statusBarTextStore.set(`Review complete! ${reviewSessionCount} cards reviewed.`);
                await loadDecks();
                if (selectedDeck) {
                    const s = await GetAnkiDeckStats(selectedDeck.id);
                    ankiDeckStatsStore.set(s);
                }
            }
        } catch (e) {
            statusBarTextStore.set(`Error: ${e}`);
        }
    }

    async function openSettings() {
        if (!selectedDeck) return;
        settingsRetention = selectedDeck.requestRetention;
        settingsMaxInterval = selectedDeck.maximumInterval;
        settingsFuzz = selectedDeck.enableFuzz;
        ankiViewModeStore.set('settings');
    }

    async function saveSettings() {
        if (!selectedDeck) return;
        try {
            await UpdateAnkiDeckParams(selectedDeck.id, settingsRetention, settingsMaxInterval, settingsFuzz);
            await loadDecks();
            // Refresh selected deck
            const updated = decks.find(d => d.id === selectedDeck.id);
            if (updated) selectedAnkiDeckStore.set(updated);
            ankiViewModeStore.set('list');
            statusBarTextStore.set('Settings saved');
        } catch (e) {
            statusBarTextStore.set(`Error: ${e}`);
        }
    }

    function startEditing(deck, event) {
        event.stopPropagation();
        editingDeckId = deck.id;
        editingName = deck.name;
        editingDescription = deck.description;
    }

    async function finishEditing(deck) {
        if (editingDeckId !== deck.id) return;
        try {
            await UpdateAnkiDeck(deck.id, editingName.trim() || deck.name, editingDescription);
            editingDeckId = null;
            await loadDecks();
        } catch (e) {
            statusBarTextStore.set(`Error: ${e}`);
        }
    }

    function cancelEditing() {
        editingDeckId = null;
    }

    // Re-execute a stored search command and return matching position IDs
    async function executeSearchForDeckSync(sourceCommand) {
        try {
            let searchData;
            try {
                searchData = JSON.parse(sourceCommand);
            } catch {
                // Legacy format: comma-separated IDs
                return sourceCommand.split(',').map(s => parseInt(s.trim(), 10)).filter(n => !isNaN(n));
            }

            // Collect stored IDs as fallback (always preserve existing cards)
            const storedIds = Array.isArray(searchData.ids) ? searchData.ids : [];

            if (!searchData.command) {
                // No search command — use stored IDs only
                return storedIds;
            }

            const command = searchData.command;
            const position = searchData.position ? JSON.parse(searchData.position) : {};

            let searchIds = [];
            if (command === 's') {
                const results = await LoadPositionsByFilters(position, false, false, '', '', '', '', '', '', '', '', '', '', '', '', '', '', '', '', false, false, '', '', '', '', '', '', false, false, '', '', '', '');
                searchIds = (results || []).map(p => p.id);
            } else {
                const filters = command.slice(1).trim().split(' ').map(f => f.trim());
                const pf = parseFilters(filters, command);

                const results = await LoadPositionsByFilters(
                    position,
                    pf.includeCube, pf.includeScore,
                    pf.pipCountFilter || '', pf.winRateFilter || '', pf.gammonRateFilter || '',
                    pf.backgammonRateFilter || '', pf.player2WinRateFilter || '', pf.player2GammonRateFilter || '',
                    pf.player2BackgammonRateFilter || '', pf.player1CheckerOffFilter || '', pf.player2CheckerOffFilter || '',
                    pf.player1BackCheckerFilter || '', pf.player2BackCheckerFilter || '',
                    pf.player1CheckerInZoneFilter || '', pf.player2CheckerInZoneFilter || '',
                    pf.searchText || '', pf.player1AbsolutePipCountFilter || '', pf.equityFilter || '',
                    pf.decisionTypeFilter || false, pf.diceRollFilter || false, pf.movePatternFilter || '',
                    pf.dateFilter || '', pf.player1OutfieldBlotFilter || '', pf.player2OutfieldBlotFilter || '',
                    pf.player1JanBlotFilter || '', pf.player2JanBlotFilter || '',
                    pf.noContactFilter || false, pf.mirrorPositionFilter || false, pf.moveErrorFilter || '',
                    pf.matchIDsFilter || '', pf.tournamentIDsFilter || '', ''
                );
                searchIds = (results || []).map(p => p.id);
            }

            // Merge: use search results PLUS any stored IDs not already included
            // This ensures we never lose previously tracked cards
            const idSet = new Set(searchIds);
            for (const id of storedIds) {
                idSet.add(id);
            }
            return Array.from(idSet);
        } catch (e) {
            console.error('Error executing search for deck sync:', e);
            return [];
        }
    }

    async function syncDeck(deck, event) {
        event.stopPropagation();
        try {
            if (deck.sourceType === 'search' && deck.sourceCommand) {
                const ids = await executeSearchForDeckSync(deck.sourceCommand);
                if (ids.length > 0) {
                    await SyncAnkiDeckWithPositions(deck.id, ids);
                }
            } else {
                await SyncAnkiDeck(deck.id);
            }
            await loadDecks();
            statusBarTextStore.set(`Deck "${deck.name}" synced`);
        } catch (e) {
            statusBarTextStore.set(`Error: ${e}`);
        }
    }

    async function resetDeck(deck, event) {
        event.stopPropagation();
        try {
            await ResetAnkiDeck(deck.id);
            await loadDecks();
            if (selectedDeck && selectedDeck.id === deck.id) {
                const s = await GetAnkiDeckStats(deck.id);
                ankiDeckStatsStore.set(s);
            }
            statusBarTextStore.set(`Deck "${deck.name}" reset`);
        } catch (e) {
            statusBarTextStore.set(`Error: ${e}`);
        }
    }

    function backToList() {
        // Save paused session if we were reviewing
        if (viewMode === 'review' && selectedDeck) {
            ankiPausedSessionStore.set({ deckId: selectedDeck.id, sessionCount: reviewSessionCount });
        }
        ankiViewModeStore.set('list');
        // Refresh stats when returning to list
        if (selectedDeck) {
            GetAnkiDeckStats(selectedDeck.id).then(s => ankiDeckStatsStore.set(s)).catch(() => {});
            loadDecks();
        }
    }

    function getStateLabel(state) {
        switch (state) {
            case 0: return 'New';
            case 1: return 'Learning';
            case 2: return 'Review';
            case 3: return 'Relearning';
            default: return '?';
        }
    }

    function getSourceLabel(deck) {
        if (deck.sourceType === 'collection') {
            const coll = collections.find(c => c.id === deck.sourceId);
            return coll ? coll.name : `Collection #${deck.sourceId}`;
        }
        return getSearchCommandLabel(deck);
    }

    function getSearchCommandLabel(deck) {
        if (!deck.sourceCommand) return 'Search';
        try {
            const data = JSON.parse(deck.sourceCommand);
            if (data.command) return data.command;
        } catch {
            // Legacy comma-separated IDs
        }
        return 'Search';
    }

    // Reactive: reload and auto-sync all decks when tab becomes active,
    // and re-display the current review card position when returning to Anki tab during review
    $: if ($activeTabStore === 'anki' && databaseLoaded) {
        syncAllDecksAndReload();
        // If a review is in progress, re-display the current card's position
        if (viewMode === 'review' && reviewCard) {
            positionStore.set(JSON.parse(JSON.stringify(reviewCard.position)));
            updatePositionIndex(reviewCard.position.id);
        }
    }

    async function syncAllDecksAndReload() {
        try {
            const result = await GetAllAnkiDecks();
            const allDecks = result || [];
            // Auto-sync all decks to pick up new/removed positions
            for (const deck of allDecks) {
                try {
                    if (deck.sourceType === 'search' && deck.sourceCommand) {
                        const ids = await executeSearchForDeckSync(deck.sourceCommand);
                        if (ids.length > 0) {
                            await SyncAnkiDeckWithPositions(deck.id, ids);
                        }
                    } else {
                        await SyncAnkiDeck(deck.id);
                    }
                } catch (e) {
                    console.error(`Error syncing deck "${deck.name}":`, e);
                }
            }
            // Reload to reflect updated card counts
            await loadDecks();
        } catch (e) {
            console.error('Error auto-syncing decks:', e);
        }
    }
</script>

<div class="anki-panel">
    {#if viewMode === 'review' && reviewCard}
        <!-- Review Mode -->
        <div class="review-header">
            <button class="btn-back" on:click={backToList} title="Back to deck list (Esc)">
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M10.5 19.5 3 12m0 0 7.5-7.5M3 12h18" />
                </svg>
            </button>
            <span class="review-title">{selectedDeck?.name}</span>
            <span class="review-count">#{reviewSessionCount + 1}</span>
            <span class="card-state state-{reviewCard.card.state}">{getStateLabel(reviewCard.card.state)}</span>
        </div>

        <div class="review-body">
            <div class="review-position-id">Position #{reviewCard.position.id}</div>
            <div class="review-buttons">
                <button class="btn-rating btn-again" on:click={() => submitReview(1)} title="Again (1)">
                    <span class="rating-label">Again</span>
                    <span class="rating-key">1</span>
                </button>
                <button class="btn-rating btn-hard" on:click={() => submitReview(2)} title="Hard (2)">
                    <span class="rating-label">Hard</span>
                    <span class="rating-key">2</span>
                </button>
                <button class="btn-rating btn-good" on:click={() => submitReview(3)} title="Good (3)">
                    <span class="rating-label">Good</span>
                    <span class="rating-key">3</span>
                </button>
                <button class="btn-rating btn-easy" on:click={() => submitReview(4)} title="Easy (4)">
                    <span class="rating-label">Easy</span>
                    <span class="rating-key">4</span>
                </button>
            </div>
        </div>

    {:else if viewMode === 'settings' && selectedDeck}
        <!-- Settings Mode -->
        <div class="settings-header">
            <button class="btn-back" on:click={backToList} title="Back">
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M10.5 19.5 3 12m0 0 7.5-7.5M3 12h18" />
                </svg>
            </button>
            <span class="settings-title">Settings: {selectedDeck.name}</span>
        </div>
        <div class="settings-body">
            <div class="settings-row">
                <label>Retention target</label>
                <input type="number" bind:value={settingsRetention} min="0.7" max="0.99" step="0.01" />
                <span class="settings-hint">{Math.round(settingsRetention * 100)}%</span>
            </div>
            <div class="settings-row">
                <label>Max interval (days)</label>
                <input type="number" bind:value={settingsMaxInterval} min="1" max="36500" step="1" />
            </div>
            <div class="settings-row">
                <label>
                    <input type="checkbox" bind:checked={settingsFuzz} />
                    Enable fuzz (randomize intervals)
                </label>
            </div>
            <div class="settings-actions">
                <button class="btn-save" on:click={saveSettings}>Save</button>
                <button class="btn-cancel" on:click={backToList}>Cancel</button>
            </div>
        </div>

    {:else}
        <!-- Deck List Mode -->
        <div class="deck-toolbar">
            {#if !showCreateForm}
                <button class="btn-add" on:click={() => { showCreateForm = true; }} title="Create a new deck">
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
                    </svg>
                    New Deck
                </button>
            {:else}
                <div class="create-form">
                    <input
                        type="text"
                        bind:value={newDeckName}
                        placeholder="Deck name"
                        class="input-name"
                        on:keydown={(e) => { if (e.key === 'Enter') createDeck(); if (e.key === 'Escape') showCreateForm = false; }}
                    />
                    <select bind:value={newDeckSourceType} class="input-source-type">
                        <option value="collection">Collection</option>
                        <option value="search">Current search</option>
                    </select>
                    {#if newDeckSourceType === 'collection'}
                        <select bind:value={newDeckSourceId} class="input-source-id">
                            <option value={0}>-- Select collection --</option>
                            {#each collections as coll}
                                <option value={coll.id}>{coll.name} ({coll.positionCount})</option>
                            {/each}
                        </select>
                    {:else}
                        <span class="search-hint">{positions.length} positions</span>
                    {/if}
                    <button class="btn-confirm" on:click={createDeck} title="Create">
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14">
                            <path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" />
                        </svg>
                    </button>
                    <button class="btn-cancel-inline" on:click={() => { showCreateForm = false; }}>
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>
            {/if}
        </div>

        <div class="deck-list">
            {#if decks.length === 0}
                <div class="empty-state">
                    No decks yet. Create one from a collection or current search.
                </div>
            {:else}
                <table class="deck-table">
                    <thead>
                        <tr>
                            <th>Name</th>
                            <th>Description</th>
                            <th>Source</th>
                            <th class="narrow-col">Cards</th>
                            <th class="narrow-col">New</th>
                            <th class="narrow-col">Due</th>
                            <th class="actions-col">Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each decks as deck}
                            <tr
                                class:selected={selectedDeck && selectedDeck.id === deck.id}
                                on:click={() => selectDeck(deck)}
                                on:dblclick={() => { selectDeck(deck); startReview(); }}
                            >
                                {#if editingDeckId === deck.id}
                                    <td colspan="7">
                                        <div class="deck-edit">
                                            <input
                                                type="text"
                                                bind:value={editingName}
                                                class="edit-name"
                                                on:keydown={(e) => { if (e.key === 'Enter') finishEditing(deck); if (e.key === 'Escape') cancelEditing(); }}
                                            />
                                            <input
                                                type="text"
                                                bind:value={editingDescription}
                                                class="edit-desc"
                                                placeholder="Description"
                                                on:keydown={(e) => { if (e.key === 'Enter') finishEditing(deck); if (e.key === 'Escape') cancelEditing(); }}
                                            />
                                            <button class="icon-btn" on:click={() => finishEditing(deck)} title="Save">
                                                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="12" height="12">
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" />
                                                </svg>
                                            </button>
                                        </div>
                                    </td>
                                {:else}
                                    <td class="name-cell">
                                        <span class="deck-name">{deck.name}</span>
                                    </td>
                                    <td class="desc-cell">{deck.description || ''}</td>
                                    <td class="source-cell">{getSourceLabel(deck)}</td>
                                    <td class="narrow-col count-cell">{deck.cardCount}</td>
                                    <td class="narrow-col count-cell">{deck.newCount || ''}</td>
                                    <td class="narrow-col count-cell">{deck.dueCount || ''}</td>
                                    <td class="actions-col">
                                        <span class="item-actions">
                                            <button class="icon-btn" on:click={(e) => startEditing(deck, e)} title="Rename">
                                                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="12" height="12">
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="m16.862 4.487 1.687-1.688a1.875 1.875 0 1 1 2.652 2.652L6.832 19.82a4.5 4.5 0 0 1-1.897 1.13l-2.685.8.8-2.685a4.5 4.5 0 0 1 1.13-1.897L16.863 4.487Z" />
                                                </svg>
                                            </button>
                                            <button class="icon-btn" on:click={(e) => syncDeck(deck, e)} title="Sync cards from source">
                                                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="12" height="12">
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.992 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182" />
                                                </svg>
                                            </button>
                                            <button class="icon-btn delete" on:click={(e) => deleteDeck(deck, e)} title="Delete deck">
                                                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="12" height="12">
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
                                                </svg>
                                            </button>
                                        </span>
                                    </td>
                                {/if}
                            </tr>
                        {/each}
                    </tbody>
                </table>
            {/if}
        </div>

        <!-- Deck detail panel (shown when a deck is selected) -->
        {#if selectedDeck && stats}
            <div class="deck-detail">
                <div class="detail-stats">
                    <div class="stat-box stat-box-new">
                        <div class="stat-number">{stats.newCount}</div>
                        <div class="stat-label">New</div>
                    </div>
                    <div class="stat-box stat-box-learning">
                        <div class="stat-number">{stats.learningCount}</div>
                        <div class="stat-label">Learning</div>
                    </div>
                    <div class="stat-box stat-box-review">
                        <div class="stat-number">{stats.reviewCount}</div>
                        <div class="stat-label">Review</div>
                    </div>
                    <div class="stat-box stat-box-total">
                        <div class="stat-number">{stats.totalCount}</div>
                        <div class="stat-label">Total</div>
                    </div>
                </div>
                <div class="detail-actions">
                    <button class="btn-study" on:click={startReview} disabled={stats.dueCount === 0}>
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.347a1.125 1.125 0 0 1 0 1.972l-11.54 6.347a1.125 1.125 0 0 1-1.667-.986V5.653Z" />
                        </svg>
                        {#if pausedSession && pausedSession.deckId === selectedDeck.id}
                            Resume ({stats.dueCount} due, {pausedSession.sessionCount} reviewed)
                        {:else}
                            Study ({stats.dueCount} due)
                        {/if}
                    </button>
                    <button class="btn-settings" on:click={openSettings} title="Deck settings">
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 0 1 1.37.49l1.296 2.247a1.125 1.125 0 0 1-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 0 1 0 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 0 1-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 0 1-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 0 1-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 0 1-1.369-.49l-1.297-2.247a1.125 1.125 0 0 1 .26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 0 1 0-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 0 1-.26-1.43l1.297-2.247a1.125 1.125 0 0 1 1.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28Z" />
                            <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
                        </svg>
                    </button>
                    <button class="btn-icon btn-reset" on:click={(e) => resetDeck(selectedDeck, e)} title="Reset all cards to new">
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.992 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182" />
                        </svg>
                    </button>
                </div>
            </div>
        {/if}
    {/if}
</div>

<style>
    .anki-panel {
        display: flex;
        flex-direction: column;
        height: 100%;
        font-size: 12px;
        overflow: hidden;
        background: white;
        user-select: none;
        -webkit-user-select: none;
    }
    .anki-panel input, .anki-panel textarea { user-select: text; -webkit-user-select: text; }

    /* Deck toolbar */
    .deck-toolbar {
        display: flex;
        align-items: center;
        padding: 4px 8px;
        border-bottom: 1px solid #e0e0e0;
        background: #fafafa;
        flex-shrink: 0;
    }

    .btn-add {
        display: flex;
        align-items: center;
        gap: 4px;
        padding: 3px 8px;
        border: 1px solid #ccc;
        border-radius: 3px;
        background: #fff;
        cursor: pointer;
        font-size: 11px;
    }
    .btn-add:hover { background: #f0f0f0; }

    .create-form {
        display: flex;
        align-items: center;
        gap: 4px;
        flex: 1;
    }

    .input-name {
        flex: 1;
        padding: 2px 6px;
        border: 1px solid #ccc;
        border-radius: 3px;
        font-size: 11px;
        min-width: 80px;
    }

    .input-source-type, .input-source-id {
        padding: 2px 4px;
        border: 1px solid #ccc;
        border-radius: 3px;
        font-size: 11px;
    }

    .search-hint {
        font-size: 10px;
        color: #888;
    }

    .btn-confirm, .btn-cancel-inline {
        display: flex;
        align-items: center;
        padding: 2px 6px;
        border: 1px solid #ccc;
        border-radius: 3px;
        background: #fff;
        cursor: pointer;
    }
    .btn-confirm:hover { background: #f0f0f0; }
    .btn-cancel-inline:hover { background: #f0f0f0; }

    /* Deck table (matches CollectionPanel/MatchPanel pattern) */
    .deck-list {
        flex: 1;
        overflow-y: auto;
        overflow-x: hidden;
        min-height: 0;
    }

    .empty-state {
        padding: 16px;
        text-align: center;
        color: #bbb;
        font-size: 11px;
        font-style: italic;
    }

    .deck-table { width: 100%; border-collapse: collapse; font-size: 12px; }
    .deck-table thead { position: sticky; top: 0; background-color: #f5f5f5; z-index: 1; }
    .deck-table th, .deck-table td { padding: 4px 8px; text-align: left; border-bottom: 1px solid #e0e0e0; }
    .deck-table th { font-weight: 600; color: #333; font-size: 11px; }
    .narrow-col { width: 1px; white-space: nowrap; padding-left: 6px; padding-right: 6px; text-align: center; }
    .actions-col { width: 80px; min-width: 80px; max-width: 80px; white-space: nowrap; text-align: center; padding: 0 4px; }
    .deck-table tbody tr { transition: background-color 0.1s; cursor: pointer; }
    .deck-table tbody tr:hover { background-color: #f9f9f9; }
    .deck-table tbody tr.selected { background-color: #e3f2fd; }
    .deck-table tbody tr.selected:hover { background-color: #bbdefb; }

    .name-cell { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 0; }
    .deck-name { font-weight: 500; }
    .desc-cell { font-size: 11px; color: #888; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 0; }
    .source-cell { font-size: 11px; color: #666; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 0; font-family: monospace; }
    .count-cell { text-align: center; color: #666; }

    .item-actions { display: inline-flex; gap: 2px; vertical-align: middle; }
    .icon-btn { background: none; border: none; cursor: pointer; font-size: 12px; color: #666; padding: 2px 4px; line-height: 1; }
    .icon-btn:hover:not(:disabled) { color: #000; }
    .icon-btn.delete:hover:not(:disabled) { color: #c55; }

    .deck-edit {
        display: flex;
        align-items: center;
        gap: 4px;
        width: 100%;
    }

    .edit-name, .edit-desc {
        padding: 2px 4px;
        border: 1px solid #ccc;
        border-radius: 3px;
        font-size: 11px;
    }
    .edit-name { flex: 1; }
    .edit-desc { flex: 1; }

    /* Deck detail */
    .deck-detail {
        border-top: 1px solid #e0e0e0;
        padding: 6px 8px;
        background: #fafafa;
        flex-shrink: 0;
    }

    .detail-stats {
        display: flex;
        gap: 8px;
        margin-bottom: 6px;
    }

    .stat-box {
        flex: 1;
        text-align: center;
        padding: 3px;
        border-radius: 3px;
        background: #fff;
        border: 1px solid #e0e0e0;
    }

    .stat-number { font-size: 14px; font-weight: 600; color: #555; }
    .stat-label { font-size: 9px; color: #888; text-transform: uppercase; }

    .detail-actions {
        display: flex;
        gap: 6px;
        align-items: center;
    }

    .btn-study {
        display: flex;
        align-items: center;
        gap: 4px;
        padding: 4px 12px;
        border: none;
        border-radius: 3px;
        background: #6c757d;
        color: #fff;
        cursor: pointer;
        font-size: 12px;
        flex: 1;
        justify-content: center;
    }
    .btn-study:hover { background: #5a6268; }
    .btn-study:disabled { background: #ccc; cursor: default; }

    .btn-settings {
        display: flex;
        align-items: center;
        padding: 4px 6px;
        border: 1px solid #ccc;
        border-radius: 3px;
        background: #fff;
        cursor: pointer;
    }
    .btn-settings:hover { background: #f0f0f0; }

    .btn-reset {
        padding: 4px 6px;
    }

    /* Review mode */
    .review-header {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 5px 8px;
        background: #f5f5f5;
        border-bottom: 1px solid #e0e0e0;
        flex-shrink: 0;
    }

    .btn-back { background: none; border: none; cursor: pointer; font-size: 16px; color: #666; padding: 2px 6px; line-height: 1; }
    .btn-back:hover { color: #333; }

    .review-title { font-size: 13px; font-weight: 600; color: #333; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .review-count { font-size: 11px; color: #888; }

    .card-state {
        font-size: 9px;
        padding: 1px 6px;
        border-radius: 3px;
        font-weight: 500;
        background: #f0f0f0;
        color: #555;
    }

    .review-body {
        flex: 1;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 12px;
        gap: 8px;
    }

    .review-position-id {
        font-size: 11px;
        color: #888;
    }

    .review-buttons {
        display: flex;
        gap: 6px;
        width: 100%;
        max-width: 320px;
    }

    .btn-rating {
        flex: 1;
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 4px 4px;
        border: 1px solid #ddd;
        border-radius: 3px;
        cursor: pointer;
        background: #fff;
        gap: 2px;
    }
    .btn-rating:hover { background: #f5f5f5; }

    .rating-label { font-size: 11px; font-weight: 500; }
    .rating-key { font-size: 9px; color: #aaa; }

    /* Settings mode */
    .settings-header {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 5px 8px;
        background: #f5f5f5;
        border-bottom: 1px solid #e0e0e0;
        flex-shrink: 0;
    }

    .settings-title { font-size: 13px; font-weight: 600; color: #333; }

    .settings-body {
        padding: 12px;
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .settings-row {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .settings-row label {
        min-width: 140px;
        font-size: 11px;
        display: flex;
        align-items: center;
        gap: 4px;
    }

    .settings-row input[type="number"] {
        width: 80px;
        padding: 2px 6px;
        border: 1px solid #ccc;
        border-radius: 3px;
        font-size: 11px;
    }

    .settings-hint {
        font-size: 10px;
        color: #888;
    }

    .settings-actions {
        display: flex;
        gap: 8px;
        margin-top: 4px;
    }

    .btn-save {
        padding: 4px 16px;
        border: none;
        border-radius: 3px;
        background: #6c757d;
        color: #fff;
        cursor: pointer;
        font-size: 12px;
    }
    .btn-save:hover { background: #5a6268; }

    .btn-cancel {
        padding: 4px 16px;
        border: 1px solid #ccc;
        border-radius: 3px;
        background: #fff;
        cursor: pointer;
        font-size: 12px;
    }
    .btn-cancel:hover { background: #f0f0f0; }
</style>
