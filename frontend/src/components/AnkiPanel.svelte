<script>
    import { createInlineEdit } from '../utils/inlineEdit.svelte.js';
    import { onMount } from 'svelte';
    import {
        ankiDecksStore,
        selectedAnkiDeckStore,
        ankiReviewCardStore,
        ankiDeckStatsStore,
        ankiViewModeStore,
        ankiReviewActionStore,
        ankiPausedSessionStore,
        ankiAnswerShownStore,
        showAnkiAnswer,
        hideAnkiAnswer
    } from '../stores/ankiStore';
    import { analysisStore, selectedMoveStore } from '../stores/analysisStore';
    import { positionStore } from '../stores/positionStore';
    import { cubeTurnability, isMoneyPosition } from '../utils/cubeDecision.js';
    import { playedMovePredicate, playedCubeActionPredicate } from '../utils/playedMarks.js';
    import { statusBarTextStore, activeTabStore } from '../stores/uiStore';
    import { databaseLoadedStore } from '../stores/databaseStore';
    import { collectionsStore } from '../stores/collectionStore';
    import { positionsStore } from '../stores/positionStore';
    import { lastSearchStore } from '../stores/searchHistoryStore';
    import { confirmAction } from '../services/confirmService.js';
    import * as anki from '../services/ankiService.js';
    import { logger } from '../utils/logger.js';
    import { t, tMsg } from '../i18n';
    import { UpdateAnkiDeck } from '../../wailsjs/go/database/Database.js';
    import PanelTable from './panels/PanelTable.svelte';
    import AnalysisView from './AnalysisView.svelte';

    // Read-only mirrors of stores — declared as $derived so Svelte tracks
    // dependencies via $store reads (the project rule, see CLAUDE.md).
    let decks = $derived($ankiDecksStore || []);
    let selectedDeck = $derived($selectedAnkiDeckStore);
    let reviewCard = $derived($ankiReviewCardStore);
    let stats = $derived($ankiDeckStatsStore);
    let viewMode = $derived($ankiViewModeStore);
    let databaseLoaded = $derived($databaseLoadedStore);
    let collections = $derived($collectionsStore || []);
    let positionIds = $derived($positionsStore?.ids || []);
    let lastSearch = $derived($lastSearchStore);
    let pausedSession = $derived($ankiPausedSessionStore);
    let answerShown = $derived($ankiAnswerShownStore);

    // ── The answer of the card under review (ADR-0025) ──────────────────
    // The stored analysis of the card's position, loaded by showCard through
    // showPosition — never a live evaluation (rule 1).
    let analysis = $derived($analysisStore);
    // Which block the answer is: outside MATCH mode the record's own type
    // decides, exactly as the Analysis tab decides it there.
    let answerKind = $derived(analysis?.analysisType === 'DoublingCube' ? 'cube' : 'checker');
    let answerMoves = $derived(analysis?.checkerAnalysis?.moves ?? []);
    // Whether this card has an answer at all. A position can be in a deck
    // without a stored analysis (typed by hand, imported bare): that is an
    // ABSENT answer, not a hidden one, so it is named rather than masked
    // (rule 3's rejected alternative) — a mask revealing nothing would be a
    // lie the interface tells.
    let hasAnswer = $derived(answerKind === 'cube' ? cubeAnalysesCount(analysis) > 0 : answerMoves.length > 0);
    let turnability = $derived(cubeTurnability($positionStore));
    let cubeValue = $derived($positionStore?.cube?.value ?? 0);
    let onRoll = $derived($positionStore?.player_on_roll ?? 0);
    // ADR-0016 point 6 / #190/C.3: same referential and rule flags as
    // AnalysisPanel, read off the card's own position.
    let isMoney = $derived(isMoneyPosition($positionStore));
    let jacoby = $derived(isMoney && $positionStore?.has_jacoby === 1);
    let beaver = $derived(isMoney && $positionStore?.has_beaver === 1);

    // A review is never in MATCH mode, so every play recorded on this position
    // is highlighted — including the blunder that put the card in the deck.
    let isPlayedMove = $derived(playedMovePredicate(analysis));
    let isPlayedCubeAction = $derived(playedCubeActionPredicate(analysis));

    // Clicking a candidate move puts it on the board — the moment of learning,
    // and the same gesture as the Analysis tab. Toggles off on a second click.
    function handleMoveRowClick(move) {
        selectedMoveStore.set($selectedMoveStore === move.move ? null : move.move);
    }

    function cubeAnalysesCount(a) {
        if (!a) return 0;
        if (a.allCubeAnalyses && a.allCubeAnalyses.length > 0) return a.allCubeAnalyses.length;
        return a.doublingCubeAnalysis ? 1 : 0;
    }

    // Heroicons outlines, drawn by the `icon` snippet below.
    const ICON = {
        back: 'M10.5 19.5 3 12m0 0 7.5-7.5M3 12h18',
        plus: 'M12 4.5v15m7.5-7.5h-15',
        check: 'm4.5 12.75 6 6 9-13.5',
        cross: 'M6 18 18 6M6 6l12 12',
        edit: 'm16.862 4.487 1.687-1.688a1.875 1.875 0 1 1 2.652 2.652L6.832 19.82a4.5 4.5 0 0 1-1.897 1.13l-2.685.8.8-2.685a4.5 4.5 0 0 1 1.13-1.897L16.863 4.487Z',
        sync: 'M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.992 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182',
        trash: 'm14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0',
        play: 'M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.347a1.125 1.125 0 0 1 0 1.972l-11.54 6.347a1.125 1.125 0 0 1-1.667-.986V5.653Z',
        gear: [
            'M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 0 1 1.37.49l1.296 2.247a1.125 1.125 0 0 1-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 0 1 0 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 0 1-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 0 1-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 0 1-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 0 1-1.369-.49l-1.297-2.247a1.125 1.125 0 0 1 .26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 0 1 0-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 0 1-.26-1.43l1.297-2.247a1.125 1.125 0 0 1 1.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28Z',
            'M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z'
        ]
    };

    // Create deck form
    let newDeckName = $state('');
    let newDeckSourceType = $state('collection');
    let newDeckSourceId = $state(0);
    let showCreateForm = $state(false);

    // Edit deck (name + description inline)
    const deckEdit = createInlineEdit({
        onSave: async (deckId, draft) => {
            const deck = decks.find((d) => d.id === deckId);
            if (!deck) return;
            try {
                await UpdateAnkiDeck(deck.id, draft.name.trim() || deck.name, draft.description);
                await anki.loadDecks();
            } catch (e) {
                statusBarTextStore.set(tMsg('common.errorWithMsg', { msg: e }));
            }
        }
    });

    // Settings
    let settingsRetention = $state(0.9);
    let settingsMaxInterval = $state(36500);
    let settingsFuzz = $state(true);
    // Session limit as the form holds it: a checkbox for "limited at all" plus
    // a number. Two controls because nil and 0 are different states — one field
    // alone would make "no limit" and "serve nothing" the same value
    // (ADR-0026 rule 3).
    let settingsLimited = $state(false);
    let settingsSessionLimit = $state(20);
    // What the deck's log measures, read when the settings view opens. Null
    // until it answers, and reported as unavailable below the sample floor.
    let retention = $state(null);

    const deckColumns = $derived([
        { key: 'name', label: $t('anki.colName') },
        { key: 'description', label: $t('anki.colDescription') },
        { key: 'source', label: $t('anki.colSource') },
        { key: 'cards', label: $t('anki.colCards'), narrow: true, align: 'center' },
        { key: 'new', label: $t('anki.colNew'), narrow: true, align: 'center' },
        { key: 'due', label: $t('anki.colDue'), narrow: true, align: 'center' },
        { key: 'actions', label: $t('anki.colActions'), actions: true }
    ]);

    // Review state
    let reviewSessionCount = $state(0);
    // Cram (free drill): serves random cards without touching the FSRS schedule.
    let cramMode = $state(false);

    // Listen for review key actions routed from App.svelte
    $effect(() => {
        const v = $ankiReviewActionStore;
        if (v !== null) {
            ankiReviewActionStore.set(null);
            if (v === 'back') {
                backToList();
            } else if (typeof v === 'number' && v >= 1 && v <= 4) {
                submitReview(v);
            }
        }
    });

    // Reload and auto-sync all decks when the tab becomes active, and put the
    // current review card back on the board when returning mid-review.
    $effect(() => {
        if ($activeTabStore === 'anki' && databaseLoaded) {
            anki.syncAllDecksAndReload();
            if (viewMode === 'review' && reviewCard) anki.showCard(reviewCard);
        }
    });

    onMount(() => {
        if (databaseLoaded) loadDecks();
    });

    async function loadDecks() {
        try {
            await anki.loadDecks();
        } catch (e) {
            logger.error('Error loading anki decks:', e);
        }
    }

    function fail(e) {
        statusBarTextStore.set(tMsg('common.errorWithMsg', { msg: e }));
    }

    async function createDeck() {
        if (!newDeckName.trim()) return;
        try {
            await anki.createDeck({
                name: newDeckName.trim(),
                sourceType: newDeckSourceType,
                sourceId: newDeckSourceId,
                lastSearch,
                positionIds
            });
            newDeckName = '';
            newDeckSourceType = 'collection';
            newDeckSourceId = 0;
            showCreateForm = false;
            statusBarTextStore.set(tMsg('anki.deckCreated'));
        } catch (e) {
            fail(e);
        }
    }

    async function deleteDeck(deck, event) {
        event.stopPropagation();
        if (!(await confirmAction($t('anki.confirmDeleteDeck', { name: deck.name }), { confirmLabel: $t('common.delete') }))) return;
        try {
            await anki.deleteDeck(deck.id);
            statusBarTextStore.set(tMsg('anki.deckDeleted', { name: deck.name }));
        } catch (e) {
            fail(e);
        }
    }

    async function selectDeck(deck) {
        try {
            await anki.selectDeck(deck);
        } catch (e) {
            logger.error(e);
        }
    }

    // A study session walks the due cards through FSRS; a cram session draws
    // random cards and never schedules anything.
    async function startSession(cram) {
        if (!selectedDeck) return;
        // A limit of 0 serves nothing, which is a state of its own and not a
        // mistake (ADR-0026 rule 3). Said rather than shown as an empty deck.
        if (!cram && anki.sessionLimitReached(selectedDeck, 0)) {
            statusBarTextStore.set(tMsg('anki.sessionLimitZero'));
            return;
        }
        cramMode = cram;
        try {
            const card = await anki.startSession(selectedDeck, { cram });
            if (!card) {
                statusBarTextStore.set(tMsg(cram ? 'anki.deckEmpty' : 'anki.noCardsDue'));
                cramMode = false;
                if (!cram) ankiPausedSessionStore.set(null);
                return;
            }
            reviewSessionCount = anki.resumedSessionCount(pausedSession, selectedDeck, cram);
            ankiPausedSessionStore.set(null);
            ankiViewModeStore.set('review');
        } catch (e) {
            cramMode = false;
            fail(e);
        }
    }

    async function submitReview(rating) {
        if (!reviewCard) return;
        try {
            // In cram mode the rating is ignored — just advance, never schedule.
            const next = cramMode ? await anki.nextCramCard(selectedDeck, reviewCard) : await anki.reviewCard(reviewCard, rating);
            reviewSessionCount++;

            // The session limit stops the sitting here, and says so: reusing
            // the ordinary "review complete" message would claim the queue is
            // empty when the limit is what emptied it (ADR-0026 rule 4). No
            // "keep going" button — cram already serves more positions, and
            // unlike an override it schedules nothing.
            if (next && anki.sessionLimitReached(selectedDeck, reviewSessionCount, { cram: cramMode })) {
                ankiViewModeStore.set('list');
                ankiPausedSessionStore.set(null);
                await anki.loadDecks();
                const fresh = selectedDeck ? await anki.refreshDeckStats(selectedDeck.id) : null;
                statusBarTextStore.set(
                    tMsg('anki.sessionLimitReached', {
                        count: reviewSessionCount,
                        remaining: fresh?.dueCount ?? stats?.dueCount ?? 0
                    })
                );
                return;
            }
            if (next) return;
            ankiViewModeStore.set('list');
            if (cramMode) {
                cramMode = false;
                return;
            }
            ankiPausedSessionStore.set(null);
            statusBarTextStore.set(tMsg('anki.reviewComplete', { count: reviewSessionCount }));
            await anki.loadDecks();
            if (selectedDeck) await anki.refreshDeckStats(selectedDeck.id);
        } catch (e) {
            fail(e);
        }
    }

    function openSettings() {
        if (!selectedDeck) return;
        settingsRetention = selectedDeck.requestRetention;
        settingsMaxInterval = selectedDeck.maximumInterval;
        settingsFuzz = selectedDeck.enableFuzz;
        const limit = anki.sessionLimitOf(selectedDeck);
        settingsLimited = limit !== null;
        settingsSessionLimit = limit === null ? 20 : limit;
        retention = null;
        anki.deckRetention(selectedDeck.id)
            .then((r) => (retention = r))
            .catch(() => (retention = null));
        ankiViewModeStore.set('settings');
    }

    async function saveSettings() {
        if (!selectedDeck) return;
        try {
            await anki.saveDeckParams(selectedDeck.id, {
                requestRetention: settingsRetention,
                maximumInterval: settingsMaxInterval,
                enableFuzz: settingsFuzz,
                sessionLimit: settingsLimited ? Math.max(0, Math.trunc(settingsSessionLimit)) : null
            });
            ankiViewModeStore.set('list');
            statusBarTextStore.set(tMsg('anki.settingsSaved'));
        } catch (e) {
            fail(e);
        }
    }

    function startEditing(deck, event) {
        event.stopPropagation();
        deckEdit.start(deck.id, { name: deck.name, description: deck.description || '' });
    }

    async function syncDeck(deck, event) {
        event.stopPropagation();
        try {
            await anki.syncDeckCards(deck);
            await anki.loadDecks();
            statusBarTextStore.set(tMsg('anki.deckSynced', { name: deck.name }));
        } catch (e) {
            fail(e);
        }
    }

    async function resetDeck(deck, event) {
        event.stopPropagation();
        if (!(await confirmAction($t('anki.confirmResetDeck', { name: deck.name }), { confirmLabel: $t('common.reset') }))) return;
        try {
            await anki.resetDeck(deck.id);
            statusBarTextStore.set(tMsg('anki.deckReset', { name: deck.name }));
        } catch (e) {
            fail(e);
        }
    }

    function backToList() {
        // Save a paused session if we were reviewing — but cram never
        // schedules, so it leaves no resumable session.
        if (viewMode === 'review' && selectedDeck && !cramMode) {
            ankiPausedSessionStore.set({ deckId: selectedDeck.id, sessionCount: reviewSessionCount });
        }
        cramMode = false;
        // Leaving the review is a change of question (ADR-0025 rule 5). The
        // move selection must go with it: left set, selectedMoveStore freezes
        // j/k position browsing app-wide — the regression AnalysisPanel's
        // onDestroy already had to fix once.
        hideAnkiAnswer();
        selectedMoveStore.set(null);
        ankiViewModeStore.set('list');
        if (selectedDeck) {
            anki.refreshDeckStats(selectedDeck.id).catch(() => {});
            loadDecks();
        }
    }
</script>

{#snippet icon(path, size = 14)}
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width={size} height={size}>
        {#each Array.isArray(path) ? path : [path] as d, i (i)}
            <path stroke-linecap="round" stroke-linejoin="round" {d} />
        {/each}
    </svg>
{/snippet}

<div class="anki-panel">
    {#if viewMode === 'review' && reviewCard}
        <!-- Review Mode -->
        <div class="view-header">
            <button class="btn-back" onclick={backToList} title={$t('anki.backToDeckList') + ' (Esc)'}>{@render icon(ICON.back)}</button>
            <span class="view-title">{selectedDeck?.name}</span>
            <span class="review-count">#{reviewSessionCount + 1}</span>
            {#if cramMode}
                <span class="card-state state-cram">{$t('anki.cramBadge')}</span>
            {:else}
                <span class="card-state state-{reviewCard.card.state}">{anki.stateLabel(reviewCard.card.state)}</span>
            {/if}
        </div>

        <!-- The grading strip stays put and the answer scrolls under it
             (ADR-0025): a checker analysis can run twenty rows, and in a side
             column it would push the buttons out of reach on every card. -->
        <div class="review-body">
            <div class="review-strip">
                <div class="review-position-id">{$t('anki.positionNumber', { id: reviewCard.position.id })}</div>
                {#if cramMode}
                    <div class="review-buttons">
                        <button class="btn-rating" onclick={() => submitReview(1)} title={$t('anki.next') + ' (1-4)'}>
                            <span class="rating-label">{$t('anki.next')}</span>
                        </button>
                    </div>
                {:else}
                    <div class="review-buttons">
                        {#each [['anki.again', 1], ['anki.hard', 2], ['anki.good', 3], ['anki.easy', 4]] as [key, rating] (rating)}
                            <button class="btn-rating" onclick={() => submitReview(rating)} title={$t(key) + ` (${rating})`}>
                                <span class="rating-label">{$t(key)}</span>
                                <span class="rating-key">{rating}</span>
                            </button>
                        {/each}
                    </div>
                {/if}
            </div>

            <div class="review-answer">
                {#if !hasAnswer}
                    <div class="answer-absent">{$t('anki.noAnalysis')}</div>
                {:else if answerShown}
                    <AnalysisView
                        {analysis}
                        kind={answerKind}
                        {turnability}
                        {cubeValue}
                        {onRoll}
                        moves={answerMoves}
                        selectedMove={$selectedMoveStore}
                        {isPlayedMove}
                        {isPlayedCubeAction}
                        onRowClick={handleMoveRowClick}
                        {isMoney}
                        {jacoby}
                        {beaver}
                    />
                {:else}
                    <button class="answer-masked" onclick={showAnkiAnswer} title={$t('anki.clickToReveal')}>···</button>
                {/if}
            </div>
        </div>
    {:else if viewMode === 'settings' && selectedDeck}
        <!-- Settings Mode -->
        <div class="view-header">
            <button class="btn-back" onclick={backToList} title={$t('common.back')}>{@render icon(ICON.back)}</button>
            <span class="view-title">{$t('anki.settingsTitle', { name: selectedDeck.name })}</span>
        </div>
        <div class="settings-body">
            <div class="settings-row">
                <label>{$t('anki.retentionTarget')}</label>
                <input type="number" bind:value={settingsRetention} min="0.7" max="0.99" step="0.01" />
                <span class="settings-hint">{Math.round(settingsRetention * 100)}%</span>
            </div>
            <!-- The target is a choice; this is its outcome, shown and never
                 acted upon (ADR-0026 rule 5). -->
            <div class="settings-note">
                {#if retention && retention.sampleSize >= anki.RETENTION_MIN_SAMPLE}
                    {$t('anki.retentionMeasured', { measured: Math.round(retention.observedRetention * 100), sample: retention.sampleSize })}
                {:else if retention}
                    {$t('anki.retentionNotEnough', { sample: retention.sampleSize, needed: anki.RETENTION_MIN_SAMPLE })}
                {/if}
            </div>
            <!-- Without this line the change looks broken: it takes effect one
                 review at a time and moves no existing due date (rule 8). -->
            <div class="settings-note">{$t('anki.retentionNotRetroactive')}</div>
            <div class="settings-row">
                <label>{$t('anki.maxInterval')}</label>
                <input type="number" bind:value={settingsMaxInterval} min="1" max="36500" step="1" />
            </div>
            <div class="settings-row">
                <label>
                    <input type="checkbox" bind:checked={settingsFuzz} />
                    {$t('anki.enableFuzz')}
                </label>
            </div>
            <div class="settings-row">
                <label>
                    <input type="checkbox" bind:checked={settingsLimited} />
                    {$t('anki.limitSession')}
                </label>
                {#if settingsLimited}
                    <input type="number" bind:value={settingsSessionLimit} min="0" max="9999" step="1" />
                {/if}
            </div>
            <div class="settings-note">{$t('anki.limitSessionHint')}</div>
            <div class="settings-actions">
                <button class="btn-primary wide" onclick={saveSettings}>{$t('common.save')}</button>
                <button class="btn-outline wide" onclick={backToList}>{$t('common.cancel')}</button>
            </div>
        </div>
    {:else}
        <!-- Deck List Mode -->
        <div class="deck-toolbar">
            {#if !showCreateForm}
                <button class="btn-outline" onclick={() => (showCreateForm = true)} title={$t('anki.createNewDeckTooltip')}>
                    {@render icon(ICON.plus)}
                    {$t('anki.newDeck')}
                </button>
            {:else}
                <div class="create-form">
                    <input
                        type="text"
                        bind:value={newDeckName}
                        placeholder={$t('anki.deckNamePlaceholder')}
                        class="input-name"
                        onkeydown={(e) => {
                            if (e.key === 'Enter') createDeck();
                            if (e.key === 'Escape') showCreateForm = false;
                        }}
                    />
                    <select bind:value={newDeckSourceType} class="input-source">
                        <option value="collection">{$t('anki.sourceCollection')}</option>
                        <option value="search">{$t('anki.sourceCurrentSearch')}</option>
                    </select>
                    {#if newDeckSourceType === 'collection'}
                        <select bind:value={newDeckSourceId} class="input-source">
                            <option value={0}>{$t('anki.selectCollection')}</option>
                            {#each collections as coll (coll.id)}
                                <option value={coll.id}>{coll.name} ({coll.positionCount})</option>
                            {/each}
                        </select>
                    {:else}
                        <span class="search-hint">{$t('anki.positionsCount', { count: positionIds.length })}</span>
                    {/if}
                    <button class="btn-outline" onclick={createDeck} title={$t('common.create')}>{@render icon(ICON.check)}</button>
                    <button class="btn-outline" onclick={() => (showCreateForm = false)}>{@render icon(ICON.cross)}</button>
                </div>
            {/if}
        </div>

        <PanelTable
            rows={decks}
            columns={deckColumns}
            selectedKey={selectedDeck?.id}
            pointerRows
            onSelect={(deck) => selectDeck(deck)}
            onActivate={(deck) => {
                selectDeck(deck);
                startSession(false);
            }}
            emptyText={$t('anki.empty')}
        >
            {#snippet cells(deck)}
                {#if deckEdit.isEditing(deck.id)}
                    <td colspan="7">
                        <div class="deck-edit">
                            <input type="text" bind:value={deckEdit.draft.name} class="edit-field" onkeydown={deckEdit.onKeyDown} />
                            <input type="text" bind:value={deckEdit.draft.description} class="edit-field" placeholder={$t('anki.colDescription')} onkeydown={deckEdit.onKeyDown} />
                            <button class="icon-btn" onclick={() => deckEdit.save()} title={$t('common.save')}>{@render icon(ICON.check, 12)}</button>
                        </div>
                    </td>
                {:else}
                    <td class="name-cell"><span class="deck-name">{deck.name}</span></td>
                    <td class="desc-cell">{deck.description || ''}</td>
                    <td class="source-cell">{anki.sourceLabel(deck, collections)}</td>
                    <td class="narrow-col count-cell">{deck.cardCount}</td>
                    <td class="narrow-col count-cell">{deck.newCount || ''}</td>
                    <td class="narrow-col count-cell">{deck.dueCount || ''}</td>
                    <td class="actions-col">
                        <span class="item-actions">
                            <button class="icon-btn" onclick={(e) => startEditing(deck, e)} title={$t('anki.renameTooltip')}>{@render icon(ICON.edit, 12)}</button>
                            <button class="icon-btn" onclick={(e) => syncDeck(deck, e)} title={$t('anki.syncTooltip')}>{@render icon(ICON.sync, 12)}</button>
                            <button class="icon-btn delete" onclick={(e) => deleteDeck(deck, e)} title={$t('anki.deleteDeckTooltip')}>{@render icon(ICON.trash, 12)}</button>
                        </span>
                    </td>
                {/if}
            {/snippet}
        </PanelTable>

        <!-- Deck detail panel (shown when a deck is selected) -->
        {#if selectedDeck && stats}
            <div class="deck-detail">
                <div class="detail-stats">
                    {#each [['newCount', 'anki.statNew'], ['learningCount', 'anki.statLearning'], ['reviewCount', 'anki.statReview'], ['totalCount', 'anki.statTotal']] as [field, labelKey] (field)}
                        <div class="stat-box">
                            <div class="stat-number">{stats[field]}</div>
                            <div class="stat-label">{$t(labelKey)}</div>
                        </div>
                    {/each}
                </div>
                <div class="detail-actions">
                    <button class="btn-primary btn-study" onclick={() => startSession(false)} disabled={!anki.canStudy(stats, selectedDeck)}>
                        {@render icon(ICON.play)}
                        {#if pausedSession && pausedSession.deckId === selectedDeck.id}
                            {$t('anki.resume', { due: stats.dueCount, reviewed: pausedSession.sessionCount })}
                        {:else}
                            {$t('anki.study', { due: stats.dueCount })}
                        {/if}
                    </button>
                    <button class="btn-cram" onclick={() => startSession(true)} disabled={!anki.canCram(stats)} title={$t('anki.cramTooltip')}>
                        {@render icon(ICON.sync)}
                        {$t('anki.cram')}
                    </button>
                    <button class="btn-outline" onclick={openSettings} title={$t('anki.deckSettingsTooltip')}>{@render icon(ICON.gear)}</button>
                    <button class="btn-outline" onclick={(e) => resetDeck(selectedDeck, e)} title={$t('anki.resetTooltip')}>{@render icon(ICON.sync)}</button>
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
        font-size: var(--font-size-base);
        overflow: hidden;
        background: white;
        user-select: none;
        -webkit-user-select: none;
    }
    .anki-panel input {
        user-select: text;
        -webkit-user-select: text;
    }

    /* --- Buttons: one outlined family, one filled --- */
    .btn-outline {
        display: flex;
        align-items: center;
        gap: 4px;
        padding: 3px 8px;
        border: 1px solid var(--color-border);
        border-radius: 3px;
        background: #fff;
        cursor: pointer;
        font-size: var(--font-size-small);
    }
    .btn-outline:hover {
        background: #f0f0f0;
    }

    .btn-primary {
        display: flex;
        align-items: center;
        gap: 4px;
        padding: 4px 12px;
        border: none;
        border-radius: 3px;
        background: #6c757d;
        color: #fff;
        cursor: pointer;
        font-size: var(--font-size-base);
    }
    .btn-primary:hover {
        background: #5a6268;
    }
    .btn-primary:disabled {
        background: #ccc;
        cursor: default;
    }

    .wide {
        padding: 4px 16px;
        font-size: var(--font-size-base);
    }

    /* --- Deck toolbar --- */
    .deck-toolbar {
        display: flex;
        align-items: center;
        padding: 4px 8px;
        border-bottom: 1px solid #e0e0e0;
        background: #fafafa;
        flex-shrink: 0;
    }

    .create-form {
        display: flex;
        align-items: center;
        gap: 4px;
        flex: 1;
    }

    .input-name,
    .input-source,
    .edit-field {
        padding: 2px 6px;
        border: 1px solid var(--color-border);
        border-radius: 3px;
        font-size: var(--font-size-small);
    }
    .input-name,
    .edit-field {
        flex: 1;
        min-width: 80px;
    }

    .search-hint {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    /* --- Deck table cells --- */
    .name-cell,
    .desc-cell,
    .source-cell {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        max-width: 0;
    }
    .deck-name {
        font-weight: 500;
    }
    .desc-cell {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }
    .source-cell {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        font-family: var(--font-family-mono);
    }

    .deck-edit {
        display: flex;
        align-items: center;
        gap: 4px;
        width: 100%;
    }

    /* --- Deck detail --- */
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

    .stat-number {
        font-size: var(--font-size-base);
        font-weight: 600;
        color: #555;
    }
    .stat-label {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        text-transform: uppercase;
    }

    .detail-actions {
        display: flex;
        gap: 6px;
        align-items: center;
    }

    .btn-study {
        flex: 1;
        justify-content: center;
    }

    .btn-cram {
        display: flex;
        align-items: center;
        gap: 4px;
        padding: 4px 12px;
        border: 1px solid #17a2b8;
        border-radius: 3px;
        background: #fff;
        color: #17a2b8;
        cursor: pointer;
        font-size: var(--font-size-base);
        justify-content: center;
    }
    .btn-cram:hover {
        background: #e8f7fa;
    }
    .btn-cram:disabled {
        border-color: var(--color-border);
        color: #ccc;
        cursor: default;
    }

    /* --- Review and settings views --- */
    .view-header {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 5px 8px;
        background: #f5f5f5;
        border-bottom: 1px solid #e0e0e0;
        flex-shrink: 0;
    }

    .btn-back {
        background: none;
        border: none;
        cursor: pointer;
        font-size: var(--font-size-title);
        color: var(--color-text-muted);
        padding: 2px 6px;
        line-height: 1;
    }
    .btn-back:hover {
        color: var(--color-text);
    }

    .view-title {
        font-size: var(--font-size-base);
        font-weight: 600;
        color: var(--color-text);
        flex: 1;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .review-count {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .card-state {
        font-size: var(--font-size-small);
        padding: 1px 6px;
        border-radius: 3px;
        font-weight: 500;
        background: #f0f0f0;
        color: #555;
    }
    .state-cram {
        background: #17a2b8;
        color: #fff;
    }

    .review-body {
        flex: 1;
        display: flex;
        flex-direction: column;
        padding: 12px;
        gap: 8px;
        min-height: 0;
        /* Query container so the revealed answer lays itself out on the panel's
           own width — bottom band or side column — exactly as it does in the
           Analysis tab. */
        container-type: inline-size;
    }

    /* The grading strip: fixed, above the answer, never scrolled away. */
    .review-strip {
        flex: 0 0 auto;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 8px;
    }

    .review-position-id {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .review-buttons {
        display: flex;
        gap: 6px;
        width: 100%;
        max-width: 320px;
    }

    /* The answer, below the strip: the only part that scrolls, on both axes.
       A candidate table is wider than a side column, and clipping it would
       hide the very columns being revealed — wide content scrolls inside its
       own box, it is never cut off. Content sits at the TOP: centred in a tall
       side column, the answer floated half a panel away from the buttons that
       grade it. */
    .review-answer {
        flex: 1;
        min-height: 0;
        overflow: auto;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: flex-start;
        padding-top: 10px;
    }

    /* One opaque stand-in for the whole answer (ADR-0025 rule 3), not the Eval
       panel's in-place mask: a checker table's rows ARE the moves, ordered by
       equity, so masking in place would leave the answer on the first line. */
    .answer-masked {
        width: 100%;
        max-width: 320px;
        padding: 14px 0;
        border: 1px dashed #ccc;
        border-radius: 3px;
        background: #f7f7f7;
        color: var(--color-text-muted);
        letter-spacing: 3px;
        cursor: pointer;
    }
    .answer-masked:hover {
        background: #eee;
        color: var(--color-text-muted);
    }

    .answer-absent {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
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
    .btn-rating:hover {
        background: #f5f5f5;
    }

    .rating-label {
        font-size: var(--font-size-small);
        font-weight: 500;
    }
    .rating-key {
        font-size: var(--font-size-small);
        color: #aaa;
    }

    .settings-body {
        padding: 12px;
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .settings-note {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        margin: -2px 0 6px;
    }

    .settings-row {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .settings-row label {
        min-width: 140px;
        font-size: var(--font-size-small);
        display: flex;
        align-items: center;
        gap: 4px;
    }

    .settings-row input[type='number'] {
        width: 80px;
        padding: 2px 6px;
        border: 1px solid var(--color-border);
        border-radius: 3px;
        font-size: var(--font-size-small);
    }

    .settings-hint {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .settings-actions {
        display: flex;
        gap: 8px;
        margin-top: 4px;
    }
</style>
