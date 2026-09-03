// Spaced-repetition decks: everything AnkiPanel does that is not drawing —
// what a deck's stored search means, how a deck is (re)synced with its
// source, how a review or cram session walks its cards, and the bookkeeping
// of the stores the rest of the app reads (the board shows the card, the
// status bar counts positions). The panel keeps the UI state (form fields,
// session counter, cram flag) and the status-bar messages; it calls here and
// catches.
//
// Pure helpers first (no I/O, unit-tested as such), then the functions that
// call the Wails bindings and write the stores.

import { get } from 'svelte/store';
import {
    CreateAnkiDeck,
    GetAllAnkiDecks,
    UpdateAnkiDeckParams,
    DeleteAnkiDeck,
    SyncAnkiDeck,
    SyncAnkiDeckWithPositions,
    GetAnkiDeckStats,
    GetAnkiDeckRetention,
    GetAnkiDeckPositions,
    GetNextAnkiCard,
    GetRandomAnkiCard,
    ReviewAnkiCard,
    ResetAnkiDeck,
    GetAllCollections,
    LoadPositionIDsByFilters
} from '../../wailsjs/go/database/Database.js';
import { ankiDecksStore, selectedAnkiDeckStore, ankiReviewCardStore, ankiDeckStatsStore, ankiViewModeStore, hideAnkiAnswer } from '../stores/ankiStore.js';
import { collectionsStore } from '../stores/collectionStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { currentPositionIndexStore } from '../stores/uiStore.js';
import { showPosition } from './positionService.js';
import { parseFilters } from '../commandProcessor.js';
import { buildSearchFilterPayload } from './searchFilterService.js';
import { logger } from '../utils/logger.js';

// ---------------------------------------------------------------------------
// Pure helpers
// ---------------------------------------------------------------------------

/**
 * What a search-backed deck stored as its source. Current decks store a JSON
 * document `{ command, position, ids }`; the first ones stored a plain
 * comma-separated list of position ids, which is read as ids only.
 *
 * @param {string} sourceCommand
 * @returns {{ ids: number[], command: string | null, position: object }}
 */
export function parseSourceCommand(sourceCommand) {
    let data;
    try {
        data = JSON.parse(sourceCommand);
    } catch {
        return { ids: parseLegacyIds(sourceCommand), command: null, position: {} };
    }
    if (!data || typeof data !== 'object') return { ids: [], command: null, position: {} };
    let position = {};
    if (data.position) {
        try {
            position = typeof data.position === 'string' ? JSON.parse(data.position) : data.position;
        } catch {
            position = {};
        }
    }
    return {
        ids: Array.isArray(data.ids) ? data.ids : [],
        command: typeof data.command === 'string' && data.command ? data.command : null,
        position
    };
}

function parseLegacyIds(text) {
    return String(text ?? '')
        .split(',')
        .map((s) => parseInt(s.trim(), 10))
        .filter((n) => !isNaN(n));
}

/**
 * The source document of a deck created from the current search: the command
 * and board that produced it, plus the ids it matched at creation time so a
 * card is never lost when the search stops matching its position.
 *
 * @param {{ command?: string, position?: string } | null} lastSearch
 * @param {number[]} positionIds
 * @returns {string}
 */
export function buildSearchSource(lastSearch, positionIds) {
    if (lastSearch && lastSearch.command) {
        return JSON.stringify({ command: lastSearch.command, position: lastSearch.position, ids: positionIds });
    }
    return JSON.stringify({ ids: positionIds });
}

/** Search results plus every stored id not among them, in that order, without duplicates. */
export function mergeIds(searchIds, storedIds) {
    const seen = new Set();
    const out = [];
    for (const id of [...searchIds, ...storedIds]) {
        if (seen.has(id)) continue;
        seen.add(id);
        out.push(id);
    }
    return out;
}

/** The FSRS card state as a word. */
export function stateLabel(state) {
    switch (state) {
        case 0:
            return 'New';
        case 1:
            return 'Learning';
        case 2:
            return 'Review';
        case 3:
            return 'Relearning';
        default:
            return '?';
    }
}

/** What a deck draws its cards from, as shown in the deck list. */
export function sourceLabel(deck, collections = []) {
    if (deck.sourceType === 'collection') {
        const coll = collections.find((c) => c.id === deck.sourceId);
        return coll ? coll.name : `Collection #${deck.sourceId}`;
    }
    if (!deck.sourceCommand) return 'Search';
    return parseSourceCommand(deck.sourceCommand).command ?? 'Search';
}

/**
 * The count a session starts at: where a paused FSRS session on the same deck
 * left off, otherwise zero. Cram sessions never resume.
 */
export function resumedSessionCount(pausedSession, deck, cram = false) {
    if (cram || !pausedSession || !deck || pausedSession.deckId !== deck.id) return 0;
    return pausedSession.sessionCount;
}

/** A study session needs due cards; a cram session only needs cards. */
/**
 * Whether a study session can serve anything right now.
 *
 * A deck whose session limit is 0 has cards due and serves none of them
 * (ADR-0026 rule 3), so the button must be inactive: without this, the user
 * clicks and nothing happens, and discovers the limit by bumping into it.
 * A null/undefined limit is no limit.
 */
export function canStudy(stats, deck) {
    if (!stats || stats.dueCount <= 0) return false;
    return sessionLimitOf(deck) !== 0;
}

/**
 * A deck's session limit as a number, or null when the deck has none.
 * `0` is a limit — it is not "no limit" — so this must not use `||`.
 */
export function sessionLimitOf(deck) {
    const v = deck?.sessionLimit;
    return v === null || v === undefined ? null : v;
}

/**
 * Whether a session that has already served `count` cards has reached the
 * deck's limit. Cram is never bounded by it (ADR-0026 rule 2): free drill
 * schedules nothing, so there is nothing to pace.
 */
export function sessionLimitReached(deck, count, { cram = false } = {}) {
    if (cram) return false;
    const limit = sessionLimitOf(deck);
    return limit !== null && count >= limit;
}

export function canCram(stats) {
    return !!stats && stats.totalCount > 0;
}

// ---------------------------------------------------------------------------
// Wails-backed operations
// ---------------------------------------------------------------------------

/** Reload the decks (and the collections the create form offers) into their stores. */
export async function loadDecks() {
    const decks = (await GetAllAnkiDecks()) || [];
    ankiDecksStore.set(decks);
    const colls = (await GetAllCollections()) || [];
    collectionsStore.set(colls);
    return decks;
}

/**
 * Re-run a search-backed deck's stored search and return the position ids
 * its cards should cover: the current results plus every id stored with the
 * deck. Errors are logged and yield an empty list, which `syncDeckCards`
 * treats as "leave the deck alone".
 */
export async function resolveSearchDeckIds(sourceCommand) {
    try {
        const { ids: storedIds, command, position } = parseSourceCommand(sourceCommand);
        if (!command) return storedIds;

        let payload;
        if (command === 's') {
            // Bare `s`: the board structure is the whole filter.
            payload = buildSearchFilterPayload(position);
        } else {
            const filters = command
                .slice(1)
                .trim()
                .split(' ')
                .map((f) => f.trim());
            payload = buildSearchFilterPayload(position, parseFilters(filters, command), filters);
        }
        // Ids only (D.8, #208): this only ever reads .id off the results, so
        // there is no reason to ship every matching position whole.
        const ids = await LoadPositionIDsByFilters(payload);
        return mergeIds(ids || [], storedIds);
    } catch (e) {
        logger.error('Error executing search for deck sync:', e);
        return [];
    }
}

/** Bring a deck's cards in line with its source (collection or stored search). */
export async function syncDeckCards(deck) {
    if (deck.sourceType === 'search' && deck.sourceCommand) {
        const ids = await resolveSearchDeckIds(deck.sourceCommand);
        if (ids.length > 0) await SyncAnkiDeckWithPositions(deck.id, ids);
    } else {
        await SyncAnkiDeck(deck.id);
    }
}

/** Sync every deck (each failure logged, the others still synced) and reload the list. */
export async function syncAllDecksAndReload() {
    try {
        const decks = (await GetAllAnkiDecks()) || [];
        for (const deck of decks) {
            try {
                await syncDeckCards(deck);
            } catch (e) {
                logger.error(`Error syncing deck "${deck.name}":`, e);
            }
        }
        await loadDecks();
    } catch (e) {
        logger.error('Error auto-syncing decks:', e);
    }
}

/**
 * Create a deck and fill it from its source.
 * @returns {Promise<number>} the new deck's id
 */
export async function createDeck({ name, sourceType, sourceId, lastSearch = null, positionIds = [] }) {
    const search = sourceType === 'search';
    const sourceCommand = search ? buildSearchSource(lastSearch, positionIds) : '';
    const deckId = await CreateAnkiDeck(name, '', sourceType, search ? 0 : sourceId, sourceCommand);
    await syncDeckCards({ id: deckId, sourceType, sourceCommand });
    await loadDecks();
    return deckId;
}

/** Delete a deck; if it was the selected one, leave the detail and review views. */
export async function deleteDeck(deckId) {
    await DeleteAnkiDeck(deckId);
    if (get(selectedAnkiDeckStore)?.id === deckId) clearSelection();
    await loadDecks();
}

export function clearSelection() {
    selectedAnkiDeckStore.set(null);
    ankiReviewCardStore.set(null);
    ankiDeckStatsStore.set(null);
    ankiViewModeStore.set('list');
}

export async function refreshDeckStats(deckId) {
    const stats = await GetAnkiDeckStats(deckId);
    ankiDeckStatsStore.set(stats);
    return stats;
}

/**
 * Add one position's card to a deck (the board's context menu, #215) — every
 * other path grows a deck through its bound source (search/collection) via
 * syncDeckCards. SyncWithPositions is `INSERT OR IGNORE`, never a removal, so
 * this is safe to call on a deck of any sourceType: it only ever adds the one
 * card, never touches the rest of the deck.
 */
export async function addPositionToDeck(deckId, positionId) {
    await SyncAnkiDeckWithPositions(deckId, [positionId]);
    // Refresh the visible stats only if this is the deck currently open in
    // the Anki panel — otherwise there is nothing on screen to update, and
    // stomping a different deck's stats with these would be a bug.
    if (get(selectedAnkiDeckStore)?.id === deckId) {
        await refreshDeckStats(deckId);
    }
}

/** Select a deck: its stats, and its positions as the list the status bar counts. */
export async function selectDeck(deck) {
    selectedAnkiDeckStore.set(deck);
    await refreshDeckStats(deck.id);
    const deckPositions = (await GetAnkiDeckPositions(deck.id)) || [];
    positionsStore.set(deckPositions);
    if (deckPositions.length > 0) currentPositionIndexStore.set(0);
}

/** Wipe a deck's schedule; refresh its stats if it is the selected one. */
export async function resetDeck(deckId) {
    await ResetAnkiDeck(deckId);
    await loadDecks();
    if (get(selectedAnkiDeckStore)?.id === deckId) await refreshDeckStats(deckId);
}

/** Save FSRS parameters and refresh the selected deck from the reloaded list. */
export async function saveDeckParams(deckId, { requestRetention, maximumInterval, enableFuzz, sessionLimit = null }) {
    await UpdateAnkiDeckParams(deckId, requestRetention, maximumInterval, enableFuzz, sessionLimit);
    const decks = await loadDecks();
    const updated = decks.find((d) => d.id === deckId);
    if (updated && get(selectedAnkiDeckStore)?.id === deckId) selectedAnkiDeckStore.set(updated);
}

/**
 * Put a card's position on the board and point the status bar at it.
 *
 * Goes through showPosition rather than setting positionStore directly: it is
 * the one function that knows what "display a position" means, and the only
 * one that loads the position's analysis and comment. Setting the store by
 * hand left analysisStore holding whatever position was browsed last, so the
 * Analysis tab — one Ctrl+L away, the review key guard lets Ctrl combos
 * through — showed another position's numbers during a review, and the
 * revealed answer of ADR-0025 rule 1 would have inherited that lie.
 */
export async function showCard(card) {
    await showPosition(card.position);
    const idx = positionsStore.indexOf(card.position.id);
    if (idx >= 0) currentPositionIndexStore.set(idx);
}

/**
 * A new question: hide its answer and drop the move picked on the previous one
 * (ADR-0025 rule 5). Deliberately NOT inside showCard — the panel re-shows the
 * current card every time the Anki tab regains focus, and going to look at the
 * Eval panel is not a new question.
 *
 * Left set, selectedMoveStore freezes j/k position browsing app-wide.
 */
function newQuestion() {
    hideAnkiAnswer();
    selectedMoveStore.set(null);
}

/**
 * Start a session on a deck: resync it, load its positions, draw the first
 * card (the next due one, or a random one when cramming) and show it.
 * @returns {Promise<object | null>} the first card, or null when there is none
 */
export async function startSession(deck, { cram = false } = {}) {
    await syncDeckCards(deck);
    positionsStore.set((await GetAnkiDeckPositions(deck.id)) || []);
    const card = cram ? await GetRandomAnkiCard(deck.id, 0) : await GetNextAnkiCard(deck.id);
    if (!card) return null;
    ankiReviewCardStore.set(card);
    newQuestion();
    await showCard(card);
    return card;
}

/**
 * Grade the current card and move to the next due one.
 * @returns {Promise<object | null>} the next card, or null when the session is over
 */
export async function reviewCard(card, rating) {
    const next = await ReviewAnkiCard(card.card.id, rating);
    return await advance(next);
}

/** Draw the next random card of a cram session, never the one just shown. */
export async function nextCramCard(deck, card) {
    const next = await GetRandomAnkiCard(deck.id, card.position.id);
    return await advance(next);
}

async function advance(next) {
    if (next) {
        ankiReviewCardStore.set(next);
        newQuestion();
        await showCard(next);
    } else {
        ankiReviewCardStore.set(null);
    }
    return next ?? null;
}

/**
 * What a deck's review log measures, against the target its owner chose.
 *
 * A reading, never a control (ADR-0026 rule 5): nothing here writes the target
 * back. Below RETENTION_MIN_SAMPLE review-state reviews the measurement is not
 * reported at all — a pass rate over three reviews reads as a fact while being
 * noise — so callers show the absence rather than a number.
 */
export const RETENTION_MIN_SAMPLE = 20;

export async function deckRetention(deckId) {
    return await GetAnkiDeckRetention(deckId);
}
