// Spaced-repetition decks: everything AnkiPanel does that is not drawing —
// deck sources, (re)sync, walking review and cram sessions, and the stores the
// rest of the app reads. The panel keeps UI state and status messages.
// Pure helpers first, then the functions calling Wails and writing stores.

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
    GetLinkedAnkiCard,
    GetRandomAnkiCard,
    ReviewAnkiCard,
    SetAnkiCardSuspended,
    BuryAnkiCard,
    RemoveAnkiCard,
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
 * What a search-backed deck stored as its source: a JSON document
 * `{ command, position, ids }`, or (legacy) a comma-separated id list read as
 * ids only.
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
 * The source document of a deck created from the current search. The ids
 * matched at creation are kept so a card survives the search no longer
 * matching its position.
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
    if (deck.sourceType === SOURCE_SCORES) return 'Score sheets';
    if (!deck.sourceCommand) return 'Search';
    return parseSourceCommand(deck.sourceCommand).command ?? 'Search';
}

/**
 * A deck of score sheets (ADR-0042 rule 2), filled with the 36 unordered
 * scores of 2 to 9 away. Its cards hold no position, hence the checks on a
 * card's KIND below.
 */
export const SOURCE_SCORES = 'scores';

/** A card that asks about a score rather than about a position. */
export function isScoreCard(card) {
    return card?.card?.kind === 'score';
}

/**
 * The two aways of a score card, smaller first, or null. Read from the key:
 * the deck states its scores, the view only renders.
 */
export function scoreCardAways(card) {
    const key = card?.card?.key ?? '';
    const [a, b] = key.split(':').map((n) => Number.parseInt(n, 10));
    if (!Number.isFinite(a) || !Number.isFinite(b)) return null;
    return a <= b ? [a, b] : [b, a];
}

/**
 * Where a paused FSRS session on the same deck left off, else zero. Cram never
 * resumes.
 */
export function resumedSessionCount(pausedSession, deck, cram = false) {
    if (cram || !pausedSession || !deck || pausedSession.deckId !== deck.id) return 0;
    return pausedSession.sessionCount;
}

/**
 * Whether a study session can serve anything: due cards, and a session limit
 * other than 0 (ADR-0026 rule 3) — otherwise the button is inactive. A null
 * limit is no limit.
 */
export function canStudy(stats, deck) {
    if (!stats || stats.dueCount <= 0) return false;
    return sessionLimitOf(deck) !== 0;
}

/**
 * The session limit, or null for none. `0` is a limit, hence no `||`.
 */
export function sessionLimitOf(deck) {
    const v = deck?.sessionLimit;
    return v === null || v === undefined ? null : v;
}

/**
 * Whether `count` served cards reach the limit. Cram is never bounded
 * (ADR-0026 rule 2): it schedules nothing.
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
 * Re-run a search deck's stored search: the current results plus every stored
 * id. On error, an empty list, which `syncDeckCards` reads as "leave it alone".
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
        // Ids only: this only ever reads .id off the results, so
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
    // A score deck names no source of its own: syncDeckCards below asks the
    // backend to state its 36 cards.
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
 * Add one position's card to a deck. SyncWithPositions is `INSERT OR IGNORE`,
 * so this is safe on any sourceType and never touches the other cards.
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
 * Put a card's position on the board and point the status bar at it. Through
 * showPosition, the only path that also loads analysis and comment: set by
 * hand, the Analysis tab would show the previously browsed position's numbers
 * during review (ADR-0025 rule 1).
 */
export async function showCard(card) {
    // A score card has no position: leave the board as is; the review view
    // renders the score sheet.
    if (isScoreCard(card)) return;
    await showPosition(card.position);
    const idx = positionsStore.indexOf(card.position.id);
    if (idx >= 0) currentPositionIndexStore.set(idx);
}

/**
 * A new question: hide the answer and drop the previously picked move
 * (ADR-0025 rule 5). Not in showCard, which reruns whenever the Anki tab
 * regains focus. A leftover selectedMoveStore freezes j/k browsing app-wide.
 */
function newQuestion() {
    hideAnkiAnswer();
    selectedMoveStore.set(null);
}

/**
 * Start a session: resync the deck, load its positions, draw and show the
 * first card (next due, or random when cramming).
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
 * Grade the current card and move to the next one.
 *
 * Une décision de videau est deux positions (« double ? », « prend ? ») : si
 * la seconde est due dans le même paquet, elle suit immédiatement (ADR-0025).
 * Le chaînage ordonne les cartes dues sans avancer aucune échéance, pour ne
 * pas fausser FSRS. Pas de chaînage en cram (tirage aléatoire).
 *
 * @returns {Promise<object | null>} the next card, or null when the session is over
 */
export async function reviewCard(card, rating, { cram = false } = {}) {
    const next = await ReviewAnkiCard(card.card.id, rating);
    if (!cram) {
        let linked = null;
        try {
            linked = await GetLinkedAnkiCard(card.card.deckId, card.card.id);
        } catch (error) {
            // Le chaînage est un confort, pas le contrat de la révision : une
            // panne ici ne doit pas interrompre une session.
            logger.error('could not look up the linked card:', error);
        }
        if (linked) return await advance(linked);
    }
    return await advance(next);
}

/**
 * Suspend, bury or remove: take a card out of the session without grading it —
 * the scheduler is told nothing. Cram advances with another random card,
 * never the one just seen.
 *
 * @param {object} card the card under review
 * @param {object} deck the deck being reviewed
 * @param {boolean} cram whether this is a cram session
 */
async function setAsideAndAdvance(card, deck, cram) {
    const next = cram ? await GetRandomAnkiCard(deck.id, isScoreCard(card) ? 0 : card.position.id) : await GetNextAnkiCard(deck.id);
    return await advance(next);
}

/**
 * Suspend the current card: it keeps its schedule and never comes up.
 *
 * @param {any} card
 * @param {any} deck
 * @param {{cram?: boolean}} [opts]
 */
export async function suspendCard(card, deck, { cram = false } = {}) {
    await SetAnkiCardSuspended(card.card.id, true);
    return await setAsideAndAdvance(card, deck, cram);
}

/**
 * Bury the current card until the start of the next day.
 *
 * @param {any} card
 * @param {any} deck
 * @param {{cram?: boolean}} [opts]
 */
export async function buryCard(card, deck, { cram = false } = {}) {
    await BuryAnkiCard(card.card.id);
    return await setAsideAndAdvance(card, deck, cram);
}

/**
 * Remove the current card from its deck. The position itself is untouched.
 *
 * @param {any} card
 * @param {any} deck
 * @param {{cram?: boolean}} [opts]
 */
export async function removeCard(card, deck, { cram = false } = {}) {
    await RemoveAnkiCard(card.card.id);
    return await setAsideAndAdvance(card, deck, cram);
}

/** Draw the next random card of a cram session, never the one just shown. */
export async function nextCramCard(deck, card) {
    // The exclusion names the POSITION just served, so a card that has none
    // excludes nothing — a score deck simply draws freely from its 36 cards.
    const next = await GetRandomAnkiCard(deck.id, isScoreCard(card) ? 0 : card.position.id);
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
 * Minimum review-state reviews before a deck's retention is reported; below
 * it, callers show the absence rather than a noisy rate. Retention is a
 * reading, never a control (ADR-0026 rule 5).
 */
export const RETENTION_MIN_SAMPLE = 20;

export async function deckRetention(deckId) {
    return await GetAnkiDeckRetention(deckId);
}
