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
    import { UpdateAnkiDeck, GetAnkiReviewLog } from '../../wailsjs/go/database/Database.js';
    import PanelTable from './panels/PanelTable.svelte';
    import ContextMenu from './ContextMenu.svelte';
    import AnalysisView from './AnalysisView.svelte';
    import ScoreCard from './ScoreCard.svelte';
    import { buildScoreCard, UNORDERED_SCORES } from '../services/scoreCard.js';

    // Read-only store mirrors.
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

    // The answer of the card under review (ADR-0025). Board/analysis reads below
    // concern position cards; a score card's answer is the Training tab's sheet (ADR-0042).
    let isScoreCard = $derived(anki.isScoreCard(reviewCard));
    let scoreAways = $derived(anki.scoreCardAways(reviewCard));
    let scoreSheet = $derived(scoreAways ? buildScoreCard(scoreAways[0], scoreAways[1]) : null);
    // The stored analysis, never a live evaluation (rule 1).
    let analysis = $derived($analysisStore);
    // The record's own type picks the block, as in the Analysis tab outside MATCH mode.
    let answerKind = $derived(analysis?.analysisType === 'DoublingCube' ? 'cube' : 'checker');
    let answerMoves = $derived(analysis?.checkerAnalysis?.moves ?? []);
    // A position without stored analysis has an absent answer: named, not masked (rule 3).
    let hasAnswer = $derived(isScoreCard ? scoreSheet !== null : answerKind === 'cube' ? cubeAnalysesCount(analysis) > 0 : answerMoves.length > 0);
    let turnability = $derived(cubeTurnability($positionStore));
    let cubeValue = $derived($positionStore?.cube?.value ?? 0);
    let onRoll = $derived($positionStore?.player_on_roll ?? 0);
    // Referential and rule flags as in AnalysisPanel (ADR-0016 point 6).
    let isMoney = $derived(isMoneyPosition($positionStore));
    let jacoby = $derived(isMoney && $positionStore?.has_jacoby === 1);
    let beaver = $derived(isMoney && $positionStore?.has_beaver === 1);
    // The cube ceiling applies at any score: read off the position.
    let maxCube = $derived($positionStore?.max_cube ?? 0);

    // Never MATCH mode: every recorded play is highlighted, the card's blunder included.
    let isPlayedMove = $derived(playedMovePredicate(analysis));
    let isPlayedCubeAction = $derived(playedCubeActionPredicate(analysis));

    // A candidate click shows it on the board, as in the Analysis tab; a second click clears it.
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
    // Ids tying the settings labels to their inputs.
    const settingsId = $props.id();
    let settingsRetention = $state(0.9);
    let settingsMaxInterval = $state(36500);
    let settingsFuzz = $state(true);
    // Checkbox + number: nil (no limit) and 0 (serve nothing) differ (ADR-0026 rule 3).
    let settingsLimited = $state(false);
    let settingsSessionLimit = $state(20);
    // Deck log measurement; null until loaded, unavailable below the sample floor.
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

    // The review log: what the scheduler was told, the only place a mistaken
    // grade shows, since the schedule stays out of reach (ADR-0026).
    const REVIEW_LOG_LIMIT = 200;
    /** @type {any[]} */
    let reviewLog = $state([]);

    const reviewLogColumns = $derived([
        { key: 'reviewedAt', label: $t('anki.colReviewedAt') },
        { key: 'subject', label: $t('anki.colSubject'), narrow: true, align: 'center' },
        { key: 'rating', label: $t('anki.colRating'), narrow: true, align: 'center' },
        { key: 'state', label: $t('anki.colState'), narrow: true, align: 'center' },
        { key: 'scheduledDays', label: $t('anki.colInterval'), narrow: true, align: 'center' }
    ]);

    const RATING_KEYS = ['', 'anki.again', 'anki.hard', 'anki.good', 'anki.easy'];
    /** @type {[string, number][]} The four review buttons: i18n key, rating. */
    const RATING_BUTTONS = [
        ['anki.again', 1],
        ['anki.hard', 2],
        ['anki.good', 3],
        ['anki.easy', 4]
    ];

    let reviewLogRows = $derived(
        reviewLog.map((/** @type {any} */ e) => ({
            id: e.id,
            reviewedAt: (e.reviewedAt || '').replace('T', ' ').slice(0, 19),
            // Position number or score (ADR-0042); falls back on the id for older journals.
            subject: e.key || e.positionId,
            rating: RATING_KEYS[e.rating] ? $t(RATING_KEYS[e.rating]) : String(e.rating),
            state: anki.stateLabel(e.state),
            // Days granted; 0 = back in the same session, not missing.
            scheduledDays: `${e.scheduledDays} ${$t('anki.days')}`
        }))
    );

    async function openReviewLog() {
        if (!selectedDeck) return;
        try {
            reviewLog = (await GetAnkiReviewLog(selectedDeck.id, REVIEW_LOG_LIMIT)) || [];
            ankiViewModeStore.set('log');
        } catch (e) {
            fail(e);
        }
    }

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
        // A limit of 0 serves nothing, and says so (ADR-0026 rule 3).
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

            // The limit ended the sitting, not an empty queue: its own message
            // (ADR-0026 rule 4). No "keep going": cram serves more.
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

    // Suspend / bury / remove, in a context menu rather than beside the grading
    // buttons, where they would be pressed by mistake at speed.
    /** @type {{x: number, y: number, items: {label: string, onClick: () => void}[]} | null} */
    let cardMenu = $state(null);

    /** @param {MouseEvent} event */
    function openCardMenu(event) {
        if (!reviewCard) return;
        event.preventDefault();
        cardMenu = {
            x: event.clientX,
            y: event.clientY,
            items: [
                { label: $t('anki.suspendCard'), onClick: () => setAside(anki.suspendCard, 'anki.cardSuspended') },
                { label: $t('anki.buryCard'), onClick: () => setAside(anki.buryCard, 'anki.cardBuried') },
                { label: $t('anki.removeCard'), onClick: () => confirmRemoveCard() }
            ]
        };
    }

    // Removing is the only one not undoable from here.
    async function confirmRemoveCard() {
        if (!(await confirmAction($t('anki.removeCardConfirm'), { confirmLabel: $t('common.delete') }))) return;
        await setAside(anki.removeCard, 'anki.cardRemoved');
    }

    /**
     * @param {(card: any, deck: any, opts: any) => Promise<any>} action
     * @param {string} messageKey
     */
    async function setAside(action, messageKey) {
        if (!reviewCard || !selectedDeck) return;
        try {
            const next = await action(reviewCard, selectedDeck, { cram: cramMode });
            statusBarTextStore.set(tMsg(messageKey));
            // Not an answer: the session count does not move.
            if (!next) {
                ankiViewModeStore.set('list');
                ankiPausedSessionStore.set(null);
                await anki.loadDecks();
                if (selectedDeck) await anki.refreshDeckStats(selectedDeck.id);
            }
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
        // Leaving clears the move selection (ADR-0025 rule 5): a set
        // selectedMoveStore freezes j/k browsing app-wide.
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
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="view-header" oncontextmenu={openCardMenu} title={$t('anki.cardMenuHint')}>
            <button class="btn-back" onclick={backToList} title={$t('anki.backToDeckList') + ' (Esc)'}>{@render icon(ICON.back)}</button>
            <span class="view-title">{selectedDeck?.name}</span>
            <span class="review-count">#{reviewSessionCount + 1}</span>
            {#if cramMode}
                <span class="card-state state-cram">{$t('anki.cramBadge')}</span>
            {:else}
                <span class="card-state state-{reviewCard.card.state}">{anki.stateLabel(reviewCard.card.state)}</span>
            {/if}
        </div>

        <!-- The grading strip stays put; the answer scrolls under it (ADR-0025). -->
        <div class="review-body">
            <div class="review-strip">
                <div class="review-position-id">
                    {#if isScoreCard}
                        {$t('anki.scoreQuestion', { a: scoreAways?.[0] ?? 0, b: scoreAways?.[1] ?? 0 })}
                    {:else}
                        {$t('anki.positionNumber', { id: reviewCard.position.id })}
                    {/if}
                </div>
                {#if cramMode}
                    <div class="review-buttons">
                        <button class="btn-rating" onclick={() => submitReview(1)} title={$t('anki.next') + ' (1-4)'}>
                            <span class="rating-label">{$t('anki.next')}</span>
                        </button>
                    </div>
                {:else}
                    <div class="review-buttons">
                        {#each RATING_BUTTONS as [key, rating] (rating)}
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
                    <div class="answer-absent">{isScoreCard ? $t('anki.badScoreKey') : $t('anki.noAnalysis')}</div>
                {:else if answerShown && isScoreCard}
                    <!-- The whole sheet, no fault ticking: Anki schedules a memory (ADR-0042 rule 3). -->
                    <ScoreCard card={scoreSheet} revealed locked />
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
                        {maxCube}
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
                <label for="{settingsId}-retention">{$t('anki.retentionTarget')}</label>
                <input id="{settingsId}-retention" type="number" bind:value={settingsRetention} min="0.7" max="0.99" step="0.01" />
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
            <!-- The change applies review by review, moving no due date (rule 8). -->
            <div class="settings-note">{$t('anki.retentionNotRetroactive')}</div>
            <div class="settings-row">
                <label for="{settingsId}-max-interval">{$t('anki.maxInterval')}</label>
                <input id="{settingsId}-max-interval" type="number" bind:value={settingsMaxInterval} min="1" max="36500" step="1" />
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
            <div class="settings-row">
                <button class="btn-outline wide" onclick={openReviewLog}>{$t('anki.openReviewLog')}</button>
            </div>
            <div class="settings-note">{$t('anki.reviewLogHint')}</div>
        </div>
    {:else if viewMode === 'log' && selectedDeck}
        <!-- Review log: a reading of what was answered, never a control. -->
        <div class="view-header">
            <button class="btn-back" onclick={() => ankiViewModeStore.set('settings')} title={$t('common.back')}>{@render icon(ICON.back)}</button>
            <span class="view-title">{$t('anki.reviewLogTitle', { name: selectedDeck.name })}</span>
        </div>
        {#if reviewLog.length === 0}
            <div class="settings-note">{$t('anki.reviewLogEmpty')}</div>
        {:else}
            <PanelTable rows={reviewLogRows} columns={reviewLogColumns}>
                {#snippet cells(/** @type {any} */ entry)}
                    <td>{entry.reviewedAt}</td>
                    <td class="num">{entry.subject}</td>
                    <td class="num">{entry.rating}</td>
                    <td class="num">{entry.state}</td>
                    <td class="num">{entry.scheduledDays}</td>
                {/snippet}
            </PanelTable>
            <div class="settings-note">{$t('anki.reviewLogCount', { n: reviewLog.length, limit: REVIEW_LOG_LIMIT })}</div>
        {/if}
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
                        <option value={anki.SOURCE_SCORES}>{$t('anki.sourceScores')}</option>
                    </select>
                    {#if newDeckSourceType === 'collection'}
                        <select bind:value={newDeckSourceId} class="input-source">
                            <option value={0}>{$t('anki.selectCollection')}</option>
                            {#each collections as coll (coll.id)}
                                <option value={coll.id}>{coll.name} ({coll.positionCount})</option>
                            {/each}
                        </select>
                    {:else if newDeckSourceType === anki.SOURCE_SCORES}
                        <!-- Fixed: the 36 unordered scores of 2 to 9 away (ADR-0042 rule 2). -->
                        <span class="search-hint">{$t('anki.scoresCount', { count: UNORDERED_SCORES.length })}</span>
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

    {#if cardMenu}
        <ContextMenu x={cardMenu.x} y={cardMenu.y} items={cardMenu.items} onClose={() => (cardMenu = null)} />
    {/if}
</div>

<style>
    .anki-panel {
        display: flex;
        flex-direction: column;
        height: 100%;
        font-size: var(--font-size-base);
        overflow: hidden;
        background: var(--color-surface);
        user-select: none;
        -webkit-user-select: none;
    }
    .anki-panel input {
        user-select: text;
        -webkit-user-select: text;
    }

    /* --- Buttons: one outlined family; the primary one carries the accent border, as in the other panels --- */
    .btn-outline {
        display: flex;
        align-items: center;
        gap: 4px;
        padding: 3px 8px;
        border: 1px solid var(--color-border);
        border-radius: 3px;
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
        font-size: var(--font-size-small);
    }
    .btn-outline:hover {
        background: var(--color-surface-alt);
    }

    .btn-primary {
        display: flex;
        align-items: center;
        gap: 4px;
        padding: 4px 12px;
        border: 1px solid var(--color-primary);
        border-radius: 3px;
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
        font-size: var(--font-size-base);
    }
    .btn-primary:hover:not(:disabled) {
        background: var(--color-surface-alt);
    }
    .btn-primary:disabled {
        opacity: 0.5;
        cursor: not-allowed;
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
        border-bottom: 1px solid var(--color-border);
        background: var(--color-surface-alt);
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
        border-top: 1px solid var(--color-border);
        padding: 6px 8px;
        background: var(--color-surface-alt);
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
        background: var(--color-surface);
        border: 1px solid var(--color-border);
    }

    .stat-number {
        font-size: var(--font-size-base);
        font-weight: 600;
        color: var(--color-text);
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
        background: var(--color-surface);
        color: #17a2b8;
        cursor: pointer;
        font-size: var(--font-size-base);
        justify-content: center;
    }
    .btn-cram:hover {
        background: color-mix(in srgb, #17a2b8 10%, var(--color-surface));
    }
    .btn-cram:disabled {
        border-color: var(--color-border);
        color: var(--color-border);
        cursor: default;
    }

    /* --- Review and settings views --- */
    .view-header {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 5px 8px;
        background: var(--color-surface-alt);
        border-bottom: 1px solid var(--color-border);
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
        /* Query container: the answer lays out on the panel's own width. */
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

    /* The only part that scrolls, both axes (never clip revealed columns);
       top-aligned so the answer stays near the grading buttons. */
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

    /* One opaque stand-in (ADR-0025 rule 3): masking rows in place would still
       reveal the best move by its position. */
    .answer-masked {
        width: 100%;
        max-width: 320px;
        padding: 14px 0;
        border: 1px dashed var(--color-border);
        border-radius: 3px;
        background: var(--color-surface-alt);
        color: var(--color-text-muted);
        letter-spacing: 3px;
        cursor: pointer;
    }
    .answer-masked:hover {
        background: color-mix(in srgb, var(--color-text) 6%, var(--color-surface-alt));
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
        border: 1px solid var(--color-border);
        border-radius: 3px;
        cursor: pointer;
        background: var(--color-surface);
        color: var(--color-text);
        gap: 2px;
    }
    .btn-rating:hover {
        background: var(--color-surface-alt);
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
